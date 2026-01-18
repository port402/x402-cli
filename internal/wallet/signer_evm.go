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

// EVMSigner implements the Signer interface for EVM-compatible chains.
// It uses EIP-712 typed data signing for EIP-3009 TransferWithAuthorization.
type EVMSigner struct {
	privateKey *ecdsa.PrivateKey
}

// NewEVMSigner creates a new EVM signer from an ECDSA private key.
func NewEVMSigner(key *ecdsa.PrivateKey) *EVMSigner {
	return &EVMSigner{privateKey: key}
}

// Sign creates an EIP-712 signature for EIP-3009 TransferWithAuthorization.
// This enables gasless token transfers: the signer authorizes a transfer off-chain,
// and a third party (the facilitator) executes it on-chain, paying the gas.
func (s *EVMSigner) Sign(params SignParams) (*SignResult, error) {
	// Generate random nonce (32 bytes)
	nonceBytes := make([]byte, 32)
	if _, err := rand.Read(nonceBytes); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	nonce := common.BytesToHash(nonceBytes)
	nonceHex := "0x" + common.Bytes2Hex(nonce.Bytes())

	// Calculate validBefore from timeout if not explicitly set
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
	if _, ok := value.SetString(params.Value, 10); !ok {
		return nil, fmt.Errorf("invalid payment value: %q", params.Value)
	}

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
	signature, err := crypto.Sign(hash.Bytes(), s.privateKey)
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
			Nonce:       nonceHex,
		},
		Nonce: nonceHex,
	}, nil
}

// Address returns the Ethereum address for this signer.
func (s *EVMSigner) Address() string {
	return crypto.PubkeyToAddress(s.privateKey.PublicKey).Hex()
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
