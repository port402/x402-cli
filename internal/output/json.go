package output

import (
	"encoding/json"
	"fmt"
	"os"
)

// PrintJSON outputs any value as formatted JSON to stdout.
func PrintJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// PrintJSONCompact outputs any value as compact JSON to stdout.
func PrintJSONCompact(v interface{}) error {
	return json.NewEncoder(os.Stdout).Encode(v)
}

// FormatJSON returns formatted JSON as a string.
func FormatJSON(v interface{}) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// PrintJSONError outputs an error as JSON to stdout.
func PrintJSONError(err error, exitCode int) {
	PrintJSON(map[string]interface{}{
		"error":    err.Error(),
		"exitCode": exitCode,
	})
}

// BatchHealthResult represents the result of a batch health check.
type BatchHealthResult struct {
	TotalURLs  int            `json:"totalUrls"`
	Passed     int            `json:"passed"`
	PassedWarn int            `json:"passedWithWarnings"`
	Failed     int            `json:"failed"`
	Results    []HealthResult `json:"results"`
	Duration   int64          `json:"durationMs"`
}

// PrintBatchHealthResult outputs batch health results.
func PrintBatchHealthResult(result *BatchHealthResult, jsonOutput bool) {
	if jsonOutput {
		PrintJSON(result)
		return
	}

	fmt.Println()
	fmt.Println("x402 Batch Health Check")
	fmt.Println("───────────────────────")
	fmt.Printf("Total:   %d URLs\n", result.TotalURLs)
	fmt.Printf("Passed:  %d\n", result.Passed)
	if result.PassedWarn > 0 {
		fmt.Printf("Warned:  %d\n", result.PassedWarn)
	}
	fmt.Printf("Failed:  %d\n", result.Failed)
	fmt.Printf("Time:    %dms\n", result.Duration)

	fmt.Println()
	for _, r := range result.Results {
		icon := "✓"
		if r.ExitCode != 0 {
			icon = "✗"
		} else {
			for _, c := range r.Checks {
				if c.Status == StatusWarn {
					icon = "⚠"
					break
				}
			}
		}
		fmt.Printf("  %s %s\n", icon, r.URL)
	}
}
