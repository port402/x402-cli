// Package tokens provides token metadata, formatting, and block explorer URLs.
package tokens

import "strings"

// TokenInfo contains metadata for a known token.
type TokenInfo struct {
	Symbol   string
	Decimals int
	Name     string
}

// knownTokens maps "network:asset" to token metadata.
// Keys are lowercase for case-insensitive lookup.
// Supports both CAIP-2 format (eip155:*) and simple network names (base).
var knownTokens = map[string]TokenInfo{
	// Base Mainnet (CAIP-2)
	"eip155:8453:0x833589fcd6edb6e08f4c7c32d4f71b54bda02913": {
		Symbol:   "USDC",
		Decimals: 6,
		Name:     "USD Coin",
	},
	// Base Mainnet (simple name)
	"base:0x833589fcd6edb6e08f4c7c32d4f71b54bda02913": {
		Symbol:   "USDC",
		Decimals: 6,
		Name:     "USD Coin",
	},

	// Base Sepolia (CAIP-2)
	"eip155:84532:0x036cbd53842c5426634e7929541ec2318f3dcf7e": {
		Symbol:   "USDC",
		Decimals: 6,
		Name:     "USDC (Testnet)",
	},
	// Base Sepolia (simple names)
	"base-sepolia:0x036cbd53842c5426634e7929541ec2318f3dcf7e": {
		Symbol:   "USDC",
		Decimals: 6,
		Name:     "USDC (Testnet)",
	},
	"basesepolia:0x036cbd53842c5426634e7929541ec2318f3dcf7e": {
		Symbol:   "USDC",
		Decimals: 6,
		Name:     "USDC (Testnet)",
	},

	// Ethereum Mainnet (CAIP-2)
	"eip155:1:0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48": {
		Symbol:   "USDC",
		Decimals: 6,
		Name:     "USD Coin",
	},
	// Ethereum Mainnet (simple names)
	"ethereum:0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48": {
		Symbol:   "USDC",
		Decimals: 6,
		Name:     "USD Coin",
	},
	"mainnet:0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48": {
		Symbol:   "USDC",
		Decimals: 6,
		Name:     "USD Coin",
	},

	// Ethereum Sepolia (CAIP-2)
	"eip155:11155111:0x1c7d4b196cb0c7b01d743fbc6116a902379c7238": {
		Symbol:   "USDC",
		Decimals: 6,
		Name:     "USDC (Testnet)",
	},
	// Ethereum Sepolia (simple name)
	"sepolia:0x1c7d4b196cb0c7b01d743fbc6116a902379c7238": {
		Symbol:   "USDC",
		Decimals: 6,
		Name:     "USDC (Testnet)",
	},

	// Solana Mainnet USDC (CAIP-2 format)
	"solana:5eykt4usfv8p8njdtrepyvzqkqzkvdp:epjfwdd5aufqssqem2qn1xzybapC8g4weggkzwytdt1v": {
		Symbol:   "USDC",
		Decimals: 6,
		Name:     "USD Coin",
	},
	// Solana Mainnet USDC (simple network names)
	"solana:epjfwdd5aufqssqem2qn1xzybapC8g4weggkzwytdt1v": {
		Symbol:   "USDC",
		Decimals: 6,
		Name:     "USD Coin",
	},
	"solana-mainnet-beta:epjfwdd5aufqssqem2qn1xzybapC8g4weggkzwytdt1v": {
		Symbol:   "USDC",
		Decimals: 6,
		Name:     "USD Coin",
	},
	"solana-mainnet:epjfwdd5aufqssqem2qn1xzybapC8g4weggkzwytdt1v": {
		Symbol:   "USDC",
		Decimals: 6,
		Name:     "USD Coin",
	},
	"mainnet-beta:epjfwdd5aufqssqem2qn1xzybapC8g4weggkzwytdt1v": {
		Symbol:   "USDC",
		Decimals: 6,
		Name:     "USD Coin",
	},

	// Solana Devnet USDC (CAIP-2 format)
	"solana:etwtraBzayq6imfeykourU166vu2xqa1:4zmmc9srt5ri5x14gagxhahii3gnpaeeryPjgzjdncdu": {
		Symbol:   "USDC",
		Decimals: 6,
		Name:     "USDC (Devnet)",
	},
	// Solana Devnet USDC (simple network names)
	"solana-devnet:4zmmc9srt5ri5x14gagxhahii3gnpaeeryPjgzjdncdu": {
		Symbol:   "USDC",
		Decimals: 6,
		Name:     "USDC (Devnet)",
	},
	"devnet:4zmmc9srt5ri5x14gagxhahii3gnpaeeryPjgzjdncdu": {
		Symbol:   "USDC",
		Decimals: 6,
		Name:     "USDC (Devnet)",
	},
}

