package output

import (
	"fmt"
	"os"
	"strings"

	"github.com/port402/x402-cli/internal/tokens"
)

// CheckStatus represents the result of a validation check.
type CheckStatus string

const (
	StatusPass CheckStatus = "pass"
	StatusWarn CheckStatus = "warn"
	StatusFail CheckStatus = "fail"
)

// Check represents a single validation check result.
type Check struct {
	Name    string      `json:"name"`
	Status  CheckStatus `json:"status"`
	Message string      `json:"message"`
}

// PaymentOptionDisplay contains formatted payment option info for display.
type PaymentOptionDisplay struct {
	Index       int    `json:"index"`
	Scheme      string `json:"scheme"`
	Network     string `json:"network"`
	NetworkName string `json:"networkName"`
	Amount      string `json:"amount"`
	AmountHuman string `json:"amountHuman"`
	Asset       string `json:"asset"`
	AssetSymbol string `json:"assetSymbol"`
	PayTo       string `json:"payTo"`
	Supported   bool   `json:"supported"`
}

// HealthResult contains the complete health check result.
type HealthResult struct {
	URL            string                 `json:"url"`
	Method         string                 `json:"method"`
	Status         int                    `json:"status"`
	StatusText     string                 `json:"statusText"`
	Latency        int64                  `json:"latency"`
	LatencyMs      int64                  `json:"latencyMs"`
	Protocol       string                 `json:"protocol"`
	PaymentOptions []PaymentOptionDisplay `json:"paymentOptions,omitempty"`
	Checks         []Check                `json:"checks"`
	ExitCode       int                    `json:"exitCode"`
	Error          string                 `json:"error,omitempty"`
}

// TestResult contains the complete test payment result.
type TestResult struct {
	URL             string               `json:"url"`
	Status          int                  `json:"status"`
	StatusText      string               `json:"statusText"`
	Protocol        string               `json:"protocol"`
	PaymentOption   PaymentOptionDisplay `json:"paymentOption"`
	Transaction     string               `json:"transaction,omitempty"`
	TransactionURL  string               `json:"transactionUrl,omitempty"`
	ResponseBody    string               `json:"responseBody,omitempty"`
	PaymentResponse interface{}          `json:"paymentResponse,omitempty"`
	DryRun          bool                 `json:"dryRun,omitempty"`
	ExitCode        int                  `json:"exitCode"`
	Error           string               `json:"error,omitempty"`
}

// PrintHealthResult outputs the health check result in human-readable format.
func PrintHealthResult(result *HealthResult, verbose bool) {
	fmt.Println()
	fmt.Println("x402 Health Check")
	fmt.Println("─────────────────")
	fmt.Printf("URL:        %s\n", result.URL)
	if result.Method != "" {
		fmt.Printf("Method:     %s\n", result.Method)
	}

	if result.Status > 0 {
		fmt.Printf("Status:     %s\n", result.StatusText)
	}

	if result.Protocol != "" {
		fmt.Printf("Protocol:   %s\n", formatProtocol(result.Protocol))
	}

	fmt.Printf("Latency:    %dms\n", result.LatencyMs)

	// Payment options
	if len(result.PaymentOptions) > 0 {
		fmt.Println()
		fmt.Println("Payment Options:")
		for _, opt := range result.PaymentOptions {
			supported := ""
			if opt.Supported {
				supported = " ✓ supported"
			} else {
				supported = " ✗ unsupported"
			}
			fmt.Printf("  [%d] %s on %s%s\n", opt.Index, opt.AmountHuman, opt.NetworkName, supported)

			if verbose {
				fmt.Printf("      Scheme: %s\n", opt.Scheme)
				fmt.Printf("      Asset:  %s\n", opt.Asset)
				fmt.Printf("      PayTo:  %s\n", opt.PayTo)
			}
		}
	}

	// Checks
	fmt.Println()
	fmt.Println("Checks:")
	for _, check := range result.Checks {
		icon := statusIcon(check.Status)
		fmt.Printf("  %s %s\n", icon, check.Name)
		if verbose || check.Status != StatusPass {
			fmt.Printf("    %s\n", check.Message)
		}
	}

	// Summary
	fmt.Println()
	failCount, warnCount := countChecks(result.Checks)
	if failCount > 0 {
		fmt.Printf("Result: FAILED (%d check(s) failed)\n", failCount)
	} else if warnCount > 0 {
		fmt.Printf("Result: PASSED with warnings (%d warning(s))\n", warnCount)
	} else {
		fmt.Println("Result: PASSED")
	}
}

