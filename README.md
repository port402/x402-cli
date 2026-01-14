# x402-cli

[![CI](https://github.com/port402/x402-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/port402/x402-cli/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/port402/x402-cli?v=1)](https://goreportcard.com/report/github.com/port402/x402-cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

> Test x402 payment-gated APIs from your terminal. No ETH needed for gas.

## Features

- **Health Check** — Validate x402 endpoints without a wallet
- **Agent Discovery** — Discover A2A protocol agent cards
- **Test Payments** — Execute gasless EIP-3009 payments
- **Batch Testing** — Check multiple endpoints in parallel
- **Keystore Support** — Works with Foundry/Geth keystores
- **CI/CD Ready** — JSON output and exit codes for automation
- **Dry Run Mode** — Preview payments before executing

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Sample Output](#sample-output)
- [Commands](#commands)
- [Configuration](#configuration)
- [Examples](#examples)
- [How x402 Works](#how-x402-works)
- [Creating a Wallet](#creating-a-wallet)
- [Development](#development)
- [Contributing](#contributing)

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

### Check if an endpoint is x402-enabled (no wallet needed)

```bash
x402 health https://api.example.com/resource
x402 health https://api.example.com/resource --agent   # Also check for agent card
```

### Discover A2A agent card

```bash
x402 agent https://api.example.com
```

### Make a test payment

```bash
x402 test https://api.example.com/resource --keystore ~/.foundry/keystores/my-wallet
```

### Preview payment without executing

```bash
x402 test https://api.example.com/resource --keystore ~/.foundry/keystores/my-wallet --dry-run
```

## Sample Output

### Health Check

```
$ x402 health https://x402-dotnet.azurewebsites.net/resource/middleware

✓ https://x402-dotnet.azurewebsites.net/resource/middleware

  Status:   402 Payment Required
  Protocol: v2 (current)
  Latency:  150ms
  Payment:  0.001 USDC on Base Sepolia

  Checks:
    ✓ Endpoint reachable
    ✓ Returns 402
    ✓ Valid payment header
    ✓ Has payment options
    ✓ Has EVM option
    ✓ Known token
```

### Health Check with Agent Discovery

```
$ x402 health https://api.example.com/endpoint --agent

✓ https://api.example.com/endpoint

  Status:   402 Payment Required
  Protocol: v2 (current)
  Latency:  120ms
  Payment:  0.01 USDC on Base

  Checks:
    ✓ Endpoint reachable
    ✓ Returns 402
    ✓ Valid payment header
    ✓ Has payment options
    ✓ Has EVM option
    ✓ Known token

  Agent:    Recipe Agent v1.0.0
            Agent that helps with recipes
  Provider: Example Inc (https://example.com)

  Skills:
    • recipe-search
      Find recipes by ingredients or cuisine type
    • meal-plan
      Generate weekly meal plans based on preferences

  Capabilities: streaming
```

### Dry Run

```
$ x402 test https://x402-dotnet.azurewebsites.net/resource/middleware \
    --keystore ~/.foundry/keystores/test --dry-run

✓ x402 endpoint detected (v2)

Payment Requirements:
  Network:   Base Sepolia (eip155:84532)
  Token:     USDC (0x036c...cf7e)
  Amount:    0.001000 USDC
  Recipient: 0xB889...64fc

⚠ Dry run mode - no payment will be made
```

### Successful Payment

```
$ x402 test https://x402-dotnet.azurewebsites.net/resource/middleware \
    --keystore ~/.foundry/keystores/test

✓ x402 endpoint detected (v2)

Payment Requirements:
  Network:   Base Sepolia (eip155:84532)
  Token:     USDC (0x036c...cf7e)
  Amount:    0.001000 USDC
  Recipient: 0xB889...64fc

? Proceed with payment? [y/N] y

  • Loading wallet...
  • Signing payment authorization...
  • Sending payment...

✓ Payment successful!
  Transaction: 0xb945a8c...
  Explorer:    https://sepolia.basescan.org/tx/0xb945a8c...

Response (200 OK):
{
  "message": "Access granted",
  "timestamp": "2026-01-12T10:30:00Z"
}
```

### Agent Discovery (Found)

```
$ x402 agent https://api.example.com

✓ Agent card found (/.well-known/agent.json)

Agent:
  Name:        Recipe Agent
  Version:     1.0.0
  Description: Agent that helps with recipes

Skills:
  • recipe-search — Find recipes by ingredients
  • meal-plan — Generate weekly meal plans

Docs: https://docs.example.com/agent
```

### Agent Discovery (Not Found)

```
$ x402 agent https://example.com

⚠ No agent card found

Tried:
  • /.well-known/agent.json (404)
  • /.well-known/agent-card.json (404)
  • /.well-known/agents.json (404)

Hint: Use --card-url to specify a custom location
```

## Commands

### `x402 health <url>`

Check if an endpoint is x402-enabled and validate its payment requirements.

```bash
x402 health https://api.example.com/endpoint
x402 health https://api.example.com/endpoint --json          # JSON output
x402 health https://api.example.com/endpoint --method POST   # POST-only APIs
x402 health https://api.example.com/endpoint --agent         # Also discover agent card
```

| Flag | Description |
|------|-------------|
| `--agent` | Also discover A2A agent card from the endpoint |
| `--method` | HTTP method (default: GET) |
| `--timeout` | Request timeout in seconds (default: 30) |

### `x402 agent <url>`

Discover A2A (Agent-to-Agent) protocol agent cards from endpoints.

```bash
x402 agent https://api.example.com                           # Auto-discover
x402 agent https://api.example.com --json                    # JSON output
x402 agent https://api.example.com --card-url /custom/agent.json  # Custom path
```

| Flag | Description |
|------|-------------|
| `--card-url` | Custom agent card path (overrides discovery) |
| `--timeout` | Request timeout in seconds (default: 5) |

**Discovery paths** (tried in order):
1. `/.well-known/agent.json` (A2A v0.1)
2. `/.well-known/agent-card.json` (A2A v0.2+)
3. `/.well-known/agents.json` (Wildcard spec)

### `x402 test <url>`

Make a test payment to an x402 endpoint.

```bash
x402 test <url> --keystore <path>                    # Interactive mode
x402 test <url> --keystore <path> --dry-run          # Preview only
x402 test <url> --keystore <path> --max-amount 0.05  # Safety cap
```

| Flag | Description |
|------|-------------|
| `--keystore` | Path to Web3 keystore file |
| `--wallet` | Hex-encoded private key (or `PRIVATE_KEY` env) |
| `--dry-run` | Show payment details without executing |
| `--skip-payment-confirmation` | Skip interactive prompt |
| `--max-amount` | Maximum payment amount (safety cap) |
| `--method` | HTTP method (GET, POST, PUT) |
| `--header` | Custom HTTP header (repeatable) |
| `--data` | Request body for POST/PUT |
| `--timeout` | Request timeout in seconds |

### `x402 batch-health <file>`

Check multiple endpoints from a JSON file.

```bash
x402 batch-health urls.json
x402 batch-health urls.json --parallel 5    # Parallel execution
x402 batch-health urls.json --fail-fast     # Stop on first failure
```

**Input formats:**

```json
["https://api1.example.com", "https://api2.example.com"]
```

```json
[{"url": "https://api.example.com", "method": "POST"}]
```

### `x402 version`

```bash
x402 version
x402 version --json
```

## Configuration

### Private Key Sources (priority order)

1. `--keystore ~/.foundry/keystores/my-wallet`
2. `--wallet 0xac0974...`
3. `PRIVATE_KEY=0xac0974...` environment variable
4. Stdin: `echo "0xac0974..." | x402 test ...`

### Supported Networks

| Network | Chain ID |
|---------|----------|
| Ethereum | eip155:1 |
| Base | eip155:8453 |
| Base Sepolia | eip155:84532 |
| Polygon | eip155:137 |
| Arbitrum One | eip155:42161 |
| Optimism | eip155:10 |

### Supported Tokens

USDC on all supported networks.

### Exit Codes

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
# Health check with JSON parsing
x402 health https://api.example.com/endpoint --json | jq '.exitCode'

# Dry-run in CI
x402 test https://api.example.com/endpoint \
  --keystore ./test-wallet.json \
  --dry-run \
  --skip-payment-confirmation
```

### Batch Testing Script

```bash
#!/bin/bash
for url in "https://api1.example.com" "https://api2.example.com"; do
  x402 test "$url" \
    --keystore ~/.foundry/keystores/test \
    --max-amount 0.01 \
    --skip-payment-confirmation
done
```

### Getting Test USDC

1. Go to [Circle's Faucet](https://faucet.circle.com)
2. Select **Base Sepolia**
3. Choose **USDC**
4. Enter your wallet address

## How x402 Works

The x402 protocol enables payment-gated APIs using HTTP 402 responses and gasless EIP-3009 token transfers.

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
| **HTTP 402** | Status code indicating payment is required |
| **EIP-3009** | Gasless token transfer standard (no ETH needed) |
| **Facilitator** | Service that verifies and settles payments on-chain |
| **X-PAYMENT** | Header containing the signed payment authorization |

### Learn More

- [x402 Protocol](https://github.com/coinbase/x402)
- [Coinbase x402 Docs](https://docs.cdp.coinbase.com/x402/welcome)
- [A2A Protocol](https://a2a-protocol.org) — Agent-to-Agent discovery standard
- [EIP-3009 Spec](https://eips.ethereum.org/EIPS/eip-3009)
- [EIP-712 Spec](https://eips.ethereum.org/EIPS/eip-712)

## Creating a Wallet

The `test` command requires a wallet. We recommend using a **keystore file**.

### Using Foundry (Recommended)

```bash
# Install Foundry
curl -L https://foundry.paradigm.xyz | bash
foundryup

# Create a new keystore
cast wallet new ~/.foundry/keystores/

# Or import existing key
cast wallet import my-wallet --interactive
```

### Using Geth

```bash
geth account new --keystore ~/.keystores/
```

> **Security:** Never share your keystore or password. Use a dedicated wallet with only test funds.

## Development

### Building from Source

```bash
git clone https://github.com/port402/x402-cli.git
cd x402-cli
make build
```

### Make Targets

| Target | Description |
|--------|-------------|
| `make build` | Build the binary |
| `make test` | Run tests |
| `make lint` | Run linter |
| `make check` | Run all checks |
| `make help` | Show all targets |

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Security

Report vulnerabilities via [SECURITY.md](SECURITY.md). Do not open public issues for security concerns.

## License

MIT — see [LICENSE](LICENSE)
