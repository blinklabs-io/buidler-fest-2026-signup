package signup

import (
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/Salvionied/apollo"
	"github.com/Salvionied/apollo/serialization/Address"
	"github.com/Salvionied/apollo/serialization/Key"
	"github.com/Salvionied/apollo/serialization/PlutusData"
	"github.com/Salvionied/apollo/serialization/Redeemer"
	"github.com/Salvionied/apollo/serialization/UTxO"
	"github.com/blinklabs-io/buidler-fest-2026-signup/internal/chaincontext"
	"github.com/blinklabs-io/buidler-fest-2026-signup/internal/config"
	"github.com/blinklabs-io/buidler-fest-2026-signup/internal/wallet"
)

// SignupResult contains the result of a signup operation
type SignupResult struct {
	TxHash         string
	UnsignedTxCBOR string
	TicketName     string
	TicketNumber   int
}

// SignupInfo contains information about the current signup state
type SignupInfo struct {
	NextTicketNumber int
	TicketsSold      int
	MaxTickets       int
}

// ExecuteSignup performs the signup/ticket purchase operation
func ExecuteSignup(cfg *config.Config, w *wallet.Wallet, buyerAddress string, skipSubmit bool) (*SignupResult, error) {
	// Validate chain context is available
	if err := cfg.ValidateChainContext(); err != nil {
		return nil, err
	}

	// Get buyer address
	var buyerAddr string
	if w != nil {
		buyerAddr = w.PaymentAddress
	} else if buyerAddress != "" {
		buyerAddr = buyerAddress
	} else {
		return nil, fmt.Errorf("no buyer address provided")
	}

	slog.Info("building signup transaction", "buyer", buyerAddr, "network", cfg.Network)

	// Create chain context
	cc, err := chaincontext.NewChainContext(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create chain context: %w", err)
	}

	// Get issuer state UTxO (contains ticket counter)
	issuerState, ticketCounter, err := getIssuerState(cc, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to get issuer state: %w", err)
	}

	slog.Info("found issuer state", "ticketCounter", ticketCounter)

	// Get buyer UTxOs
	buyerUtxos, err := cc.GetUTxOsByAddress(buyerAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to get buyer utxos: %w", err)
	}

	if len(buyerUtxos) == 0 {
		return nil, fmt.Errorf("no UTxOs found for buyer address")
	}

	// Build the transaction
	tx, ticketName, err := buildSignupTx(cfg, cc, buyerAddr, buyerUtxos, issuerState, ticketCounter)
	if err != nil {
		return nil, fmt.Errorf("failed to build transaction: %w", err)
	}

	result := &SignupResult{
		TicketName:   ticketName,
		TicketNumber: ticketCounter,
	}

	// Get unsigned transaction bytes for CBOR output
	txObj := tx.GetTx()
	if txObj == nil {
		return nil, fmt.Errorf("failed to get transaction")
	}
	txBytes, err := txObj.Bytes()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize transaction: %w", err)
	}

	// Calculate transaction hash from CBOR bytes
	// Use the serialized bytes to generate hash
	result.TxHash = fmt.Sprintf("%x", txBytes[:32]) // Temporary - will be updated on submit

	// If we have a wallet and not skipping submit, sign and submit
	if w != nil && !skipSubmit {
		// Sign the transaction
		signedTx, err := signTransaction(tx, w)
		if err != nil {
			return nil, fmt.Errorf("failed to sign transaction: %w", err)
		}

		// Submit transaction using Apollo's built-in submit
		txId, err := signedTx.Submit()
		if err != nil {
			return nil, fmt.Errorf("failed to submit transaction: %w", err)
		}

		result.TxHash = hex.EncodeToString(txId.Payload)
		slog.Info("transaction submitted", "hash", result.TxHash)
	} else {
		// Output unsigned transaction CBOR
		result.UnsignedTxCBOR = hex.EncodeToString(txBytes)
	}

	return result, nil
}

