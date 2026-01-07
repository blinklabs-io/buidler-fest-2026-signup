# Buidler Fest 2026 Signup

A CLI tool to sign up for Buidler Fest 2026 by purchasing a ticket NFT on Cardano.

This project serves as both a functional signup application and a demonstration of "vibe coding" - building a Cardano application using Go with Claude Code, showcasing AI-assisted development with minimal human intervention.

## Features

- Purchase Buidler Fest 2026 ticket NFTs on Cardano
- Support for Preview (testnet) and Mainnet networks
- Multiple chain context backends: Blockfrost, Kupo/Ogmios, UTxO RPC
- Interactive and non-interactive modes
- Generate unsigned transactions for external wallet signing
- Auto-submit when mnemonic is provided

## Installation

### From Source

```bash
git clone https://github.com/blinklabs-io/buidler-fest-2026-signup.git
cd buidler-fest-2026-signup
make build
```

### From Releases

Download the latest binary for your platform from the [Releases](https://github.com/blinklabs-io/buidler-fest-2026-signup/releases) page.

## Configuration

Set your chain context backend via environment variables or `.env.{profile}` files:

```bash
# Blockfrost (recommended)
export BLOCKFROST_API_KEY=your_api_key

# Or Kupo/Ogmios
export KUPO_URL=http://localhost:1442
export OGMIOS_URL=ws://localhost:1337

# Or UTxO RPC
export UTXORPC_URL=localhost:50051
```

## Usage

```bash
# Show help
./bin/buidlerfest --help

# Show signup information and current ticket status
./bin/buidlerfest info --profile preview

# Sign up interactively (prompts for wallet info)
./bin/buidlerfest signup --profile preview

# Sign up with mnemonic (auto-signs and submits)
./bin/buidlerfest signup --profile preview --mnemonic "your 24 word mnemonic..."

# Generate unsigned transaction for external signing
./bin/buidlerfest signup --profile preview --address addr_test1... --skip-submit

# Use mainnet
./bin/buidlerfest signup --profile mainnet
```

## Ticket Price

- **400 ADA** (both Preview and Mainnet)

## How It Works

1. The CLI queries the issuer state UTxO to find the current ticket counter
2. Builds a transaction that:
   - Pays 400 ADA to the treasury
   - Mints a new TICKET{n} NFT
   - Updates the issuer state with incremented counter
3. Signs with your wallet (if mnemonic provided) or outputs unsigned CBOR
4. Submits to the network

## Development

```bash
# Build
make build

# Run tests
make test

# Lint
make lint

# Format code
make fmt
```

## Architecture

See [CLAUDE.md](CLAUDE.md) for detailed architecture documentation and the development prompt log.

## References

- [txpipe/buidler-fest-2026-buy-ticket](https://github.com/txpipe/buidler-fest-2026-buy-ticket) - Original Tx3 implementation
- [blinklabs-io/buidler-fest-2024-workshop](https://github.com/blinklabs-io/buidler-fest-2024-workshop) - Go patterns reference

## License

Apache-2.0
