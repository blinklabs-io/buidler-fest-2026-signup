package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/kelseyhightower/envconfig"
)

// Config holds all configuration for the signup application
type Config struct {
	// Network settings
	Network string `envconfig:"NETWORK" default:"preview"`
	Profile string `envconfig:"PROFILE" default:"preview"`

	// Chain context backends (priority: UTxORPC > Ogmios+Kupo > Blockfrost)
	BlockfrostAPIKey string `envconfig:"BLOCKFROST_API_KEY"`
	OgmiosURL        string `envconfig:"OGMIOS_URL"`
	KupoURL          string `envconfig:"KUPO_URL"`
	UTxORPCURL       string `envconfig:"UTXORPC_URL"`

	// Transaction submission
	SubmitTCPAddress string `envconfig:"SUBMIT_TCP_ADDRESS"`
	SubmitSocketPath string `envconfig:"SUBMIT_SOCKET_PATH"`
	SubmitURL        string `envconfig:"SUBMIT_URL"`

	// Ticketing parameters (from .env.{profile})
	IssuerBeaconPolicy string `envconfig:"ISSUER_BEACON_POLICY"`
	IssuerBeaconName   string `envconfig:"ISSUER_BEACON_NAME"`
	Treasury           string `envconfig:"TREASURY"`
	IssuerScriptRef    string `envconfig:"ISSUER_SCRIPT_REF"`
	TicketPolicy       string `envconfig:"TICKET_POLICY"`
	IssuerAddress      string `envconfig:"ISSUER"`
	TicketPrice        uint64 `envconfig:"TICKET_PRICE" default:"400000000"`
}

// NetworkConfig holds network-specific parameters
type NetworkConfig struct {
	Name               string
	NetworkMagic       uint32
	BlockfrostBaseURL  string
	BootstrapPeerHost  string
	BootstrapPeerPort  uint16
	AddressPrefix      string
	IssuerBeaconPolicy string
	IssuerBeaconName   string
	Treasury           string
	IssuerScriptRef    string
	TicketPolicy       string
	IssuerAddress      string
	TicketPrice        uint64
}

var networks = map[string]NetworkConfig{
	"preview": {
		Name:               "preview",
		NetworkMagic:       2,
		BlockfrostBaseURL:  "https://cardano-preview.blockfrost.io/api",
		BootstrapPeerHost:  "preview-node.play.dev.cardano.org",
		BootstrapPeerPort:  3001,
		AddressPrefix:      "addr_test",
		IssuerBeaconPolicy: "eb5c99eee64509431ace535f61fa5aca27e28181dc1e38c0b6f65c09",
		IssuerBeaconName:   "425549444c45524645535432303236",
		Treasury:           "addr_test1vp5hwqva2u8mglkygpmea5jjcag7lmd4ylf5rfkfgsm746qj9khza",
		IssuerScriptRef:    "2f85786f7211411a2f740ae1bb2d6283166b28dd287301f6bd99a918ff98adda#0",
		TicketPolicy:       "4672f28ce36e492e722392359f71e9a9442646f81d856f92d7e163f1",
		IssuerAddress:      "addr_test1wpr89u5vudhyjtnjywfrt8m3ax55gfjxlqwc2muj6lsk8ugwajhze",
		TicketPrice:        400000000,
	},
	"mainnet": {
		Name:               "mainnet",
		NetworkMagic:       764824073,
		BlockfrostBaseURL:  "https://cardano-mainnet.blockfrost.io/api",
		BootstrapPeerHost:  "backbone.mainnet.cardano.org",
		BootstrapPeerPort:  3001,
		AddressPrefix:      "addr",
		IssuerBeaconPolicy: "e1ddde8138579e255482791d9fba0778cb1f5c7b435be7b3e42069de",
		IssuerBeaconName:   "425549444c45524645535432303236",
		Treasury:           "addr1qx0decp93g2kwym5cz0p68thamd2t9pehlxqe02qae5r6nycv42qmjppm2rr8fj6qlzfhm6ljkd5f0tjlgudtmt5kzyqmy8x82",
		IssuerScriptRef:    "31596ecbdcf102c8e5c17e75c65cf9780996285879d18903f035964f3a7499a8#0",
		TicketPolicy:       "1d9c0b541adc300c19ddc6b9fb63c0bfe32b1508305ba65b8762dc7b",
		IssuerAddress:      "addr1wywecz65rtwrqrqemhrtn7mrczl7x2c4pqc9hfjmsa3dc7cr5pvqw",
		TicketPrice:        400000000,
	},
}

