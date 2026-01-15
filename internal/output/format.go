package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/port402/x402-cli/internal/a2a"
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
	AgentCard      *a2a.Result            `json:"agentCard,omitempty"`
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
	failCount, warnCount := countChecks(result.Checks)

	// Status line with URL
	if failCount > 0 {
		fmt.Printf("✗ %s\n", result.URL)
	} else if warnCount > 0 {
		fmt.Printf("⚠ %s\n", result.URL)
	} else {
		fmt.Printf("✓ %s\n", result.URL)
	}

	// Details section
	fmt.Println()
	if result.Status > 0 {
		fmt.Printf("  Status:   %s\n", result.StatusText)
	}
	if result.Protocol != "" {
		fmt.Printf("  Protocol: %s\n", formatProtocol(result.Protocol))
	}
	fmt.Printf("  Latency:  %dms\n", result.LatencyMs)

	// Payment - consolidated single line
	if len(result.PaymentOptions) > 0 {
		opt := result.PaymentOptions[0]
		fmt.Printf("  Payment:  %s on %s\n", opt.AmountHuman, tokens.GetNetworkName(opt.Network))

		// Show additional options in verbose mode
		if verbose && len(result.PaymentOptions) > 1 {
			for i, opt := range result.PaymentOptions[1:] {
				fmt.Printf("            [%d] %s on %s\n", i+2, opt.AmountHuman, tokens.GetNetworkName(opt.Network))
			}
		}
	}

	// Checks
	fmt.Println()
	fmt.Println("  Checks:")
	for _, check := range result.Checks {
		icon := statusIcon(check.Status)
		fmt.Printf("    %s %s\n", icon, check.Name)
		if check.Status != StatusPass {
			fmt.Printf("      %s\n", check.Message)
		}
	}

	// Agent card section (when --agent flag used)
	if result.AgentCard != nil {
		printAgentSection(result.AgentCard)
	}

	// Error line for failures (no "Result:" footer)
	if failCount > 0 {
		fmt.Println()
		fmt.Printf("Error: endpoint is not x402-enabled\n")
	}
}

// printAgentSection outputs the agent card discovery result.
func printAgentSection(result *a2a.Result) {
	fmt.Println()
	if !result.Found {
		fmt.Println("  Agent:    not found")
		return
	}

	card := result.Card
	fmt.Printf("  Agent:    %s v%s\n", card.Name, card.Version)
	if card.Description != "" {
		fmt.Printf("            %s\n", card.Description)
	}

	// Provider info
	if card.Provider != nil && card.Provider.Organization != "" {
		provider := card.Provider.Organization
		if card.Provider.URL != "" {
			provider = fmt.Sprintf("%s (%s)", provider, card.Provider.URL)
		}
		fmt.Printf("  Provider: %s\n", provider)
	}

	// Skills list
	fmt.Println()
	if len(card.Skills) == 0 {
		fmt.Println("  Skills:   none")
	} else {
		fmt.Println("  Skills:")
		for _, skill := range card.Skills {
			fmt.Printf("    • %s\n", skill.Name)
			if skill.Description != "" {
				fmt.Printf("      %s\n", truncateText(skill.Description, 60))
			}
		}
	}

	// Capabilities
	caps := formatCapabilities(card.Capabilities)
	if caps != "" {
		fmt.Println()
		fmt.Printf("  Capabilities: %s\n", caps)
	}
}

// formatCapabilities returns a comma-separated list of enabled capabilities.
func formatCapabilities(caps *a2a.Capabilities) string {
	if caps == nil {
		return ""
	}

	var enabled []string
	if caps.Streaming {
		enabled = append(enabled, "streaming")
	}
	if caps.PushNotifications {
		enabled = append(enabled, "push")
	}
	return strings.Join(enabled, ", ")
}

// PrintTestResult outputs the test payment result in human-readable format.
func PrintTestResult(result *TestResult, verbose bool) {
	// Status line
	if result.Error != "" {
		fmt.Println("✗ Payment failed")
	} else if result.DryRun {
		fmt.Println("• Dry run complete")
	} else {
		fmt.Println("✓ Payment successful")
	}

	// Details section
	fmt.Println()
	fmt.Printf("  URL:      %s\n", result.URL)
	fmt.Printf("  Status:   %s\n", result.StatusText)
	fmt.Printf("  Payment:  %s on %s\n", result.PaymentOption.AmountHuman, tokens.GetNetworkName(result.PaymentOption.Network))

	// Transaction info (on success)
	if result.Transaction != "" {
		fmt.Printf("  TxHash:   %s\n", result.Transaction)
		if result.TransactionURL != "" {
			fmt.Printf("  View:     %s\n", result.TransactionURL)
		}
	}

	// Response body
	if result.ResponseBody != "" && !result.DryRun && result.Error == "" {
		fmt.Println()
		fmt.Println("Response:")
		fmt.Println(formatResponseBody(result.ResponseBody))
	}

	// Dry run notice
	if result.DryRun {
		fmt.Println()
		fmt.Println("No payment was made (dry run)")
	}

	// Error with hint
	if result.Error != "" {
		fmt.Println()
		fmt.Printf("Error: %s\n", cleanErrorMessage(result.Error))
		// Add helpful hint for server errors
		if strings.Contains(result.Error, "500") || strings.Contains(result.Error, "Internal Server Error") {
			fmt.Println("Hint:  your funds were not transferred (authorization was not settled)")
		}
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

// truncateText truncates a string to maxLen characters, adding "..." if truncated.
func truncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
}

// errorMessageReplacements maps status codes to clean error messages for payment failures.
var errorMessageReplacements = map[string]string{
	"500": "server returned 500 during payment verification",
	"401": "payment authorization rejected (401)",
	"403": "payment forbidden (403)",
}

// cleanErrorMessage removes redundant status code duplication from error messages.
// e.g., "Payment failed: 500 500 Internal Server Error" -> "server returned 500 during payment verification"
func cleanErrorMessage(msg string) string {
	if !strings.HasPrefix(msg, "Payment failed:") {
		return msg
	}

	for code, replacement := range errorMessageReplacements {
		if strings.Contains(msg, code) {
			return replacement
		}
	}
	return msg
}

// maxPrettyPrintSize is the maximum response size (in bytes) to pretty-print.
// Larger responses are returned raw to avoid terminal lag and memory issues.
const maxPrettyPrintSize = 50 * 1024 // 50KB

// formatResponseBody pretty-prints JSON when outputting to a terminal,
// otherwise returns the raw body for piping to other tools.
func formatResponseBody(body string) string {
	// Only pretty-print for TTY output and small responses
	if !IsTTY() || len(body) > maxPrettyPrintSize {
		return body
	}

	// json.Indent returns an error for invalid JSON, so no need to pre-validate
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, []byte(body), "", "  "); err != nil {
		return body
	}
	return pretty.String()
}
