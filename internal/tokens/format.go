package tokens

import (
	"fmt"
	"math/big"
	"strings"
)

// FormatAmount converts a raw token amount to human-readable format.
// Example: FormatAmount("10000", 6, "USDC") → "0.01 USDC"
func FormatAmount(rawAmount string, decimals int, symbol string) string {
	if rawAmount == "" {
		return "0 " + symbol
	}

	// Use big.Int for precision
	amount := new(big.Int)
	_, ok := amount.SetString(rawAmount, 10)
	if !ok {
		return rawAmount + " " + symbol + " (invalid)"
	}

	// Calculate divisor (10^decimals)
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)

	// Integer and remainder parts
	intPart := new(big.Int).Div(amount, divisor)
	remainder := new(big.Int).Mod(amount, divisor)

	// Format decimal part with leading zeros
	decStr := fmt.Sprintf("%0*d", decimals, remainder)

	// Trim trailing zeros but keep at least 2 decimal places
	decStr = strings.TrimRight(decStr, "0")
	if len(decStr) < 2 {
		decStr = decStr + strings.Repeat("0", 2-len(decStr))
	}

	return fmt.Sprintf("%s.%s %s", intPart.String(), decStr, symbol)
}

// FormatAmountWithToken formats amount using token registry lookup.
// Falls back to raw units if token is unknown.
func FormatAmountWithToken(rawAmount, network, asset string) (formatted string, known bool) {
	info := GetTokenInfo(network, asset)
	if info == nil {
		return fmt.Sprintf("%s raw units", rawAmount), false
	}
	return FormatAmount(rawAmount, info.Decimals, info.Symbol), true
}

// ParseHumanAmount converts human-readable amount to raw units.
// Example: ParseHumanAmount("0.01", 6) → "10000"
func ParseHumanAmount(humanAmount string, decimals int) (string, error) {
	// Handle empty input
	humanAmount = strings.TrimSpace(humanAmount)
	if humanAmount == "" {
		return "", fmt.Errorf("empty amount")
	}

	// Split on decimal point
	parts := strings.Split(humanAmount, ".")
	if len(parts) > 2 {
		return "", fmt.Errorf("invalid amount format: %s", humanAmount)
	}

	intPart := parts[0]
	decPart := ""
	if len(parts) == 2 {
		decPart = parts[1]
	}

	// Pad or truncate decimal part to match decimals
	if len(decPart) < decimals {
		decPart = decPart + strings.Repeat("0", decimals-len(decPart))
	} else if len(decPart) > decimals {
		decPart = decPart[:decimals]
	}

	// Combine and parse
	rawStr := intPart + decPart

	// Validate it's a valid number
	raw := new(big.Int)
	_, ok := raw.SetString(rawStr, 10)
	if !ok {
		return "", fmt.Errorf("invalid amount: %s", humanAmount)
	}

	return raw.String(), nil
}

// CompareAmounts compares two raw amounts.
// Returns: -1 if a < b, 0 if a == b, 1 if a > b.
func CompareAmounts(a, b string) int {
	amountA := new(big.Int)
	amountA.SetString(a, 10)

	amountB := new(big.Int)
	amountB.SetString(b, 10)

	return amountA.Cmp(amountB)
}

// FormatShortAddress truncates an address for display.
// Example: "0x64c2310BD1151266AA2Ad2410447E133b7F84e29" → "0x64c2...4e29"
func FormatShortAddress(address string) string {
	if len(address) <= 12 {
		return address
	}
	return address[:6] + "..." + address[len(address)-4:]
}
