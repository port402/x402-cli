package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/port402/x402-cli/internal/output"
)

// BatchEntry represents an endpoint to check with optional method.
type BatchEntry struct {
	URL    string `json:"url"`
	Method string `json:"method,omitempty"`
}

// Batch health command flags
var (
	batchParallel int
	batchDelay    int
	batchFailFast bool
	batchTimeout  int
)

var batchHealthCmd = &cobra.Command{
	Use:   "batch-health <file>",
	Short: "Check multiple endpoints from a JSON file",
	Long: `Batch health check for multiple x402-enabled endpoints.

The input file can be either:

  1. Simple array of URLs (uses GET):
     ["https://api1.example.com/endpoint", "https://api2.example.com/endpoint"]

  2. Array of objects with URL and method:
     [
       {"url": "https://api.example.com/get-endpoint"},
       {"url": "https://api.example.com/post-endpoint", "method": "POST"}
     ]

Examples:
  x402 batch-health urls.json
  x402 batch-health urls.json --parallel 5
  x402 batch-health urls.json --json
  x402 batch-health urls.json --fail-fast`,
	Args: cobra.ExactArgs(1),
	RunE: runBatchHealth,
}

func init() {
	batchHealthCmd.Flags().IntVar(&batchParallel, "parallel", 1, "Number of parallel checks")
	batchHealthCmd.Flags().IntVar(&batchDelay, "delay", 0, "Delay between requests in milliseconds")
	batchHealthCmd.Flags().BoolVar(&batchFailFast, "fail-fast", false, "Stop on first failure")
	batchHealthCmd.Flags().IntVar(&batchTimeout, "timeout", 30, "Request timeout in seconds")

	rootCmd.AddCommand(batchHealthCmd)
}

func runBatchHealth(cmd *cobra.Command, args []string) error {
	filePath := args[0]
	timeout := time.Duration(batchTimeout) * time.Second

	// Read and parse input file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	entries, err := parseBatchInput(data)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		return fmt.Errorf("no URLs in file")
	}

	// Run checks
	startTime := time.Now()
	results := runBatchChecks(entries, timeout, batchParallel, batchDelay, batchFailFast)
	duration := time.Since(startTime)

	// Count results
	passed := 0
	passedWarn := 0
	failed := 0
	for _, r := range results {
		if r.ExitCode != 0 {
			failed++
		} else {
			hasWarn := false
			for _, c := range r.Checks {
				if c.Status == output.StatusWarn {
					hasWarn = true
					break
				}
			}
			if hasWarn {
				passedWarn++
			} else {
				passed++
			}
		}
	}

	batchResult := &output.BatchHealthResult{
		TotalURLs:  len(entries),
		Passed:     passed,
		PassedWarn: passedWarn,
		Failed:     failed,
		Results:    results,
		Duration:   duration.Milliseconds(),
	}

	output.PrintBatchHealthResult(batchResult, GetJSONOutput())

	if failed > 0 {
		return fmt.Errorf("%d endpoint(s) failed", failed)
	}

	return nil
}

// parseBatchInput parses JSON input supporting both simple URL arrays and object arrays.
func parseBatchInput(data []byte) ([]BatchEntry, error) {
	// Try parsing as array of objects first
	var entries []BatchEntry
	if err := json.Unmarshal(data, &entries); err == nil {
		// Validate and normalize entries
		for i := range entries {
			if entries[i].URL == "" {
				return nil, fmt.Errorf("entry %d: missing URL", i+1)
			}
			// Default to GET if method not specified
			if entries[i].Method == "" {
				entries[i].Method = "GET"
			} else {
				entries[i].Method = strings.ToUpper(entries[i].Method)
			}
		}
		return entries, nil
	}

	// Try parsing as simple string array (backward compatible)
	var urls []string
	if err := json.Unmarshal(data, &urls); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: expected array of URLs or array of {url, method} objects")
	}

	// Convert to BatchEntry with default GET method
	entries = make([]BatchEntry, len(urls))
	for i, url := range urls {
		entries[i] = BatchEntry{URL: url, Method: "GET"}
	}
	return entries, nil
}

func runBatchChecks(entries []BatchEntry, timeout time.Duration, parallel int, delayMs int, failFast bool) []output.HealthResult {
	results := make([]output.HealthResult, len(entries))

	if parallel <= 1 {
		// Sequential execution
		for i, entry := range entries {
			results[i] = *CheckHealthForBatchWithMethod(entry.URL, entry.Method, timeout)
			if failFast && results[i].ExitCode != 0 {
				break
			}
			if delayMs > 0 && i < len(entries)-1 {
				time.Sleep(time.Duration(delayMs) * time.Millisecond)
			}
		}
		return results
	}

	// Parallel execution
	var wg sync.WaitGroup
	var mu sync.Mutex
	semaphore := make(chan struct{}, parallel)
	stopChan := make(chan struct{})
	stopped := false

	for i, entry := range entries {
		// Check if we should stop (fail-fast)
		select {
		case <-stopChan:
			return results[:i]
		default:
		}

		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore

		go func(idx int, e BatchEntry) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore

			result := CheckHealthForBatchWithMethod(e.URL, e.Method, timeout)

			mu.Lock()
			results[idx] = *result
			if failFast && result.ExitCode != 0 && !stopped {
				stopped = true
				close(stopChan)
			}
			mu.Unlock()
		}(i, entry)

		if delayMs > 0 {
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
		}
	}

	wg.Wait()
	return results
}
