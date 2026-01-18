package commands

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/port402/x402-cli/internal/client"
	"github.com/port402/x402-cli/internal/output"
	"github.com/port402/x402-cli/internal/tokens"
	"github.com/port402/x402-cli/internal/wallet"
	"github.com/port402/x402-cli/internal/x402"
)

// Test command flags
var (
	keystorePath            string
	walletKey               string
	solanaKeypairPath       string
	requestData             string
	requestMethod           string
	requestHeaders          []string
	testTimeout             int
	dryRun                  bool
	skipPaymentConfirmation bool
	maxAmount               string
)

var testCmd = &cobra.Command{
	Use:   "test <url>",
	Short: "Make a test payment to an x402 endpoint",
	Long: `Test the full payment flow for an x402 endpoint.

This command:
  1. Makes an initial request to get payment requirements
  2. Signs the payment authorization (EIP-3009 for EVM, transaction for Solana)
  3. Retries with the payment signature
  4. Displays the result with transaction link

Supports both EVM (Ethereum, Base, etc.) and Solana payments.

Examples:
  # EVM: Using keystore file
  x402 test https://api.example.com/endpoint --keystore ~/.foundry/keystores/my-wallet

  # EVM: Using hex private key
  x402 test https://api.example.com/endpoint --wallet 0x...

  # Solana: Using keypair file
  x402 test https://api.example.com/endpoint --solana-keypair ~/.config/solana/id.json

  # Dry run (show payment details without paying)
  x402 test https://api.example.com/endpoint --keystore ~/.foundry/keystores/my-wallet --dry-run

  # Skip confirmation prompt (for scripting)
  x402 test https://api.example.com/endpoint --keystore ~/.foundry/keystores/my-wallet --skip-payment-confirmation

  # Set maximum payment amount
  x402 test https://api.example.com/endpoint --keystore ~/.foundry/keystores/my-wallet --max-amount 0.05`,
	Args: cobra.ExactArgs(1),
	RunE: runTest,
}

func init() {
	testCmd.Flags().StringVar(&keystorePath, "keystore", "", "Path to EVM keystore file")
	testCmd.Flags().StringVar(&walletKey, "wallet", "", "EVM hex private key (or use PRIVATE_KEY env)")
	testCmd.Flags().StringVar(&solanaKeypairPath, "solana-keypair", "", "Path to Solana keypair file")
	testCmd.Flags().StringVarP(&requestData, "data", "d", "", "Request body data")
	testCmd.Flags().StringVarP(&requestMethod, "method", "X", "GET", "HTTP method")
	testCmd.Flags().StringArrayVarP(&requestHeaders, "header", "H", nil, "Custom headers (repeatable)")
	testCmd.Flags().IntVar(&testTimeout, "timeout", 30, "Request timeout in seconds")
	testCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show payment details without paying")
	testCmd.Flags().BoolVar(&skipPaymentConfirmation, "skip-payment-confirmation", false, "Skip payment confirmation prompt")
	testCmd.Flags().StringVar(&maxAmount, "max-amount", "", "Maximum payment amount (e.g., 0.05)")

	rootCmd.AddCommand(testCmd)
}

