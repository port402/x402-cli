package x402

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockResponse creates a mock http.Response for testing
func mockResponse(statusCode int, body string, headers map[string]string) *http.Response {
	resp := &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
	for k, v := range headers {
		resp.Header.Set(k, v)
	}
	return resp
}

func TestParsePaymentRequired_V2_Success(t *testing.T) {
	pr := PaymentRequired{
		X402Version: 2,
		Resource: ResourceInfo{
			URL: "https://example.com/api",
		},
		Accepts: []PaymentRequirement{{
			Scheme:  "exact",
			Network: "eip155:84532",
			Amount:  "1000",
			Asset:   "0x036cbd53842c5426634e7929541ec2318f3dcf7e",
			PayTo:   "0x64c2310BD1151266AA2Ad2410447E133b7F84e29",
		}},
	}

	// Encode as base64 JSON
	jsonBytes, err := json.Marshal(pr)
	require.NoError(t, err)
	encoded := base64.StdEncoding.EncodeToString(jsonBytes)

	resp := mockResponse(402, "", map[string]string{
		HeaderPaymentRequired: encoded,
	})

	result, err := ParsePaymentRequired(resp)
	require.NoError(t, err)
	assert.Equal(t, ProtocolV2, result.ProtocolVersion)
	assert.Equal(t, encoded, result.RawHeader)
	assert.Len(t, result.PaymentRequired.Accepts, 1)
	assert.Equal(t, "1000", result.PaymentRequired.Accepts[0].Amount)
	assert.Equal(t, "eip155:84532", result.PaymentRequired.Accepts[0].Network)
}

func TestParsePaymentRequired_V1_Success(t *testing.T) {
	pr := PaymentRequired{
		X402Version: 1,
		Accepts: []PaymentRequirement{{
			Scheme:            "exact",
			Network:           "eip155:84532",
			MaxAmountRequired: "2000",
			Asset:             "0x036cbd53842c5426634e7929541ec2318f3dcf7e",
			PayTo:             "0x64c2310BD1151266AA2Ad2410447E133b7F84e29",
		}},
	}

	jsonBytes, err := json.Marshal(pr)
	require.NoError(t, err)

	// v1: No header, body contains JSON
	resp := mockResponse(402, string(jsonBytes), nil)

	result, err := ParsePaymentRequired(resp)
	require.NoError(t, err)
	assert.Equal(t, ProtocolV1, result.ProtocolVersion)
	assert.Empty(t, result.RawHeader)
	assert.Len(t, result.PaymentRequired.Accepts, 1)
	assert.Equal(t, "2000", result.PaymentRequired.Accepts[0].MaxAmountRequired)
}

