# x402-cli Specification

A command-line tool for testing x402-enabled APIs, written in Go.

## Overview

x402-cli provides utilities to test and debug x402 payment-gated API endpoints. It handles the complete payment flow including health checks, payment negotiation, transaction execution, and response verification.

## Commands

### `x402 health <url>`

Health check for x402-enabled endpoints with full discovery.

**Behavior:**
- HTTP connectivity check (status code, response time)
- Parse 402 response if returned (payment requirements, accepted tokens, facilitator)
- Check `/.well-known/x402` discovery endpoint if available
- Validate facilitator contract reachability
- Warn if endpoint returns 200 (no payment required)

**Output:**
- Status code and latency
- Agent metadata (from x402 headers)
- Payment requirements (if 402)
- Discovery endpoint data (if available)
- Facilitator status

---

### `x402 test <url>`

Execute full payment flow test against an x402-enabled API.

**Flags:**
| Flag | Description | Required |
|------|-------------|----------|
| `--wallet <key>` | Private key (hex) | Yes* |
| `--keystore <file>` | Path to Web3 Secret Storage keystore file | Yes* |
| `--data <json>` | Request body JSON | No |
| `--method <method>` | HTTP method (GET/POST), auto-POST if --data provided | No |
| `--rpc-url <url>` | RPC endpoint URL | No** |
| `--facilitator <addr>` | Override facilitator address from 402 response | No |
| `--timeout <seconds>` | Operation timeout (default: 120) | No |
| `--auto-approve` | Skip ERC20 approval prompt | No |
| `--verbose` | Show detailed payment negotiation | No |
| `--json` | Machine-readable JSON output | No |

\* One of `--wallet`, `--keystore`, or stdin required
\** Uses `ETH_RPC_URL` env var if not provided

**Payment Flow:**
1. Send initial HTTP request to endpoint
2. Receive 402 response with payment requirements
3. If multiple payment options, prompt user to select (unless only one matches)
4. Check ERC20 allowance for facilitator
5. If approval needed, prompt user (unless `--auto-approve`)
6. Execute payment transaction via facilitator contract
7. Wait for 2 block confirmations
8. Retry original request with payment proof header (per x402 spec)
9. Return response

**Verbose Mode (`--verbose`):**
Outputs to stderr:
- Full HTTP request/response headers (both directions)
- 402 payment requirements parsed
- Payment option selection
- ERC20 approval transaction (if applicable)
- Payment transaction details
- Token exchange details (human readable + raw units)
- Confirmation wait status
- Retry request with proof

**Success Output:**
- Payment summary (tx hash, amount in both formats, gas used)
- Response status and latency
- Response body

**Error Scenarios:**
- Non-402 initial response: Warn and show response
- Insufficient balance: Transaction fails with decoded revert reason
- Double 402 after proof: Error with tx hash, suggest checking facilitator for refund
- Transaction reverts: Best-effort decode of revert reason, fall back to raw data

---

### `x402 batch-health <file>`

