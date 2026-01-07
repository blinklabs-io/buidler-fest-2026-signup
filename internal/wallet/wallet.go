package wallet

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/blinklabs-io/bursa"
)

// Wallet holds wallet information for signing transactions
type Wallet struct {
	Mnemonic              string
	PaymentAddress        string
	PaymentVKey           []byte
	PaymentSKey           []byte
	PaymentExtendedSKey   []byte
	StakeAddress          string
	Network               string
}

// LoadWallet loads a wallet from mnemonic string or file
func LoadWallet(mnemonic, mnemonicFile, network string) (*Wallet, error) {
	var mnemonicStr string

	if mnemonicFile != "" {
		data, err := os.ReadFile(mnemonicFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read mnemonic file: %w", err)
		}
		mnemonicStr = strings.TrimSpace(string(data))
	} else if mnemonic != "" {
		mnemonicStr = strings.TrimSpace(mnemonic)
	} else {
		return nil, fmt.Errorf("no mnemonic provided")
	}

	// Validate mnemonic word count
	words := strings.Fields(mnemonicStr)
	if len(words) != 24 && len(words) != 15 && len(words) != 12 {
		return nil, fmt.Errorf("invalid mnemonic: expected 12, 15, or 24 words, got %d", len(words))
	}

	return createWalletFromMnemonic(mnemonicStr, network)
}

// createWalletFromMnemonic creates a wallet from a mnemonic phrase
func createWalletFromMnemonic(mnemonic, network string) (*Wallet, error) {
	// Determine network name for bursa
	var bursaNetwork string
	switch network {
	case "mainnet":
		bursaNetwork = "mainnet"
	case "preview":
		bursaNetwork = "preview"
	case "preprod":
		bursaNetwork = "preprod"
	default:
		bursaNetwork = "preview"
	}

	// Create wallet using bursa with default derivation paths
	// NewWallet(mnemonic, network, password string, accountId uint, paymentId, stakeId, addressId uint32)
	bursaWallet, err := bursa.NewWallet(mnemonic, bursaNetwork, "", 0, 0, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}

	// Extract keys
	vKeyBytes, err := hex.DecodeString(bursaWallet.PaymentVKey.CborHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode vkey: %w", err)
	}
	// Strip CBOR wrapper (2 bytes)
	if len(vKeyBytes) > 2 {
		vKeyBytes = vKeyBytes[2:]
	}

	sKeyBytes, err := hex.DecodeString(bursaWallet.PaymentExtendedSKey.CborHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode skey: %w", err)
	}
	// Strip CBOR wrapper and extended key portion
	if len(sKeyBytes) > 2 {
		sKeyBytes = sKeyBytes[2:]
	}

	// Store full extended skey for signing
	extSKey := make([]byte, len(sKeyBytes))
	copy(extSKey, sKeyBytes)

	// Trim to 64 bytes for standard signing key
	if len(sKeyBytes) > 64 {
		sKeyBytes = slices.Delete(sKeyBytes, 64, len(sKeyBytes))
	}

	return &Wallet{
		Mnemonic:            mnemonic,
		PaymentAddress:      bursaWallet.PaymentAddress,
		PaymentVKey:         vKeyBytes,
		PaymentSKey:         sKeyBytes,
		PaymentExtendedSKey: extSKey,
		StakeAddress:        bursaWallet.StakeAddress,
		Network:             network,
	}, nil
}

// InteractiveWalletSetup prompts the user for wallet information
func InteractiveWalletSetup(network string) (*Wallet, string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\nWallet Setup")
	fmt.Println("============")
	fmt.Println("Choose one of the following options:")
	fmt.Println("1. Enter a 24-word mnemonic (for signing and submitting)")
	fmt.Println("2. Enter a wallet address (for generating unsigned transaction)")
	fmt.Print("\nSelect option (1 or 2): ")

	option, err := reader.ReadString('\n')
	if err != nil {
		return nil, "", fmt.Errorf("failed to read option: %w", err)
	}
	option = strings.TrimSpace(option)

	switch option {
	case "1":
		fmt.Print("\nEnter your 24-word mnemonic: ")
		mnemonic, err := reader.ReadString('\n')
		if err != nil {
			return nil, "", fmt.Errorf("failed to read mnemonic: %w", err)
		}
		mnemonic = strings.TrimSpace(mnemonic)

		w, err := createWalletFromMnemonic(mnemonic, network)
		if err != nil {
			return nil, "", err
		}
		return w, w.PaymentAddress, nil

	case "2":
		fmt.Print("\nEnter your wallet address: ")
		address, err := reader.ReadString('\n')
		if err != nil {
			return nil, "", fmt.Errorf("failed to read address: %w", err)
		}
		address = strings.TrimSpace(address)

		// Validate address format
		if !isValidCardanoAddress(address, network) {
			return nil, "", fmt.Errorf("invalid Cardano address for network %s", network)
		}
		return nil, address, nil

	default:
		return nil, "", fmt.Errorf("invalid option: %s", option)
	}
}

// isValidCardanoAddress performs basic validation of a Cardano address
func isValidCardanoAddress(address, network string) bool {
	switch network {
	case "mainnet":
		return strings.HasPrefix(address, "addr1")
	case "preview", "preprod", "testnet":
		return strings.HasPrefix(address, "addr_test1")
	default:
		return strings.HasPrefix(address, "addr")
	}
}

// GenerateNewWallet creates a new wallet with a random mnemonic
func GenerateNewWallet(network string) (*Wallet, error) {
	// Generate new mnemonic
	mnemonic, err := bursa.NewMnemonic()
	if err != nil {
		return nil, fmt.Errorf("failed to generate mnemonic: %w", err)
	}

	return createWalletFromMnemonic(mnemonic, network)
}

// SaveMnemonicToFile saves the mnemonic to a file
func (w *Wallet) SaveMnemonicToFile(path string) error {
	return os.WriteFile(path, []byte(w.Mnemonic), 0600)
}
