package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/port402/x402-cli/internal/output"
	"github.com/port402/x402-cli/internal/tokens"
)

var networksCmd = &cobra.Command{
	Use:   "networks",
	Short: "List supported networks",
	Long: `List all supported blockchain networks with their CAIP-2 identifiers,
tokens, and block explorer URLs.

Examples:
  x402 networks
  x402 networks --json`,
	Args: cobra.NoArgs,
	RunE: runNetworks,
}

func init() {
	rootCmd.AddCommand(networksCmd)
}

func runNetworks(cmd *cobra.Command, args []string) error {
	entries := tokens.ListNetworks()

	if GetJSONOutput() {
		return output.PrintJSON(entries)
	}

	fmt.Println("Supported Networks")

	currentChain := ""
	for _, e := range entries {
		if e.Chain != currentChain {
			currentChain = e.Chain
			fmt.Println()
			switch currentChain {
			case "evm":
				fmt.Println("  EVM")
			case "solana":
				fmt.Println("  Solana")
			}
		}

		explorer := tokens.GetExplorerHost(e.ID)
		testnet := ""
		if e.IsTestnet {
			testnet = "  (testnet)"
		}

		tokenStr := e.Token
		if tokenStr == "" {
			tokenStr = "-"
		}

		fmt.Printf("    %-22s %-45s %-6s %s%s\n", e.Name, e.ID, tokenStr, explorer, testnet)
	}

	fmt.Println()
	return nil
}
