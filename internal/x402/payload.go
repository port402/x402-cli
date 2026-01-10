package x402

import (
	"encoding/base64"
	"encoding/json"
)

// BuildPayloadV2 constructs the v2 payment payload for the PAYMENT-SIGNATURE header.
func BuildPayloadV2(resource ResourceInfo, option *PaymentRequirement, signature string, auth Authorization) *PaymentPayloadV2 {
	return &PaymentPayloadV2{
		X402Version: ProtocolV2,
		Resource:    resource,
		Accepted: AcceptedOption{
			Scheme:            option.Scheme,
			Network:           option.Network,
			Amount:            option.GetAmount(),
			Asset:             option.Asset,
			PayTo:             option.PayTo,
			MaxTimeoutSeconds: option.MaxTimeoutSeconds,
			Extra:             option.Extra,
		},
		Payload: ExactEvmPayload{
			Signature:     signature,
			Authorization: auth,
		},
	}
}

// BuildPayloadV1 constructs the v1 payment payload for the X-PAYMENT header.
func BuildPayloadV1(option *PaymentRequirement, signature string, auth Authorization) *PaymentPayloadV1 {
	return &PaymentPayloadV1{
		X402Version: ProtocolV1,
		Scheme:      option.Scheme,
		Network:     option.Network,
		Payload: ExactEvmPayload{
			Signature:     signature,
			Authorization: auth,
		},
	}
}

// EncodePayload serializes a payload to base64-encoded JSON.
func EncodePayload(payload interface{}) (string, error) {
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(jsonBytes), nil
}

// BuildAndEncodePayload builds and encodes a payment payload based on protocol version.
func BuildAndEncodePayload(
	protocolVersion int,
	resource ResourceInfo,
	option *PaymentRequirement,
	signature string,
	auth Authorization,
) (headerName string, headerValue string, err error) {
	var payload interface{}

	if protocolVersion == ProtocolV2 {
		headerName = HeaderPaymentSignature
		payload = BuildPayloadV2(resource, option, signature, auth)
	} else {
		headerName = HeaderXPayment
		payload = BuildPayloadV1(option, signature, auth)
	}

	headerValue, err = EncodePayload(payload)
	if err != nil {
		return "", "", err
	}

	return headerName, headerValue, nil
}
