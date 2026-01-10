// Package x402 implements the x402 payment protocol types and parsing.
package x402

// PaymentRequired represents the decoded payment requirements from a 402 response.
// Supports both v1 and v2 protocol formats.
type PaymentRequired struct {
	X402Version int                  `json:"x402Version"`
	Error       string               `json:"error,omitempty"`
	Resource    ResourceInfo         `json:"resource,omitempty"`
	Accepts     []PaymentRequirement `json:"accepts"`
}

// ResourceInfo describes the protected resource (v2 only, extracted from options in v1).
type ResourceInfo struct {
	URL         string `json:"url,omitempty"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// PaymentRequirement represents a single payment option from the accepts[] array.
type PaymentRequirement struct {
	Scheme            string                 `json:"scheme"`
	Network           string                 `json:"network"`
	Amount            string                 `json:"amount,omitempty"`            // v2 field
	MaxAmountRequired string                 `json:"maxAmountRequired,omitempty"` // v1 field
	Asset             string                 `json:"asset"`
	PayTo             string                 `json:"payTo"`
	MaxTimeoutSeconds int                    `json:"maxTimeoutSeconds,omitempty"`
	Extra             map[string]interface{} `json:"extra,omitempty"`

	// v1 includes resource info directly in each payment option
	ResourcePath string `json:"resource,omitempty"`
	Description  string `json:"description,omitempty"`
	MimeType     string `json:"mimeType,omitempty"`
}

// GetAmount returns the payment amount, handling v1 vs v2 field naming.
// v2 uses "amount", v1 uses "maxAmountRequired".
func (p *PaymentRequirement) GetAmount() string {
	if p.Amount != "" {
		return p.Amount
	}
	return p.MaxAmountRequired
}

// GetExtraString retrieves a string value from the Extra map.
func (p *PaymentRequirement) GetExtraString(key string) string {
	if p.Extra == nil {
		return ""
	}
	if v, ok := p.Extra[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// Authorization contains EIP-3009 TransferWithAuthorization parameters.
type Authorization struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Value       string `json:"value"`
	ValidAfter  string `json:"validAfter"`
	ValidBefore string `json:"validBefore"`
	Nonce       string `json:"nonce"`
}

// ExactEvmPayload contains the signature and authorization for EVM payments.
type ExactEvmPayload struct {
	Signature     string        `json:"signature"`
	Authorization Authorization `json:"authorization"`
}

// PaymentPayloadV2 is the v2 protocol payment payload structure.
// Sent in the PAYMENT-SIGNATURE header (base64 encoded).
type PaymentPayloadV2 struct {
	X402Version int             `json:"x402Version"`
	Resource    ResourceInfo    `json:"resource"`
	Accepted    AcceptedOption  `json:"accepted"`
	Payload     ExactEvmPayload `json:"payload"`
}

// AcceptedOption mirrors PaymentRequirement for the accepted field.
type AcceptedOption struct {
	Scheme            string                 `json:"scheme"`
	Network           string                 `json:"network"`
	Amount            string                 `json:"amount,omitempty"`
	MaxAmountRequired string                 `json:"maxAmountRequired,omitempty"`
	Asset             string                 `json:"asset"`
	PayTo             string                 `json:"payTo"`
	MaxTimeoutSeconds int                    `json:"maxTimeoutSeconds,omitempty"`
	Extra             map[string]interface{} `json:"extra,omitempty"`
}

// PaymentPayloadV1 is the v1 protocol payment payload structure.
// Sent in the X-PAYMENT header (base64 encoded).
type PaymentPayloadV1 struct {
	X402Version int             `json:"x402Version"`
	Scheme      string          `json:"scheme"`
	Network     string          `json:"network"`
	Payload     ExactEvmPayload `json:"payload"`
}

// PaymentResponse represents the server's response after successful payment.
type PaymentResponse struct {
	Success     bool   `json:"success"`
	Transaction string `json:"transaction,omitempty"`
	Network     string `json:"network,omitempty"`
	Error       string `json:"error,omitempty"`
}

// Protocol version constants.
const (
	ProtocolV1 = 1
	ProtocolV2 = 2
)

// Header names for x402 protocol.
const (
	// v2 headers
	HeaderPaymentRequired  = "Payment-Required"
	HeaderPaymentSignature = "Payment-Signature"
	HeaderPaymentResponse  = "Payment-Response"

	// v1 headers
	HeaderXPayment         = "X-Payment"
	HeaderXPaymentResponse = "X-Payment-Response"
)
