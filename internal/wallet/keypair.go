package wallet

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gagliardetto/solana-go"
	"github.com/mr-tron/base58"
)

const solanaKeypairLen = 64

// LoadSolanaKeypair loads a Solana keypair from a file.
// Supports JSON array format (Solana CLI) and base58 encoded private keys.
func LoadSolanaKeypair(path string) (solana.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read keypair file: %w", err)
	}

	// Try JSON array format first (Solana CLI format)
	var keyBytes []byte
	if err := json.Unmarshal(data, &keyBytes); err == nil {
		return validateAndCreateKey(keyBytes)
	}

	// Try base58 encoded string
	decoded, err := base58.Decode(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, fmt.Errorf("invalid keypair format: not JSON array or base58 encoded")
	}

	return validateAndCreateKey(decoded)
}

// LoadSolanaKeypairFromBase58 loads a Solana keypair from a base58 string.
func LoadSolanaKeypairFromBase58(base58Key string) (solana.PrivateKey, error) {
	decoded, err := base58.Decode(strings.TrimSpace(base58Key))
	if err != nil {
		return nil, fmt.Errorf("invalid base58 private key: %w", err)
	}
	return validateAndCreateKey(decoded)
}

// validateAndCreateKey validates the keypair length and returns the private key.
func validateAndCreateKey(keyBytes []byte) (solana.PrivateKey, error) {
	if len(keyBytes) != solanaKeypairLen {
		return nil, fmt.Errorf("invalid keypair length: expected %d bytes, got %d", solanaKeypairLen, len(keyBytes))
	}
	return solana.PrivateKey(keyBytes), nil
}

// GetSolanaAddress returns the base58-encoded public key for a Solana private key.
func GetSolanaAddress(privateKey solana.PrivateKey) string {
	return privateKey.PublicKey().String()
}
