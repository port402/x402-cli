package wallet

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test keypair - NEVER use for real funds
// Generated from a random seed for testing purposes only
var testSolanaKeypair []byte

func init() {
	// Generate a valid Ed25519 keypair for testing
	key, _ := solana.NewRandomPrivateKey()
	testSolanaKeypair = key[:]
}

func TestLoadSolanaKeypair_JSONArray(t *testing.T) {
	// Create temp file with JSON array format
	tmpDir := t.TempDir()
	keypairPath := filepath.Join(tmpDir, "test-keypair.json")

	jsonData, err := json.Marshal(testSolanaKeypair)
	require.NoError(t, err)

	err = os.WriteFile(keypairPath, jsonData, 0600)
	require.NoError(t, err)

	// Load keypair
	key, err := LoadSolanaKeypair(keypairPath)
	require.NoError(t, err)
	require.NotNil(t, key)

	// Verify length
	assert.Len(t, key, 64)
}

func TestLoadSolanaKeypair_Base58(t *testing.T) {
	// Create temp file with base58 format
	tmpDir := t.TempDir()
	keypairPath := filepath.Join(tmpDir, "test-keypair.txt")

	base58Key := base58.Encode(testSolanaKeypair)
	err := os.WriteFile(keypairPath, []byte(base58Key), 0600)
	require.NoError(t, err)

	// Load keypair
	key, err := LoadSolanaKeypair(keypairPath)
	require.NoError(t, err)
	require.NotNil(t, key)

	// Verify length
	assert.Len(t, key, 64)
}

func TestLoadSolanaKeypair_InvalidLength(t *testing.T) {
	// Create temp file with wrong length
	tmpDir := t.TempDir()
	keypairPath := filepath.Join(tmpDir, "bad-keypair.json")

	badKey := make([]byte, 32) // Should be 64
	jsonData, err := json.Marshal(badKey)
	require.NoError(t, err)

	err = os.WriteFile(keypairPath, jsonData, 0600)
	require.NoError(t, err)

	// Load should fail
	_, err = LoadSolanaKeypair(keypairPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid keypair length")
}

func TestLoadSolanaKeypair_InvalidFormat(t *testing.T) {
	// Create temp file with invalid format
	tmpDir := t.TempDir()
	keypairPath := filepath.Join(tmpDir, "bad-keypair.txt")

	err := os.WriteFile(keypairPath, []byte("not-valid-data!@#$"), 0600)
	require.NoError(t, err)

	// Load should fail
	_, err = LoadSolanaKeypair(keypairPath)
	assert.Error(t, err)
}

func TestLoadSolanaKeypair_FileNotFound(t *testing.T) {
	_, err := LoadSolanaKeypair("/nonexistent/path/keypair.json")
	assert.Error(t, err)
}

func TestLoadSolanaKeypairFromBase58(t *testing.T) {
	base58Key := base58.Encode(testSolanaKeypair)

	key, err := LoadSolanaKeypairFromBase58(base58Key)
	require.NoError(t, err)
	assert.Len(t, key, 64)
}

func TestLoadSolanaKeypairFromBase58_WithWhitespace(t *testing.T) {
	base58Key := "  " + base58.Encode(testSolanaKeypair) + "\n"

	key, err := LoadSolanaKeypairFromBase58(base58Key)
	require.NoError(t, err)
	assert.Len(t, key, 64)
}

func TestGetSolanaAddress(t *testing.T) {
	key, err := LoadSolanaKeypairFromBase58(base58.Encode(testSolanaKeypair))
	require.NoError(t, err)

	address := GetSolanaAddress(key)

	// Should be a valid base58 address
	assert.NotEmpty(t, address)
	// Solana addresses are typically 32-44 characters
	assert.GreaterOrEqual(t, len(address), 32)
	assert.LessOrEqual(t, len(address), 44)
}
