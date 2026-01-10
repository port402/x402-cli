package wallet

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test private key from Foundry/Anvil - NEVER use for real funds
const testPrivateKeyHex = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
const testPrivateKeyWithPrefix = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
const testAddress = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"

func TestLoadFromHex_WithoutPrefix(t *testing.T) {
	key, err := LoadFromHex(testPrivateKeyHex)
	require.NoError(t, err)
	require.NotNil(t, key)

	// Verify the address matches expected
	address := crypto.PubkeyToAddress(key.PublicKey).Hex()
	assert.Equal(t, testAddress, address)
}

func TestLoadFromHex_WithPrefix(t *testing.T) {
	key, err := LoadFromHex(testPrivateKeyWithPrefix)
	require.NoError(t, err)
	require.NotNil(t, key)

	// Verify the address matches expected
	address := crypto.PubkeyToAddress(key.PublicKey).Hex()
	assert.Equal(t, testAddress, address)
}

func TestLoadFromHex_WithWhitespace(t *testing.T) {
	key, err := LoadFromHex("  " + testPrivateKeyHex + "  ")
	require.NoError(t, err)
	require.NotNil(t, key)
}

func TestLoadFromHex_Invalid(t *testing.T) {
	_, err := LoadFromHex("not-hex-at-all")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid hex")
}

func TestLoadFromHex_WrongLength(t *testing.T) {
	// Too short
	_, err := LoadFromHex("abcd")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid private key")
}

func TestLoadFromHex_TooLong(t *testing.T) {
	// Too long (65 bytes instead of 32)
	_, err := LoadFromHex(testPrivateKeyHex + "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234ff")
	require.Error(t, err)
}

func TestGetAddress(t *testing.T) {
	key, err := LoadFromHex(testPrivateKeyHex)
	require.NoError(t, err)

	address := GetAddress(key)
	assert.Equal(t, testAddress, address)
}

func TestLoadPrivateKey_NoSource(t *testing.T) {
	// No keystore, no hex, no stdin
	_, err := LoadPrivateKey("", "", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no private key source provided")
}

func TestLoadPrivateKey_HexKey(t *testing.T) {
	key, err := LoadPrivateKey("", testPrivateKeyHex, false)
	require.NoError(t, err)
	require.NotNil(t, key)

	address := GetAddress(key)
	assert.Equal(t, testAddress, address)
}

// TestLoadFromKeystore_FileNotFound tests error handling for missing keystore
func TestLoadFromKeystore_FileNotFound(t *testing.T) {
	_, err := LoadFromKeystore("/nonexistent/path/keystore.json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read keystore file")
}

// Note: Testing LoadFromKeystore with a real keystore would require
// creating a test fixture with a known password. This is deferred
// to integration tests where we can use the full Foundry tooling.

// Note: LoadFromStdin is difficult to unit test because it checks
// os.Stdin.Stat() and requires actual piped input. This is better
// tested via integration tests or by refactoring to accept an io.Reader.

// Note: PromptPassword requires terminal interaction and cannot be
// easily unit tested. It's tested manually during development.
