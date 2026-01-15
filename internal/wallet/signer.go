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
// Currently supports EVM EIP-3009 TransferWithAuthorization parameters.
// Future implementations may use a subset of these fields.
type SignParams struct {
	ChainID        int64  // Chain ID (e.g., 84532 for Base Sepolia, 1 for Ethereum mainnet)
	TokenAddress   string // Token contract address
	TokenName      string // Token name for EIP-712 domain (e.g., "USDC")
	TokenVersion   string // Token version for EIP-712 domain (e.g., "2")
	From           string // Payer address (signer)
	To             string // Recipient address
	Value          string // Amount in atomic units
	ValidAfter     int64  // Unix timestamp, usually 0
	ValidBefore    int64  // Unix timestamp for expiration
	TimeoutSeconds int    // Added to current time if ValidBefore is 0
}

// SignResult contains the signature and authorization details.
type SignResult struct {
	Signature     string             // Hex-encoded signature with 0x prefix
	Authorization x402.Authorization // Authorization struct for payload
	Nonce         string             // Hex-encoded nonce with 0x prefix
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
