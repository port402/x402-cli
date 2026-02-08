// Package commands implements the CLI commands using Cobra.
package commands

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// Version information (set at build time via ldflags)
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

// Global flags
var (
	verbose    bool
	jsonOutput bool
)

// rootCmd is the base command when called without subcommands.
var rootCmd = &cobra.Command{
	Use:   "x402",
	Short: "CLI for testing x402-enabled payment-gated APIs",
	Long: `x402 is a command-line tool for testing APIs that use the x402 payment protocol.

The x402 protocol uses HTTP 402 (Payment Required) status codes with EIP-3009
gasless token transfers (EVM) and SPL token transfers (Solana) to gate access
to resources.

Commands:
  health       Check if an endpoint is x402-enabled (no wallet needed)
  test         Make a test payment to an x402 endpoint
  batch-health Check multiple endpoints from a file
  agent        Discover A2A agent card from an endpoint
  networks     List supported networks
  completion   Generate shell completion scripts
  version      Show version information

Examples:
  # Check if an endpoint requires payment
  x402 health https://api.example.com/endpoint

  # Make a test payment
  x402 test https://api.example.com/endpoint --keystore ~/.foundry/keystores/my-wallet

  # Check multiple endpoints
  x402 batch-health urls.json --parallel 5

  # List supported networks
  x402 networks`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags available to all commands
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed output")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output results as JSON")
}

// GetVerbose returns the verbose flag value.
func GetVerbose() bool {
	return verbose
}

// GetJSONOutput returns the json output flag value.
func GetJSONOutput() bool {
	return jsonOutput
}

// normalizeURL adds https:// if no scheme is present and validates the result.
func normalizeURL(raw string) (string, error) {
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		raw = "https://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("invalid URL: %s", raw)
	}
	return raw, nil
}
