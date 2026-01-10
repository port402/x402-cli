package wallet

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	"github.com/port402/x402-cli/internal/x402"
)

// SignParams contains parameters for signing a TransferWithAuthorization.
type SignParams struct {
	ChainID        int64  // EVM chain ID (e.g., 84532 for Base Sepolia)
	TokenAddress   string // ERC-20 token contract address
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
	Signature     string            // Hex-encoded signature with 0x prefix
	Authorization x402.Authorization // Authorization struct for payload
	Nonce         string            // Hex-encoded nonce with 0x prefix
}

// SignTransferAuthorization creates an EIP-712 signature for EIP-3009 TransferWithAuthorization.
//
// This enables gasless token transfers: the signer authorizes a transfer off-chain,
// and a third party (the facilitator) executes it on-chain, paying the gas.
func SignTransferAuthorization(key *ecdsa.PrivateKey, params SignParams) (*SignResult, error) {
	// Generate random nonce (32 bytes)
	nonceBytes := make([]byte, 32)
	if _, err := rand.Read(nonceBytes); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	nonce := common.BytesToHash(nonceBytes)

	// Calculate validBefore
	validBefore := params.ValidBefore
	if validBefore == 0 {
		timeout := params.TimeoutSeconds
		if timeout == 0 {
			timeout = 300 // Default 5 minutes
		}
		validBefore = time.Now().Unix() + int64(timeout)
	}

	// Parse value as big.Int
	value := new(big.Int)
	value.SetString(params.Value, 10)

	// Build EIP-712 typed data
	typedData := buildTypedData(params, nonce, validBefore, value)

	// Hash the typed data (EIP-712)
	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return nil, fmt.Errorf("failed to hash domain: %w", err)
	}

	messageHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return nil, fmt.Errorf("failed to hash message: %w", err)
	}

	// Construct final hash: keccak256("\x19\x01" || domainSeparator || messageHash)
	rawData := []byte{0x19, 0x01}
	rawData = append(rawData, domainSeparator...)
	rawData = append(rawData, messageHash...)
	hash := crypto.Keccak256Hash(rawData)

	// Sign the hash
	signature, err := crypto.Sign(hash.Bytes(), key)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}

	// Adjust v value for Ethereum (add 27)
	if signature[64] < 27 {
		signature[64] += 27
	}

	return &SignResult{
		Signature: "0x" + common.Bytes2Hex(signature),
		Authorization: x402.Authorization{
			From:        params.From,
			To:          params.To,
			Value:       params.Value,
			ValidAfter:  fmt.Sprintf("%d", params.ValidAfter),
			ValidBefore: fmt.Sprintf("%d", validBefore),
			Nonce:       "0x" + common.Bytes2Hex(nonce.Bytes()),
		},
		Nonce: "0x" + common.Bytes2Hex(nonce.Bytes()),
	}, nil
}

// buildTypedData constructs the EIP-712 typed data for TransferWithAuthorization.
func buildTypedData(params SignParams, nonce common.Hash, validBefore int64, value *big.Int) apitypes.TypedData {
	return apitypes.TypedData{
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
			Name:              params.TokenName,
			Version:           params.TokenVersion,
			ChainId:           math.NewHexOrDecimal256(params.ChainID),
			VerifyingContract: params.TokenAddress,
		},
		Message: apitypes.TypedDataMessage{
			"from":        params.From,
			"to":          params.To,
			"value":       value.String(),
			"validAfter":  fmt.Sprintf("%d", params.ValidAfter),
			"validBefore": fmt.Sprintf("%d", validBefore),
			"nonce":       nonce.Hex(),
		},
	}
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
