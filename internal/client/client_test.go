package client

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_DefaultTimeout(t *testing.T) {
	c := New()
	assert.Equal(t, 30*time.Second, c.httpClient.Timeout)
}

func TestNew_WithTimeout(t *testing.T) {
	c := New(WithTimeout(60 * time.Second))
	assert.Equal(t, 60*time.Second, c.httpClient.Timeout)
}

func TestNew_WithHeader(t *testing.T) {
	c := New(WithHeader("X-Custom", "value"))
	assert.Equal(t, "value", c.headers["X-Custom"])
}

func TestNew_WithHeaders(t *testing.T) {
	c := New(WithHeaders(map[string]string{
		"X-Custom-1": "value1",
		"X-Custom-2": "value2",
	}))
	assert.Equal(t, "value1", c.headers["X-Custom-1"])
	assert.Equal(t, "value2", c.headers["X-Custom-2"])
}

func TestClient_Get_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	c := New()
	resp, err := c.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestClient_Get_Error(t *testing.T) {
	c := New(WithTimeout(100 * time.Millisecond))

	// Try to connect to a non-existent server
	_, err := c.Get("http://localhost:99999/nonexistent")
	require.Error(t, err)
}

func TestClient_GetWithHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-value", r.Header.Get("X-Test"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New()
	resp, err := c.GetWithHeader(server.URL, "X-Test", "test-value")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestClient_Request_POST(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New()
	resp, err := c.Request(
		http.MethodPost,
		server.URL,
		map[string]string{"Content-Type": "application/json"},
		[]byte(`{"key": "value"}`),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestClient_Request_CustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "custom-value", r.Header.Get("X-Custom"))
		assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New()
	resp, err := c.Request(
		http.MethodGet,
		server.URL,
		map[string]string{
			"X-Custom":      "custom-value",
			"Authorization": "Bearer token123",
		},
		nil,
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestClient_Do_DefaultHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "default-value", r.Header.Get("X-Default"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New(WithHeader("X-Default", "default-value"))
	resp, err := c.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestClient_Do_NoOverrideExisting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The request-specific header should take precedence
		assert.Equal(t, "request-value", r.Header.Get("X-Custom"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New(WithHeader("X-Custom", "default-value"))
	resp, err := c.GetWithHeader(server.URL, "X-Custom", "request-value")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestTimedGet_Latency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New()
	result, err := c.TimedGet(server.URL)
	require.NoError(t, err)
	defer result.Response.Body.Close()

	assert.GreaterOrEqual(t, result.Latency, 10*time.Millisecond)
	assert.GreaterOrEqual(t, result.LatencyMs, int64(10))
}

func TestTimedRequest_Latency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New()
	result, err := c.TimedRequest(http.MethodGet, server.URL, nil, nil)
	require.NoError(t, err)
	defer result.Response.Body.Close()

	assert.GreaterOrEqual(t, result.Latency, 10*time.Millisecond)
}

func TestParseRetryAfter_Seconds(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{
			"Retry-After": []string{"120"},
		},
	}

	duration := ParseRetryAfter(resp)
	assert.Equal(t, 120*time.Second, duration)
}

func TestParseRetryAfter_Empty(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{},
	}

	duration := ParseRetryAfter(resp)
	assert.Equal(t, time.Duration(0), duration)
}

func TestParseRetryAfter_Invalid(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{
			"Retry-After": []string{"invalid"},
		},
	}

	duration := ParseRetryAfter(resp)
	assert.Equal(t, time.Duration(0), duration)
}

func TestParseRetryAfter_HTTPDate(t *testing.T) {
	// Use a date in the future
	futureTime := time.Now().Add(60 * time.Second).UTC()
	httpDate := futureTime.Format(http.TimeFormat)

	resp := &http.Response{
		Header: http.Header{
			"Retry-After": []string{httpDate},
		},
	}

	duration := ParseRetryAfter(resp)
	// Should be approximately 60 seconds (with some margin for test execution)
	assert.Greater(t, duration, 50*time.Second)
	assert.Less(t, duration, 70*time.Second)
}

func TestClient_402Response(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Payment-Required", "base64encodeddata")
		w.WriteHeader(http.StatusPaymentRequired)
	}))
	defer server.Close()

	c := New()
	resp, err := c.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusPaymentRequired, resp.StatusCode)
	assert.Equal(t, "base64encodeddata", resp.Header.Get("Payment-Required"))
}

func TestClient_WithPaymentHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paymentSig := r.Header.Get("Payment-Signature")
		if paymentSig != "" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("protected resource"))
		} else {
			w.WriteHeader(http.StatusPaymentRequired)
		}
	}))
	defer server.Close()

	c := New()

	// First request without payment header
	resp1, err := c.Get(server.URL)
	require.NoError(t, err)
	resp1.Body.Close()
	assert.Equal(t, http.StatusPaymentRequired, resp1.StatusCode)

	// Second request with payment header
	resp2, err := c.GetWithHeader(server.URL, "Payment-Signature", "signed-payload")
	require.NoError(t, err)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
}
