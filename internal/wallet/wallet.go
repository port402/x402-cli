// Package wallet handles private key loading and management.
package wallet

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/term"
)

// LoadPrivateKey loads a private key from various sources.
// Priority: keystore file → hex key (flag/env) → stdin
//
// Parameters:
//   - keystorePath: Path to Web3 Secret Storage keystore file
//   - hexKey: Hex-encoded private key (with or without 0x prefix)
//   - fromStdin: If true and no other source, read from stdin
func LoadPrivateKey(keystorePath, hexKey string, fromStdin bool) (*ecdsa.PrivateKey, error) {
	// Priority 1: Keystore file
	if keystorePath != "" {
		return LoadFromKeystore(keystorePath)
	}

	// Priority 2: Hex private key
	if hexKey != "" {
		return LoadFromHex(hexKey)
	}

	// Priority 3: Environment variable
	if envKey := os.Getenv("PRIVATE_KEY"); envKey != "" {
		return LoadFromHex(envKey)
	}

	// Priority 4: Stdin (for piped input)
	if fromStdin {
		return LoadFromStdin()
	}

	return nil, fmt.Errorf("no private key source provided (use --keystore, --wallet, PRIVATE_KEY env, or pipe to stdin)")
}

// LoadFromKeystore loads a private key from a Web3 Secret Storage keystore file.
// Prompts for password interactively.
func LoadFromKeystore(path string) (*ecdsa.PrivateKey, error) {
	// Read keystore file
	keystoreJSON, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read keystore file: %w", err)
	}

	// Prompt for password
	password, err := PromptPassword("Enter keystore password: ")
	if err != nil {
		return nil, fmt.Errorf("failed to read password: %w", err)
	}

	// Decrypt keystore
	key, err := keystore.DecryptKey(keystoreJSON, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt keystore (wrong password?): %w", err)
	}

	return key.PrivateKey, nil
}

// LoadFromHex loads a private key from a hex string.
// Accepts with or without 0x prefix.
func LoadFromHex(hexKey string) (*ecdsa.PrivateKey, error) {
	// Remove 0x prefix if present
	hexKey = strings.TrimPrefix(hexKey, "0x")
	hexKey = strings.TrimSpace(hexKey)

	// Decode hex
	keyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("invalid hex private key: %w", err)
	}

	// Convert to ECDSA key
	privateKey, err := crypto.ToECDSA(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	return privateKey, nil
}

// LoadFromStdin reads a private key from stdin.
// Expects hex-encoded key on a single line.
func LoadFromStdin() (*ecdsa.PrivateKey, error) {
	// Check if stdin has data
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return nil, fmt.Errorf("no private key piped to stdin")
	}

	// Read from stdin
	var hexKey string
	_, err := fmt.Scanln(&hexKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key from stdin: %w", err)
	}

	return LoadFromHex(hexKey)
}

// PromptPassword prompts for a password without echoing to terminal.
func PromptPassword(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)

	// Read password without echo
	passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr) // Newline after password

	if err != nil {
		return "", err
	}

	return string(passwordBytes), nil
}

// GetAddress returns the Ethereum address for a private key.
func GetAddress(privateKey *ecdsa.PrivateKey) string {
	return crypto.PubkeyToAddress(privateKey.PublicKey).Hex()
}
