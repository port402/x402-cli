package tokens

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatAmount(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		decimals int
		symbol   string
		expected string
	}{
		{"1 USDC", "1000000", 6, "USDC", "1.00 USDC"},
		{"0.01 USDC", "10000", 6, "USDC", "0.01 USDC"},
		{"0.000001 USDC", "1", 6, "USDC", "0.000001 USDC"},
		{"1234.567890 USDC", "1234567890", 6, "USDC", "1234.56789 USDC"},
		{"0 USDC", "0", 6, "USDC", "0.00 USDC"},
		{"Large amount", "1000000000000", 6, "USDC", "1000000.00 USDC"},
		{"18 decimals", "1000000000000000000", 18, "ETH", "1.00 ETH"},
		{"0.1 with 18 decimals", "100000000000000000", 18, "ETH", "0.10 ETH"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatAmount(tt.raw, tt.decimals, tt.symbol)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatAmount_EmptyInput(t *testing.T) {
	result := FormatAmount("", 6, "USDC")
	assert.Equal(t, "0 USDC", result)
}

func TestFormatAmount_InvalidInput(t *testing.T) {
	result := FormatAmount("not-a-number", 6, "USDC")
	assert.Equal(t, "not-a-number USDC (invalid)", result)
}

func TestFormatAmountWithToken_KnownToken(t *testing.T) {
	formatted, known := FormatAmountWithToken("1000000", "eip155:84532", "0x036cbd53842c5426634e7929541ec2318f3dcf7e")
	assert.True(t, known)
	assert.Equal(t, "1.00 USDC", formatted)
}

func TestFormatAmountWithToken_UnknownToken(t *testing.T) {
	formatted, known := FormatAmountWithToken("1000000", "eip155:8453", "0x0000000000000000000000000000000000000000")
	assert.False(t, known)
	assert.Equal(t, "1000000 raw units", formatted)
}

func TestParseHumanAmount(t *testing.T) {
	tests := []struct {
		name     string
		human    string
		decimals int
		expected string
	}{
		{"1.00", "1.00", 6, "1000000"},
		{"0.01", "0.01", 6, "10000"},
		{"0.000001", "0.000001", 6, "1"},
		{"100", "100", 6, "100000000"},
		{"no decimals", "5", 6, "5000000"},
		{"18 decimals", "1.0", 18, "1000000000000000000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseHumanAmount(tt.human, tt.decimals)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseHumanAmount_TooManyDecimals(t *testing.T) {
	// When more decimal places than token supports, truncate
	result, err := ParseHumanAmount("0.1234567", 6)
	require.NoError(t, err)
	assert.Equal(t, "123456", result) // Truncated to 6 decimals
}

func TestParseHumanAmount_Empty(t *testing.T) {
	_, err := ParseHumanAmount("", 6)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty amount")
}

func TestParseHumanAmount_Invalid(t *testing.T) {
	_, err := ParseHumanAmount("abc", 6)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid amount")
}

func TestParseHumanAmount_MultipleDecimals(t *testing.T) {
	_, err := ParseHumanAmount("1.2.3", 6)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid amount format")
}

func TestCompareAmounts(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		{"equal", "100", "100", 0},
		{"a < b", "100", "200", -1},
		{"a > b", "200", "100", 1},
		{"large numbers", "1000000000000", "1000000000001", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareAmounts(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatShortAddress(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		expected string
	}{
		{
			name:     "standard address",
			address:  "0x64c2310BD1151266AA2Ad2410447E133b7F84e29",
			expected: "0x64c2...4e29",
		},
		{
			name:     "short input",
			address:  "0x1234",
			expected: "0x1234", // Unchanged
		},
		{
			name:     "exactly 12 chars",
			address:  "0x12345678ab",
			expected: "0x12345678ab", // Unchanged
		},
		{
			name:     "13 chars",
			address:  "0x12345678abc",
			expected: "0x1234...8abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatShortAddress(tt.address)
			assert.Equal(t, tt.expected, result)
		})
	}
}
