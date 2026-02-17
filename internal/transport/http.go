package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/opendatahub-io/mlflow-go/internal/errors"
)

// Client handles HTTP communication with the MLflow API.
type Client struct {
	baseURL    *url.URL
	headers    map[string]string
	httpClient *http.Client
	logger     *slog.Logger
}

// Config holds configuration for creating a transport Client.
type Config struct {
	BaseURL    string
	Headers    map[string]string
	HTTPClient *http.Client
	Logger     *slog.Logger
	Timeout    time.Duration
}

// errorResponse represents the MLflow API error format.
type errorResponse struct {
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
}

// New creates a new transport Client.
func New(cfg Config) (*Client, error) {
	baseURL, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		timeout := cfg.Timeout
		if timeout == 0 {
			timeout = 30 * time.Second
		}
		httpClient = &http.Client{Timeout: timeout}
	}

	return &Client{
		baseURL:    baseURL,
		headers:    cfg.Headers,
		httpClient: httpClient,
		logger:     cfg.Logger,
	}, nil
}

// Get performs a GET request to the specified path with query parameters.
func (c *Client) Get(ctx context.Context, path string, query url.Values, result any) error {
	return c.do(ctx, http.MethodGet, path, query, nil, result)
}

// Post performs a POST request to the specified path with a JSON body.
func (c *Client) Post(ctx context.Context, path string, body, result any) error {
	return c.do(ctx, http.MethodPost, path, nil, body, result)
}

// Delete performs a DELETE request to the specified path with a JSON body.
func (c *Client) Delete(ctx context.Context, path string, body, result any) error {
	return c.do(ctx, http.MethodDelete, path, nil, body, result)
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body, result any) error {
	// Build request URL
	reqURL := c.baseURL.ResolveReference(&url.URL{Path: path, RawQuery: query.Encode()})

	// Encode body if present
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to encode request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, reqURL.String(), bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	// Log request
	start := time.Now()
	if c.logger != nil {
		c.logger.Debug("request",
			"method", method,
			"url", reqURL.String(),
		)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Log response
	duration := time.Since(start)
	if c.logger != nil {
		c.logger.Debug("response",
			"status", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
		)
	}

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Handle error responses
	if resp.StatusCode >= 400 {
		return c.parseError(resp.StatusCode, respBody)
	}

	// Decode successful response
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

func (c *Client) parseError(statusCode int, body []byte) error {
	var errResp errorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		// If we can't parse the error, return a generic one
		return &errors.APIError{
			StatusCode: statusCode,
			Message:    string(body),
		}
	}

	return &errors.APIError{
		StatusCode: statusCode,
		Code:       errResp.ErrorCode,
		Message:    errResp.Message,
	}
}
