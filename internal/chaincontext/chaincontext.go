package chaincontext

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/Salvionied/apollo/serialization/Address"
	"github.com/Salvionied/apollo/serialization/Amount"
	"github.com/Salvionied/apollo/serialization/Asset"
	"github.com/Salvionied/apollo/serialization/AssetName"
	"github.com/Salvionied/apollo/serialization/MultiAsset"
	"github.com/Salvionied/apollo/serialization/PlutusData"
	"github.com/Salvionied/apollo/serialization/Policy"
	"github.com/Salvionied/apollo/serialization/Transaction"
	"github.com/Salvionied/apollo/serialization/TransactionInput"
	"github.com/Salvionied/apollo/serialization/TransactionOutput"
	"github.com/Salvionied/apollo/serialization/UTxO"
	"github.com/Salvionied/apollo/serialization/Value"
	"github.com/Salvionied/apollo/txBuilding/Backend/Base"
	"github.com/Salvionied/apollo/txBuilding/Backend/BlockFrostChainContext"
	"github.com/SundaeSwap-finance/kugo"
	"github.com/blinklabs-io/buidler-fest-2026-signup/internal/config"
	"github.com/fxamacker/cbor/v2"
)

// ChainContext provides an interface for querying the blockchain
type ChainContext interface {
	GetUTxOsByAddress(address string) ([]UTxO.UTxO, error)
	GetUTxOByRef(txHash string, index int) (*UTxO.UTxO, error)
	GetProtocolParameters() (*Base.ProtocolParameters, error)
	GetTip() (uint64, error)
	SubmitTx(tx Transaction.Transaction) (string, error)
}

// NewChainContext creates a chain context based on configuration
func NewChainContext(cfg *config.Config) (ChainContext, error) {
	switch cfg.GetChainContextType() {
	case "blockfrost":
		return NewBlockfrostContext(cfg)
	case "kupo", "ogmios":
		return NewKupoContext(cfg)
	case "utxorpc":
		return NewUTxORPCContext(cfg)
	default:
		return nil, fmt.Errorf("no chain context configured")
	}
}

// BlockfrostContext implements ChainContext using Blockfrost API
type BlockfrostContext struct {
	client  *BlockFrostChainContext.BlockFrostChainContext
	cfg     *config.Config
	network int
}

// NewBlockfrostContext creates a new Blockfrost chain context
func NewBlockfrostContext(cfg *config.Config) (*BlockfrostContext, error) {
	netCfg, err := config.GetNetworkConfig(cfg.Network)
	if err != nil {
		return nil, err
	}

	var networkId int
	switch cfg.Network {
	case "mainnet":
		networkId = 1
	default:
		networkId = 0
	}

	client, err := BlockFrostChainContext.NewBlockfrostChainContext(
		netCfg.BlockfrostBaseURL,
		networkId,
		cfg.BlockfrostAPIKey,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create blockfrost client: %w", err)
	}

	return &BlockfrostContext{
		client:  &client,
		cfg:     cfg,
		network: networkId,
	}, nil
}

func (b *BlockfrostContext) GetUTxOsByAddress(address string) ([]UTxO.UTxO, error) {
	addr, err := Address.DecodeAddress(address)
	if err != nil {
		return nil, fmt.Errorf("failed to decode address: %w", err)
	}
	return b.client.Utxos(addr)
}

func (b *BlockfrostContext) GetUTxOByRef(txHash string, index int) (*UTxO.UTxO, error) {
	// Blockfrost doesn't have a direct UTxO by ref lookup, so we need to get all UTxOs
	// This is a limitation - in production, you'd want to use a different approach
	return nil, fmt.Errorf("GetUTxOByRef not implemented for Blockfrost")
}

func (b *BlockfrostContext) GetProtocolParameters() (*Base.ProtocolParameters, error) {
	params, err := b.client.GetProtocolParams()
	if err != nil {
		return nil, fmt.Errorf("failed to get protocol params: %w", err)
	}
	return &params, nil
}

func (b *BlockfrostContext) GetTip() (uint64, error) {
	// Get latest block slot from Blockfrost
	tip, err := b.client.LatestBlock()
	if err != nil {
		return 0, fmt.Errorf("failed to get latest block: %w", err)
	}
	return uint64(tip.Slot), nil
}

func (b *BlockfrostContext) SubmitTx(tx Transaction.Transaction) (string, error) {
	hash, err := b.client.SubmitTx(tx)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Payload), nil
}

// GetClient returns the underlying Blockfrost client for use with Apollo
func (b *BlockfrostContext) GetClient() *BlockFrostChainContext.BlockFrostChainContext {
	return b.client
}

// KupoContext implements ChainContext using Kupo API
type KupoContext struct {
	client *kugo.Client
	cfg    *config.Config
}

// NewKupoContext creates a new Kupo chain context
func NewKupoContext(cfg *config.Config) (*KupoContext, error) {
	client := kugo.New(kugo.WithEndpoint(cfg.KupoURL))
	if client == nil {
		return nil, fmt.Errorf("failed to create kupo client")
	}
	return &KupoContext{
		client: client,
		cfg:    cfg,
	}, nil
}

func (k *KupoContext) GetUTxOsByAddress(address string) ([]UTxO.UTxO, error) {
	matches, err := k.client.Matches(context.Background(), kugo.Pattern(address), kugo.OnlyUnspent())
	if err != nil {
		return nil, fmt.Errorf("failed to get matches: %w", err)
	}

	var utxos []UTxO.UTxO
	var conversionErrors int
	for _, match := range matches {
		utxo, err := k.kupoMatchToUTxO(match)
		if err != nil {
			conversionErrors++
			continue
		}
		utxos = append(utxos, utxo)
	}
	if conversionErrors > 0 {
		return utxos, fmt.Errorf("failed to convert %d of %d UTxOs (partial result returned)", conversionErrors, len(matches))
	}
	return utxos, nil
}

