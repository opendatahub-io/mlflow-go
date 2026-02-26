// Package mlflow provides a Go SDK for MLflow.
// Supports Prompt Registry and Experiment Tracking.
package mlflow

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/opendatahub-io/mlflow-go/internal/transport"
	"github.com/opendatahub-io/mlflow-go/mlflow/promptregistry"
	"github.com/opendatahub-io/mlflow-go/mlflow/tracking"
)

// Client is the MLflow SDK client.
// It is safe for concurrent use after construction.
type Client struct {
	transport *transport.Client
	opts      options

	promptRegistryOnce sync.Once
	promptRegistry     *promptregistry.Client

	trackingOnce sync.Once
	tracking     *tracking.Client
}

// NewClient creates a new MLflow client with the given options.
// If no options are provided, configuration is read from environment variables:
//   - MLFLOW_TRACKING_URI: MLflow server URL (required)
//   - MLFLOW_INSECURE_SKIP_TLS_VERIFY: Allow HTTP (optional, default false)
func NewClient(clientOpts ...Option) (*Client, error) {
	opts := options{}

	// Apply provided options first (they take precedence over env vars)
	for _, opt := range clientOpts {
		opt(&opts)
	}

	// Fill in missing values from environment variables
	if opts.trackingURI == "" {
		opts.trackingURI = os.Getenv("MLFLOW_TRACKING_URI")
	}
	if !opts.insecure {
		if v := os.Getenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY"); v == "true" || v == "1" {
			opts.insecure = true
		}
	}

	// Validate tracking URI is provided
	if opts.trackingURI == "" {
		return nil, fmt.Errorf("mlflow: tracking URI is required (set MLFLOW_TRACKING_URI or use WithTrackingURI)")
	}

	// Normalize bare host:port input (e.g., "localhost:5000") by prepending https://.
	// Without a scheme, url.Parse treats the host as the scheme and the port as opaque data.
	if !strings.Contains(opts.trackingURI, "://") {
		opts.trackingURI = "https://" + opts.trackingURI
	}

	// Parse and validate the URI
	parsedURL, err := url.Parse(opts.trackingURI)
	if err != nil {
		return nil, fmt.Errorf("mlflow: invalid tracking URI: %w", err)
	}

	// Enforce HTTPS unless insecure mode is enabled
	if !opts.insecure && parsedURL.Scheme == "http" {
		return nil, fmt.Errorf("mlflow: HTTP is not allowed (use HTTPS or enable insecure mode with WithInsecure)")
	}

	// Create transport client
	transportCfg := transport.Config{
		BaseURL:    opts.trackingURI,
		Headers:    opts.headers,
		HTTPClient: opts.httpClient,
		Logger:     opts.logger,
		Timeout:    opts.timeout,
	}

	transportClient, err := transport.New(transportCfg)
	if err != nil {
		return nil, fmt.Errorf("mlflow: failed to create transport: %w", err)
	}

	return &Client{
		transport: transportClient,
		opts:      opts,
	}, nil
}

// TrackingURI returns the configured MLflow tracking URI.
func (c *Client) TrackingURI() string {
	return c.opts.trackingURI
}

// IsInsecure returns whether insecure (HTTP) connections are allowed.
func (c *Client) IsInsecure() bool {
	return c.opts.insecure
}

// PromptRegistry returns the Prompt Registry client for managing prompts.
// The sub-client is created lazily on first access.
func (c *Client) PromptRegistry() *promptregistry.Client {
	c.promptRegistryOnce.Do(func() {
		c.promptRegistry = promptregistry.NewClient(c.transport)
	})
	return c.promptRegistry
}

// Tracking returns the Tracking client for experiment and run management.
// The sub-client is created lazily on first access.
func (c *Client) Tracking() *tracking.Client {
	c.trackingOnce.Do(func() {
		c.tracking = tracking.NewClient(c.transport)
	})
	return c.tracking
}
