# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This repository serves two purposes:

1. **Buidler Fest 2026 Signup Application** - A functional signup system for the Buidler Fest 2026 event
2. **Vibe Coding Example** - A demonstration of building a Cardano application using Go with Claude Code, showcasing AI-assisted development with minimal human intervention

## Technology Stack

- **Language**: Go 1.24+
- **Blockchain**: Cardano (Preview and Mainnet)
- **Key Libraries**:
  - Apollo - Cardano transaction building
  - Bursa - Wallet and key management
  - Gouroboros - Cardano node protocol
  - Kugo - Kupo/Ogmios UTxO queries
  - Cobra - CLI framework

## Development Methodology

This project follows a "vibe coding" approach:
- Plan thoroughly before implementing
- Execute implementation with minimal human interaction
- Claude Code drives development based on high-level requirements

## Architecture

```
cmd/buidlerfest/           # CLI entry point
internal/
├── config/                # Configuration management (env vars, profiles)
├── wallet/                # Wallet/key management (mnemonic, signing)
├── chaincontext/          # Chain backends (Blockfrost, Kupo, UTxO RPC)
└── signup/                # Transaction building and submission
```

### Key Design Decisions

1. **Multi-backend support**: Blockfrost, Ogmios+Kupo, UTxO RPC Apollo
2. **Network profiles**: Preview (default) and Mainnet via `.env.{profile}` files
3. **Dual mode**: Interactive (prompts) and non-interactive (CLI flags)
4. **Unsigned TX output**: When no mnemonic provided, outputs CBOR for external signing

## Development Commands

```bash
# Build
make build

# Run tests
make test

# Lint
make lint

# Show help
./bin/buidlerfest --help

# Sign up (interactive)
./bin/buidlerfest signup --profile preview

# Sign up (non-interactive with mnemonic)
./bin/buidlerfest signup --profile preview --mnemonic "your 24 words..."

# Generate unsigned transaction
./bin/buidlerfest signup --profile preview --address addr_test1... --skip-submit

# Show signup info
./bin/buidlerfest info --profile preview
```

## Configuration

Environment variables or `.env.{profile}` files:

| Variable | Description |
|----------|-------------|
| `BLOCKFROST_API_KEY` | Blockfrost API key |
| `OGMIOS_URL` | Ogmios WebSocket URL |
| `KUPO_URL` | Kupo HTTP URL |
| `UTXORPC_URL` | UTxO RPC gRPC URL |
| `NETWORK` | Network name (preview, mainnet) |

## Smart Contract Integration

This application interacts with the Buidler Fest 2026 ticketing smart contract:

- **Aiken validators**: Ticketer (spend + mint policy)
- **State machine**: Issuer UTxO with beacon token tracks ticket counter
- **Transaction flow**:
  1. Find issuer state UTxO with beacon token
  2. Extract current ticket counter from datum
  3. Build TX: pay 400 ADA to treasury, mint TICKET{n}, update state
  4. Sign and submit (or output unsigned CBOR)

## Prompt Log

This section records the prompts given to Claude and their outcomes throughout development.

### Session 1: Project Initialization
- **Prompt**: Initialize CLAUDE.md for the repository
- **Outcome**: Created initial CLAUDE.md with Go project structure
- **Prompt**: Update to reflect dual purpose (signup app + vibe coding example for Cardano/Go)
- **Outcome**: Updated CLAUDE.md with project methodology and prompt logging

### Session 2: Full Implementation
- **Prompt**: Read and understand the example application at https://github.com/txpipe/buidler-fest-2026-buy-ticket including the onchain Aiken smart contracts. Model this application on https://github.com/blinklabs-io/buidler-fest-2024-workshop as much as possible. All of the software should be in Go. Do not shell out to other applications. Build a CLI tool similar to the example, supporting preview and mainnet networks with automatic submission when a mnemonic is available with the ability to output simply the unsigned transaction. Once built, the user should not need to modify the code, rather provide inputs only to a prebuilt binary. Default to the preview profile and network. Support both interactive and non-interactive capabilities. Support Blockfrost, Ogmios, and UTxO RPC Apollo chain contexts. Generate GitHub Actions workflows to test, build, and publish the built binary.
- **Outcome**:
  - Explored both reference repositories via web fetch
  - Analyzed Aiken smart contract (ticketer.ak) and type definitions
  - Extracted configuration from .env.mainnet and .env.preview
  - Understood Tx3 protocol definition (main.tx3)
  - Reviewed mainnet transaction structure
  - Created complete Go implementation:
    - `cmd/buidlerfest/main.go` - CLI with Cobra (signup, info, version commands)
    - `internal/config/config.go` - Configuration with network profiles
    - `internal/wallet/wallet.go` - Wallet management with Bursa
    - `internal/chaincontext/chaincontext.go` - Multi-backend chain context
    - `internal/signup/signup.go` - Transaction builder with Apollo
  - Created GitHub Actions workflows:
    - `.github/workflows/ci.yml` - Test and lint on PR/push
    - `.github/workflows/release.yml` - Multi-platform binary releases
  - Created environment files:
    - `.env.preview` - Preview network configuration
    - `.env.mainnet` - Mainnet configuration
  - Created `Makefile` for development convenience

## Reference

- [txpipe/buidler-fest-2026-buy-ticket](https://github.com/txpipe/buidler-fest-2026-buy-ticket) - Original Tx3 implementation
- [blinklabs-io/buidler-fest-2024-workshop](https://github.com/blinklabs-io/buidler-fest-2024-workshop) - Go patterns reference
