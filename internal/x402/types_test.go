package x402

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPaymentRequirement_GetAmount_V2(t *testing.T) {
	// v2 uses Amount field
	pr := &PaymentRequirement{
		Amount: "1000000",
	}

	assert.Equal(t, "1000000", pr.GetAmount())
}

func TestPaymentRequirement_GetAmount_V1(t *testing.T) {
	// v1 uses MaxAmountRequired field
	pr := &PaymentRequirement{
		MaxAmountRequired: "500000",
	}

	assert.Equal(t, "500000", pr.GetAmount())
}

func TestPaymentRequirement_GetAmount_V2Priority(t *testing.T) {
	// If both are set, Amount (v2) takes priority
	pr := &PaymentRequirement{
		Amount:            "1000000",
		MaxAmountRequired: "500000",
	}

	assert.Equal(t, "1000000", pr.GetAmount())
}

func TestPaymentRequirement_GetAmount_Empty(t *testing.T) {
	// Neither set returns empty string
	pr := &PaymentRequirement{}

	assert.Equal(t, "", pr.GetAmount())
}

func TestPaymentRequirement_GetExtraString(t *testing.T) {
	pr := &PaymentRequirement{
		Extra: map[string]interface{}{
			"name":    "USDC",
			"version": "2",
		},
	}

	assert.Equal(t, "USDC", pr.GetExtraString("name"))
	assert.Equal(t, "2", pr.GetExtraString("version"))
}

func TestPaymentRequirement_GetExtraString_Missing(t *testing.T) {
	pr := &PaymentRequirement{
		Extra: map[string]interface{}{
			"name": "USDC",
		},
	}

	assert.Equal(t, "", pr.GetExtraString("nonexistent"))
}

func TestPaymentRequirement_GetExtraString_NilExtra(t *testing.T) {
	pr := &PaymentRequirement{
		Extra: nil,
	}

	assert.Equal(t, "", pr.GetExtraString("name"))
}

func TestPaymentRequirement_GetExtraString_WrongType(t *testing.T) {
	pr := &PaymentRequirement{
		Extra: map[string]interface{}{
			"name":    "USDC",
			"numeric": 123,
		},
	}

	// Non-string values return empty string
	assert.Equal(t, "", pr.GetExtraString("numeric"))
}

func TestProtocolConstants(t *testing.T) {
	assert.Equal(t, 1, ProtocolV1)
	assert.Equal(t, 2, ProtocolV2)
}

func TestHeaderConstants(t *testing.T) {
	// v2 headers
	assert.Equal(t, "Payment-Required", HeaderPaymentRequired)
	assert.Equal(t, "Payment-Signature", HeaderPaymentSignature)
	assert.Equal(t, "Payment-Response", HeaderPaymentResponse)

	// v1 headers
	assert.Equal(t, "X-Payment", HeaderXPayment)
	assert.Equal(t, "X-Payment-Response", HeaderXPaymentResponse)
}
