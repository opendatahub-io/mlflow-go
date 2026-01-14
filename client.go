// ABOUTME: Main SDK client for MLflow Prompt Registry operations.
// ABOUTME: Provides NewClient constructor and PromptRegistry accessor.

package mlflow

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/ederign/mlflow-go/internal/transport"
)

// Client is the MLflow SDK client.
// It is safe for concurrent use after construction.
type Client struct {
	transport *transport.Client
	opts      options
}

// NewClient creates a new MLflow client with the given options.
// If no options are provided, configuration is read from environment variables:
//   - MLFLOW_TRACKING_URI: MLflow server URL (required)
//   - MLFLOW_TRACKING_TOKEN: Authentication token (optional)
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
	if opts.token == "" {
		opts.token = os.Getenv("MLFLOW_TRACKING_TOKEN")
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

	// Parse and validate the URI
	parsedURL, err := url.Parse(opts.trackingURI)
	if err != nil {
		return nil, fmt.Errorf("mlflow: invalid tracking URI: %w", err)
	}

	// Enforce HTTPS unless insecure mode is enabled
	if !opts.insecure && parsedURL.Scheme == "http" {
		return nil, fmt.Errorf("mlflow: HTTP is not allowed (use HTTPS or enable insecure mode with WithInsecure)")
	}

	// Normalize scheme if missing
	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "https"
		opts.trackingURI = parsedURL.String()
	}

	// Create transport client
	transportCfg := transport.Config{
		BaseURL:    opts.trackingURI,
		Token:      opts.token,
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

// LoadPrompt loads a prompt from the registry by name.
// If no version is specified via WithVersion, loads the latest version.
func (c *Client) LoadPrompt(ctx context.Context, name string, opts ...LoadOption) (*Prompt, error) {
	loadOpts := &loadOptions{}
	for _, opt := range opts {
		opt(loadOpts)
	}

	if loadOpts.version > 0 {
		return c.loadPromptVersion(ctx, name, loadOpts.version)
	}

	return c.loadLatestPrompt(ctx, name)
}

// loadLatestPrompt loads the latest version of a prompt.
func (c *Client) loadLatestPrompt(ctx context.Context, name string) (*Prompt, error) {
	// Get the registered model to find latest version info
	var getModelResp struct {
		RegisteredModel struct {
			Name           string `json:"name"`
			LatestVersions []struct {
				Version string `json:"version"`
			} `json:"latest_versions"`
		} `json:"registered_model"`
	}

	query := url.Values{"name": []string{name}}
	err := c.transport.Get(ctx, "/api/2.0/mlflow/registered-models/get", query, &getModelResp)
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt: %w", err)
	}

	// Find the latest version number
	latestVersion := 1
	if len(getModelResp.RegisteredModel.LatestVersions) > 0 {
		v := getModelResp.RegisteredModel.LatestVersions[0].Version
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			latestVersion = parsed
		}
	}

	return c.loadPromptVersion(ctx, name, latestVersion)
}

// loadPromptVersion loads a specific version of a prompt.
func (c *Client) loadPromptVersion(ctx context.Context, name string, version int) (*Prompt, error) {
	var resp struct {
		ModelVersion modelVersionJSON `json:"model_version"`
	}

	query := url.Values{
		"name":    []string{name},
		"version": []string{strconv.Itoa(version)},
	}

	err := c.transport.Get(ctx, "/api/2.0/mlflow/model-versions/get", query, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt version: %w", err)
	}

	return resp.ModelVersion.toPrompt(), nil
}

// modelVersionJSON represents the JSON structure of a model version response.
type modelVersionJSON struct {
	Name                 string             `json:"name"`
	Version              string             `json:"version"`
	Description          string             `json:"description"`
	CreationTimestamp    int64              `json:"creation_timestamp"`
	LastUpdatedTimestamp int64              `json:"last_updated_timestamp"`
	Tags                 []modelVersionTag  `json:"tags"`
}

type modelVersionTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Prompt tag keys used by MLflow to store prompt metadata.
const (
	tagPromptText   = "mlflow.prompt.text"
	tagIsPrompt     = "mlflow.prompt.is_prompt"
	tagPromptType   = "_mlflow_prompt_type"
	tagDescription  = "mlflow.prompt.description"
)

func (mv *modelVersionJSON) toPrompt() *Prompt {
	p := &Prompt{
		Name:        mv.Name,
		Template:    "",
		Description: mv.Description,
		Tags:        make(map[string]string),
	}

	// Parse version
	if v, err := strconv.Atoi(mv.Version); err == nil {
		p.Version = v
	}

	// Convert timestamps
	if mv.CreationTimestamp > 0 {
		p.CreatedAt = time.UnixMilli(mv.CreationTimestamp)
	}
	if mv.LastUpdatedTimestamp > 0 {
		p.UpdatedAt = time.UnixMilli(mv.LastUpdatedTimestamp)
	}

	// Process tags
	for _, tag := range mv.Tags {
		switch tag.Key {
		case tagPromptText:
			p.Template = tag.Value
		case tagDescription:
			if tag.Value != "" {
				p.Description = tag.Value
			}
		case tagIsPrompt, tagPromptType:
			// Internal tags, don't expose
		default:
			p.Tags[tag.Key] = tag.Value
		}
	}

	return p
}