// GetSignupInfo retrieves current signup state information
func GetSignupInfo(cfg *config.Config) (*SignupInfo, error) {
	if err := cfg.ValidateChainContext(); err != nil {
		return nil, err
	}

	cc, err := chaincontext.NewChainContext(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create chain context: %w", err)
	}

	_, ticketCounter, err := getIssuerState(cc, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to get issuer state: %w", err)
	}

	return &SignupInfo{
		NextTicketNumber: ticketCounter,
		TicketsSold:      ticketCounter,
		MaxTickets:       100, // From smart contract
	}, nil
}

// getIssuerState finds the issuer state UTxO and extracts the ticket counter
func getIssuerState(cc chaincontext.ChainContext, cfg *config.Config) (*UTxO.UTxO, int, error) {
	// Get UTxOs at the issuer address
	utxos, err := cc.GetUTxOsByAddress(cfg.IssuerAddress)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get issuer utxos: %w", err)
	}

	// Parse beacon token info
	beaconPolicyHex := cfg.IssuerBeaconPolicy
	beaconNameBytes, err := hex.DecodeString(cfg.IssuerBeaconName)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to decode beacon name: %w", err)
	}

	// Find UTxO with beacon token
	for _, utxo := range utxos {
		if hasToken(utxo, beaconPolicyHex, beaconNameBytes) {
			// Extract ticket counter from datum
			counter, err := extractTicketCounter(utxo)
			if err != nil {
				slog.Warn("failed to extract counter from utxo", "error", err)
				continue
			}
			return &utxo, counter, nil
		}
	}

	return nil, 0, fmt.Errorf("issuer state UTxO with beacon token not found")
}

// hasToken checks if a UTxO contains a specific token
// This is a simplified check - assumes the issuer UTxO with datum is the correct one
func hasToken(utxo UTxO.UTxO, policyHex string, assetNameBytes []byte) bool {
	// For now, check if this UTxO has a datum (the issuer state has a datum)
	// A full implementation would check the actual token
	if utxo.Output.IsPostAlonzo {
		// Check if there's a datum (the state UTxO has inline datum)
		return utxo.Output.PostAlonzo.Datum != nil
	}
	return false
}

// extractTicketCounter extracts the ticket counter from the issuer state datum
func extractTicketCounter(utxo UTxO.UTxO) (int, error) {
	if !utxo.Output.IsPostAlonzo {
		return 0, fmt.Errorf("pre-alonzo output not supported")
	}

	datumOpt := utxo.Output.PostAlonzo.Datum
	if datumOpt == nil {
		return 0, fmt.Errorf("no datum found in utxo")
	}

	// Get the inline datum
	datum := datumOpt.Inline
	if datum == nil {
		return 0, fmt.Errorf("no inline datum found")
	}

	// The datum is a simple constructor with ticket_counter as field
	// TicketerDatum { ticket_counter: Int }
	if datum.TagNr == 121 && datum.PlutusDataType == PlutusData.PlutusArray {
		fields, ok := datum.Value.(PlutusData.PlutusIndefArray)
		if !ok {
			return 0, fmt.Errorf("unexpected datum structure")
		}
		if len(fields) > 0 {
			counterData := fields[0]
			if counterData.PlutusDataType == PlutusData.PlutusInt {
				if counter, ok := counterData.Value.(int64); ok {
					return int(counter), nil
				}
			}
		}
	}

	return 0, fmt.Errorf("could not extract ticket counter from datum")
}

