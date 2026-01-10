package x402

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildPayloadV2(t *testing.T) {
	resource := ResourceInfo{
		URL: "https://example.com/api",
	}

	option := &PaymentRequirement{
		Scheme:            "exact",
		Network:           "eip155:84532",
		Amount:            "1000",
		Asset:             "0x036cbd53842c5426634e7929541ec2318f3dcf7e",
		PayTo:             "0x64c2310BD1151266AA2Ad2410447E133b7F84e29",
		MaxTimeoutSeconds: 300,
		Extra: map[string]interface{}{
			"name":    "USDC",
			"version": "2",
		},
	}

	signature := "0xabc123"
	auth := Authorization{
		From:        "0xSender",
		To:          "0xRecipient",
		Value:       "1000",
		ValidAfter:  "0",
		ValidBefore: "9999999999",
		Nonce:       "0x123",
	}

	payload := BuildPayloadV2(resource, option, signature, auth)

	assert.Equal(t, ProtocolV2, payload.X402Version)
	assert.Equal(t, "https://example.com/api", payload.Resource.URL)
	assert.Equal(t, "exact", payload.Accepted.Scheme)
	assert.Equal(t, "eip155:84532", payload.Accepted.Network)
	assert.Equal(t, "1000", payload.Accepted.Amount)
	assert.Equal(t, "0xabc123", payload.Payload.Signature)
	assert.Equal(t, "0xSender", payload.Payload.Authorization.From)
}

func TestBuildPayloadV1(t *testing.T) {
	option := &PaymentRequirement{
		Scheme:            "exact",
		Network:           "eip155:84532",
		MaxAmountRequired: "2000",
		Asset:             "0x036cbd53842c5426634e7929541ec2318f3dcf7e",
		PayTo:             "0x64c2310BD1151266AA2Ad2410447E133b7F84e29",
	}

	signature := "0xdef456"
	auth := Authorization{
		From:        "0xSender",
		To:          "0xRecipient",
		Value:       "2000",
		ValidAfter:  "0",
		ValidBefore: "9999999999",
		Nonce:       "0x456",
	}

	payload := BuildPayloadV1(option, signature, auth)

	assert.Equal(t, ProtocolV1, payload.X402Version)
	assert.Equal(t, "exact", payload.Scheme)
	assert.Equal(t, "eip155:84532", payload.Network)
	assert.Equal(t, "0xdef456", payload.Payload.Signature)
	assert.Equal(t, "0xSender", payload.Payload.Authorization.From)
}

func TestEncodePayload(t *testing.T) {
	payload := map[string]string{
		"key": "value",
	}

	encoded, err := EncodePayload(payload)
	require.NoError(t, err)

	// Verify it's valid base64
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, err)

	// Verify it's valid JSON
	var result map[string]string
	err = json.Unmarshal(decoded, &result)
	require.NoError(t, err)
	assert.Equal(t, "value", result["key"])
}

func TestEncodePayload_ComplexStruct(t *testing.T) {
	payload := PaymentPayloadV2{
		X402Version: 2,
		Resource:    ResourceInfo{URL: "https://example.com"},
		Accepted:    AcceptedOption{Scheme: "exact", Network: "eip155:8453"},
		Payload: ExactEvmPayload{
			Signature: "0xsig",
			Authorization: Authorization{
				From:  "0xfrom",
				To:    "0xto",
				Value: "100",
			},
		},
	}

	encoded, err := EncodePayload(payload)
	require.NoError(t, err)

	// Decode and verify
	decoded, _ := base64.StdEncoding.DecodeString(encoded)
	var result PaymentPayloadV2
	err = json.Unmarshal(decoded, &result)
	require.NoError(t, err)
	assert.Equal(t, 2, result.X402Version)
	assert.Equal(t, "exact", result.Accepted.Scheme)
}

func TestBuildAndEncodePayload_V2(t *testing.T) {
	resource := ResourceInfo{URL: "https://example.com/api"}
	option := &PaymentRequirement{
		Scheme:  "exact",
		Network: "eip155:8453",
		Amount:  "1000",
	}
	auth := Authorization{
		From:  "0xfrom",
		To:    "0xto",
		Value: "1000",
	}

	headerName, headerValue, err := BuildAndEncodePayload(ProtocolV2, resource, option, "0xsig", auth)

	require.NoError(t, err)
	assert.Equal(t, HeaderPaymentSignature, headerName)
	assert.NotEmpty(t, headerValue)

	// Verify the encoded payload
	decoded, _ := base64.StdEncoding.DecodeString(headerValue)
	var payload PaymentPayloadV2
	err = json.Unmarshal(decoded, &payload)
	require.NoError(t, err)
	assert.Equal(t, 2, payload.X402Version)
}

func TestBuildAndEncodePayload_V1(t *testing.T) {
	resource := ResourceInfo{URL: "https://example.com/api"}
	option := &PaymentRequirement{
		Scheme:            "exact",
		Network:           "eip155:84532",
		MaxAmountRequired: "2000",
	}
	auth := Authorization{
		From:  "0xfrom",
		To:    "0xto",
		Value: "2000",
	}

	headerName, headerValue, err := BuildAndEncodePayload(ProtocolV1, resource, option, "0xsig", auth)

	require.NoError(t, err)
	assert.Equal(t, HeaderXPayment, headerName)
	assert.NotEmpty(t, headerValue)

	// Verify the encoded payload
	decoded, _ := base64.StdEncoding.DecodeString(headerValue)
	var payload PaymentPayloadV1
	err = json.Unmarshal(decoded, &payload)
	require.NoError(t, err)
	assert.Equal(t, 1, payload.X402Version)
}
