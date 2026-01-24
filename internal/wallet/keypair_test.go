package wallet

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testSolanaKeypairBytes returns a deterministic Ed25519 keypair for testing.
// NEVER use for real funds - this is a well-known test key.
func testSolanaKeypairBytes(t *testing.T) []byte {
	t.Helper()
	seed := bytes.Repeat([]byte{0x01}, ed25519.SeedSize)
	return ed25519.NewKeyFromSeed(seed)
}

func TestLoadSolanaKeypair(t *testing.T) {
	testKeypair := testSolanaKeypairBytes(t)

	tests := []struct {
		name        string
		setup       func(t *testing.T, tmpDir string) string // returns path to keypair file
		wantErr     bool
		errContains string
		wantLen     int
	}{
		{
			name: "JSON array format",
			setup: func(t *testing.T, tmpDir string) string {
				path := filepath.Join(tmpDir, "keypair.json")
				data, err := json.Marshal(testKeypair)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(path, data, 0600))
				return path
			},
			wantErr: false,
			wantLen: 64,
		},
		{
			name: "Base58 format",
			setup: func(t *testing.T, tmpDir string) string {
				path := filepath.Join(tmpDir, "keypair.txt")
				encoded := base58.Encode(testKeypair)
				require.NoError(t, os.WriteFile(path, []byte(encoded), 0600))
				return path
			},
			wantErr: false,
			wantLen: 64,
		},
		{
			name: "invalid length (32 bytes instead of 64)",
			setup: func(t *testing.T, tmpDir string) string {
				path := filepath.Join(tmpDir, "bad-keypair.json")
				badKey := make([]byte, 32)
				data, err := json.Marshal(badKey)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(path, data, 0600))
				return path
			},
			wantErr:     true,
			errContains: "invalid keypair length",
		},
		{
			name: "invalid format (garbage data)",
			setup: func(t *testing.T, tmpDir string) string {
				path := filepath.Join(tmpDir, "bad-keypair.txt")
				require.NoError(t, os.WriteFile(path, []byte("not-valid-data!@#$"), 0600))
				return path
			},
			wantErr: true,
		},
		{
			name: "file not found",
			setup: func(t *testing.T, tmpDir string) string {
				return "/nonexistent/path/keypair.json"
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path := tt.setup(t, tmpDir)

			key, err := LoadSolanaKeypair(path)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, key)
			assert.Len(t, key, tt.wantLen)
		})
	}
}

func TestLoadSolanaKeypairFromBase58(t *testing.T) {
	testKeypair := testSolanaKeypairBytes(t)

	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantLen int
	}{
		{
			name:    "valid base58",
			input:   base58.Encode(testKeypair),
			wantErr: false,
			wantLen: 64,
		},
		{
			name:    "valid base58 with whitespace",
			input:   "  " + base58.Encode(testKeypair) + "\n",
			wantErr: false,
			wantLen: 64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := LoadSolanaKeypairFromBase58(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, key, tt.wantLen)
		})
	}
}

func TestGetSolanaAddress(t *testing.T) {
	testKeypair := testSolanaKeypairBytes(t)
	key, err := LoadSolanaKeypairFromBase58(base58.Encode(testKeypair))
	require.NoError(t, err)

	address := GetSolanaAddress(key)

	// Should be a valid base58 address
	assert.NotEmpty(t, address)
	// Solana addresses are typically 32-44 characters
	assert.GreaterOrEqual(t, len(address), 32)
	assert.LessOrEqual(t, len(address), 44)
}