func (k *KupoContext) GetUTxOByRef(txHash string, index int) (*UTxO.UTxO, error) {
	pattern := fmt.Sprintf("%s@%d", txHash, index)
	matches, err := k.client.Matches(context.Background(), kugo.OnlyUnspent(), kugo.Pattern(pattern))
	if err != nil {
		return nil, fmt.Errorf("failed to get match: %w", err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("utxo not found: %s#%d", txHash, index)
	}
	utxo, err := k.kupoMatchToUTxO(matches[0])
	if err != nil {
		return nil, err
	}
	return &utxo, nil
}

func (k *KupoContext) GetProtocolParameters() (*Base.ProtocolParameters, error) {
	// Kupo doesn't provide protocol parameters - would need Ogmios for this
	return nil, fmt.Errorf("protocol parameters not available from Kupo")
}

func (k *KupoContext) GetTip() (uint64, error) {
	// Would need Ogmios for tip
	return 0, fmt.Errorf("tip not available from Kupo")
}

func (k *KupoContext) SubmitTx(tx Transaction.Transaction) (string, error) {
	// Would need Ogmios for submission
	return "", fmt.Errorf("tx submission not available from Kupo")
}

// kupoMatchToUTxO converts a Kupo match to an Apollo UTxO
// This handles multi-assets and inline datums required for script UTxOs
func (k *KupoContext) kupoMatchToUTxO(match kugo.Match) (UTxO.UTxO, error) {
	addr, err := Address.DecodeAddress(match.Address)
	if err != nil {
		return UTxO.UTxO{}, fmt.Errorf("failed to decode address: %w", err)
	}

	txIdBytes, err := hex.DecodeString(match.TransactionID)
	if err != nil {
		return UTxO.UTxO{}, fmt.Errorf("failed to decode tx id: %w", err)
	}

	// Extract lovelace and multi-assets from Kupo value
	var totalLovelace int64
	multiAsset := make(MultiAsset.MultiAsset[int64])

	for policyId, assets := range match.Value {
		if policyId == "ada" {
			for assetId, assetAmount := range assets {
				if assetId == "lovelace" {
					totalLovelace = assetAmount.Int64()
				}
			}
		} else {
			// This is a native asset
			policy, err := Policy.New(policyId)
			if err != nil {
				continue // Skip invalid policies
			}
			assetMap := make(Asset.Asset[int64])
			for assetName, assetAmount := range assets {
				an := AssetName.NewAssetNameFromHexString(assetName)
				if an != nil {
					assetMap[*an] = assetAmount.Int64()
				}
			}
			if len(assetMap) > 0 {
				multiAsset[*policy] = assetMap
			}
		}
	}

	// Create the value with multi-assets
	var alonzoAmount Amount.AlonzoAmount
	alonzoAmount.Coin = totalLovelace
	alonzoAmount.Value = multiAsset

	// Create PostAlonzo output
	output := TransactionOutput.TransactionOutput{
		IsPostAlonzo: true,
		PostAlonzo: TransactionOutput.TransactionOutputAlonzo{
			Address: addr,
			Amount: Value.AlonzoValue{
				Am:        alonzoAmount,
				Coin:      totalLovelace,
				HasAssets: len(multiAsset) > 0,
			},
		},
	}

	// Handle inline datum if present
	if match.DatumHash != "" && match.DatumType == "inline" {
		// Fetch the datum from Kupo
		datumHex, err := k.client.Datum(context.Background(), match.DatumHash)
		if err == nil && datumHex != "" {
			datumBytes, err := hex.DecodeString(datumHex)
			if err == nil {
				var pd PlutusData.PlutusData
				if err := cbor.Unmarshal(datumBytes, &pd); err == nil {
					datumOpt := PlutusData.DatumOptionInline(&pd)
					output.PostAlonzo.Datum = &datumOpt
				}
			}
		}
	}

	return UTxO.UTxO{
		Input: TransactionInput.TransactionInput{
			TransactionId: txIdBytes,
			Index:         int(match.OutputIndex),
		},
		Output: output,
	}, nil
}

// UTxORPCContext implements ChainContext using UTxO RPC (Apollo gRPC)
type UTxORPCContext struct {
	cfg *config.Config
}

// NewUTxORPCContext creates a new UTxO RPC chain context
func NewUTxORPCContext(cfg *config.Config) (*UTxORPCContext, error) {
	return &UTxORPCContext{
		cfg: cfg,
	}, nil
}

func (u *UTxORPCContext) GetUTxOsByAddress(address string) ([]UTxO.UTxO, error) {
	// Implementation would use utxorpc-go client
	return nil, fmt.Errorf("UTxO RPC not yet implemented")
}

func (u *UTxORPCContext) GetUTxOByRef(txHash string, index int) (*UTxO.UTxO, error) {
	return nil, fmt.Errorf("UTxO RPC not yet implemented")
}

func (u *UTxORPCContext) GetProtocolParameters() (*Base.ProtocolParameters, error) {
	return nil, fmt.Errorf("UTxO RPC not yet implemented")
}

func (u *UTxORPCContext) GetTip() (uint64, error) {
	return 0, fmt.Errorf("UTxO RPC not yet implemented")
}

func (u *UTxORPCContext) SubmitTx(tx Transaction.Transaction) (string, error) {
	return "", fmt.Errorf("UTxO RPC not yet implemented")
}
