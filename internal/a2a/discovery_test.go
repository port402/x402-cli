package a2a

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "full URL with path",
			input:    "https://api.example.com/v1/endpoint",
			expected: "https://api.example.com",
		},
		{
			name:     "URL with port",
			input:    "https://api.example.com:8080/path",
			expected: "https://api.example.com:8080",
		},
		{
			name:     "URL without scheme",
			input:    "api.example.com/path",
			expected: "https://api.example.com",
		},
		{
			name:     "URL with subdomain",
			input:    "https://agent.api.example.com/resource",
			expected: "https://agent.api.example.com",
		},
		{
			name:     "HTTP URL",
			input:    "http://localhost:3000/test",
			expected: "http://localhost:3000",
		},
		{
			name:     "URL with query params",
			input:    "https://api.example.com/path?query=1",
			expected: "https://api.example.com",
		},
		{
			name:     "bare domain",
			input:    "example.com",
			expected: "https://example.com",
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractBaseURL(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestDiscover_Success(t *testing.T) {
	card := AgentCard{
		Name:        "Test Agent",
		Description: "A test agent for unit tests",
		Version:     "1.0.0",
		Skills: []Skill{
			{ID: "test-skill", Name: "Test Skill", Description: "Does testing"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/agent.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(card)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	result := Discover(context.Background(), server.URL+"/some/path", "", 5*time.Second)

	assert.True(t, result.Found)
	assert.Equal(t, "/.well-known/agent.json", result.DiscoveryPath)
	assert.Equal(t, 0, result.ExitCode)
	require.NotNil(t, result.Card)
	assert.Equal(t, "Test Agent", result.Card.Name)
	assert.Equal(t, "1.0.0", result.Card.Version)
	assert.Len(t, result.Card.Skills, 1)
}

func TestDiscover_FallbackPath(t *testing.T) {
	card := AgentCard{
		Name:    "Fallback Agent",
		Version: "2.0.0",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/agent.json":
			w.WriteHeader(http.StatusNotFound)
		case "/.well-known/agent-card.json":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(card)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	result := Discover(context.Background(), server.URL, "", 5*time.Second)

	assert.True(t, result.Found)
	assert.Equal(t, "/.well-known/agent-card.json", result.DiscoveryPath)
	assert.Equal(t, 0, result.ExitCode)
	assert.Len(t, result.TriedPaths, 2) // Should have tried first path before finding second
}

func TestDiscover_CustomCardURL(t *testing.T) {
	card := AgentCard{
		Name:    "Custom Path Agent",
		Version: "3.0.0",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/custom/my-agent.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(card)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	result := Discover(context.Background(), server.URL, "/custom/my-agent.json", 5*time.Second)

	assert.True(t, result.Found)
	assert.Equal(t, "/custom/my-agent.json", result.DiscoveryPath)
	assert.Len(t, result.TriedPaths, 1) // Only tried custom path
}

func TestDiscover_CustomCardURL_WithoutLeadingSlash(t *testing.T) {
	card := AgentCard{
		Name:    "Custom Path Agent",
		Version: "3.0.0",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/custom/agent.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(card)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Pass without leading slash - should still work
	result := Discover(context.Background(), server.URL, "custom/agent.json", 5*time.Second)

	assert.True(t, result.Found)
	assert.Equal(t, "/custom/agent.json", result.DiscoveryPath)
}

func TestDiscover_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	result := Discover(context.Background(), server.URL, "", 5*time.Second)

	assert.False(t, result.Found)
	assert.Equal(t, 0, result.ExitCode) // Not found is not an error
	assert.Len(t, result.TriedPaths, 3) // Tried all 3 default paths

	// All paths should show 404
	for _, attempt := range result.TriedPaths {
		assert.Equal(t, http.StatusNotFound, attempt.Status)
	}
}

func TestDiscover_AuthRequired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	result := Discover(context.Background(), server.URL, "", 5*time.Second)

	assert.False(t, result.Found)
	assert.Equal(t, 0, result.ExitCode) // Auth required is not a failure
	assert.Equal(t, "agent card requires authentication", result.Error)
}

func TestDiscover_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/agent.json" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("not valid json"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	result := Discover(context.Background(), server.URL, "", 5*time.Second)

	assert.False(t, result.Found)
	assert.Equal(t, 4, result.ExitCode) // Invalid JSON is exit code 4
	assert.Contains(t, result.Error, "invalid JSON")
}

func TestDiscover_MissingName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/agent.json" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"version": "1.0.0", "description": "No name"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	result := Discover(context.Background(), server.URL, "", 5*time.Second)

	assert.False(t, result.Found)
	assert.Equal(t, 4, result.ExitCode)
	assert.Contains(t, result.Error, "missing required field 'name'")
}

func TestDiscover_MissingVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/agent.json" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"name": "Test Agent", "description": "No version"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	result := Discover(context.Background(), server.URL, "", 5*time.Second)

	assert.False(t, result.Found)
	assert.Equal(t, 4, result.ExitCode)
	assert.Contains(t, result.Error, "missing required field 'version'")
}

func TestDiscover_NetworkError(t *testing.T) {
	// Use an invalid URL that will fail to connect
	result := Discover(context.Background(), "http://localhost:1", "", 1*time.Second)

	assert.False(t, result.Found)
	assert.Equal(t, 3, result.ExitCode) // Network error
	assert.NotEmpty(t, result.Error)
}

func TestDiscover_NetworkError_CustomPath(t *testing.T) {
	// Custom path with network error should fail immediately
	result := Discover(context.Background(), "http://localhost:1", "/agent.json", 1*time.Second)

	assert.False(t, result.Found)
	assert.Equal(t, 3, result.ExitCode)
	assert.Len(t, result.TriedPaths, 1) // Only tried custom path
}

func TestDiscover_WithCapabilities(t *testing.T) {
	card := AgentCard{
		Name:    "Capable Agent",
		Version: "1.0.0",
		Capabilities: &Capabilities{
			Streaming:         true,
			PushNotifications: true,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/agent.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(card)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	result := Discover(context.Background(), server.URL, "", 5*time.Second)

	assert.True(t, result.Found)
	require.NotNil(t, result.Card.Capabilities)
	assert.True(t, result.Card.Capabilities.Streaming)
	assert.True(t, result.Card.Capabilities.PushNotifications)
}

func TestDiscover_WithProvider(t *testing.T) {
	card := AgentCard{
		Name:    "Provider Agent",
		Version: "1.0.0",
		Provider: &Provider{
			Organization: "Test Org",
			URL:          "https://test.org",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/agent.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(card)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	result := Discover(context.Background(), server.URL, "", 5*time.Second)

	assert.True(t, result.Found)
	require.NotNil(t, result.Card.Provider)
	assert.Equal(t, "Test Org", result.Card.Provider.Organization)
	assert.Equal(t, "https://test.org", result.Card.Provider.URL)
}

func TestDiscover_Timeout(t *testing.T) {
	// Server that sleeps longer than the timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Use a very short timeout (100ms) so the server's 500ms sleep causes a timeout
	result := Discover(context.Background(), server.URL, "/.well-known/agent.json", 100*time.Millisecond)

	assert.False(t, result.Found)
	assert.Equal(t, 3, result.ExitCode) // Network error exit code
	assert.NotEmpty(t, result.Error)
	// Go's http.Client reports timeouts as "context deadline exceeded" or "Client.Timeout"
	assert.True(t, strings.Contains(result.Error, "deadline exceeded") ||
		strings.Contains(result.Error, "Timeout"), "expected timeout error, got: %s", result.Error)
}

func TestDiscover_RedirectLimit(t *testing.T) {
	redirectCount := 0

	// Server that redirects more than 3 times
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectCount++
		if redirectCount <= 5 {
			// Keep redirecting (more than the 3 redirect limit)
			http.Redirect(w, r, r.URL.Path, http.StatusFound)
			return
		}
		// Should never reach here if redirect limit works
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	result := Discover(context.Background(), server.URL, "/.well-known/agent.json", 5*time.Second)

	assert.False(t, result.Found)
	assert.Equal(t, 3, result.ExitCode) // Network error exit code for redirect failure
	assert.NotEmpty(t, result.Error)
	assert.Contains(t, result.Error, "redirect")

	// Should have stopped after 4 requests (1 original + 3 redirects)
	assert.LessOrEqual(t, redirectCount, 4)
}
