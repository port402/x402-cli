// Package client provides HTTP client functionality for x402 requests.
package client

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client wraps http.Client with x402-specific functionality.
type Client struct {
	httpClient *http.Client
	headers    map[string]string
}

// Option configures the Client.
type Option func(*Client)

// WithTimeout sets the request timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithHeader adds a custom header to all requests.
func WithHeader(key, value string) Option {
	return func(c *Client) {
		c.headers[key] = value
	}
}

// WithHeaders adds multiple custom headers to all requests.
func WithHeaders(headers map[string]string) Option {
	return func(c *Client) {
		for k, v := range headers {
			c.headers[k] = v
		}
	}
}

// New creates a new Client with the given options.
func New(opts ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second, // Default timeout
		},
		headers: make(map[string]string),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Get performs a GET request to the given URL.
func (c *Client) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	return c.Do(req)
}

// GetWithHeader performs a GET request with an additional header.
func (c *Client) GetWithHeader(url, headerName, headerValue string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set(headerName, headerValue)

	return c.Do(req)
}

// Request performs an HTTP request with the given method, URL, headers, and body.
func (c *Client) Request(method, url string, headers map[string]string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add request-specific headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.Do(req)
}

// Do performs the HTTP request with default headers applied.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	// Apply default headers
	for k, v := range c.headers {
		if req.Header.Get(k) == "" { // Don't override if already set
			req.Header.Set(k, v)
		}
	}

	return c.httpClient.Do(req)
}

// RequestResult contains timing and response information.
type RequestResult struct {
	Response  *http.Response
	Latency   time.Duration
	LatencyMs int64
}

// TimedGet performs a GET request and records the latency.
func (c *Client) TimedGet(url string) (*RequestResult, error) {
	start := time.Now()
	resp, err := c.Get(url)
	latency := time.Since(start)

	if err != nil {
		return nil, err
	}

	return &RequestResult{
		Response:  resp,
		Latency:   latency,
		LatencyMs: latency.Milliseconds(),
	}, nil
}

// TimedRequest performs a timed HTTP request.
func (c *Client) TimedRequest(method, url string, headers map[string]string, body []byte) (*RequestResult, error) {
	start := time.Now()
	resp, err := c.Request(method, url, headers, body)
	latency := time.Since(start)

	if err != nil {
		return nil, err
	}

	return &RequestResult{
		Response:  resp,
		Latency:   latency,
		LatencyMs: latency.Milliseconds(),
	}, nil
}

// ParseRetryAfter extracts the Retry-After header value as a duration.
// Returns 0 if the header is not present or invalid.
func ParseRetryAfter(resp *http.Response) time.Duration {
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		return 0
	}

	// Try parsing as seconds
	var seconds int
	if _, err := fmt.Sscanf(retryAfter, "%d", &seconds); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as HTTP-date (RFC 7231)
	if t, err := http.ParseTime(retryAfter); err == nil {
		return time.Until(t)
	}

	return 0
}
