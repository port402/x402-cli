package commands

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/port402/x402-cli/internal/output"
	"github.com/port402/x402-cli/internal/x402"
)

func TestRunBatchChecks_MultipleURLs(t *testing.T) {
	paymentReq := &x402.PaymentRequired{
		X402Version: 2,
		Accepts: []x402.PaymentRequirement{{
			Scheme:  "exact",
			Network: "eip155:84532",
			Amount:  "1000",
			Asset:   "0x036cbd53842c5426634e7929541ec2318f3dcf7e",
			PayTo:   "0x64c2310BD1151266AA2Ad2410447E133b7F84e29",
		}},
	}

	// Create mock server
	server := createMock402Server(t, x402.ProtocolV2, paymentReq)
	defer server.Close()

	// Test with multiple URLs (same server)
	entries := []BatchEntry{
		{URL: server.URL, Method: "GET"},
		{URL: server.URL, Method: "GET"},
		{URL: server.URL, Method: "GET"},
	}
	timeout := 30 * time.Second

	results := runBatchChecks(entries, timeout, 2, 0, false)

	assert.Len(t, results, 3)
	for _, r := range results {
		assert.Equal(t, 0, r.ExitCode)
	}
}

func TestRunBatchChecks_SomeFail(t *testing.T) {
	paymentReq := &x402.PaymentRequired{
		X402Version: 2,
		Accepts: []x402.PaymentRequirement{{
			Scheme:  "exact",
			Network: "eip155:84532",
			Amount:  "1000",
			Asset:   "0x036cbd53842c5426634e7929541ec2318f3dcf7e",
			PayTo:   "0x64c2310BD1151266AA2Ad2410447E133b7F84e29",
		}},
	}

	// Create success server
	successServer := createMock402Server(t, x402.ProtocolV2, paymentReq)
	defer successServer.Close()

	// Create failure server
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failServer.Close()

	entries := []BatchEntry{
		{URL: successServer.URL, Method: "GET"},
		{URL: failServer.URL, Method: "GET"},
		{URL: successServer.URL, Method: "GET"},
	}
	results := runBatchChecks(entries, 30*time.Second, 1, 0, false)

	assert.Len(t, results, 3)

	// Count pass/fail
	passed := 0
	failed := 0
	for _, r := range results {
		if r.ExitCode == 0 {
			passed++
		} else {
			failed++
		}
	}
	assert.Equal(t, 2, passed)
	assert.Equal(t, 1, failed)
}

func TestRunBatchChecks_FailFast(t *testing.T) {
	// Create failure server
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failServer.Close()

	// With fail-fast, should stop after first failure
	entries := []BatchEntry{
		{URL: failServer.URL, Method: "GET"},
		{URL: failServer.URL, Method: "GET"},
		{URL: failServer.URL, Method: "GET"},
	}
	results := runBatchChecks(entries, 30*time.Second, 1, 0, true)

	// With sequential execution and fail-fast, should stop at first failure
	assert.GreaterOrEqual(t, len(results), 1)
	assert.NotEqual(t, 0, results[0].ExitCode)
}

func TestRunBatchChecks_Parallel(t *testing.T) {
	// Create slow server to test parallelism
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)

		paymentReq := &x402.PaymentRequired{
			X402Version: 2,
			Accepts: []x402.PaymentRequirement{{
				Scheme:  "exact",
				Network: "eip155:84532",
				Amount:  "1000",
				Asset:   "0x036cbd53842c5426634e7929541ec2318f3dcf7e",
				PayTo:   "0x64c2310BD1151266AA2Ad2410447E133b7F84e29",
			}},
		}

		jsonBytes, _ := json.Marshal(paymentReq)
		encoded := base64.StdEncoding.EncodeToString(jsonBytes)
		w.Header().Set(x402.HeaderPaymentRequired, encoded)
		w.WriteHeader(http.StatusPaymentRequired)
	}))
	defer server.Close()

	entries := []BatchEntry{
		{URL: server.URL, Method: "GET"},
		{URL: server.URL, Method: "GET"},
		{URL: server.URL, Method: "GET"},
		{URL: server.URL, Method: "GET"},
	}

	start := time.Now()
	results := runBatchChecks(entries, 30*time.Second, 4, 0, false) // 4 parallel
	duration := time.Since(start)

	assert.Len(t, results, 4)
	for _, r := range results {
		assert.Equal(t, 0, r.ExitCode)
	}

	// With parallelism, should take ~50-100ms, not 200ms+ (sequential)
	assert.Less(t, duration, 200*time.Millisecond)
}

func TestRunBatchChecks_CountResults(t *testing.T) {
	paymentReq := &x402.PaymentRequired{
		X402Version: 2,
		Accepts: []x402.PaymentRequirement{{
			Scheme:  "exact",
			Network: "eip155:84532",
			Amount:  "1000",
			Asset:   "0x036cbd53842c5426634e7929541ec2318f3dcf7e",
			PayTo:   "0x64c2310BD1151266AA2Ad2410447E133b7F84e29",
		}},
	}

	server := createMock402Server(t, x402.ProtocolV2, paymentReq)
	defer server.Close()

	entries := []BatchEntry{
		{URL: server.URL, Method: "GET"},
		{URL: server.URL, Method: "GET"},
	}
	results := runBatchChecks(entries, 30*time.Second, 1, 0, false)

	// Simulate counting logic from runBatchHealth
	passed := 0
	passedWarn := 0
	failed := 0
	for _, r := range results {
		if r.ExitCode != 0 {
			failed++
		} else {
			hasWarn := false
			for _, c := range r.Checks {
				if c.Status == output.StatusWarn {
					hasWarn = true
					break
				}
			}
			if hasWarn {
				passedWarn++
			} else {
				passed++
			}
		}
	}

	assert.Equal(t, 2, passed)
	assert.Equal(t, 0, passedWarn)
	assert.Equal(t, 0, failed)
}

func TestParseBatchInput_SimpleArray(t *testing.T) {
	// Test backward-compatible simple URL array format
	input := `["https://api1.example.com", "https://api2.example.com"]`

	entries, err := parseBatchInput([]byte(input))

	assert.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.Equal(t, "https://api1.example.com", entries[0].URL)
	assert.Equal(t, "GET", entries[0].Method) // Default method
	assert.Equal(t, "https://api2.example.com", entries[1].URL)
	assert.Equal(t, "GET", entries[1].Method)
}

func TestParseBatchInput_ObjectArray(t *testing.T) {
	// Test new object array format with method specification
	input := `[
		{"url": "https://api1.example.com", "method": "POST"},
		{"url": "https://api2.example.com"},
		{"url": "https://api3.example.com", "method": "put"}
	]`

	entries, err := parseBatchInput([]byte(input))

	assert.NoError(t, err)
	assert.Len(t, entries, 3)
	assert.Equal(t, "https://api1.example.com", entries[0].URL)
	assert.Equal(t, "POST", entries[0].Method)
	assert.Equal(t, "https://api2.example.com", entries[1].URL)
	assert.Equal(t, "GET", entries[1].Method) // Default when not specified
	assert.Equal(t, "https://api3.example.com", entries[2].URL)
	assert.Equal(t, "PUT", entries[2].Method) // Normalized to uppercase
}

func TestParseBatchInput_MissingURL(t *testing.T) {
	// Test error when URL is missing in object format
	input := `[{"method": "POST"}]`

	_, err := parseBatchInput([]byte(input))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing URL")
}

func TestParseBatchInput_InvalidJSON(t *testing.T) {
	// Test error for invalid JSON
	input := `not valid json`

	_, err := parseBatchInput([]byte(input))

	assert.Error(t, err)
}
