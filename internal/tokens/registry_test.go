package tokens

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTokenInfo_KnownToken(t *testing.T) {
	tests := []struct {
		name    string
		network string
		asset   string
		symbol  string
	}{
		{
			name:    "Base Mainnet USDC",
			network: "eip155:8453",
			asset:   "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
			symbol:  "USDC",
		},
		{
			name:    "Base Sepolia USDC",
			network: "eip155:84532",
			asset:   "0x036cbd53842c5426634e7929541ec2318f3dcf7e",
			symbol:  "USDC",
		},
		{
			name:    "Ethereum Mainnet USDC",
			network: "eip155:1",
			asset:   "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			symbol:  "USDC",
		},
		{
			name:    "Ethereum Sepolia USDC",
			network: "eip155:11155111",
			asset:   "0x1c7d4b196cb0c7b01d743fbc6116a902379c7238",
			symbol:  "USDC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := GetTokenInfo(tt.network, tt.asset)
			assert.NotNil(t, info)
			assert.Equal(t, tt.symbol, info.Symbol)
			assert.Equal(t, 6, info.Decimals)
		})
	}
}

func TestGetTokenInfo_CaseInsensitive(t *testing.T) {
	// Test with uppercase
	info1 := GetTokenInfo("eip155:84532", "0x036CBD53842C5426634E7929541EC2318F3DCF7E")
	assert.NotNil(t, info1)

	// Test with lowercase
	info2 := GetTokenInfo("eip155:84532", "0x036cbd53842c5426634e7929541ec2318f3dcf7e")
	assert.NotNil(t, info2)

	// Both should return same token
	assert.Equal(t, info1.Symbol, info2.Symbol)
}

func TestGetTokenInfo_UnknownToken(t *testing.T) {
	info := GetTokenInfo("eip155:8453", "0x0000000000000000000000000000000000000000")
	assert.Nil(t, info)
}

func TestGetTokenInfo_UnknownNetwork(t *testing.T) {
	info := GetTokenInfo("solana:mainnet", "0x036cbd53842c5426634e7929541ec2318f3dcf7e")
	assert.Nil(t, info)
}

func TestGetNetworkInfo_KnownNetwork(t *testing.T) {
	tests := []struct {
		network   string
		name      string
		isTestnet bool
	}{
		{"eip155:1", "Ethereum Mainnet", false},
		{"eip155:8453", "Base Mainnet", false},
		{"eip155:84532", "Base Sepolia", true},
		{"eip155:11155111", "Ethereum Sepolia", true},
		{"eip155:137", "Polygon Mainnet", false},
		{"eip155:42161", "Arbitrum One", false},
		{"eip155:10", "Optimism", false},
	}

	for _, tt := range tests {
		t.Run(tt.network, func(t *testing.T) {
			info := GetNetworkInfo(tt.network)
			assert.NotNil(t, info)
			assert.Equal(t, tt.name, info.Name)
			assert.Equal(t, tt.isTestnet, info.IsTestnet)
		})
	}
}

func TestGetNetworkInfo_UnknownNetwork(t *testing.T) {
	info := GetNetworkInfo("eip155:999999")
	assert.Nil(t, info)
}

func TestGetNetworkName_KnownNetwork(t *testing.T) {
	assert.Equal(t, "Ethereum Mainnet", GetNetworkName("eip155:1"))
	assert.Equal(t, "Base Mainnet", GetNetworkName("eip155:8453"))
	assert.Equal(t, "Base Sepolia", GetNetworkName("eip155:84532"))
}

func TestGetNetworkName_UnknownNetwork(t *testing.T) {
	// Falls back to raw network identifier
	assert.Equal(t, "eip155:999999", GetNetworkName("eip155:999999"))
	assert.Equal(t, "solana:mainnet", GetNetworkName("solana:mainnet"))
}

func TestIsTestnet(t *testing.T) {
	tests := []struct {
		network  string
		expected bool
	}{
		{"eip155:1", false},      // Ethereum Mainnet
		{"eip155:8453", false},   // Base Mainnet
		{"eip155:84532", true},   // Base Sepolia
		{"eip155:11155111", true}, // Ethereum Sepolia
		{"eip155:137", false},    // Polygon Mainnet
		{"eip155:999999", false}, // Unknown (defaults to false)
	}

	for _, tt := range tests {
		t.Run(tt.network, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsTestnet(tt.network))
		})
	}
}
