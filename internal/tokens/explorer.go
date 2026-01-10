package tokens

import "fmt"

// explorerURLs maps network identifiers to block explorer base URLs.
// Supports both CAIP-2 format (eip155:*) and simple network names.
var explorerURLs = map[string]string{
	// CAIP-2 format
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
}

// GetExplorerURL returns the block explorer URL for a transaction.
// Returns empty string if the network is not in the registry.
func GetExplorerURL(network, txHash string) string {
	baseURL, ok := explorerURLs[network]
	if !ok {
		return ""
	}
	return fmt.Sprintf("%s/tx/%s", baseURL, txHash)
}

// GetAddressExplorerURL returns the block explorer URL for an address.
func GetAddressExplorerURL(network, address string) string {
	baseURL, ok := explorerURLs[network]
	if !ok {
		return ""
	}
	return fmt.Sprintf("%s/address/%s", baseURL, address)
}

// GetTokenExplorerURL returns the block explorer URL for a token contract.
func GetTokenExplorerURL(network, tokenAddress string) string {
	baseURL, ok := explorerURLs[network]
	if !ok {
		return ""
	}
	return fmt.Sprintf("%s/token/%s", baseURL, tokenAddress)
}

// HasExplorer returns true if we have a block explorer URL for the network.
func HasExplorer(network string) bool {
	_, ok := explorerURLs[network]
	return ok
}
