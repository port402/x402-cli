package tokens

import (
	"fmt"
	"strings"
)

// explorerURLs maps network identifiers to block explorer base URLs.
// Supports both CAIP-2 format (eip155:*) and simple network names.
var explorerURLs = map[string]string{
	// CAIP-2 format - EVM
	"eip155:1":        "https://etherscan.io",
	"eip155:8453":     "https://basescan.org",
	"eip155:84532":    "https://sepolia.basescan.org",
	"eip155:11155111": "https://sepolia.etherscan.io",
	"eip155:137":      "https://polygonscan.com",
	"eip155:42161":    "https://arbiscan.io",
	"eip155:10":       "https://optimistic.etherscan.io",

	// Simple names (v1 protocol compatibility)
	"ethereum":     "https://etherscan.io",
	"mainnet":      "https://etherscan.io",
	"base":         "https://basescan.org",
	"base-sepolia": "https://sepolia.basescan.org",
	"base_sepolia": "https://sepolia.basescan.org",
	"basesepolia":  "https://sepolia.basescan.org",
	"sepolia":      "https://sepolia.etherscan.io",
	"polygon":      "https://polygonscan.com",
	"arbitrum":     "https://arbiscan.io",
	"optimism":     "https://optimistic.etherscan.io",

	// Solana networks (CAIP-2 format)
	"solana:5eykt4UsFv8P8NJdTREpY1vzqKqZKvdp": "https://solscan.io",
	"solana:EtWTRABZaYq6iMfeYKouRu166VU2xqa1": "https://solscan.io?cluster=devnet",
	"solana:4uhcVJyU9pJkvQyS88uRDiswHXSCkY3z": "https://solscan.io?cluster=testnet",
}

// GetExplorerURL returns the block explorer URL for a transaction.
// Returns empty string if the network is not in the registry.
func GetExplorerURL(network, txHash string) string {
	baseURL, ok := explorerURLs[network]
	if !ok {
		return ""
	}
	return buildExplorerURL(network, baseURL, "tx", txHash)
}

// buildExplorerURL constructs an explorer URL, handling Solana's query param format.
// Solscan uses /tx/<sig>?cluster=devnet and /account/<addr>?cluster=devnet
func buildExplorerURL(network, baseURL, pathType, value string) string {
	if strings.HasPrefix(network, "solana:") {
		// Solana uses /account/ not /address/
		if pathType == "address" {
			pathType = "account"
		}
		// Split base URL into host and query string
		if idx := strings.Index(baseURL, "?"); idx != -1 {
			host := baseURL[:idx]
			query := baseURL[idx:]
			return fmt.Sprintf("%s/%s/%s%s", host, pathType, value, query)
		}
		return fmt.Sprintf("%s/%s/%s", baseURL, pathType, value)
	}
	return fmt.Sprintf("%s/%s/%s", baseURL, pathType, value)
}

// GetAddressExplorerURL returns the block explorer URL for an address.
func GetAddressExplorerURL(network, address string) string {
	baseURL, ok := explorerURLs[network]
	if !ok {
		return ""
	}
	return buildExplorerURL(network, baseURL, "address", address)
}

// GetTokenExplorerURL returns the block explorer URL for a token contract.
func GetTokenExplorerURL(network, tokenAddress string) string {
	baseURL, ok := explorerURLs[network]
	if !ok {
		return ""
	}
	return buildExplorerURL(network, baseURL, "token", tokenAddress)
}

// HasExplorer returns true if we have a block explorer URL for the network.
func HasExplorer(network string) bool {
	_, ok := explorerURLs[network]
	return ok
}

// GetExplorerHost returns just the explorer hostname for display (e.g., "basescan.org").
// Returns empty string if the network is not in the registry.
func GetExplorerHost(network string) string {
	baseURL, ok := explorerURLs[network]
	if !ok {
		return ""
	}
	// Strip protocol
	host := strings.TrimPrefix(baseURL, "https://")
	host = strings.TrimPrefix(host, "http://")
	// Keep query params for Solana cluster display (e.g., "solscan.io?cluster=devnet")
	return host
}
