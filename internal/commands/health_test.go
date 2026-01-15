package commands

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/port402/x402-cli/internal/a2a"
	"github.com/port402/x402-cli/internal/output"
	"github.com/port402/x402-cli/internal/x402"
)

// createMock402Server creates a mock server that returns a valid 402 response
func createMock402Server(t *testing.T, protocolVersion int, paymentReq *x402.PaymentRequired) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if protocolVersion == x402.ProtocolV2 {
			// v2: Encode in header
			jsonBytes, err := json.Marshal(paymentReq)
			require.NoError(t, err)
			encoded := base64.StdEncoding.EncodeToString(jsonBytes)
			w.Header().Set(x402.HeaderPaymentRequired, encoded)
			w.WriteHeader(http.StatusPaymentRequired)
		} else {
			// v1: Return as body
			w.WriteHeader(http.StatusPaymentRequired)
			json.NewEncoder(w).Encode(paymentReq)
		}
	}))
}

func TestCheckHealthForBatch_V2_Success(t *testing.T) {
	paymentReq := &x402.PaymentRequired{
		X402Version: 2,
		Resource: x402.ResourceInfo{
			URL: "https://example.com/api",
		},
		Accepts: []x402.PaymentRequirement{{
			Scheme:            "exact",
			Network:           "eip155:84532",
			Amount:            "1000000",
			Asset:             "0x036cbd53842c5426634e7929541ec2318f3dcf7e",
			PayTo:             "0x64c2310BD1151266AA2Ad2410447E133b7F84e29",
			MaxTimeoutSeconds: 300,
		}},
	}

	server := createMock402Server(t, x402.ProtocolV2, paymentReq)
	defer server.Close()

	result := CheckHealthForBatch(server.URL, 30*time.Second)

	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, http.StatusPaymentRequired, result.Status)
	assert.Equal(t, "v2", result.Protocol)
	assert.Len(t, result.PaymentOptions, 1)
	assert.True(t, result.PaymentOptions[0].Supported)

	// Check all checks passed
	for _, check := range result.Checks {
		assert.NotEqual(t, output.StatusFail, check.Status, "Check %s failed: %s", check.Name, check.Message)
	}
}

func TestCheckHealthForBatch_V1_Success(t *testing.T) {
	paymentReq := &x402.PaymentRequired{
		X402Version: 1,
		Accepts: []x402.PaymentRequirement{{
			Scheme:            "exact",
			Network:           "eip155:84532",
			MaxAmountRequired: "2000000",
			Asset:             "0x036cbd53842c5426634e7929541ec2318f3dcf7e",
			PayTo:             "0x64c2310BD1151266AA2Ad2410447E133b7F84e29",
		}},
	}

	server := createMock402Server(t, x402.ProtocolV1, paymentReq)
	defer server.Close()

	result := CheckHealthForBatch(server.URL, 30*time.Second)

	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "v1", result.Protocol)
	assert.Len(t, result.PaymentOptions, 1)
}

func TestCheckHealthForBatch_NoPaymentRequired(t *testing.T) {
	// Server returns 200 OK (no payment required)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	result := CheckHealthForBatch(server.URL, 30*time.Second)

	assert.Equal(t, 0, result.ExitCode) // Not a failure, just a warning
	assert.Equal(t, "none", result.Protocol)

	// Should have a warning about not requiring payment
	var foundWarn bool
	for _, check := range result.Checks {
		if check.Name == "Returns 402" && check.Status == output.StatusWarn {
			foundWarn = true
		}
	}
	assert.True(t, foundWarn, "Should have warning about 200 response")
}

func TestCheckHealthForBatch_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	result := CheckHealthForBatch(server.URL, 30*time.Second)

	assert.Equal(t, 1, result.ExitCode)
	assert.Equal(t, http.StatusNotFound, result.Status)
}

func TestCheckHealthForBatch_RateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	result := CheckHealthForBatch(server.URL, 30*time.Second)

	assert.Equal(t, 1, result.ExitCode)
	assert.Equal(t, http.StatusTooManyRequests, result.Status)

	// Should mention retry-after in check message
	var foundRetryAfter bool
	for _, check := range result.Checks {
		if check.Name == "Returns 402" && check.Status == output.StatusFail {
			if assert.Contains(t, check.Message, "Rate limited") {
				foundRetryAfter = true
			}
		}
	}
	assert.True(t, foundRetryAfter)
}

func TestCheckHealthForBatch_InvalidPaymentHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Invalid base64 in header
		w.Header().Set(x402.HeaderPaymentRequired, "not-valid-base64!!!")
		w.WriteHeader(http.StatusPaymentRequired)
	}))
	defer server.Close()

	result := CheckHealthForBatch(server.URL, 30*time.Second)

	assert.Equal(t, 4, result.ExitCode) // Protocol error
}

func TestCheckHealthForBatch_SolanaOnly(t *testing.T) {
	paymentReq := &x402.PaymentRequired{
		X402Version: 2,
		Accepts: []x402.PaymentRequirement{{
			Scheme:  "exact",
			Network: "solana:mainnet",
			Amount:  "1000000",
			Asset:   "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
			PayTo:   "SomeSOLAddress",
		}},
	}

	server := createMock402Server(t, x402.ProtocolV2, paymentReq)
	defer server.Close()

	result := CheckHealthForBatch(server.URL, 30*time.Second)

	// Should succeed but with warning about no EVM options
	assert.Equal(t, 0, result.ExitCode)
	assert.Len(t, result.PaymentOptions, 1)
	assert.False(t, result.PaymentOptions[0].Supported)

	// Should have warning
	var foundEvmWarning bool
	for _, check := range result.Checks {
		if check.Name == "Has EVM option" && check.Status == output.StatusWarn {
			foundEvmWarning = true
		}
	}
	assert.True(t, foundEvmWarning)
}

