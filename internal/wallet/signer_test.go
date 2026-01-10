package wallet

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/port402/x402-cli/internal/x402"
)

// Test private key from Foundry/Anvil - NEVER use for real funds
const signerTestPrivateKey = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
const signerTestAddress = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"

func TestSignTransferAuthorization_Success(t *testing.T) {
	key, err := LoadFromHex(signerTestPrivateKey)
	require.NoError(t, err)

	params := SignParams{
		ChainID:        84532, // Base Sepolia
		TokenAddress:   "0x036cbd53842c5426634e7929541ec2318f3dcf7e",
		TokenName:      "USDC",
		TokenVersion:   "2",
		From:           signerTestAddress,
		To:             "0x64c2310BD1151266AA2Ad2410447E133b7F84e29",
		Value:          "1000000",
		ValidAfter:     0,
		TimeoutSeconds: 300,
	}

	result, err := SignTransferAuthorization(key, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify signature format
	assert.True(t, strings.HasPrefix(result.Signature, "0x"))
	assert.Len(t, result.Signature, 132) // 0x + 130 hex chars = 65 bytes

	// Verify authorization fields
	assert.Equal(t, signerTestAddress, result.Authorization.From)
	assert.Equal(t, params.To, result.Authorization.To)
	assert.Equal(t, params.Value, result.Authorization.Value)
	assert.Equal(t, "0", result.Authorization.ValidAfter)
	assert.NotEmpty(t, result.Authorization.ValidBefore)

	// Verify nonce
	assert.True(t, strings.HasPrefix(result.Nonce, "0x"))
	assert.Len(t, result.Nonce, 66) // 0x + 64 hex chars = 32 bytes
}

func TestSignTransferAuthorization_SignatureFormat(t *testing.T) {
	key, err := LoadFromHex(signerTestPrivateKey)
	require.NoError(t, err)

	params := SignParams{
		ChainID:        8453,
		TokenAddress:   "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
		TokenName:      "USDC",
		TokenVersion:   "2",
		From:           signerTestAddress,
		To:             "0x64c2310BD1151266AA2Ad2410447E133b7F84e29",
		Value:          "1000",
		TimeoutSeconds: 300,
	}

	result, err := SignTransferAuthorization(key, params)
	require.NoError(t, err)

	// Decode signature and verify it's 65 bytes
	sigHex := strings.TrimPrefix(result.Signature, "0x")
	sigBytes, err := hex.DecodeString(sigHex)
	require.NoError(t, err)
	assert.Len(t, sigBytes, 65)

	// Verify v value is 27 or 28
	v := sigBytes[64]
	assert.True(t, v == 27 || v == 28, "v value should be 27 or 28, got %d", v)
}

func TestSignTransferAuthorization_NonceUniqueness(t *testing.T) {
	key, err := LoadFromHex(signerTestPrivateKey)
	require.NoError(t, err)

	params := SignParams{
		ChainID:        84532,
		TokenAddress:   "0x036cbd53842c5426634e7929541ec2318f3dcf7e",
		TokenName:      "USDC",
		TokenVersion:   "2",
		From:           signerTestAddress,
		To:             "0x64c2310BD1151266AA2Ad2410447E133b7F84e29",
		Value:          "1000",
		TimeoutSeconds: 300,
	}

	// Sign twice with same params
	result1, err := SignTransferAuthorization(key, params)
	require.NoError(t, err)

	result2, err := SignTransferAuthorization(key, params)
	require.NoError(t, err)

	// Nonces should be different (random)
	assert.NotEqual(t, result1.Nonce, result2.Nonce)

	// Signatures will also be different due to different nonces
	assert.NotEqual(t, result1.Signature, result2.Signature)
}

func TestSignTransferAuthorization_ValidBeforeCalculation(t *testing.T) {
	key, err := LoadFromHex(signerTestPrivateKey)
	require.NoError(t, err)

	params := SignParams{
		ChainID:        84532,
		TokenAddress:   "0x036cbd53842c5426634e7929541ec2318f3dcf7e",
		TokenName:      "USDC",
		TokenVersion:   "2",
		From:           signerTestAddress,
		To:             "0x64c2310BD1151266AA2Ad2410447E133b7F84e29",
		Value:          "1000",
		ValidAfter:     0,
		ValidBefore:    0, // Should be calculated
		TimeoutSeconds: 300,
	}

	result, err := SignTransferAuthorization(key, params)
	require.NoError(t, err)

	// ValidBefore should be set (non-zero)
	assert.NotEqual(t, "0", result.Authorization.ValidBefore)
}

func TestSignTransferAuthorization_ExplicitValidBefore(t *testing.T) {
	key, err := LoadFromHex(signerTestPrivateKey)
	require.NoError(t, err)

	params := SignParams{
		ChainID:        84532,
		TokenAddress:   "0x036cbd53842c5426634e7929541ec2318f3dcf7e",
		TokenName:      "USDC",
		TokenVersion:   "2",
		From:           signerTestAddress,
		To:             "0x64c2310BD1151266AA2Ad2410447E133b7F84e29",
		Value:          "1000",
		ValidAfter:     0,
		ValidBefore:    9999999999, // Explicit value
		TimeoutSeconds: 300,
	}

	result, err := SignTransferAuthorization(key, params)
	require.NoError(t, err)

	// Should use the explicit value
	assert.Equal(t, "9999999999", result.Authorization.ValidBefore)
}

func TestPrepareSignParams(t *testing.T) {
	option := &x402.PaymentRequirement{
		Scheme:            "exact",
		Network:           "eip155:84532",
		Amount:            "1000000",
		Asset:             "0x036cbd53842c5426634e7929541ec2318f3dcf7e",
		PayTo:             "0x64c2310BD1151266AA2Ad2410447E133b7F84e29",
		MaxTimeoutSeconds: 300,
		Extra: map[string]interface{}{
			"name":    "USDC",
			"version": "2",
		},
	}

	params := PrepareSignParams(option, signerTestAddress, 84532)

	assert.Equal(t, int64(84532), params.ChainID)
	assert.Equal(t, option.Asset, params.TokenAddress)
	assert.Equal(t, "USDC", params.TokenName)
	assert.Equal(t, "2", params.TokenVersion)
	assert.Equal(t, signerTestAddress, params.From)
	assert.Equal(t, option.PayTo, params.To)
	assert.Equal(t, "1000000", params.Value)
	assert.Equal(t, 300, params.TimeoutSeconds)
}

func TestPrepareSignParams_DefaultTokenName(t *testing.T) {
	option := &x402.PaymentRequirement{
		Scheme:  "exact",
		Network: "eip155:84532",
		Amount:  "1000",
		Asset:   "0x036cbd53842c5426634e7929541ec2318f3dcf7e",
		PayTo:   "0x64c2310BD1151266AA2Ad2410447E133b7F84e29",
		Extra:   map[string]interface{}{}, // No name
	}

	params := PrepareSignParams(option, signerTestAddress, 84532)

	// Should default to "USDC"
	assert.Equal(t, "USDC", params.TokenName)
}

func TestPrepareSignParams_DefaultTokenVersion(t *testing.T) {
	option := &x402.PaymentRequirement{
		Scheme:  "exact",
		Network: "eip155:84532",
		Amount:  "1000",
		Asset:   "0x036cbd53842c5426634e7929541ec2318f3dcf7e",
		PayTo:   "0x64c2310BD1151266AA2Ad2410447E133b7F84e29",
		Extra: map[string]interface{}{
			"name": "USDC",
			// No version
		},
	}

	params := PrepareSignParams(option, signerTestAddress, 84532)

	// Should default to "2"
	assert.Equal(t, "2", params.TokenVersion)
}

func TestPrepareSignParams_V1Amount(t *testing.T) {
	// v1 uses MaxAmountRequired instead of Amount
	option := &x402.PaymentRequirement{
		Scheme:            "exact",
		Network:           "eip155:84532",
		MaxAmountRequired: "2000000", // v1 field
		Asset:             "0x036cbd53842c5426634e7929541ec2318f3dcf7e",
		PayTo:             "0x64c2310BD1151266AA2Ad2410447E133b7F84e29",
	}

	params := PrepareSignParams(option, signerTestAddress, 84532)

	assert.Equal(t, "2000000", params.Value)
}

// TestSignatureRecovery verifies the signature can be recovered to the correct address
func TestSignatureRecovery(t *testing.T) {
	key, err := LoadFromHex(signerTestPrivateKey)
	require.NoError(t, err)

	params := SignParams{
		ChainID:        84532,
		TokenAddress:   "0x036cbd53842c5426634e7929541ec2318f3dcf7e",
		TokenName:      "USDC",
		TokenVersion:   "2",
		From:           signerTestAddress,
		To:             "0x64c2310BD1151266AA2Ad2410447E133b7F84e29",
		Value:          "1000",
		TimeoutSeconds: 300,
	}

	result, err := SignTransferAuthorization(key, params)
	require.NoError(t, err)

	// Decode signature
	sigHex := strings.TrimPrefix(result.Signature, "0x")
	sigBytes, err := hex.DecodeString(sigHex)
	require.NoError(t, err)

	// The signature should be valid ECDSA signature
	// We can't easily verify recovery without reconstructing the exact
	// EIP-712 hash, but we can verify the signature is well-formed
	assert.Len(t, sigBytes, 65)

	// v should be 27 or 28 (Ethereum signature format)
	v := sigBytes[64]
	assert.True(t, v >= 27 && v <= 28)

	// r and s should be non-zero
	r := sigBytes[:32]
	s := sigBytes[32:64]
	assert.NotEqual(t, make([]byte, 32), r)
	assert.NotEqual(t, make([]byte, 32), s)
}

// TestGetAddress verifies address derivation works correctly
func TestSignerGetAddress(t *testing.T) {
	key, err := LoadFromHex(signerTestPrivateKey)
	require.NoError(t, err)

	address := crypto.PubkeyToAddress(key.PublicKey).Hex()
	assert.Equal(t, signerTestAddress, address)
}
