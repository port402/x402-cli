# x402-cli

[![CI](https://github.com/port402/x402-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/port402/x402-cli/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/port402/x402-cli)](https://goreportcard.com/report/github.com/port402/x402-cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A command-line tool for testing x402-enabled payment-gated APIs.

The x402 protocol uses HTTP 402 (Payment Required) status codes with EIP-3009 gasless token transfers to gate access to resources. This CLI helps developers test and validate x402 endpoints.

## How x402 Works

The x402 protocol enables payment-gated APIs using HTTP 402 responses and gasless EIP-3009 token transfers.

### Payment Flow

```
┌─────────┐                    ┌─────────┐                    ┌─────────────┐
│  Client │                    │  Server │                    │ Facilitator │
└────┬────┘                    └────┬────┘                    └──────┬──────┘
     │                              │                                │
     │  1. GET /resource            │                                │
     │ ───────────────────────────> │                                │
     │                              │                                │
     │  2. 402 Payment Required     │                                │
     │     + payment requirements   │                                │
     │ <─────────────────────────── │                                │
     │                              │                                │
     │  3. Sign EIP-3009 auth       │                                │
     │     (gasless, off-chain)     │                                │
     │                              │                                │
     │  4. GET /resource            │                                │
     │     + X-PAYMENT header       │                                │
     │ ───────────────────────────> │                                │
     │                              │                                │
     │                              │  5. Verify & settle payment    │
     │                              │ ──────────────────────────────>│
     │                              │                                │
     │                              │  6. Payment confirmed          │
     │                              │ <──────────────────────────────│
     │                              │                                │
     │  7. 200 OK + resource        │                                │
     │ <─────────────────────────── │                                │
     │                              │                                │
```

### Key Concepts

| Term | Description |
|------|-------------|
| **402 Response** | HTTP status indicating payment is required |
| **EIP-3009** | Gasless token transfer standard (no ETH needed for gas) |
| **Facilitator** | Service that verifies and settles payments on-chain |
| **X-PAYMENT** | Header containing the signed payment authorization |

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap port402/tap
brew install x402-cli
```

### Go Install

```bash
go install github.com/port402/x402-cli/cmd/x402@latest
```

### Binary Download

Download pre-built binaries from the [releases page](https://github.com/port402/x402-cli/releases).

## Quick Start

### Health Check (No wallet required)

Check if an endpoint is x402-enabled:

```bash
x402 health https://api.example.com/endpoint
```

### Test Payment

Make a test payment to an x402 endpoint:

```bash
x402 test https://api.example.com/endpoint --keystore ~/.foundry/keystores/my-wallet
```

## Commands

### `x402 health <url>`

Check if an endpoint is x402-enabled and validate its payment requirements.

```bash
# Basic health check
x402 health https://api.example.com/endpoint

# JSON output
x402 health https://api.example.com/endpoint --json

# Verbose output
x402 health https://api.example.com/endpoint --verbose

# Custom timeout
x402 health https://api.example.com/endpoint --timeout 60

# For POST-only APIs (like Elsa)
x402 health https://api.example.com/endpoint --method POST
x402 health https://api.example.com/endpoint -X POST
```

**Validates:**
- Endpoint is reachable
- Returns 402 Payment Required
- Has valid payment requirements
- Has EVM payment options
- Uses known tokens

### `x402 test <url>`

Make a full test payment to an x402 endpoint.

```bash
# Interactive mode (prompts for confirmation)
x402 test https://api.example.com/endpoint --keystore ~/.foundry/keystores/my-wallet

# Dry run (no actual payment)
x402 test https://api.example.com/endpoint --keystore ~/.foundry/keystores/my-wallet --dry-run

# Skip confirmation prompt (for scripting)
x402 test https://api.example.com/endpoint --keystore ~/.foundry/keystores/my-wallet --skip-payment-confirmation

# With safety cap
x402 test https://api.example.com/endpoint --keystore ~/.foundry/keystores/my-wallet --max-amount 0.05

# Custom HTTP method and headers
x402 test https://api.example.com/endpoint --keystore ~/.foundry/keystores/my-wallet \
  --method POST \
  --header "Authorization: Bearer token" \
  --data '{"key": "value"}'
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--keystore` | Path to Web3 Secret Storage keystore file |
| `--wallet` | Hex-encoded private key (or use `PRIVATE_KEY` env var) |
| `--dry-run` | Show payment details without executing |
| `--skip-payment-confirmation` | Skip interactive confirmation prompt |
| `--max-amount` | Maximum amount willing to pay (safety cap) |
| `--method` | HTTP method (GET, POST, PUT) |
| `--header` | Custom HTTP header (repeatable) |
| `--data` | Request body for POST/PUT requests |
| `--timeout` | Request timeout in seconds |

### `x402 batch-health <file>`

Check multiple endpoints from a JSON file.

```bash
# Basic batch check
x402 batch-health urls.json

# Parallel execution
x402 batch-health urls.json --parallel 5

# Stop on first failure
x402 batch-health urls.json --fail-fast

# JSON output
x402 batch-health urls.json --json
```

**Input file formats:**

Simple URL array (backward compatible):
```json
[
  "https://api1.example.com/endpoint",
  "https://api2.example.com/endpoint"
]
```

Object array with HTTP method support:
```json
[
  {"url": "https://api.example.com/get-endpoint"},
  {"url": "https://api.example.com/post-endpoint", "method": "POST"}
]
```

### `x402 version`

Display version information.

```bash
x402 version
x402 version --json
```

## Configuration

### Private Key Sources (in order of priority)

1. **Keystore file**: `--keystore ~/.foundry/keystores/my-wallet`
2. **Hex key flag**: `--wallet 0xac0974...`
3. **Environment variable**: `PRIVATE_KEY=0xac0974...`
4. **Stdin**: `echo "0xac0974..." | x402 test ...`

### Supported Networks

| Network | Chain ID | Status |
|---------|----------|--------|
| Ethereum Mainnet | eip155:1 | Supported |
| Base Mainnet | eip155:8453 | Supported |
| Base Sepolia | eip155:84532 | Supported |
| Ethereum Sepolia | eip155:11155111 | Supported |
| Polygon Mainnet | eip155:137 | Supported |
| Arbitrum One | eip155:42161 | Supported |
| Optimism | eip155:10 | Supported |
| Solana | solana:* | Planned |

### Supported Tokens

| Token | Networks |
|-------|----------|
| USDC | Ethereum, Base, Polygon, Arbitrum, Optimism |

## Creating a Wallet

The `x402 test` command requires a wallet to sign payments. The recommended approach is using a **keystore file** (Web3 Secret Storage format).

### Using Foundry (Recommended)

[Foundry](https://book.getfoundry.sh/) provides tools to create and manage keystores:

```bash
# Install Foundry
curl -L https://foundry.paradigm.xyz | bash
foundryup

# Create a new keystore (will prompt for password)
cast wallet new ~/.foundry/keystores/

# Or import an existing private key into a keystore
cast wallet import my-wallet --interactive

# List your keystores
ls ~/.foundry/keystores/
```

### Using Geth

If you have [Geth](https://geth.ethereum.org/) installed:

```bash
# Create a new account (stores in default datadir)
geth account new

# Create with custom keystore directory
geth account new --keystore ~/.keystores/
```

### Using the Keystore

Once created, use your keystore with:

```bash
x402 test https://api.example.com/endpoint --keystore ~/.foundry/keystores/my-wallet
```

You'll be prompted for the keystore password securely (input is hidden).

> **Security Tip:** Never share your keystore file or password. For testing, create a dedicated wallet with only test funds.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General failure |
| 2 | Input validation error |
| 3 | Network error |
| 4 | Protocol error |
| 5 | Payment rejected |

## Examples

### CI/CD Integration

```bash
# Check endpoint health in CI (JSON output, no TTY)
x402 health https://api.example.com/endpoint --json | jq '.exitCode'

# Dry-run payment test in CI
x402 test https://api.example.com/endpoint \
  --keystore ./test-wallet.json \
  --dry-run \
  --skip-payment-confirmation
```

### Scripted Testing

```bash
#!/bin/bash
# Test multiple endpoints with a safety cap
for url in "https://api1.example.com" "https://api2.example.com"; do
  x402 test "$url" \
    --keystore ~/.foundry/keystores/test \
    --max-amount 0.01 \
    --skip-payment-confirmation
done
```

### Getting Test USDC

1. Go to [Circle's Faucet](https://faucet.circle.com)
2. Select Base Sepolia
3. Choose USDC
4. Enter your wallet address

## Development

### Building from Source

```bash
git clone https://github.com/port402/x402-cli.git
cd x402-cli
make build
```

### Available Make Targets

| Target | Description |
|--------|-------------|
| `make build` | Build the binary |
| `make install` | Install to $GOPATH/bin |
| `make test` | Run all tests |
| `make test-cover` | Run tests with coverage report |
| `make lint` | Run linter |
| `make fmt` | Format code |
| `make check` | Run all checks (fmt, vet, lint, test) |
| `make clean` | Remove build artifacts |
| `make help` | Show all available targets |

### Common Workflows

```bash
# Run all checks before committing
make check

# Build with version info embedded
make build
./x402 version

# Generate coverage report
make test-cover
open coverage.html
```

## Protocol Reference

- [x402 Protocol](https://github.com/coinbase/x402)
- [Coinbase x402 Documentation](https://docs.cdp.coinbase.com/x402/welcome)
- [EIP-3009: Transfer With Authorization](https://eips.ethereum.org/EIPS/eip-3009)
- [EIP-712: Typed Data Signing](https://eips.ethereum.org/EIPS/eip-712)

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on:

- Branch naming conventions
- Commit message format
- Pull request process
- Code style requirements

## Security

To report security vulnerabilities, please see [SECURITY.md](SECURITY.md) for our responsible disclosure policy. Do not open public issues for security concerns.

## License

MIT License - see [LICENSE](LICENSE) for details.
