package commands

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/port402/x402-cli/internal/a2a"
	"github.com/port402/x402-cli/internal/client"
	"github.com/port402/x402-cli/internal/output"
	"github.com/port402/x402-cli/internal/tokens"
	"github.com/port402/x402-cli/internal/x402"
)

var (
	healthTimeout int
	healthMethod  string
	healthAgent   bool
)

var healthCmd = &cobra.Command{
	Use:   "health <url>",
	Short: "Check if an endpoint is x402-enabled",
	Long: `Health check for x402-enabled endpoints. No wallet required.

Validates:
  - Endpoint is reachable
  - Returns 402 Payment Required
  - Has valid payment requirements
  - Has EVM payment options
  - Uses known tokens

Use --agent to also discover A2A agent cards from the endpoint.

Examples:
  x402 health https://api.example.com/endpoint
  x402 health https://api.example.com/endpoint --json
  x402 health https://api.example.com/endpoint --verbose
  x402 health https://api.example.com/endpoint --method POST
  x402 health https://api.example.com/endpoint --agent`,
	Args: cobra.ExactArgs(1),
	RunE: runHealth,
}

func init() {
	healthCmd.Flags().IntVar(&healthTimeout, "timeout", 30, "Request timeout in seconds")
	healthCmd.Flags().StringVarP(&healthMethod, "method", "X", "GET", "HTTP method")
	healthCmd.Flags().BoolVar(&healthAgent, "agent", false, "Also discover A2A agent card")
	rootCmd.AddCommand(healthCmd)
}

func runHealth(cmd *cobra.Command, args []string) error {
	endpoint, err := normalizeURL(args[0])
	if err != nil {
		return fmt.Errorf("normalizing endpoint URL: %w", err)
	}
	timeout := time.Duration(healthTimeout) * time.Second

	result := checkHealth(endpoint, timeout, healthMethod)

	// Optionally discover agent card
	var agentResult *a2a.Result
	if healthAgent {
		agentResult = a2a.Discover(cmd.Context(), endpoint, "", timeout)
		result.AgentCard = agentResult
	}

	if GetJSONOutput() {
		return output.PrintJSON(result)
	}

	output.PrintHealthResult(result, GetVerbose())

	if result.ExitCode != 0 {
		cmd.SilenceUsage = true
		return fmt.Errorf("health check failed")
	}

	return nil
}