func TestParsePaymentRequired_V2_InvalidBase64(t *testing.T) {
	resp := mockResponse(402, "", map[string]string{
		HeaderPaymentRequired: "not-valid-base64!!!",
	})

	_, err := ParsePaymentRequired(resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid base64")
}

func TestParsePaymentRequired_V2_InvalidJSON(t *testing.T) {
	// Valid base64 but invalid JSON
	encoded := base64.StdEncoding.EncodeToString([]byte("not json"))

	resp := mockResponse(402, "", map[string]string{
		HeaderPaymentRequired: encoded,
	})

	_, err := ParsePaymentRequired(resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON")
}

func TestParsePaymentRequired_V1_EmptyBody(t *testing.T) {
	resp := mockResponse(402, "", nil)

	_, err := ParsePaymentRequired(resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty response body")
}

func TestParsePaymentRequired_V1_InvalidJSON(t *testing.T) {
	resp := mockResponse(402, "not valid json", nil)

	_, err := ParsePaymentRequired(resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON")
}

func TestParsePaymentRequired_EmptyAccepts(t *testing.T) {
	pr := PaymentRequired{
		X402Version: 1,
		Accepts:     []PaymentRequirement{}, // Empty array
	}

	jsonBytes, _ := json.Marshal(pr)
	resp := mockResponse(402, string(jsonBytes), nil)

	_, err := ParsePaymentRequired(resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no payment options")
}

func TestParsePaymentRequired_MultipleOptions(t *testing.T) {
	pr := PaymentRequired{
		X402Version: 2,
		Accepts: []PaymentRequirement{
			{Scheme: "exact", Network: "eip155:1", Amount: "1000"},
			{Scheme: "exact", Network: "eip155:8453", Amount: "2000"},
			{Scheme: "exact", Network: "solana:mainnet", Amount: "3000"},
		},
	}

	jsonBytes, _ := json.Marshal(pr)
	encoded := base64.StdEncoding.EncodeToString(jsonBytes)

	resp := mockResponse(402, "", map[string]string{
		HeaderPaymentRequired: encoded,
	})

	result, err := ParsePaymentRequired(resp)
	require.NoError(t, err)
	assert.Len(t, result.PaymentRequired.Accepts, 3)
}

func TestParsePaymentResponse_V2_Success(t *testing.T) {
	pr := PaymentResponse{
		Success:     true,
		Transaction: "0xabc123",
		Network:     "eip155:84532",
	}

	jsonBytes, _ := json.Marshal(pr)
	encoded := base64.StdEncoding.EncodeToString(jsonBytes)

	resp := mockResponse(200, "", map[string]string{
		HeaderPaymentResponse: encoded,
	})

	result, err := ParsePaymentResponse(resp, ProtocolV2)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, "0xabc123", result.Transaction)
}

func TestParsePaymentResponse_V1_Success(t *testing.T) {
	pr := PaymentResponse{
		Success:     true,
		Transaction: "0xdef456",
	}

	jsonBytes, _ := json.Marshal(pr)
	encoded := base64.StdEncoding.EncodeToString(jsonBytes)

	resp := mockResponse(200, "", map[string]string{
		HeaderXPaymentResponse: encoded,
	})

	result, err := ParsePaymentResponse(resp, ProtocolV1)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "0xdef456", result.Transaction)
}

func TestParsePaymentResponse_NoHeader(t *testing.T) {
	resp := mockResponse(200, "", nil)

	result, err := ParsePaymentResponse(resp, ProtocolV2)
	assert.NoError(t, err)
	assert.Nil(t, result) // No error, but no response
}

func TestIsEVMNetwork(t *testing.T) {
	tests := []struct {
		network  string
		expected bool
	}{
		{"eip155:8453", true},
		{"eip155:1", true},
		{"eip155:84532", true},
		{"eip155:11155111", true},
		{"solana:mainnet", false},
		{"solana:devnet", false},
		{"", false},
		{"eip155", false}, // Missing chain ID
		{"eip155:", false},
	}

	for _, tt := range tests {
		t.Run(tt.network, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsEVMNetwork(tt.network))
		})
	}
}

func TestExtractChainID(t *testing.T) {
	tests := []struct {
		network    string
		expectedID int64
		wantErr    bool
	}{
		{"eip155:8453", 8453, false},
		{"eip155:1", 1, false},
		{"eip155:84532", 84532, false},
		{"eip155:11155111", 11155111, false},
		{"solana:mainnet", 0, true},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.network, func(t *testing.T) {
			id, err := ExtractChainID(tt.network)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, id)
			}
		})
	}
}

func TestFindEVMOption(t *testing.T) {
	pr := &PaymentRequired{
		Accepts: []PaymentRequirement{
			{Network: "solana:mainnet", Amount: "1000"},
			{Network: "eip155:8453", Amount: "2000"},
			{Network: "eip155:1", Amount: "3000"},
		},
	}

	option := FindEVMOption(pr)
	require.NotNil(t, option)
	assert.Equal(t, "eip155:8453", option.Network)
	assert.Equal(t, "2000", option.Amount)
}

func TestFindEVMOption_None(t *testing.T) {
	pr := &PaymentRequired{
		Accepts: []PaymentRequirement{
			{Network: "solana:mainnet", Amount: "1000"},
			{Network: "solana:devnet", Amount: "2000"},
		},
	}

	option := FindEVMOption(pr)
	assert.Nil(t, option)
}

func TestHasOnlySolanaOptions(t *testing.T) {
	tests := []struct {
		name     string
		accepts  []PaymentRequirement
		expected bool
	}{
		{
			name: "only solana",
			accepts: []PaymentRequirement{
				{Network: "solana:mainnet"},
				{Network: "solana:devnet"},
			},
			expected: true,
		},
		{
			name: "mixed",
			accepts: []PaymentRequirement{
				{Network: "solana:mainnet"},
				{Network: "eip155:8453"},
			},
			expected: false,
		},
		{
			name: "only evm",
			accepts: []PaymentRequirement{
				{Network: "eip155:1"},
				{Network: "eip155:8453"},
			},
			expected: false,
		},
		{
			name:     "empty",
			accepts:  []PaymentRequirement{},
			expected: false, // No options means not "only Solana"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := &PaymentRequired{Accepts: tt.accepts}
			assert.Equal(t, tt.expected, HasOnlySolanaOptions(pr))
		})
	}
}
