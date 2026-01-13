// Package a2a provides types and discovery logic for the A2A (Agent-to-Agent) protocol.
package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DiscoveryPaths lists A2A agent card locations in priority order.
var DiscoveryPaths = []string{
	"/.well-known/agent.json",      // A2A v0.1
	"/.well-known/agent-card.json", // A2A v0.2+
	"/.well-known/agents.json",     // Wildcard spec
}

// Discover attempts to find an agent card at the given URL.
// If customPath is provided, only that path is tried.
// Returns a Result with all attempted paths and any discovered card.
func Discover(ctx context.Context, rawURL string, customPath string, timeout time.Duration) *Result {
	result := &Result{
		URL:        rawURL,
		TriedPaths: []PathAttempt{},
	}

	baseURL, err := ExtractBaseURL(rawURL)
	if err != nil {
		result.Error = err.Error()
		result.ExitCode = 4
		return result
	}
	result.BaseURL = baseURL

	httpClient := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("stopped after 3 redirects")
			}
			return nil
		},
	}
	paths := buildPaths(customPath)

	for _, path := range paths {
		card, status, fetchErr := fetchAgentCard(ctx, httpClient, baseURL+path)

		attempt := PathAttempt{Path: path, Status: status}
		if fetchErr != nil {
			attempt.Error = fetchErr.Error()
		}
		result.TriedPaths = append(result.TriedPaths, attempt)

		if card != nil {
			result.Found = true
			result.DiscoveryPath = path
			result.Card = card
			return result
		}

		if exitCode, errMsg := evaluateAttempt(status, fetchErr, customPath); exitCode > 0 {
			result.Error = errMsg
			result.ExitCode = exitCode
			return result
		}

		if status == http.StatusUnauthorized || status == http.StatusForbidden {
			result.Error = "agent card requires authentication"
		}
	}

	if allNetworkErrors(result.TriedPaths) {
		result.ExitCode = 3
		if result.Error == "" && len(result.TriedPaths) > 0 {
			result.Error = result.TriedPaths[0].Error
		}
	}

	return result
}

// buildPaths returns the discovery paths to try.
func buildPaths(customPath string) []string {
	if customPath == "" {
		return DiscoveryPaths
	}
	if !strings.HasPrefix(customPath, "/") {
		customPath = "/" + customPath
	}
	return []string{customPath}
}

// evaluateAttempt checks if the attempt should terminate discovery with an error.
func evaluateAttempt(status int, err error, customPath string) (exitCode int, errMsg string) {
	if status == 0 && err != nil && customPath != "" {
		return 3, err.Error()
	}
	if status == http.StatusOK && err != nil {
		return 4, err.Error()
	}
	return 0, ""
}

// allNetworkErrors returns true if all attempts had network errors (status 0).
func allNetworkErrors(attempts []PathAttempt) bool {
	if len(attempts) == 0 {
		return false
	}
	for _, attempt := range attempts {
		if attempt.Status != 0 {
			return false
		}
	}
	return true
}

// ExtractBaseURL extracts scheme://host[:port] from a full URL.
// Example: "https://api.example.com/v1/endpoint" -> "https://api.example.com"
func ExtractBaseURL(rawURL string) (string, error) {
	// Ensure URL has scheme
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	if parsed.Host == "" {
		return "", fmt.Errorf("invalid URL: missing host")
	}

	// Reconstruct base: scheme://host[:port]
	return fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host), nil
}

// fetchAgentCard attempts to fetch and parse an agent card from a specific URL.
// Returns the card, HTTP status code, and any error.
func fetchAgentCard(ctx context.Context, client *http.Client, url string) (*AgentCard, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err // Network error
	}
	defer resp.Body.Close()

	// Not found
	if resp.StatusCode == http.StatusNotFound {
		return nil, resp.StatusCode, nil
	}

	// Auth required
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, resp.StatusCode, nil
	}

	// Other non-200
	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Read body (limit to 1MB to prevent memory exhaustion)
	const maxBodySize = 1 << 20 // 1MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse JSON
	var card AgentCard
	if err := json.Unmarshal(body, &card); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate required fields
	if err := validateAgentCard(&card); err != nil {
		return nil, resp.StatusCode, err
	}

	return &card, resp.StatusCode, nil
}

// validateAgentCard checks that required fields are present.
func validateAgentCard(card *AgentCard) error {
	if card.Name == "" {
		return fmt.Errorf("invalid agent card: missing required field 'name'")
	}
	if card.Version == "" {
		return fmt.Errorf("invalid agent card: missing required field 'version'")
	}
	return nil
}