func runTest(cmd *cobra.Command, args []string) error {
	url := args[0]
	timeout := time.Duration(testTimeout) * time.Second

	// Set up interrupt handler
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	signatureSent := false

	go func() {
		<-sigChan
		fmt.Fprintln(os.Stderr)
		if signatureSent {
			fmt.Fprintln(os.Stderr, "⚠ Warning: Payment signature was already sent to the server.")
			fmt.Fprintln(os.Stderr, "  The payment may still be processed. Check your wallet balance.")
		} else {
			fmt.Fprintln(os.Stderr, "Cancelled by user. No payment was made.")
		}
		os.Exit(1)
	}()

	// Step 1: Make initial request
	if GetVerbose() && !GetJSONOutput() {
		fmt.Fprintln(os.Stderr, "• Fetching payment requirements...")
	}

	httpClient := client.New(client.WithTimeout(timeout))

	// Build headers from "Key: Value" format
	headers := make(map[string]string)
	for _, h := range requestHeaders {
		if key, value, found := strings.Cut(h, ":"); found {
			headers[key] = strings.TrimPrefix(value, " ")
		}
	}

	var body []byte
	if requestData != "" {
		body = []byte(requestData)
	}

	reqResult, err := httpClient.TimedRequest(requestMethod, url, headers, body)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer reqResult.Response.Body.Close()

	// Check for 402
	if reqResult.Response.StatusCode != http.StatusPaymentRequired {
		if reqResult.Response.StatusCode == http.StatusOK {
			bodyBytes, _ := io.ReadAll(reqResult.Response.Body)
			if GetJSONOutput() {
				return output.PrintJSON(map[string]interface{}{
					"url":      url,
					"status":   200,
					"message":  "Endpoint does not require payment",
					"body":     string(bodyBytes),
					"exitCode": 0,
				})
			}
			fmt.Printf("Endpoint returned 200 OK (no payment required)\n")
			fmt.Printf("Response: %s\n", string(bodyBytes))
			return nil
		}
		return fmt.Errorf("expected 402 Payment Required, got %d", reqResult.Response.StatusCode)
	}

	// Step 2: Parse payment requirements
	if GetVerbose() && !GetJSONOutput() {
		fmt.Fprintln(os.Stderr, "• Parsing 402 response...")
	}

	parseResult, err := x402.ParsePaymentRequired(reqResult.Response)
	if err != nil {
		return fmt.Errorf("failed to parse payment requirements: %w", err)
	}

	// Find payment option based on provided credentials
	solanaOption := x402.FindSolanaOption(parseResult.PaymentRequired)
	evmOption := x402.FindEVMOption(parseResult.PaymentRequired)

	paymentOption, isSolana, err := selectPaymentOption(solanaOption, evmOption, solanaKeypairPath != "")
	if err != nil {
		return err
	}

	// Get chain ID for EVM or network name for Solana
	var chainID int64
	if !isSolana {
		var err error
		chainID, err = x402.ExtractChainID(paymentOption.Network)
		if err != nil {
			return fmt.Errorf("invalid network: %w", err)
		}
	}

	// Format payment info
	var amountHuman string
	var tokenKnown bool
	if tokenInfo := tokens.GetTokenInfo(paymentOption.Network, paymentOption.Asset); tokenInfo != nil {
		amountHuman = tokens.FormatAmount(paymentOption.GetAmount(), tokenInfo.Decimals, tokenInfo.Symbol)
		tokenKnown = true
	} else {
		amountHuman = fmt.Sprintf("%s raw units (unknown token)", paymentOption.GetAmount())
		tokenKnown = false
	}

	networkName := tokens.GetNetworkName(paymentOption.Network)

	// Check max-amount
	if maxAmount != "" && tokenKnown {
		tokenInfo := tokens.GetTokenInfo(paymentOption.Network, paymentOption.Asset)
		maxRaw, err := tokens.ParseHumanAmount(maxAmount, tokenInfo.Decimals)
		if err != nil {
			return fmt.Errorf("invalid --max-amount: %w", err)
		}
		if tokens.CompareAmounts(paymentOption.GetAmount(), maxRaw) > 0 {
			return fmt.Errorf("payment amount %s exceeds maximum %s %s", amountHuman, maxAmount, tokenInfo.Symbol)
		}
	}

	// Build result for display/output
	result := &output.TestResult{
		URL:        url,
		Status:     reqResult.Response.StatusCode,
		StatusText: reqResult.Response.Status,
		Protocol:   fmt.Sprintf("v%d", parseResult.ProtocolVersion),
		PaymentOption: output.PaymentOptionDisplay{
			Index:       1,
			Scheme:      paymentOption.Scheme,
			Network:     paymentOption.Network,
			NetworkName: networkName,
			Amount:      paymentOption.GetAmount(),
			AmountHuman: amountHuman,
			Asset:       paymentOption.Asset,
			PayTo:       paymentOption.PayTo,
			Supported:   true,
		},
		DryRun:   dryRun,
		ExitCode: 0,
	}

	// Show payment details
	if !GetJSONOutput() {
		fmt.Println()
		fmt.Printf("  Payment:  %s → %s\n", amountHuman, tokens.FormatShortAddress(paymentOption.PayTo))
		fmt.Printf("  Network:  %s\n", networkName)
		if !tokenKnown {
			fmt.Println()
			output.PrintWarning("Unknown token - verify amount manually before proceeding")
		}
		fmt.Println()
	}

	// Dry run - stop here
	if dryRun {
		if GetJSONOutput() {
			return output.PrintJSON(result)
		}
		fmt.Println("(Dry run - no payment will be made)")
		return nil
	}

	// Load wallet and create signer
	var fromAddress string
	var signer wallet.Signer

	if isSolana {
		// Load Solana keypair
		if GetVerbose() && !GetJSONOutput() {
			fmt.Fprintln(os.Stderr, "• Loading Solana keypair...")
		}

		solanaKey, err := wallet.LoadSolanaKeypair(solanaKeypairPath)
		if err != nil {
			return fmt.Errorf("failed to load Solana keypair: %w", err)
		}

		fromAddress = wallet.GetSolanaAddress(solanaKey)
		rpcURL := x402.GetSolanaRPCURL(paymentOption.Network)
		signer = wallet.NewSolanaSigner(solanaKey, rpcURL)
	} else {
		// Load EVM wallet
		if GetVerbose() && !GetJSONOutput() {
			fmt.Fprintln(os.Stderr, "• Loading wallet...")
		}

		privateKeyLoaded, err := wallet.LoadPrivateKey(keystorePath, walletKey, !output.IsStdinTTY())
		if err != nil {
			return fmt.Errorf("failed to load wallet: %w", err)
		}

		fromAddress = wallet.GetAddress(privateKeyLoaded)
		signer = wallet.NewEVMSigner(privateKeyLoaded)
	}

	if GetVerbose() && !GetJSONOutput() {
		fmt.Fprintf(os.Stderr, "  Wallet: %s\n", fromAddress)
	}

	// Confirmation prompt
	if !skipPaymentConfirmation && output.IsTTY() {
		if !output.PromptConfirm("Proceed with payment?") {
			fmt.Println("Cancelled by user. No payment was made.")
			return nil
		}
		fmt.Println()
	}

	// Step 4: Sign authorization
	if GetVerbose() && !GetJSONOutput() {
		if isSolana {
			fmt.Fprintln(os.Stderr, "• Building Solana transaction...")
		} else {
			fmt.Fprintln(os.Stderr, "• Signing EIP-3009 authorization...")
		}
	}

	// Prepare sign params based on chain type
	var signParams wallet.SignParams
	if isSolana {
		signParams = wallet.PrepareSolanaSignParams(paymentOption, fromAddress)
	} else {
		signParams = wallet.PrepareSignParams(paymentOption, fromAddress, chainID)
	}

	signResult, err := signer.Sign(signParams)
	if err != nil {
		return fmt.Errorf("failed to sign authorization: %w", err)
	}

	// Step 5: Build payment payload
	if GetVerbose() && !GetJSONOutput() {
		fmt.Fprintln(os.Stderr, "• Building payment payload...")
	}

	resource := parseResult.PaymentRequired.Resource
	if parseResult.ProtocolVersion == x402.ProtocolV1 {
		// v1 doesn't have resource in top-level
		resource = x402.ResourceInfo{
			URL: url,
		}
	}

	var headerName, headerValue string
	if isSolana {
		// Solana uses the transaction as the payload
		payload := x402.BuildPayloadV2Solana(resource, paymentOption, signResult.Signature)
		headerValue, err = x402.EncodePayload(payload)
		if err != nil {
			return fmt.Errorf("failed to encode Solana payload: %w", err)
		}
		headerName = x402.HeaderPaymentSignature
	} else {
		// EVM uses signature and authorization
		headerName, headerValue, err = x402.BuildAndEncodePayload(
			parseResult.ProtocolVersion,
			resource,
			paymentOption,
			signResult.Signature,
			signResult.Authorization,
		)
		if err != nil {
			return fmt.Errorf("failed to build payment payload: %w", err)
		}
	}

	// Step 6: Retry with payment
	if GetVerbose() && !GetJSONOutput() {
		fmt.Fprintln(os.Stderr, "• Sending payment...")
	}

	// Mark that signature has been sent (for Ctrl+C warning)
	signatureSent = true

	// Add payment header to existing headers
	headers[headerName] = headerValue

	retryResult, err := httpClient.TimedRequest(requestMethod, url, headers, body)
	if err != nil {
		return fmt.Errorf("retry request failed: %w", err)
	}
	defer retryResult.Response.Body.Close()

	// Read response body
	responseBody, _ := io.ReadAll(retryResult.Response.Body)
	result.ResponseBody = string(responseBody)
	result.Status = retryResult.Response.StatusCode
	result.StatusText = retryResult.Response.Status

	// Parse payment response header
	paymentResp, _ := x402.ParsePaymentResponse(retryResult.Response, parseResult.ProtocolVersion)
	if paymentResp != nil {
		result.PaymentResponse = paymentResp
		if paymentResp.Transaction != "" {
			result.Transaction = paymentResp.Transaction
			result.TransactionURL = tokens.GetExplorerURL(paymentOption.Network, paymentResp.Transaction)
		}
	}

	// Check success
	if retryResult.Response.StatusCode != http.StatusOK {
		result.ExitCode = 5 // Payment rejected
		result.Error = fmt.Sprintf("Payment failed: %d %s", retryResult.Response.StatusCode, retryResult.Response.Status)

		if GetJSONOutput() {
			return output.PrintJSON(result)
		}

		output.PrintTestResult(result, GetVerbose())
		return errors.New(result.Error)
	}

	// Success!
	if GetJSONOutput() {
		return output.PrintJSON(result)
	}

	// TTY vs pipe output
	if output.IsTTY() {
		output.PrintTestResult(result, GetVerbose())
	} else {
		// Pipe mode: response body to stdout, summary to stderr
		fmt.Print(result.ResponseBody)
		if result.Transaction != "" {
			fmt.Fprintf(os.Stderr, "Transaction: %s\n", result.Transaction)
			if result.TransactionURL != "" {
				fmt.Fprintf(os.Stderr, "View: %s\n", result.TransactionURL)
			}
		}
	}

	return nil
}

// selectPaymentOption chooses the appropriate payment option based on available options
// and whether the user provided a Solana keypair.
func selectPaymentOption(solanaOpt, evmOpt *x402.PaymentRequirement, hasSolanaKeypair bool) (*x402.PaymentRequirement, bool, error) {
	if hasSolanaKeypair {
		if solanaOpt != nil {
			return solanaOpt, true, nil
		}
		if evmOpt != nil {
			return nil, false, fmt.Errorf("endpoint does not accept Solana payments, but --solana-keypair was provided")
		}
		return nil, false, fmt.Errorf("no supported payment options found")
	}

	// Default to EVM
	if evmOpt != nil {
		return evmOpt, false, nil
	}
	if solanaOpt != nil {
		return nil, false, fmt.Errorf("endpoint only accepts Solana payments (use --solana-keypair)")
	}
	return nil, false, fmt.Errorf("no supported payment options found")
}
