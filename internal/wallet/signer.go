package wallet

import (
	"crypto/ecdsa"

	"github.com/port402/x402-cli/internal/x402"
)

// Signer is the interface for signing payment authorizations.
// Implementations exist for different blockchain networks (EVM, Solana, etc.).
type Signer interface {
	// Sign creates a signature for the given payment parameters.
	// Returns the signature and authorization details needed for the payment payload.
	Sign(params SignParams) (*SignResult, error)

	// Address returns the signer's address in the appropriate format for the chain.
	Address() string
}

// SignParams contains parameters for signing a payment authorization.
// Supports both EVM (EIP-3009) and Solana payment parameters.
// Different signers use different subsets of these fields.
type SignParams struct {
	// Common fields (all chains)
	TokenAddress   string // Token contract/mint address
	From           string // Payer address (signer)
	To             string // Recipient address
	Value          string // Amount in atomic units
	TimeoutSeconds int    // Timeout for payment validity

	// EVM-specific fields
	ChainID      int64  // EVM chain ID (e.g., 84532 for Base Sepolia)
	TokenName    string // Token name for EIP-712 domain (e.g., "USDC")
	TokenVersion string // Token version for EIP-712 domain (e.g., "2")
	ValidAfter   int64  // Unix timestamp, usually 0
	ValidBefore  int64  // Unix timestamp for expiration

	// Solana-specific fields
	FeePayer string // Fee payer public key (facilitator)
}

// SignResult contains the signature and authorization details.
type SignResult struct {
	// Signature is the signed authorization.
	// Format varies by chain:
	//   - EVM: Hex-encoded signature with 0x prefix
	//   - Solana: Base64-encoded partially-signed transaction
	Signature     string
	Authorization x402.Authorization // Authorization struct for payload
	// Nonce is the transaction nonce.
	// Format varies by chain:
	//   - EVM: Hex-encoded nonce with 0x prefix
	//   - Solana: Base58-encoded recent blockhash
	Nonce string
}

// PrepareSignParams builds SignParams from payment requirement and signer address.
func PrepareSignParams(option *x402.PaymentRequirement, fromAddress string, chainID int64) SignParams {
	// Get token name and version from extra field
	tokenName := option.GetExtraString("name")
	if tokenName == "" {
		tokenName = "USDC" // Default fallback
	}

	tokenVersion := option.GetExtraString("version")
	if tokenVersion == "" {
		tokenVersion = "2" // Default fallback
	}

	return SignParams{
		ChainID:        chainID,
		TokenAddress:   option.Asset,
		TokenName:      tokenName,
		TokenVersion:   tokenVersion,
		From:           fromAddress,
		To:             option.PayTo,
		Value:          option.GetAmount(),
		ValidAfter:     0,
		ValidBefore:    0, // Will be calculated from TimeoutSeconds
		TimeoutSeconds: option.MaxTimeoutSeconds,
	}
}

// SignTransferAuthorization creates an EIP-712 signature for EIP-3009 TransferWithAuthorization.
// This is a convenience function that creates an EVMSigner and signs the authorization.
// It maintains backward compatibility with existing code.
func SignTransferAuthorization(key *ecdsa.PrivateKey, params SignParams) (*SignResult, error) {
	signer := NewEVMSigner(key)
	return signer.Sign(params)
}