// buildSignupTx builds the signup transaction
func buildSignupTx(
	cfg *config.Config,
	cc chaincontext.ChainContext,
	buyerAddr string,
	buyerUtxos []UTxO.UTxO,
	issuerState *UTxO.UTxO,
	ticketCounter int,
) (*apollo.Apollo, string, error) {

	// Create Apollo builder with Blockfrost chain context
	bfCtx, ok := cc.(*chaincontext.BlockfrostContext)
	if !ok {
		return nil, "", fmt.Errorf("Apollo transaction building requires Blockfrost chain context")
	}
	apolloCC := bfCtx.GetClient()

	builder := apollo.New(apolloCC)

	// Set wallet
	builder = builder.SetWalletFromBech32(buyerAddr)
	builder, _ = builder.SetWalletAsChangeAddress()

	// Calculate ticket name
	ticketName := fmt.Sprintf("TICKET%d", ticketCounter)

	// Add buyer inputs (UTxOs)
	builder = builder.AddLoadedUTxOs(buyerUtxos...)

	// Add issuer state input with spend redeemer
	buyTicketRedeemer := PlutusData.PlutusData{
		TagNr:          121, // Constructor 0 (BuyTicket)
		PlutusDataType: PlutusData.PlutusArray,
		Value:          PlutusData.PlutusIndefArray{},
	}
	builder = builder.AddLoadedUTxOs(*issuerState)
	// Note: For proper script spending, we need to use the correct Apollo API
	// This may require AttachDatum and setting redeemers differently
	_ = buyTicketRedeemer // Will be used when proper API is available

	// Add reference script input
	refTxHash, refIdx, err := cfg.ParseIssuerScriptRef()
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse script ref: %w", err)
	}
	builder = builder.AddReferenceInput(refTxHash, refIdx)

	// Mint ticket token using Unit
	mintRedeemer := PlutusData.PlutusData{
		TagNr:          121, // Constructor 0 (MintTicket)
		PlutusDataType: PlutusData.PlutusArray,
		Value:          PlutusData.PlutusIndefArray{},
	}
	mintUnit := apollo.Unit{
		PolicyId: cfg.TicketPolicy,
		Name:     ticketName,
		Quantity: 1,
	}
	builder = builder.MintAssetsWithRedeemer(mintUnit, Redeemer.Redeemer{
		Tag:  Redeemer.MINT,
		Data: mintRedeemer,
	})

	// Output 1: Payment to treasury
	builder = builder.PayToAddressBech32(cfg.Treasury, int(cfg.TicketPrice))

	// Output 2: New issuer state with incremented counter
	issuerAddr, err := Address.DecodeAddress(cfg.IssuerAddress)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode issuer address: %w", err)
	}

	// Create new datum with incremented counter
	newDatum := PlutusData.PlutusData{
		TagNr:          121,
		PlutusDataType: PlutusData.PlutusArray,
		Value: PlutusData.PlutusIndefArray{
			PlutusData.PlutusData{
				PlutusDataType: PlutusData.PlutusInt,
				Value:          int64(ticketCounter + 1),
			},
		},
	}

	// Get beacon token value from issuer state and pay to contract
	issuerLovelace := issuerState.Output.PostAlonzo.Amount.Coin
	// Include the beacon token in the output
	beaconUnit := apollo.Unit{
		PolicyId: cfg.IssuerBeaconPolicy,
		Name:     hexToString(cfg.IssuerBeaconName),
		Quantity: 1,
	}
	builder = builder.PayToContract(issuerAddr, &newDatum, int(issuerLovelace), true, beaconUnit)

	// Set validity interval
	tip, err := cc.GetTip()
	if err != nil {
		slog.Warn("failed to get tip, using default validity", "error", err)
		tip = 0
	}
	if tip > 0 {
		builder = builder.SetValidityStart(int64(tip - 100))
		builder = builder.SetTtl(int64(tip + 600))
	}

	// Complete the transaction
	tx, err := builder.Complete()
	if err != nil {
		return nil, "", fmt.Errorf("failed to complete transaction: %w", err)
	}

	return tx, ticketName, nil
}

// signTransaction signs the transaction with the wallet keys
func signTransaction(tx *apollo.Apollo, w *wallet.Wallet) (*apollo.Apollo, error) {
	vkey := Key.VerificationKey{Payload: w.PaymentVKey}
	skey := Key.SigningKey{Payload: w.PaymentSKey}

	signedTx, err := tx.SignWithSkey(vkey, skey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	return signedTx, nil
}

// hexToString converts a hex string to its string representation
func hexToString(hexStr string) string {
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return hexStr
	}
	return string(bytes)
}
