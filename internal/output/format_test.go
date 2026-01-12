package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckStatus_Constants(t *testing.T) {
	assert.Equal(t, CheckStatus("pass"), StatusPass)
	assert.Equal(t, CheckStatus("warn"), StatusWarn)
	assert.Equal(t, CheckStatus("fail"), StatusFail)
}

func TestFormatProtocol(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"v1", "v1 (legacy)"},
		{"v2", "v2 (current)"},
		{"none", "N/A (no payment required)"},
		{"unknown", "unknown"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatProtocol(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		status   CheckStatus
		expected string
	}{
		{StatusPass, "✓"},
		{StatusWarn, "⚠"},
		{StatusFail, "✗"},
		{CheckStatus("unknown"), "?"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := statusIcon(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCountChecks(t *testing.T) {
	tests := []struct {
		name         string
		checks       []Check
		expectedFail int
		expectedWarn int
	}{
		{
			name: "all pass",
			checks: []Check{
				{Status: StatusPass},
				{Status: StatusPass},
			},
			expectedFail: 0,
			expectedWarn: 0,
		},
		{
			name: "one fail",
			checks: []Check{
				{Status: StatusPass},
				{Status: StatusFail},
			},
			expectedFail: 1,
			expectedWarn: 0,
		},
		{
			name: "one warn",
			checks: []Check{
				{Status: StatusPass},
				{Status: StatusWarn},
			},
			expectedFail: 0,
			expectedWarn: 1,
		},
		{
			name: "mixed",
			checks: []Check{
				{Status: StatusPass},
				{Status: StatusWarn},
				{Status: StatusFail},
				{Status: StatusFail},
			},
			expectedFail: 2,
			expectedWarn: 1,
		},
		{
			name:         "empty",
			checks:       []Check{},
			expectedFail: 0,
			expectedWarn: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failCount, warnCount := countChecks(tt.checks)
			assert.Equal(t, tt.expectedFail, failCount)
			assert.Equal(t, tt.expectedWarn, warnCount)
		})
	}
}

func TestHealthResult_Structure(t *testing.T) {
	result := HealthResult{
		URL:        "https://example.com/api",
		Status:     402,
		StatusText: "402 Payment Required",
		Latency:    150,
		LatencyMs:  150,
		Protocol:   "v2",
		PaymentOptions: []PaymentOptionDisplay{
			{
				Index:       0,
				Scheme:      "exact",
				Network:     "eip155:8453",
				NetworkName: "Base Mainnet",
				Amount:      "1000000",
				AmountHuman: "1.00 USDC",
				Supported:   true,
			},
		},
		Checks: []Check{
			{Name: "Reachable", Status: StatusPass, Message: "200ms"},
		},
		ExitCode: 0,
	}

	assert.Equal(t, "https://example.com/api", result.URL)
	assert.Equal(t, 402, result.Status)
	assert.Len(t, result.PaymentOptions, 1)
	assert.Len(t, result.Checks, 1)
}

func TestTestResult_Structure(t *testing.T) {
	result := TestResult{
		URL:        "https://example.com/api",
		Status:     200,
		StatusText: "200 OK",
		Protocol:   "v2",
		PaymentOption: PaymentOptionDisplay{
			AmountHuman: "1.00 USDC",
			NetworkName: "Base Mainnet",
		},
		Transaction:    "0xabc123",
		TransactionURL: "https://basescan.org/tx/0xabc123",
		ExitCode:       0,
	}

	assert.Equal(t, "https://example.com/api", result.URL)
	assert.Equal(t, "0xabc123", result.Transaction)
	assert.Equal(t, 0, result.ExitCode)
}

func TestPaymentOptionDisplay_Structure(t *testing.T) {
	opt := PaymentOptionDisplay{
		Index:       0,
		Scheme:      "exact",
		Network:     "eip155:84532",
		NetworkName: "Base Sepolia",
		Amount:      "1000",
		AmountHuman: "0.001 USDC",
		Asset:       "0x036cbd53842c5426634e7929541ec2318f3dcf7e",
		AssetSymbol: "USDC",
		PayTo:       "0x64c2310BD1151266AA2Ad2410447E133b7F84e29",
		Supported:   true,
	}

	assert.Equal(t, 0, opt.Index)
	assert.Equal(t, "exact", opt.Scheme)
	assert.True(t, opt.Supported)
}

func TestCleanErrorMessage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Payment failed: 500 500 Internal Server Error", "server returned 500 during payment verification"},
		{"Payment failed: 500 Internal Server Error", "server returned 500 during payment verification"},
		{"Payment failed: 401 Unauthorized", "payment authorization rejected (401)"},
		{"Payment failed: 403 Forbidden", "payment forbidden (403)"},
		{"Some other error", "Some other error"},
		{"Connection timeout", "Connection timeout"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := cleanErrorMessage(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatResponseBody(t *testing.T) {
	// Note: formatResponseBody only pretty-prints when IsTTY() is true.
	// In test environment, IsTTY() returns false, so we test the non-TTY path.

	// Non-JSON should pass through unchanged
	plainText := "Hello, world!"
	assert.Equal(t, plainText, formatResponseBody(plainText))

	// Invalid JSON should pass through unchanged
	invalidJSON := "{invalid json"
	assert.Equal(t, invalidJSON, formatResponseBody(invalidJSON))

	// Valid JSON passes through unchanged when not TTY (test environment)
	validJSON := `{"key":"value"}`
	result := formatResponseBody(validJSON)
	assert.Equal(t, validJSON, result)
}

// Note: PrintHealthResult, PrintTestResult, PrintError, PrintWarning,
// PrintInfo, PromptConfirm, and PromptSelect output to stdout/stderr
// and require terminal interaction. These are better tested manually
// or via integration tests.