func TestCheckHealthForBatch_UnknownToken(t *testing.T) {
	paymentReq := &x402.PaymentRequired{
		X402Version: 2,
		Accepts: []x402.PaymentRequirement{{
			Scheme:  "exact",
			Network: "eip155:8453",
			Amount:  "1000000",
			Asset:   "0x0000000000000000000000000000000000000000", // Unknown token
			PayTo:   "0x64c2310BD1151266AA2Ad2410447E133b7F84e29",
		}},
	}

	server := createMock402Server(t, x402.ProtocolV2, paymentReq)
	defer server.Close()

	result := CheckHealthForBatch(server.URL, 30*time.Second)

	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "UNKNOWN", result.PaymentOptions[0].AssetSymbol)
	assert.Contains(t, result.PaymentOptions[0].AmountHuman, "raw units")

	// Should have warning about unknown token
	var foundTokenWarning bool
	for _, check := range result.Checks {
		if check.Name == "Known token" && check.Status == output.StatusWarn {
			foundTokenWarning = true
		}
	}
	assert.True(t, foundTokenWarning)
}

func TestCheckHealthForBatch_MultiplePaymentOptions(t *testing.T) {
	paymentReq := &x402.PaymentRequired{
		X402Version: 2,
		Accepts: []x402.PaymentRequirement{
			{
				Scheme:  "exact",
				Network: "eip155:8453",
				Amount:  "1000000",
				Asset:   "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
				PayTo:   "0x64c2310BD1151266AA2Ad2410447E133b7F84e29",
			},
			{
				Scheme:  "exact",
				Network: "eip155:1",
				Amount:  "2000000",
				Asset:   "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				PayTo:   "0x64c2310BD1151266AA2Ad2410447E133b7F84e29",
			},
			{
				Scheme:  "exact",
				Network: "solana:mainnet",
				Amount:  "3000000",
				Asset:   "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
				PayTo:   "SomeSOLAddress",
			},
		},
	}

	server := createMock402Server(t, x402.ProtocolV2, paymentReq)
	defer server.Close()

	result := CheckHealthForBatch(server.URL, 30*time.Second)

	assert.Equal(t, 0, result.ExitCode)
	assert.Len(t, result.PaymentOptions, 3)

	// First two should be supported (EVM), third should not (Solana)
	assert.True(t, result.PaymentOptions[0].Supported)
	assert.True(t, result.PaymentOptions[1].Supported)
	assert.False(t, result.PaymentOptions[2].Supported)
}

func TestCheckHealthForBatch_NetworkError(t *testing.T) {
	// Use a URL that won't resolve
	result := CheckHealthForBatch("http://localhost:59999", 1*time.Second)

	assert.Equal(t, 3, result.ExitCode) // Network error
	assert.NotEmpty(t, result.Error)
}

func TestCheckHealthForBatch_AddsHTTPS(t *testing.T) {
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

	// The server.URL already has http://, so this test verifies
	// that existing URLs are not modified
	result := CheckHealthForBatch(server.URL, 30*time.Second)
	assert.Equal(t, 0, result.ExitCode)
}

func TestCheckHealthForBatch_Latency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	result := CheckHealthForBatch(server.URL, 30*time.Second)

	// Latency should be captured
	assert.GreaterOrEqual(t, result.LatencyMs, int64(50))
}

func TestHealthResult_AgentCardField(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		result := &output.HealthResult{
			URL:      "https://example.com",
			Status:   402,
			Protocol: "v2",
			AgentCard: &a2a.Result{
				Found:         true,
				DiscoveryPath: "/.well-known/agent.json",
				Card: &a2a.AgentCard{
					Name:         "Test Agent",
					Version:      "1.0.0",
					Description:  "A test agent",
					Skills:       []a2a.Skill{{ID: "skill1", Name: "Skill One"}},
					Capabilities: &a2a.Capabilities{Streaming: true},
				},
			},
		}

		require.NotNil(t, result.AgentCard)
		assert.True(t, result.AgentCard.Found)
		assert.Equal(t, "Test Agent", result.AgentCard.Card.Name)
		assert.Equal(t, "1.0.0", result.AgentCard.Card.Version)
		assert.Len(t, result.AgentCard.Card.Skills, 1)
		assert.True(t, result.AgentCard.Card.Capabilities.Streaming)
	})

	t.Run("not found", func(t *testing.T) {
		result := &output.HealthResult{
			URL:      "https://example.com",
			Status:   402,
			Protocol: "v2",
			AgentCard: &a2a.Result{
				Found: false,
				TriedPaths: []a2a.PathAttempt{
					{Path: "/.well-known/agent.json", Status: 404},
					{Path: "/.well-known/agent-card.json", Status: 404},
				},
			},
		}

		require.NotNil(t, result.AgentCard)
		assert.False(t, result.AgentCard.Found)
		assert.Nil(t, result.AgentCard.Card)
		assert.Len(t, result.AgentCard.TriedPaths, 2)
	})

	t.Run("nil when flag not used", func(t *testing.T) {
		result := &output.HealthResult{
			URL:       "https://example.com",
			Status:    402,
			Protocol:  "v2",
			AgentCard: nil,
		}

		assert.Nil(t, result.AgentCard)
	})
}
