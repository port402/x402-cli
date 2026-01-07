# x402-cli Specification

A command-line tool for testing x402-enabled APIs, written in Go.

## Overview

x402-cli provides utilities to test and debug x402 payment-gated API endpoints. It handles the complete payment flow including health checks, payment negotiation, EIP-3009 signature generation, and response verification.

**Key Insight:** The x402 protocol uses **gasless payments** via EIP-3009 `TransferWithAuthorization`. The client only signs an off-chain authorization message—the facilitator (Coinbase's service) executes the actual blockchain transaction. This means:
- No gas fees for the client
- No ERC20 approval transactions needed
- No waiting for on-chain confirmations

---

## x402 Protocol Reference

The CLI supports both x402 **v1** and **v2** protocols with automatic detection.

### Version Detection

| Check | v1 | v2 |
|-------|----|----|
| 402 response | Payment info in **body** | `PAYMENT-REQUIRED` **header** |
| Version field | `x402Version: 1` | `x402Version: 2` |

### HTTP Headers

**v2 Headers (current standard):**

| Header | Direction | Description |
|--------|-----------|-------------|
| `PAYMENT-REQUIRED` | Server → Client | Base64-encoded `PaymentRequired` JSON |
| `PAYMENT-SIGNATURE` | Client → Server | Base64-encoded `PaymentPayload` JSON |
| `PAYMENT-RESPONSE` | Server → Client | Base64-encoded `SettlementResponse` JSON |

**v1 Headers (legacy):**

| Header | Direction | Description |
|--------|-----------|-------------|
| *(body)* | Server → Client | JSON `PaymentRequirementsResponse` in 402 body |
| `X-PAYMENT` | Client → Server | Base64-encoded `PaymentPayload` JSON |
| `X-PAYMENT-RESPONSE` | Server → Client | Base64-encoded `SettlementResponse` JSON |

### Auto-Detection Logic

```
1. Send initial request
2. Receive 402 response
3. Check for PAYMENT-REQUIRED header:
   - Present → use v2 flow
   - Absent → parse body as JSON, use v1 flow
4. Send payment with appropriate header (PAYMENT-SIGNATURE or X-PAYMENT)
5. Parse response with appropriate header (PAYMENT-RESPONSE or X-PAYMENT-RESPONSE)
```

### PAYMENT-REQUIRED Payload (402 Response)

```json
{
  "x402Version": 2,
  "error": "Payment required to access this resource",
  "resource": {
    "url": "/api/endpoint",
    "description": "Premium API access",
    "mimeType": "application/json"
  },
  "accepts": [
    {
      "scheme": "exact",
      "network": "eip155:8453",
      "amount": "1000000",
      "asset": "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
      "payTo": "0x...",
      "maxTimeoutSeconds": 300,
      "extra": {}
    }
  ],
  "extensions": {}
}
```

### PAYMENT-SIGNATURE Payload (Client Retry)

```json
{
  "x402Version": 2,
  "scheme": "exact",
  "network": "eip155:8453",
  "payload": {
    "signature": "0x...",
    "from": "0x...",
    "to": "0x...",
    "value": "1000000",
    "validAfter": 0,
    "validBefore": 1704067200,
    "nonce": "0x..."
  }
}
```

### PAYMENT-RESPONSE Payload (Success)

```json
{
  "success": true,
  "transaction": "0x...",
  "network": "eip155:8453",
  "payer": "0x..."
}
```

### Network Identifiers (CAIP-2 Format)

| Network | Identifier |
|---------|------------|
| Base Mainnet | `eip155:8453` |
| Base Sepolia | `eip155:84532` |
| Ethereum Mainnet | `eip155:1` |
| Solana Mainnet | `solana:5eykt4UsFv8P8NJdTREpY1vzqKqZKvdp` |

### EIP-3009 TransferWithAuthorization

The "exact" scheme uses EIP-3009 for gasless token transfers. The client signs an EIP-712 typed message authorizing the facilitator to transfer tokens on their behalf:

```
TransferWithAuthorization(
  from: address,      // Payer wallet
  to: address,        // Payment recipient
  value: uint256,     // Amount in atomic units
  validAfter: uint256,  // Unix timestamp (0 = immediate)
  validBefore: uint256, // Unix timestamp (expiration)
  nonce: bytes32      // Random value for replay prevention
)
```

See: [x402 Protocol Specification](https://github.com/coinbase/x402) | [Coinbase x402 Docs](https://docs.cdp.coinbase.com/x402/welcome)

---

## Payment Flow Diagram

```
┌──────────────────────────────────────────────────────────────────────────┐
│                        x402 test <url> --wallet ...                       │
└──────────────────────────────────────────────────────────────────────────┘
                                     │
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│ 1. INITIALIZATION                                                         │
│    • Load private key (flag/env/keystore/stdin)                          │
│    • Validate key format, derive address                                 │
└──────────────────────────────────────────────────────────────────────────┘
                                     │
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│ 2. INITIAL REQUEST                                                        │
│    GET/POST https://api.example.com/endpoint                             │
│    Content-Type: application/json                                         │
│    Body: {"input": "..."}                                                 │
└──────────────────────────────────────────────────────────────────────────┘
                                     │
                         ┌───────────┴───────────┐
                         ▼                       ▼
                   ┌──────────┐            ┌──────────┐
                   │   200    │            │   402    │
                   │ Success  │            │ Payment  │
                   └──────────┘            │ Required │
                         │                 └──────────┘
                         ▼                       │
                ┌─────────────────┐              │
                │ ⚠ WARN: No      │              │
                │ payment needed  │              │
                │ Show response   │              │
                │ Exit 0          │              │
                └─────────────────┘              │
                                                 ▼
┌──────────────────────────────────────────────────────────────────────────┐
│ 3. PARSE PAYMENT-REQUIRED HEADER                                          │
│    • Decode base64 → JSON                                                 │
│    • Extract accepts[] array with payment options                        │
│    • Validate x402Version == 2                                           │
└──────────────────────────────────────────────────────────────────────────┘
                                     │
                                     ▼
                       ┌─────────────────────────┐
                       │ Multiple payment        │
                       │ options in accepts[]?   │
                       └─────────────────────────┘
                          │              │
                         Yes             No
                          │              │
                          ▼              │
                 ┌─────────────────┐     │
                 │ Prompt user to  │     │
                 │ select option   │     │
                 └─────────────────┘     │
                          │              │
                          └──────┬───────┘
                                 ▼
┌──────────────────────────────────────────────────────────────────────────┐
│ 4. SIGN EIP-3009 AUTHORIZATION (off-chain, no gas)                        │
│    • Generate random 32-byte nonce                                        │
│    • Set validAfter = 0 (immediate)                                       │
│    • Set validBefore = now + maxTimeoutSeconds                           │
│    • Construct EIP-712 typed data message                                │
│    • Sign with wallet private key                                         │
└──────────────────────────────────────────────────────────────────────────┘
                                     │
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│ 5. RETRY WITH PAYMENT-SIGNATURE HEADER                                    │
│    GET/POST https://api.example.com/endpoint                             │
│    PAYMENT-SIGNATURE: <base64 PaymentPayload>                            │
│    Content-Type: application/json                                         │
│    Body: {"input": "..."}                                                 │
└──────────────────────────────────────────────────────────────────────────┘
                                     │
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│ 6. SERVER-SIDE (transparent to CLI)                                       │
│    • Server calls facilitator POST /verify                               │
│    • Facilitator validates signature                                      │
│    • Server calls facilitator POST /settle                               │
│    • Facilitator executes on-chain tx, waits for confirmation            │
│    • Server returns resource with PAYMENT-RESPONSE header                │
└──────────────────────────────────────────────────────────────────────────┘
                                     │
                         ┌───────────┴───────────┐
                         ▼                       ▼
                   ┌──────────┐            ┌──────────┐
                   │   200    │            │   402    │
                   │ Success  │            │  Again   │
                   └──────────┘            └──────────┘
                         │                       │
                         ▼                       ▼
┌─────────────────────────────────┐  ┌────────────────────────────────────┐
│ 7a. SUCCESS OUTPUT              │  │ 7b. PAYMENT REJECTED               │
│ Parse PAYMENT-RESPONSE header   │  │ Parse error from response          │
│ • Transaction hash              │  │ • Show errorReason                 │
│ • Amount paid (human + raw)     │  │ • Suggest checking wallet balance  │
│ • Network                       │  │ Exit 1                             │
│ • Response body                 │  └────────────────────────────────────┘
│ Exit 0                          │
└─────────────────────────────────┘
```

---

## Commands

### `x402 health <url>`

Health check for x402-enabled endpoints with full discovery.

**Behavior:**
- HTTP connectivity check (status code, response time)
- Parse 402 response if returned (decode `PAYMENT-REQUIRED` header)
- Check `/.well-known/x402` discovery endpoint if available
- Validate facilitator reachability (GET /supported)
- Warn if endpoint returns 200 (no payment required)

**Output:**
- Status code and latency
- Payment requirements (if 402): scheme, network, amount, asset, payTo
- Supported networks/schemes from facilitator
- Discovery endpoint data (if available)

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
| `--timeout <seconds>` | Operation timeout (default: 120) | No |
| `--verbose` | Show detailed payment negotiation | No |
| `--json` | Machine-readable JSON output | No |

\* One of `--wallet`, `--keystore`, or stdin required

**Payment Flow:**
1. Send initial HTTP request to endpoint
2. Receive 402 response with `PAYMENT-REQUIRED` header
3. Decode base64 JSON, extract payment options from `accepts[]`
4. If multiple options, prompt user to select
5. Sign EIP-3009 `TransferWithAuthorization` message (off-chain, no gas)
6. Retry request with `PAYMENT-SIGNATURE` header
7. Parse `PAYMENT-RESPONSE` header from success response
8. Display results

**Verbose Mode (`--verbose`):**
Outputs to stderr:
- Detected protocol version (v1 or v2)
- Full HTTP request/response headers (both directions)
- Decoded payment requirements JSON (pretty-printed)
- Payment option selection
- EIP-712 typed data being signed
- Generated signature
- Retry request with payment header
- Decoded settlement response JSON

**Success Output:**
- Payment summary: tx hash, amount (human readable + atomic units), network
- Response status and latency
- Response body

**Error Scenarios:**
- Non-402 initial response: Warn and show response
- Invalid signature: Show facilitator error reason
- Insufficient balance: `insufficient_funds` error from facilitator
- Expired authorization: `invalid_exact_evm_payload_authorization_valid_before` error
- Double 402 after signature: Show error reason from response

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
| `PRIVATE_KEY` | Default wallet private key |

Flags always override environment variables.

### Wallet Input Methods

1. **Flag:** `--wallet 0x...` (private key in hex)
2. **Environment:** `PRIVATE_KEY` env var
3. **Keystore:** `--keystore /path/to/keystore.json` (Web3 Secret Storage format, prompts for password)
4. **Stdin:** Pipe private key via stdin

**Security:**
- Early validation of key format before any operations
- Derive and display address for user confirmation
- Warning displayed when reading key from interactive TTY via stdin

---

## Multi-Chain Support

The CLI supports any network specified in the 402 response using CAIP-2 identifiers.

**Network Detection:**
- Network identifier from `accepts[].network` field (e.g., `eip155:8453`)
- No RPC URL needed—client only signs messages (no on-chain interaction)
- Facilitator handles all blockchain communication

**Supported Networks:** Any network the facilitator supports. Query facilitator's `GET /supported` endpoint.

---

## Token Support

**Primary:** USDC (identified by contract address in `accepts[].asset`)

**No Approval Needed:** EIP-3009 `TransferWithAuthorization` is a gasless mechanism that doesn't require prior ERC20 approval. The client signs an authorization, and the facilitator calls the token contract's `transferWithAuthorization` function.

**Amount Display:** Both human-readable (e.g., `1.00 USDC`) and raw atomic units (e.g., `1000000`)

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
- **Ethereum Signing:** go-ethereum/crypto (for EIP-712 signing)
- **Keystore:** Web3 Secret Storage (geth-compatible)
- **HTTP:** net/http standard library

### EIP-712 Signing

The CLI must implement EIP-712 typed data signing for the `TransferWithAuthorization` message:

```go
typedData := apitypes.TypedData{
    Types: apitypes.Types{
        "EIP712Domain": {
            {Name: "name", Type: "string"},
            {Name: "version", Type: "string"},
            {Name: "chainId", Type: "uint256"},
            {Name: "verifyingContract", Type: "address"},
        },
        "TransferWithAuthorization": {
            {Name: "from", Type: "address"},
            {Name: "to", Type: "address"},
            {Name: "value", Type: "uint256"},
            {Name: "validAfter", Type: "uint256"},
            {Name: "validBefore", Type: "uint256"},
            {Name: "nonce", Type: "bytes32"},
        },
    },
    PrimaryType: "TransferWithAuthorization",
    Domain: apitypes.TypedDataDomain{
        Name:              "USD Coin",  // Token name
        Version:           "2",
        ChainId:           math.NewHexOrDecimal256(8453),
        VerifyingContract: "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
    },
    Message: map[string]interface{}{
        "from":        walletAddress,
        "to":          payTo,
        "value":       amount,
        "validAfter":  0,
        "validBefore": validBefore,
        "nonce":       randomNonce,
    },
}
```

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
│   ├── client/        # HTTP client with x402 header handling
│   ├── wallet/        # Key loading, EIP-712 signing
│   ├── x402/          # PaymentRequired/PaymentPayload types
│   ├── discovery/     # .well-known/x402 parsing
│   └── output/        # Formatting, TTY detection
├── main.go
├── go.mod
└── go.sum
```

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

# Using environment variable
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
| No 402 response | `warning: endpoint returned 200 OK (no payment required)` |
| Invalid x402 version | `error: unsupported x402 version (expected 2, got X)` |
| Signature rejected | `error: payment signature rejected: <errorReason from facilitator>` |
| Insufficient balance | `error: insufficient_funds - wallet has insufficient token balance` |
| Authorization expired | `error: authorization expired before facilitator could settle` |
| Network unsupported | `error: network eip155:XXXX not supported by facilitator` |
| Timeout | `error: operation timed out after Xs` |
| Malformed header | `error: failed to decode PAYMENT-REQUIRED header: <details>` |

---

## Removed Features (vs Initial Draft)

The following features from the initial draft are **not needed** due to the gasless EIP-3009 architecture:

| Feature | Reason Removed |
|---------|----------------|
| `--rpc-url` flag | Client doesn't interact with blockchain |
| `--facilitator` flag | Facilitator URL derived from server, not user-configured |
| `--auto-approve` flag | No ERC20 approval needed with EIP-3009 |
| `ETH_RPC_URL` env var | No RPC needed |
| Gas estimation | Facilitator pays gas |
| Nonce management | No on-chain tx from client |
| Confirmation waiting | Facilitator handles settlement |
| ERC20 allowance checks | TransferWithAuthorization doesn't need approval |
