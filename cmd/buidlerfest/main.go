package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/blinklabs-io/buidler-fest-2026-signup/internal/config"
	"github.com/blinklabs-io/buidler-fest-2026-signup/internal/signup"
	"github.com/blinklabs-io/buidler-fest-2026-signup/internal/wallet"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var (
	// Version information (set by ldflags)
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func main() {
	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	rootCmd := &cobra.Command{
		Use:   "buidlerfest",
		Short: "Buidler Fest 2026 Signup CLI",
		Long: `A CLI tool to sign up for Buidler Fest 2026 by purchasing a ticket NFT on Cardano.

This tool demonstrates "vibe coding" with Claude Code to build Cardano applications in Go.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Load environment from profile-specific .env file
			profile, _ := cmd.Flags().GetString("profile")
			envFile := fmt.Sprintf(".env.%s", profile)
			if err := godotenv.Load(envFile); err != nil {
				// Try loading default .env
				_ = godotenv.Load()
			}
			return nil
		},
	}

	// Global flags
	rootCmd.PersistentFlags().StringP("profile", "p", "preview", "Network profile (preview, mainnet)")
	rootCmd.PersistentFlags().String("blockfrost-api-key", "", "Blockfrost API key")
	rootCmd.PersistentFlags().String("ogmios-url", "", "Ogmios WebSocket URL")
	rootCmd.PersistentFlags().String("kupo-url", "", "Kupo HTTP URL")
	rootCmd.PersistentFlags().String("utxorpc-url", "", "UTxO RPC (Apollo) gRPC URL")
	rootCmd.PersistentFlags().String("mnemonic", "", "Wallet mnemonic (24 words) - WARNING: visible in shell history/process list, prefer --mnemonic-file")
	rootCmd.PersistentFlags().String("mnemonic-file", "", "Path to file containing mnemonic")
	rootCmd.PersistentFlags().Bool("skip-submit", false, "Output unsigned transaction without submitting")

	// Signup command
	signupCmd := &cobra.Command{
		Use:   "signup",
		Short: "Sign up for Buidler Fest 2026",
		Long:  `Purchase a ticket NFT to sign up for Buidler Fest 2026.`,
		RunE:  runSignup,
	}
	signupCmd.Flags().String("address", "", "Buyer wallet address (required if no mnemonic)")

	// Version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("buidlerfest %s\n", Version)
			fmt.Printf("  commit: %s\n", Commit)
			fmt.Printf("  built:  %s\n", BuildDate)
		},
	}

	// Info command - show current configuration and ticket status
	infoCmd := &cobra.Command{
		Use:   "info",
		Short: "Show signup information and current ticket status",
		RunE:  runInfo,
	}

	rootCmd.AddCommand(signupCmd, versionCmd, infoCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runSignup(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Load or create wallet
	var w *wallet.Wallet
	mnemonic, _ := cmd.Flags().GetString("mnemonic")
	mnemonicFile, _ := cmd.Flags().GetString("mnemonic-file")
	address, _ := cmd.Flags().GetString("address")
	skipSubmit, _ := cmd.Flags().GetBool("skip-submit")

	if mnemonic != "" || mnemonicFile != "" {
		w, err = wallet.LoadWallet(mnemonic, mnemonicFile, cfg.Network)
		if err != nil {
			return fmt.Errorf("failed to load wallet: %w", err)
		}
		slog.Info("loaded wallet", "address", w.PaymentAddress)
	} else if address == "" {
		// Interactive mode - prompt for address or mnemonic
		w, address, err = wallet.InteractiveWalletSetup(cfg.Network)
		if err != nil {
			return fmt.Errorf("failed to setup wallet: %w", err)
		}
		if w != nil {
			slog.Info("loaded wallet", "address", w.PaymentAddress)
		}
	}

	// Build and optionally submit transaction
	result, err := signup.ExecuteSignup(cfg, w, address, skipSubmit)
	if err != nil {
		return fmt.Errorf("signup failed: %w", err)
	}

	if skipSubmit || w == nil {
		fmt.Printf("\nUnsigned Transaction (CBOR):\n%s\n", result.UnsignedTxCBOR)
		fmt.Printf("\nTransaction Hash: %s\n", result.TxHash)
		fmt.Println("\nImport this CBOR into your wallet to sign and submit.")
	} else {
		fmt.Printf("\nTransaction submitted successfully!\n")
		fmt.Printf("Transaction Hash: %s\n", result.TxHash)
		fmt.Printf("Ticket: %s\n", result.TicketName)
	}

	return nil
}

func runInfo(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	info, err := signup.GetSignupInfo(cfg)
	if err != nil {
		return fmt.Errorf("failed to get signup info: %w", err)
	}

	fmt.Printf("Buidler Fest 2026 Signup Information\n")
	fmt.Printf("=====================================\n")
	fmt.Printf("Network:        %s\n", cfg.Network)
	fmt.Printf("Ticket Price:   %d ADA\n", cfg.TicketPrice/1_000_000)
	fmt.Printf("Treasury:       %s\n", cfg.Treasury)
	fmt.Printf("Ticket Policy:  %s\n", cfg.TicketPolicy)
	fmt.Printf("\nCurrent Status:\n")
	fmt.Printf("  Next Ticket:  TICKET%d\n", info.NextTicketNumber)
	fmt.Printf("  Tickets Sold: %d\n", info.TicketsSold)

	return nil
}

func loadConfig(cmd *cobra.Command) (*config.Config, error) {
	profile, _ := cmd.Flags().GetString("profile")

	cfg, err := config.Load(profile)
	if err != nil {
		return nil, err
	}

	// Override with command line flags
	if key, _ := cmd.Flags().GetString("blockfrost-api-key"); key != "" {
		cfg.BlockfrostAPIKey = key
	}
	if url, _ := cmd.Flags().GetString("ogmios-url"); url != "" {
		cfg.OgmiosURL = url
	}
	if url, _ := cmd.Flags().GetString("kupo-url"); url != "" {
		cfg.KupoURL = url
	}
	if url, _ := cmd.Flags().GetString("utxorpc-url"); url != "" {
		cfg.UTxORPCURL = url
	}

	return cfg, nil
}
