package x402

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// caip2EVMPrefix is the prefix for EVM chains in CAIP-2 format.
const caip2EVMPrefix = "eip155:"

// caip2SolanaPrefix is the prefix for Solana chains in CAIP-2 format.
const caip2SolanaPrefix = "solana:"

// Solana network identifiers (CAIP-2 format uses genesis hash)
const (
	// SolanaMainnet is the CAIP-2 identifier for Solana mainnet-beta
	SolanaMainnet = "solana:5eykt4UsFv8P8NJdTREpY1vzqKqZKvdp"
	// SolanaDevnet is the CAIP-2 identifier for Solana devnet
	SolanaDevnet = "solana:EtWTRABZaYq6iMfeYKouRu166VU2xqa1"
	// SolanaTestnet is the CAIP-2 identifier for Solana testnet
	SolanaTestnet = "solana:4uhcVJyU9pJkvQyS88uRDiswHXSCkY3z"
)

// solanaNetworkAliases maps common Solana network names to their CAIP-2 identifiers.
// This handles v1 protocol which may use simple names instead of CAIP-2 format.
var solanaNetworkAliases = map[string]string{
	// Mainnet aliases
	"solana":              SolanaMainnet,
	"solana-mainnet":      SolanaMainnet,
	"solana-mainnet-beta": SolanaMainnet,
	"mainnet-beta":        SolanaMainnet,
	// Devnet aliases
	"solana-devnet": SolanaDevnet,
	"devnet":        SolanaDevnet,
	// Testnet aliases
	"solana-testnet": SolanaTestnet,
	"testnet":        SolanaTestnet,
}

// ParseResult contains the parsed payment requirements and metadata.
type ParseResult struct {
	PaymentRequired *PaymentRequired
	ProtocolVersion int
	RawHeader       string // Original header value (v2) or empty (v1)
	RawBody         []byte // Response body
}

// ParsePaymentRequired extracts payment requirements from a 402 response.
// Auto-detects v1 vs v2 based on the presence of the Payment-Required header.
//
// Protocol detection:
//   - v2: Payment-Required header present (base64 encoded JSON)
//   - v1: No header, payment requirements in response body (plain JSON)
func ParsePaymentRequired(resp *http.Response) (*ParseResult, error) {
	result := &ParseResult{}

	// Read body (needed for v1, useful for debugging v2)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	result.RawBody = body

	// Check for v2 header
	paymentRequiredHeader := resp.Header.Get(HeaderPaymentRequired)

	if paymentRequiredHeader != "" {
		// v2: Decode base64 header
		result.ProtocolVersion = ProtocolV2
		result.RawHeader = paymentRequiredHeader

		decoded, err := base64.StdEncoding.DecodeString(paymentRequiredHeader)
		if err != nil {
			return nil, fmt.Errorf("invalid base64 in %s header: %w", HeaderPaymentRequired, err)
		}

		var pr PaymentRequired
		if err := json.Unmarshal(decoded, &pr); err != nil {
			return nil, fmt.Errorf("invalid JSON in %s header: %w", HeaderPaymentRequired, err)
		}
		result.PaymentRequired = &pr
	} else {
		// v1: Parse body as JSON
		result.ProtocolVersion = ProtocolV1

		if len(body) == 0 {
			return nil, fmt.Errorf("empty response body (expected JSON payment requirements)")
		}

		var pr PaymentRequired
		if err := json.Unmarshal(body, &pr); err != nil {
			return nil, fmt.Errorf("invalid JSON in response body: %w", err)
		}
		result.PaymentRequired = &pr
	}

	// Validate we have payment options
	if len(result.PaymentRequired.Accepts) == 0 {
		return nil, fmt.Errorf("no payment options in accepts[] array")
	}

	return result, nil
}

// ParsePaymentResponse extracts the payment response from a successful response.
// Checks the appropriate header based on protocol version.
func ParsePaymentResponse(resp *http.Response, protocolVersion int) (*PaymentResponse, error) {
	var headerName string
	if protocolVersion == ProtocolV2 {
		headerName = HeaderPaymentResponse
	} else {
		headerName = HeaderXPaymentResponse
	}

	headerValue := resp.Header.Get(headerName)
	if headerValue == "" {
		// No payment response header - may still be success
		return nil, nil
	}

	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(headerValue)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 in %s header: %w", headerName, err)
	}

	var pr PaymentResponse
	if err := json.Unmarshal(decoded, &pr); err != nil {
		return nil, fmt.Errorf("invalid JSON in %s header: %w", headerName, err)
	}

	return &pr, nil
}

