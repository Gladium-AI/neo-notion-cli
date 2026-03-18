// Package httpx provides a Notion-aware HTTP client with retry, rate-limit
// handling, and standard header injection.
package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/paoloanzn/neo-notion-cli/internal/config"
)

// Client wraps a retryable HTTP client configured for the Notion API.
type Client struct {
	inner   *retryablehttp.Client
	cfg     *config.Config
	baseURL string
	logger  zerolog.Logger
}

// New creates a Client from the resolved config.
func New(cfg *config.Config) *Client {
	rc := retryablehttp.NewClient()
	rc.RetryMax = cfg.Retry
	rc.RetryWaitMin = 500 * time.Millisecond
	rc.RetryWaitMax = 10 * time.Second
	rc.Logger = nil // silence default logger; we use zerolog

	// Respect Retry-After from Notion 429 responses.
	rc.CheckRetry = retryablehttp.DefaultRetryPolicy
	rc.Backoff = retryablehttp.DefaultBackoff

	rc.HTTPClient.Timeout = cfg.Timeout

	return &Client{
		inner:   rc,
		cfg:     cfg,
		baseURL: cfg.BaseURL,
		logger:  log.With().Str("component", "httpx").Logger(),
	}
}

// SetAuthToken overrides the bearer token on this client.
func (c *Client) SetAuthToken(token string) {
	c.cfg.AuthToken = token
}

// NotionError represents an error response from the Notion API.
type NotionError struct {
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *NotionError) Error() string {
	return fmt.Sprintf("notion api error %d (%s): %s", e.Status, e.Code, e.Message)
}

// Do executes an HTTP request against the Notion API and returns the raw
// response body. It injects standard headers and handles error responses.
func (c *Client) Do(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := retryablehttp.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Standard headers.
	if c.cfg.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.AuthToken)
	}
	req.Header.Set("Notion-Version", c.cfg.NotionVersion)
	req.Header.Set("Content-Type", "application/json")

	if c.cfg.IdempotencyKey != "" {
		req.Header.Set("Idempotency-Key", c.cfg.IdempotencyKey)
	}

	// Extra user-supplied headers.
	for k, v := range c.cfg.ExtraHeaders {
		req.Header.Set(k, v)
	}

	c.logger.Debug().Str("method", method).Str("url", url).Msg("request")

	resp, err := c.inner.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var ne NotionError
		if json.Unmarshal(respBody, &ne) == nil && ne.Code != "" {
			ne.Status = resp.StatusCode
			return nil, &ne
		}
		return nil, fmt.Errorf("notion api error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// DoRaw executes an HTTP request with a raw io.Reader body (for file uploads).
func (c *Client) DoRaw(ctx context.Context, method, path string, body io.Reader, contentType string) ([]byte, error) {
	url := c.baseURL + path

	req, err := retryablehttp.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if c.cfg.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.AuthToken)
	}
	req.Header.Set("Notion-Version", c.cfg.NotionVersion)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	for k, v := range c.cfg.ExtraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := c.inner.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var ne NotionError
		if json.Unmarshal(respBody, &ne) == nil && ne.Code != "" {
			ne.Status = resp.StatusCode
			return nil, &ne
		}
		return nil, fmt.Errorf("notion api error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// DoBasicAuth executes a request using HTTP Basic Auth (for OAuth endpoints).
func (c *Client) DoBasicAuth(ctx context.Context, method, path string, body interface{}, clientID, clientSecret string) ([]byte, error) {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := retryablehttp.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", c.cfg.NotionVersion)
	req.SetBasicAuth(clientID, clientSecret)

	resp, err := c.inner.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var ne NotionError
		if json.Unmarshal(respBody, &ne) == nil && ne.Code != "" {
			ne.Status = resp.StatusCode
			return nil, &ne
		}
		return nil, fmt.Errorf("notion api error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
