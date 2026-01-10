package tokens

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetExplorerURL(t *testing.T) {
	tests := []struct {
		network  string
		txHash   string
		expected string
	}{
		{
			network:  "eip155:1",
			txHash:   "0xabc123",
			expected: "https://etherscan.io/tx/0xabc123",
		},
		{
			network:  "eip155:8453",
			txHash:   "0xdef456",
			expected: "https://basescan.org/tx/0xdef456",
		},
		{
			network:  "eip155:84532",
			txHash:   "0x789",
			expected: "https://sepolia.basescan.org/tx/0x789",
		},
		{
			network:  "eip155:11155111",
			txHash:   "0xghi",
			expected: "https://sepolia.etherscan.io/tx/0xghi",
		},
		{
			network:  "eip155:137",
			txHash:   "0xpoly",
			expected: "https://polygonscan.com/tx/0xpoly",
		},
		{
			network:  "eip155:42161",
			txHash:   "0xarb",
			expected: "https://arbiscan.io/tx/0xarb",
		},
		{
			network:  "eip155:10",
			txHash:   "0xopt",
			expected: "https://optimistic.etherscan.io/tx/0xopt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.network, func(t *testing.T) {
			result := GetExplorerURL(tt.network, tt.txHash)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetExplorerURL_UnknownNetwork(t *testing.T) {
	result := GetExplorerURL("eip155:999999", "0xabc")
	assert.Empty(t, result)

	result = GetExplorerURL("solana:mainnet", "sig123")
	assert.Empty(t, result)
}

func TestGetAddressExplorerURL(t *testing.T) {
	result := GetAddressExplorerURL("eip155:8453", "0x64c2310BD1151266AA2Ad2410447E133b7F84e29")
	assert.Equal(t, "https://basescan.org/address/0x64c2310BD1151266AA2Ad2410447E133b7F84e29", result)
}

func TestGetAddressExplorerURL_UnknownNetwork(t *testing.T) {
	result := GetAddressExplorerURL("eip155:999999", "0xabc")
	assert.Empty(t, result)
}

func TestGetTokenExplorerURL(t *testing.T) {
	result := GetTokenExplorerURL("eip155:8453", "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913")
	assert.Equal(t, "https://basescan.org/token/0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", result)
}

func TestGetTokenExplorerURL_UnknownNetwork(t *testing.T) {
	result := GetTokenExplorerURL("eip155:999999", "0xabc")
	assert.Empty(t, result)
}

func TestHasExplorer(t *testing.T) {
	tests := []struct {
		network  string
		expected bool
	}{
		{"eip155:1", true},
		{"eip155:8453", true},
		{"eip155:84532", true},
		{"eip155:11155111", true},
		{"eip155:137", true},
		{"eip155:42161", true},
		{"eip155:10", true},
		{"eip155:999999", false},
		{"solana:mainnet", false},
	}

	for _, tt := range tests {
		t.Run(tt.network, func(t *testing.T) {
			assert.Equal(t, tt.expected, HasExplorer(tt.network))
		})
	}
}
