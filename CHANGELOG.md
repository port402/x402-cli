# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2025-01-10

### Added

#### Commands
- `x402 health <url>` - Health check for x402-enabled endpoints (no wallet required)
- `x402 test <url>` - Full payment flow test with EIP-3009 signing
- `x402 batch-health <file>` - Batch health check from JSON file with parallel execution
- `x402 version` - Display version, commit, and build information

#### Protocol Support
- Automatic v1/v2 protocol detection via `Payment-Required` header
- v1 protocol: JSON body parsing with `maxAmountRequired` field
- v2 protocol: Base64 header decoding with `amount` field
- EIP-712 typed data signing for `TransferWithAuthorization` (EIP-3009)
- Support for both CAIP-2 (`eip155:8453`) and simple (`base`) network formats

#### Wallet Support
- Keystore files (Web3 Secret Storage format) via `--keystore`
- Raw hex private keys via `--wallet`
- Environment variable via `PRIVATE_KEY`
- Stdin input for piped workflows
- Secure password prompt for encrypted keystores

#### Safety Features
- Interactive payment confirmation prompt (default: enabled)
- `--dry-run` mode to preview payment details without signing
- `--max-amount` safety cap to prevent unexpected large payments
- `--skip-payment-confirmation` for scripted/CI usage
- Ctrl+C handling with post-signing warning

#### Output
- Human-readable TTY output with colored formatting
- `--json` flag for machine-readable JSON output
- `--verbose` flag for detailed request/response logging
- Transaction explorer links for successful payments
- Response body piping support (body to stdout, status to stderr)

#### Batch Health Features
- JSON input with optional HTTP method per URL
- Configurable parallelism via `--parallel`
- Summary statistics (passed/failed/total/time)

#### Token Support
- Built-in registry for USDC on major EVM networks
- Human-readable amount formatting (e.g., "0.001 USDC")
- Block explorer URL generation for known networks
- Graceful handling of unknown tokens with raw unit display

### Technical Details

- Built with Go 1.21+
- Uses [Cobra](https://github.com/spf13/cobra) for CLI framework
- Uses [go-ethereum](https://github.com/ethereum/go-ethereum) for EIP-712 signing
- Comprehensive test coverage with `testify/assert`
- CI/CD via GitHub Actions with CodeQL security scanning

[Unreleased]: https://github.com/port402/x402-cli/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/port402/x402-cli/releases/tag/v1.0.0