// NetworkInfo contains metadata for a known network.
type NetworkInfo struct {
	Name      string
	IsTestnet bool
}

// networkNames maps network identifiers to human-readable names.
// Supports both CAIP-2 format (eip155:*) and simple names (base).
var networkNames = map[string]NetworkInfo{
	// CAIP-2 format
	"eip155:1":        {Name: "Ethereum Mainnet", IsTestnet: false},
	"eip155:8453":     {Name: "Base Mainnet", IsTestnet: false},
	"eip155:84532":    {Name: "Base Sepolia", IsTestnet: true},
	"eip155:11155111": {Name: "Ethereum Sepolia", IsTestnet: true},
	"eip155:137":      {Name: "Polygon Mainnet", IsTestnet: false},
	"eip155:42161":    {Name: "Arbitrum One", IsTestnet: false},
	"eip155:10":       {Name: "Optimism", IsTestnet: false},

	// Simple names (v1 protocol compatibility)
	"ethereum":     {Name: "Ethereum Mainnet", IsTestnet: false},
	"mainnet":      {Name: "Ethereum Mainnet", IsTestnet: false},
	"base":         {Name: "Base Mainnet", IsTestnet: false},
	"base-sepolia": {Name: "Base Sepolia", IsTestnet: true},
	"base_sepolia": {Name: "Base Sepolia", IsTestnet: true},
	"basesepolia":  {Name: "Base Sepolia", IsTestnet: true},
	"sepolia":      {Name: "Ethereum Sepolia", IsTestnet: true},
	"polygon":      {Name: "Polygon Mainnet", IsTestnet: false},
	"arbitrum":     {Name: "Arbitrum One", IsTestnet: false},
	"optimism":     {Name: "Optimism", IsTestnet: false},

	// Solana networks (CAIP-2 format)
	"solana:5eykt4UsFv8P8NJdTREpY1vzqKqZKvdp": {Name: "Solana Mainnet", IsTestnet: false},
	"solana:EtWTRABZaYq6iMfeYKouRu166VU2xqa1": {Name: "Solana Devnet", IsTestnet: true},
	"solana:4uhcVJyU9pJkvQyS88uRDiswHXSCkY3z": {Name: "Solana Testnet", IsTestnet: true},

	// Solana networks (simple name aliases)
	"solana":              {Name: "Solana Mainnet", IsTestnet: false},
	"solana-mainnet":      {Name: "Solana Mainnet", IsTestnet: false},
	"solana-mainnet-beta": {Name: "Solana Mainnet", IsTestnet: false},
	"mainnet-beta":        {Name: "Solana Mainnet", IsTestnet: false},
	"solana-devnet":       {Name: "Solana Devnet", IsTestnet: true},
	"solana-testnet":      {Name: "Solana Testnet", IsTestnet: true},
}

// GetTokenInfo looks up token metadata by network and asset address.
// Returns nil if the token is not in the registry.
func GetTokenInfo(network, asset string) *TokenInfo {
	key := strings.ToLower(network + ":" + asset)
	if info, ok := knownTokens[key]; ok {
		return &info
	}
	return nil
}

// GetNetworkInfo looks up network metadata by CAIP-2 identifier.
// Returns nil if the network is not in the registry.
func GetNetworkInfo(network string) *NetworkInfo {
	if info, ok := networkNames[network]; ok {
		return &info
	}
	return nil
}

// GetNetworkName returns a human-readable network name.
// Falls back to the raw network identifier if not found.
func GetNetworkName(network string) string {
	if info := GetNetworkInfo(network); info != nil {
		return info.Name
	}
	return network
}

// IsTestnet returns true if the network is a known testnet.
func IsTestnet(network string) bool {
	if info := GetNetworkInfo(network); info != nil {
		return info.IsTestnet
	}
	return false
}