// Load loads configuration from environment variables and applies network defaults
func Load(profile string) (*Config, error) {
	cfg := &Config{}
	if err := envconfig.Process("", cfg); err != nil {
		return nil, fmt.Errorf("failed to process env config: %w", err)
	}

	// Override profile if specified
	if profile != "" {
		cfg.Profile = profile
	}

	// Determine network from profile
	network := strings.ToLower(cfg.Profile)
	if network == "" {
		network = "preview"
	}
	cfg.Network = network

	// Apply network defaults for ticketing parameters if not set
	netCfg, ok := networks[network]
	if !ok {
		return nil, fmt.Errorf("unknown network: %s", network)
	}

	if cfg.IssuerBeaconPolicy == "" {
		cfg.IssuerBeaconPolicy = netCfg.IssuerBeaconPolicy
	}
	if cfg.IssuerBeaconName == "" {
		cfg.IssuerBeaconName = netCfg.IssuerBeaconName
	}
	if cfg.Treasury == "" {
		cfg.Treasury = netCfg.Treasury
	}
	if cfg.IssuerScriptRef == "" {
		cfg.IssuerScriptRef = netCfg.IssuerScriptRef
	}
	if cfg.TicketPolicy == "" {
		cfg.TicketPolicy = netCfg.TicketPolicy
	}
	if cfg.IssuerAddress == "" {
		cfg.IssuerAddress = netCfg.IssuerAddress
	}
	if cfg.TicketPrice == 0 {
		cfg.TicketPrice = netCfg.TicketPrice
	}

	return cfg, nil
}

// GetNetworkConfig returns the network configuration for the given network name
func GetNetworkConfig(network string) (*NetworkConfig, error) {
	netCfg, ok := networks[network]
	if !ok {
		return nil, fmt.Errorf("unknown network: %s", network)
	}
	return &netCfg, nil
}

// HasChainContext returns true if at least one chain context backend is configured
func (c *Config) HasChainContext() bool {
	return c.BlockfrostAPIKey != "" || c.OgmiosURL != "" || c.KupoURL != "" || c.UTxORPCURL != ""
}

// GetChainContextType returns the configured chain context type
func (c *Config) GetChainContextType() string {
	if c.UTxORPCURL != "" {
		return "utxorpc"
	}
	if c.OgmiosURL != "" && c.KupoURL != "" {
		return "ogmios"
	}
	if c.KupoURL != "" {
		return "kupo"
	}
	if c.BlockfrostAPIKey != "" {
		return "blockfrost"
	}
	return "none"
}

// ValidateChainContext validates that required chain context configuration is present
func (c *Config) ValidateChainContext() error {
	if !c.HasChainContext() {
		return fmt.Errorf("no chain context configured; set BLOCKFROST_API_KEY, OGMIOS_URL+KUPO_URL, or UTXORPC_URL")
	}
	return nil
}

// ParseIssuerScriptRef parses the issuer script reference into tx hash and index
func (c *Config) ParseIssuerScriptRef() (string, int, error) {
	parts := strings.Split(c.IssuerScriptRef, "#")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid issuer script ref format: %s", c.IssuerScriptRef)
	}
	var idx int
	if _, err := fmt.Sscanf(parts[1], "%d", &idx); err != nil {
		return "", 0, fmt.Errorf("invalid issuer script ref index: %s", parts[1])
	}
	if idx < 0 {
		return "", 0, fmt.Errorf("invalid issuer script ref index: must be non-negative, got %d", idx)
	}
	return parts[0], idx, nil
}

// LoadFromEnvFile loads additional configuration from a .env file
func LoadFromEnvFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// Strip surrounding quotes (both single and double)
			if len(value) >= 2 {
				if (value[0] == '"' && value[len(value)-1] == '"') ||
					(value[0] == '\'' && value[len(value)-1] == '\'') {
					value = value[1 : len(value)-1]
				}
			}
			if os.Getenv(key) == "" {
				_ = os.Setenv(key, value)
			}
		}
	}
	return nil
}