func checkHealth(url string, timeout time.Duration, method string) *output.HealthResult {
	result := &output.HealthResult{
		URL:      url,
		Method:   method,
		Checks:   []output.Check{},
		ExitCode: 0,
	}

	// Create HTTP client
	httpClient := client.New(client.WithTimeout(timeout))

	// Make request and measure latency
	reqResult, err := httpClient.TimedRequest(method, url, nil, nil)
	if err != nil {
		result.Checks = append(result.Checks, output.Check{
			Name:    "Endpoint reachable",
			Status:  output.StatusFail,
			Message: fmt.Sprintf("Connection failed: %v", err),
		})
		result.Error = err.Error()
		result.ExitCode = 3 // Network error
		return result
	}
	defer reqResult.Response.Body.Close()

	result.Latency = reqResult.LatencyMs
	result.LatencyMs = reqResult.LatencyMs
	result.Status = reqResult.Response.StatusCode
	result.StatusText = reqResult.Response.Status

	// Check 1: Endpoint reachable
	result.Checks = append(result.Checks, output.Check{
		Name:    "Endpoint reachable",
		Status:  output.StatusPass,
		Message: fmt.Sprintf("Connected in %dms", result.LatencyMs),
	})

	// Check 2: Returns 402
	if reqResult.Response.StatusCode == http.StatusPaymentRequired {
		result.Checks = append(result.Checks, output.Check{
			Name:    "Returns 402",
			Status:  output.StatusPass,
			Message: "402 Payment Required",
		})
	} else if reqResult.Response.StatusCode == http.StatusOK {
		result.Checks = append(result.Checks, output.Check{
			Name:    "Returns 402",
			Status:  output.StatusWarn,
			Message: fmt.Sprintf("Got %d - endpoint may not require payment", reqResult.Response.StatusCode),
		})
		result.Protocol = "none"
		return result
	} else if reqResult.Response.StatusCode == http.StatusTooManyRequests {
		retryAfter := client.ParseRetryAfter(reqResult.Response)
		msg := "Rate limited (429)"
		if retryAfter > 0 {
			msg = fmt.Sprintf("Rate limited (429) - retry after %v", retryAfter)
		}
		result.Checks = append(result.Checks, output.Check{
			Name:    "Returns 402",
			Status:  output.StatusFail,
			Message: msg,
		})
		result.ExitCode = 1
		return result
	} else {
		result.Checks = append(result.Checks, output.Check{
			Name:    "Returns 402",
			Status:  output.StatusFail,
			Message: fmt.Sprintf("Got %d instead of 402", reqResult.Response.StatusCode),
		})
		result.ExitCode = 1
		return result
	}

	// Check 3: Parse payment requirements
	parseResult, err := x402.ParsePaymentRequired(reqResult.Response)
	if err != nil {
		checkName := "Valid payment header"
		if reqResult.Response.Header.Get(x402.HeaderPaymentRequired) == "" {
			checkName = "Valid payment body"
		}
		result.Checks = append(result.Checks, output.Check{
			Name:    checkName,
			Status:  output.StatusFail,
			Message: err.Error(),
		})
		result.ExitCode = 4 // Protocol error
		return result
	}

	// Set protocol version
	if parseResult.ProtocolVersion == x402.ProtocolV2 {
		result.Protocol = "v2"
		result.Checks = append(result.Checks, output.Check{
			Name:    "Valid payment header",
			Status:  output.StatusPass,
			Message: "PAYMENT-REQUIRED header decoded successfully",
		})
	} else {
		result.Protocol = "v1"
		result.Checks = append(result.Checks, output.Check{
			Name:    "Valid payment body",
			Status:  output.StatusPass,
			Message: "Response body parsed as JSON successfully",
		})
	}

	// Check 4: Has payment options
	result.Checks = append(result.Checks, output.Check{
		Name:    "Has payment options",
		Status:  output.StatusPass,
		Message: fmt.Sprintf("%d payment option(s) found", len(parseResult.PaymentRequired.Accepts)),
	})

	// Process payment options
	hasEvmOption := false
	hasSolanaOption := false
	hasKnownToken := false

	for i, opt := range parseResult.PaymentRequired.Accepts {
		po := output.PaymentOptionDisplay{
			Index:   i + 1,
			Scheme:  opt.Scheme,
			Network: opt.Network,
			Amount:  opt.GetAmount(),
			Asset:   opt.Asset,
			PayTo:   opt.PayTo,
		}

		// Get network name (shows human name with raw identifier)
		humanName := tokens.GetNetworkName(opt.Network)
		po.NetworkName = fmt.Sprintf("%s (%s)", humanName, opt.Network)

		// Check if EVM or Solana network
		if x402.IsEVMNetwork(opt.Network) {
			po.Supported = true
			hasEvmOption = true
		} else if x402.IsSolanaNetwork(opt.Network) {
			po.Supported = true
			hasSolanaOption = true
		}

		// Look up token info
		if tokenInfo := tokens.GetTokenInfo(opt.Network, opt.Asset); tokenInfo != nil {
			po.AssetSymbol = tokenInfo.Symbol
			po.AmountHuman = tokens.FormatAmount(opt.GetAmount(), tokenInfo.Decimals, tokenInfo.Symbol)
			hasKnownToken = true
		} else {
			po.AssetSymbol = "UNKNOWN"
			po.AmountHuman = fmt.Sprintf("%s raw units", opt.GetAmount())
		}

		result.PaymentOptions = append(result.PaymentOptions, po)
	}

	// Check 5: Has supported network options
	if hasEvmOption {
		result.Checks = append(result.Checks, output.Check{
			Name:    "Has EVM option",
			Status:  output.StatusPass,
			Message: "EVM network supported",
		})
	}
	if hasSolanaOption {
		result.Checks = append(result.Checks, output.Check{
			Name:    "Has Solana option",
			Status:  output.StatusPass,
			Message: "Solana network supported",
		})
	}
	if !hasEvmOption && !hasSolanaOption {
		result.Checks = append(result.Checks, output.Check{
			Name:    "Has supported option",
			Status:  output.StatusWarn,
			Message: "No EVM or Solana options found",
		})
	}

	// Check 6: Known token
	if hasKnownToken {
		result.Checks = append(result.Checks, output.Check{
			Name:    "Known token",
			Status:  output.StatusPass,
			Message: "Token recognized in registry",
		})
	} else {
		result.Checks = append(result.Checks, output.Check{
			Name:    "Known token",
			Status:  output.StatusWarn,
			Message: "Token not in registry (amount displayed as raw units)",
		})
	}

	return result
}

// CheckHealthForBatch is exported for use by batch-health command.
// Always uses GET method for batch operations (backward compatible).
func CheckHealthForBatch(url string, timeout time.Duration) *output.HealthResult {
	return CheckHealthForBatchWithMethod(url, "GET", timeout)
}

// CheckHealthForBatchWithMethod is exported for use by batch-health command.
// Allows specifying the HTTP method.
func CheckHealthForBatchWithMethod(rawURL string, method string, timeout time.Duration) *output.HealthResult {
	normalized, err := normalizeURL(rawURL)
	if err != nil {
		return &output.HealthResult{
			URL:      rawURL,
			ExitCode: 3,
			Error:    err.Error(),
			Checks: []output.Check{{
				Name:    "Endpoint reachable",
				Status:  output.StatusFail,
				Message: err.Error(),
			}},
		}
	}
	if method == "" {
		method = "GET"
	}
	return checkHealth(normalized, timeout, strings.ToUpper(method))
}