// networkNameToChainID maps common network names to their chain IDs.
// This handles v1 protocol which may use simple names instead of CAIP-2 format.
var networkNameToChainID = map[string]int64{
	// Mainnets
	"ethereum":  1,
	"mainnet":   1,
	"base":      8453,
	"polygon":   137,
	"arbitrum":  42161,
	"optimism":  10,
	"avalanche": 43114,
	"bsc":       56,
	// Testnets
	"sepolia":      11155111,
	"goerli":       5,
	"base-sepolia": 84532,
	"base_sepolia": 84532,
	"basesepolia":  84532,
	"mumbai":       80001,
}

// IsEVMNetwork checks if the network is an EVM-compatible chain.
// Supports both CAIP-2 format (eip155:*) and common network names.
func IsEVMNetwork(network string) bool {
	// Check CAIP-2 format (must have content after prefix)
	if strings.HasPrefix(network, caip2EVMPrefix) && len(network) > len(caip2EVMPrefix) {
		return true
	}
	_, ok := networkNameToChainID[network]
	return ok
}

// ExtractChainID extracts the numeric chain ID from a network string.
// Supports both CAIP-2 format (eip155:8453) and common names (base).
func ExtractChainID(network string) (int64, error) {
	// Try CAIP-2 format first
	if strings.HasPrefix(network, caip2EVMPrefix) {
		var chainID int64
		_, err := fmt.Sscanf(network, caip2EVMPrefix+"%d", &chainID)
		if err != nil {
			return 0, fmt.Errorf("invalid chain ID in network %s: %w", network, err)
		}
		return chainID, nil
	}

	// Try known network names
	if chainID, ok := networkNameToChainID[network]; ok {
		return chainID, nil
	}

	return 0, fmt.Errorf("unknown network: %s", network)
}

// FindEVMOption returns the first EVM-compatible payment option.
// Returns nil if no EVM options are available.
func FindEVMOption(pr *PaymentRequired) *PaymentRequirement {
	for i := range pr.Accepts {
		if IsEVMNetwork(pr.Accepts[i].Network) {
			return &pr.Accepts[i]
		}
	}
	return nil
}

// HasOnlySolanaOptions returns true if all payment options are Solana-based.
func HasOnlySolanaOptions(pr *PaymentRequired) bool {
	for _, opt := range pr.Accepts {
		if IsEVMNetwork(opt.Network) {
			return false
		}
	}
	// Check if any options exist and none are EVM
	return len(pr.Accepts) > 0
}

// IsSolanaNetwork checks if the network is a Solana chain.
// Supports both CAIP-2 format (solana:*) and common network names.
func IsSolanaNetwork(network string) bool {
	// Check CAIP-2 format (must have content after prefix)
	if strings.HasPrefix(network, caip2SolanaPrefix) && len(network) > len(caip2SolanaPrefix) {
		return true
	}
	// Check known aliases
	_, ok := solanaNetworkAliases[network]
	return ok
}

// FindSolanaOption returns the first Solana payment option.
// Returns nil if no Solana options are available.
func FindSolanaOption(pr *PaymentRequired) *PaymentRequirement {
	for i := range pr.Accepts {
		if IsSolanaNetwork(pr.Accepts[i].Network) {
			return &pr.Accepts[i]
		}
	}
	return nil
}

// NormalizeSolanaNetwork converts a Solana network name to its CAIP-2 identifier.
// Returns the original string if already in CAIP-2 format or unknown.
func NormalizeSolanaNetwork(network string) string {
	// Already in CAIP-2 format
	if strings.HasPrefix(network, caip2SolanaPrefix) {
		return network
	}
	// Check aliases
	if caip2, ok := solanaNetworkAliases[network]; ok {
		return caip2
	}
	return network
}

// GetSolanaRPCURL returns the appropriate RPC URL for a Solana network.
// Supports both CAIP-2 format and common network names.
func GetSolanaRPCURL(network string) string {
	// Normalize to CAIP-2 format first
	normalized := NormalizeSolanaNetwork(network)

	switch normalized {
	case SolanaMainnet:
		return "https://api.mainnet-beta.solana.com"
	case SolanaDevnet:
		return "https://api.devnet.solana.com"
	case SolanaTestnet:
		return "https://api.testnet.solana.com"
	default:
		// Default to mainnet for unknown networks
		return "https://api.mainnet-beta.solana.com"
	}
}
