package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatJSON(t *testing.T) {
	data := map[string]interface{}{
		"key":    "value",
		"number": 42,
	}

	result, err := FormatJSON(data)
	require.NoError(t, err)
	assert.Contains(t, result, `"key": "value"`)
	assert.Contains(t, result, `"number": 42`)
}

func TestFormatJSON_HealthResult(t *testing.T) {
	result := HealthResult{
		URL:        "https://example.com/api",
		Status:     402,
		StatusText: "402 Payment Required",
		Protocol:   "v2",
		ExitCode:   0,
	}

	jsonStr, err := FormatJSON(result)
	require.NoError(t, err)
	assert.Contains(t, jsonStr, `"url": "https://example.com/api"`)
	assert.Contains(t, jsonStr, `"status": 402`)
	assert.Contains(t, jsonStr, `"protocol": "v2"`)
}

func TestFormatJSON_TestResult(t *testing.T) {
	result := TestResult{
		URL:         "https://example.com/api",
		Transaction: "0xabc123",
		DryRun:      true,
		ExitCode:    0,
	}

	jsonStr, err := FormatJSON(result)
	require.NoError(t, err)
	assert.Contains(t, jsonStr, `"url": "https://example.com/api"`)
	assert.Contains(t, jsonStr, `"transaction": "0xabc123"`)
	assert.Contains(t, jsonStr, `"dryRun": true`)
}

func TestFormatJSON_Nested(t *testing.T) {
	result := HealthResult{
		URL: "https://example.com",
		PaymentOptions: []PaymentOptionDisplay{
			{
				Index:       0,
				Network:     "eip155:8453",
				NetworkName: "Base Mainnet",
				AmountHuman: "1.00 USDC",
			},
		},
		Checks: []Check{
			{Name: "Reachable", Status: StatusPass, Message: "OK"},
		},
	}

	jsonStr, err := FormatJSON(result)
	require.NoError(t, err)
	assert.Contains(t, jsonStr, `"networkName": "Base Mainnet"`)
	assert.Contains(t, jsonStr, `"status": "pass"`)
}

func TestBatchHealthResult_Structure(t *testing.T) {
	result := BatchHealthResult{
		TotalURLs:  3,
		Passed:     2,
		PassedWarn: 1,
		Failed:     0,
		Results: []HealthResult{
			{URL: "https://example1.com", ExitCode: 0},
			{URL: "https://example2.com", ExitCode: 0},
			{URL: "https://example3.com", ExitCode: 0},
		},
		Duration: 500,
	}

	assert.Equal(t, 3, result.TotalURLs)
	assert.Equal(t, 2, result.Passed)
	assert.Equal(t, 1, result.PassedWarn)
	assert.Equal(t, 0, result.Failed)
	assert.Len(t, result.Results, 3)
}

func TestFormatJSON_BatchHealthResult(t *testing.T) {
	result := BatchHealthResult{
		TotalURLs: 2,
		Passed:    2,
		Duration:  100,
		Results: []HealthResult{
			{URL: "https://example1.com"},
			{URL: "https://example2.com"},
		},
	}

	jsonStr, err := FormatJSON(result)
	require.NoError(t, err)
	assert.Contains(t, jsonStr, `"totalUrls": 2`)
	assert.Contains(t, jsonStr, `"passed": 2`)
	assert.Contains(t, jsonStr, `"durationMs": 100`)
}

// Note: PrintJSON, PrintJSONCompact, PrintJSONError, and PrintBatchHealthResult
// write directly to stdout. These are better tested via integration tests
// that capture stdout output.