Batch health check from JSON file.

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--parallel <n>` | Concurrent checks | 1 |
| `--delay <ms>` | Pause between batches (ms) | 0 |
| `--fail-fast` | Exit on first failure | false |
| `--json` | JSON output | false |

**File Format:**
```json
["https://api1.example.com", "https://api2.example.com"]
```

**Behavior:**
- Run health checks with specified parallelism
- Rate limiting via `--delay` between batches
- Default: complete all checks, report summary
- With `--fail-fast`: exit immediately on first failure
- Exit code 1 if any check fails

---

### `x402 version`

Display version and check for updates.

**Behavior:**
- Show current version
- Check GitHub releases for newer versions
- Display update notice if available

---

## Configuration

### Environment Variables

| Variable | Description |
|----------|-------------|
| `ETH_RPC_URL` | Default RPC endpoint URL |
| `PRIVATE_KEY` | Default wallet private key |

Flags always override environment variables.

### Wallet Input Methods

1. **Flag:** `--wallet 0x...` (private key in hex)
2. **Environment:** `PRIVATE_KEY` env var
3. **Keystore:** `--keystore /path/to/keystore.json` (Web3 Secret Storage format, prompts for password)
4. **Stdin:** Pipe private key via stdin

**Security:**
- Early validation of key format before network calls
- Derive and display address for user confirmation
- Warning displayed when reading key from interactive TTY via stdin

---

## Multi-Chain Support

The CLI supports any EVM-compatible chain via RPC URL configuration.

**Network Detection:**
- Chain ID detected from RPC endpoint
- Facilitator address from 402 response (or `--facilitator` override)
- No hardcoded network assumptions

---

## Token Support

**Primary:** USDC

**ERC20 Approval Flow:**
1. Check current allowance for facilitator address
2. If insufficient, prompt: `Approval needed for X.XX USDC. Proceed? [y/N]`
3. Send approval for exact payment amount
4. Wait for confirmation
5. Proceed with payment

Use `--auto-approve` to skip interactive prompt.

---

## Transaction Handling

**Gas:** Automatic estimation via `eth_estimateGas` and current gas price

**Nonce:** Auto-detect pending transaction count, queue behind any pending transactions

**Confirmations:** Wait for 2 block confirmations before retry with proof

**Amount Display:** Both human-readable (e.g., `0.001 USDC`) and raw atomic units

---

## Output Behavior

### TTY Detection (Pipe-Friendly)

When stdout is not a TTY:
- Output only response body (no summary/decoration)
- Suitable for piping to `jq` or other tools

When stdout is a TTY:
- Full summary + response body output

### Streams

- **stdout:** Results, response body
- **stderr:** Verbose/debug logs, warnings, prompts

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Any failure |

---

## Technical Implementation

### Framework & Libraries

- **CLI Framework:** Cobra
- **Ethereum:** go-ethereum (ethclient)
- **Keystore:** Web3 Secret Storage (geth-compatible)

### Project Structure

```
x402-cli/
├── cmd/
│   ├── root.go
│   ├── health.go
│   ├── test.go
│   ├── batch_health.go
│   └── version.go
├── internal/
│   ├── client/        # HTTP client with x402 handling
│   ├── wallet/        # Key loading, signing
│   ├── payment/       # Facilitator interaction, tx execution
│   ├── discovery/     # .well-known/x402 parsing
│   └── output/        # Formatting, TTY detection
├── main.go
├── go.mod
└── go.sum
```

---

## Protocol Reference

This CLI implements the x402 payment protocol. For protocol details including:
- Payment requirement header format
- Payment proof header format
- Facilitator contract ABI
- Token exchange mechanics

See: [x402 Protocol Specification](https://github.com/coinbase/x402)

---

## Example Usage

```bash
# Health check single endpoint
x402 health https://api.example.com/paid-endpoint

# Full payment test
x402 test https://api.example.com/paid-endpoint \
  --wallet 0xabc123... \
  --data '{"prompt": "Hello"}' \
  --verbose

# Using keystore file
x402 test https://api.example.com/paid-endpoint \
  --keystore ~/.ethereum/keystore/UTC--2024... \
  --data '{"input": "test"}'

# Using environment variables
export ETH_RPC_URL=https://base-mainnet.infura.io/v3/...
export PRIVATE_KEY=0xabc123...
x402 test https://api.example.com/paid-endpoint

# Batch health check
x402 batch-health services.json --parallel 10 --delay 100

# JSON output for CI
x402 test https://api.example.com/paid-endpoint \
  --wallet 0xabc123... \
  --json

# Pipe response to jq
x402 test https://api.example.com/paid-endpoint \
  --wallet 0xabc123... | jq '.result'

# Check version
x402 version
```

---

## Error Messages

The CLI provides clear, actionable error messages:

| Scenario | Message |
|----------|---------|
| Invalid private key | `error: invalid private key format (expected 64 hex chars or 0x-prefixed)` |
| Missing RPC URL | `error: RPC URL required (--rpc-url or ETH_RPC_URL env var)` |
| Insufficient balance | `error: insufficient USDC balance (have: X.XX, need: Y.YY)` |
| Approval rejected | `error: ERC20 approval cancelled by user` |
| Payment not recognized | `error: payment not recognized by server (tx: 0x...). Check facilitator for potential refund.` |
| Contract revert | `error: facilitator contract reverted: <decoded reason or raw data>` |
| Timeout | `error: operation timed out after Xs` |
