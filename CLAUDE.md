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

### Session 3: PR Creation and CI Fixes
- **Prompt**: Update any documentation as necessary and commit using git commit -s and without GPG signature.
- **Outcome**: Committed changes with signed-off-by header

- **Prompt**: Create a pull request.
- **Outcome**: Created PR #1 with implementation

- **Prompt**: Check if CI passes on the PR
- **Outcome**:
  - Discovered lint timeout issues
  - Created `.golangci.yml` with 5m timeout and v2 config format
  - Fixed gofmt formatting issues in config.go and wallet.go
  - Fixed errcheck: explicitly ignoring os.Setenv return value
  - Fixed staticcheck: lowercase error string in signup.go
  - Updated CI to use golangci-lint-action v7 with golangci-lint v2.1.6
  - All CI checks passed after fixes

### Session 4: Code Review Feedback
- **Prompt**: Make sure you read and process the review feedback as well. You may need to ask the reviewer bots to review your code. Leave no TODOs or unfinished code.
- **Outcome**: Addressed all 11 issues from cubic-dev-ai bot:
  - **Security fixes**:
    - Added warning about --mnemonic flag visibility in shell history/process list
    - Updated README to recommend --mnemonic-file over --mnemonic
    - Changed wallet to return error for unsupported networks instead of silent fallback
  - **Code quality fixes**:
    - Fixed transaction hash calculation using blake2b.Sum256()
    - Implemented proper token verification in hasToken() using policy.Value and assetName.HexString()
    - Added quote stripping for .env file values (single and double quotes)
    - Added error tracking and reporting in Kupo UTxO conversion
    - Fixed potential integer underflow when tip < 100
  - **Build/CI fixes**:
    - Updated Go version to 1.24+ (required by Apollo dependency)
    - Fixed Windows binary filename in release workflow
    - Pinned golangci-lint to v2.1.6

- **Prompt**: Update with additional prompts you've been given and check the PR reviews, again. The review bots can take 10 to 15 minutes. Ensure you update our documentation as you go. Request reviews from the bots after making code changes.
- **Outcome**: Addressed second round of review feedback (6 issues from cubic-dev-ai):
  - Fixed integer division for ADA display (now shows decimal precision)
  - Added validation for non-negative script ref index
  - Added missing .PHONY targets to Makefile (fmt, mod-tidy, run-info, install)
  - Handle SetWalletAsChangeAddress error with logging
  - Fixed spend redeemer attachment using CollectFrom() instead of AddLoadedUTxOs()
  - Removed .claude/settings.local.json from git tracking (added to .gitignore)
  - Updated CLAUDE.md with prompt log

### Session 5: Kupo/Ogmios Support Fix
- **Prompt**: I've updated the .env files to include my Kupo and Ogmios URLs, but running the binary fails. I cannot get info from either preview or mainnet profile. It appears to be a UTxO lookup error.
- **Outcome**: Fixed Kupo chain context implementation:
  - Rewrote `kupoMatchToUTxO` to properly handle multi-assets (beacon tokens)
  - Added support for inline datums by fetching from Kupo's datum endpoint
  - Created PostAlonzo UTxO outputs instead of simple outputs
  - Added `kugo.OnlyUnspent()` filter to Kupo queries
  - Added imports for Apollo types: Amount, Asset, AssetName, MultiAsset, Policy, PlutusData
  - Added cbor library for datum deserialization

### Session 6: Datum Parsing and OgmiosChainContext Support
- **Prompt**: It's still failing. Maybe you should test your work. At the very least, we should be able to get the latest info. Deep dive and discover the root cause and fix the issues.
- **Outcome**: Fixed datum extraction and added Apollo's OgmiosChainContext:
  - **Datum parsing fix**: Changed type assertion from `PlutusIndefArray` to support both `PlutusDefArray` and `PlutusIndefArray` (CBOR can decode to either)
  - **Counter value fix**: Changed type assertion from `int64` to support both `int64` and `uint64` (CBOR unsigned integers decode to uint64)
  - Tested successfully with mainnet Kupo (preview Kupo was 48 days behind)

- **Prompt**: I am getting a failure in transaction building: "apollo transaction building requires Blockfrost chain context" which means you didn't implement all the required functionality. A Kupo and Ogmios address should suffice for Apollo.
- **Outcome**: Integrated Apollo's built-in OgmiosChainContext:
  - Added `OgmiosContext` wrapper type that uses Apollo's `OgmiosChainContext.NewOgmiosChainContext()`
  - `OgmiosContext` wraps both `ogmigo` (Ogmios) and `kugo` (Kupo) clients
  - Implements all methods of our `ChainContext` interface
  - Updated `buildSignupTx` to accept both `BlockfrostContext` and `OgmiosContext`
  - Successfully tested with mainnet Kupo+Ogmios configuration

- **Prompt**: Script integrity hash mismatch error when submitting signed transaction
- **Outcome**: Fixed Plutus version for reference script:
  - Changed `AddReferenceInput` to `AddReferenceInputV3` since Aiken compiles to Plutus V3 by default
  - The script integrity hash is computed based on Plutus version and cost models
  - V2 vs V3 use different cost models, causing hash mismatch

## Reference

- [txpipe/buidler-fest-2026-buy-ticket](https://github.com/txpipe/buidler-fest-2026-buy-ticket) - Original Tx3 implementation
- [blinklabs-io/buidler-fest-2024-workshop](https://github.com/blinklabs-io/buidler-fest-2024-workshop) - Go patterns reference