// PrintTestResult outputs the test payment result in human-readable format.
func PrintTestResult(result *TestResult, verbose bool) {
	fmt.Println()
	if result.DryRun {
		fmt.Println("x402 Payment Test (DRY RUN)")
	} else {
		fmt.Println("x402 Payment Test")
	}
	fmt.Println("──────────────────")
	fmt.Printf("URL:        %s\n", result.URL)
	fmt.Printf("Status:     %s\n", result.StatusText)
	fmt.Printf("Protocol:   %s\n", formatProtocol(result.Protocol))

	fmt.Println()
	fmt.Println("Payment:")
	fmt.Printf("  Amount:   %s\n", result.PaymentOption.AmountHuman)
	fmt.Printf("  Network:  %s\n", result.PaymentOption.NetworkName)
	fmt.Printf("  Asset:    %s\n", tokens.FormatShortAddress(result.PaymentOption.Asset))
	fmt.Printf("  PayTo:    %s\n", tokens.FormatShortAddress(result.PaymentOption.PayTo))

	if result.Transaction != "" {
		fmt.Println()
		fmt.Println("Transaction:")
		fmt.Printf("  Hash: %s\n", result.Transaction)
		if result.TransactionURL != "" {
			fmt.Printf("  View: %s\n", result.TransactionURL)
		}
	}

	// Show response body (the actual API response after payment)
	if result.ResponseBody != "" && !result.DryRun {
		fmt.Println()
		fmt.Println("Response:")
		fmt.Println("─────────")
		fmt.Println(result.ResponseBody)
	}

	if result.DryRun {
		fmt.Println()
		fmt.Println("(Dry run - no payment was made)")
	}

	if result.Error != "" {
		fmt.Println()
		fmt.Printf("Error: %s\n", result.Error)
	}
}

// PrintError outputs an error message to stderr.
func PrintError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
}

// PrintWarning outputs a warning message to stderr.
func PrintWarning(msg string) {
	fmt.Fprintf(os.Stderr, "Warning: %s\n", msg)
}

// PrintInfo outputs an info message to stderr (for TTY vs pipe awareness).
func PrintInfo(msg string) {
	fmt.Fprintf(os.Stderr, "%s\n", msg)
}

// PromptConfirm prompts the user for yes/no confirmation.
// Returns true if user enters y/Y/yes.
func PromptConfirm(prompt string) bool {
	fmt.Fprintf(os.Stderr, "%s [y/N]: ", prompt)

	var response string
	fmt.Scanln(&response)

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// PromptSelect prompts the user to select from a list of options.
// Returns the 0-based index of the selected option.
func PromptSelect(prompt string, options []string) int {
	fmt.Fprintln(os.Stderr, prompt)
	for i, opt := range options {
		fmt.Fprintf(os.Stderr, "  [%d] %s\n", i+1, opt)
	}
	fmt.Fprint(os.Stderr, "Enter choice: ")

	var choice int
	fmt.Scanln(&choice)

	if choice < 1 || choice > len(options) {
		return 0 // Default to first option
	}
	return choice - 1
}

// Helper functions

func formatProtocol(protocol string) string {
	switch protocol {
	case "v1":
		return "v1 (legacy)"
	case "v2":
		return "v2 (current)"
	case "none":
		return "N/A (no payment required)"
	default:
		return protocol
	}
}

func statusIcon(status CheckStatus) string {
	switch status {
	case StatusPass:
		return "✓"
	case StatusWarn:
		return "⚠"
	case StatusFail:
		return "✗"
	default:
		return "?"
	}
}

func countChecks(checks []Check) (failCount, warnCount int) {
	for _, check := range checks {
		switch check.Status {
		case StatusFail:
			failCount++
		case StatusWarn:
			warnCount++
		}
	}
	return
}
