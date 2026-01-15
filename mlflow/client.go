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
	if name == "" {
		return nil, fmt.Errorf("mlflow: prompt name is required")
	}

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

	// Try to find the latest version from the registered model response
	latestVersion := 0
	if len(getModelResp.RegisteredModel.LatestVersions) > 0 {
		v := getModelResp.RegisteredModel.LatestVersions[0].Version
		if parsed, parseErr := strconv.Atoi(v); parseErr == nil && parsed > 0 {
			latestVersion = parsed
		}
	}

	// If latest_versions is empty, search for versions directly
	if latestVersion == 0 {
		latestVersion, err = c.findLatestVersion(ctx, name)
		if err != nil {
			return nil, err
		}
	}

	return c.loadPromptVersion(ctx, name, latestVersion)
}

// findLatestVersion searches for the highest version number of a prompt.
func (c *Client) findLatestVersion(ctx context.Context, name string) (int, error) {
	var searchResp struct {
		ModelVersions []struct {
			Version string `json:"version"`
		} `json:"model_versions"`
	}

	query := url.Values{
		"filter":      []string{fmt.Sprintf("name='%s'", name)},
		"order_by":    []string{"version_number DESC"},
		"max_results": []string{"1"},
	}

	err := c.transport.Get(ctx, "/api/2.0/mlflow/model-versions/search", query, &searchResp)
	if err != nil {
		return 0, fmt.Errorf("failed to search versions: %w", err)
	}

	if len(searchResp.ModelVersions) == 0 {
		return 0, fmt.Errorf("prompt %q has no versions", name)
	}

	version, err := strconv.Atoi(searchResp.ModelVersions[0].Version)
	if err != nil || version <= 0 {
		return 0, fmt.Errorf("invalid version number for prompt %q", name)
	}

	return version, nil
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
	Name                 string            `json:"name"`
	Version              string            `json:"version"`
	Description          string            `json:"description"`
	CreationTimestamp    int64             `json:"creation_timestamp"`
	LastUpdatedTimestamp int64             `json:"last_updated_timestamp"`
	Tags                 []modelVersionTag `json:"tags"`
}

type modelVersionTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Prompt tag keys used by MLflow to store prompt metadata.
const (
	tagPromptText  = "mlflow.prompt.text"
	tagIsPrompt    = "mlflow.prompt.is_prompt"
	tagPromptType  = "_mlflow_prompt_type"
	tagDescription = "mlflow.prompt.description"
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

// RegisterPrompt registers a prompt in the registry.
// If the prompt doesn't exist, it creates a new one with version 1.
// If the prompt exists, it creates a new version.
func (c *Client) RegisterPrompt(ctx context.Context, name, template string, opts ...RegisterOption) (*Prompt, error) {
	if name == "" {
		return nil, fmt.Errorf("mlflow: prompt name is required")
	}
	if template == "" {
		return nil, fmt.Errorf("mlflow: prompt template is required")
	}

	regOpts := &registerOptions{}
	for _, opt := range opts {
		opt(regOpts)
	}

	// Step 1: Ensure the RegisteredModel exists
	if err := c.ensureRegisteredModel(ctx, name); err != nil {
		return nil, err
	}

	// Step 2: Create a new ModelVersion with the template
	return c.createModelVersion(ctx, name, template, regOpts)
}

// ensureRegisteredModel creates the RegisteredModel if it doesn't exist.
func (c *Client) ensureRegisteredModel(ctx context.Context, name string) error {
	req := createRegisteredModelRequest{
		Name: name,
		Tags: []modelVersionTag{
			{Key: tagIsPrompt, Value: "true"},
		},
	}

	var resp struct {
		RegisteredModel struct {
			Name string `json:"name"`
		} `json:"registered_model"`
	}

	err := c.transport.Post(ctx, "/api/2.0/mlflow/registered-models/create", req, &resp)
	if err != nil {
		// Ignore 409 (already exists) - that's expected for existing prompts
		if IsAlreadyExists(err) {
			return nil
		}
		return fmt.Errorf("failed to create prompt: %w", err)
	}

	return nil
}

// createModelVersion creates a new version of the prompt with the template.
func (c *Client) createModelVersion(ctx context.Context, name, template string, opts *registerOptions) (*Prompt, error) {
	// Build tags for the version
	tags := []modelVersionTag{
		{Key: tagPromptText, Value: template},
		{Key: tagPromptType, Value: "text"},
		{Key: tagIsPrompt, Value: "true"},
	}

	// Add user-provided tags
	for k, v := range opts.tags {
		tags = append(tags, modelVersionTag{Key: k, Value: v})
	}

	req := createModelVersionRequest{
		Name:        name,
		Source:      "mlflow-artifacts:/" + name,
		Description: opts.description,
		Tags:        tags,
	}

	var resp struct {
		ModelVersion modelVersionJSON `json:"model_version"`
	}

	err := c.transport.Post(ctx, "/api/2.0/mlflow/model-versions/create", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt version: %w", err)
	}

	return resp.ModelVersion.toPrompt(), nil
}

// createRegisteredModelRequest is the request body for creating a RegisteredModel.
type createRegisteredModelRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Tags        []modelVersionTag `json:"tags,omitempty"`
}

// createModelVersionRequest is the request body for creating a ModelVersion.
type createModelVersionRequest struct {
	Name        string            `json:"name"`
	Source      string            `json:"source"`
	Description string            `json:"description,omitempty"`
	Tags        []modelVersionTag `json:"tags,omitempty"`
}

// registeredModelJSON represents the JSON structure of a registered model response.
type registeredModelJSON struct {
	Name                 string `json:"name"`
	Description          string `json:"description"`
	CreationTimestamp    int64  `json:"creation_timestamp"`
	LastUpdatedTimestamp int64  `json:"last_updated_timestamp"`
	LatestVersions       []struct {
		Version string `json:"version"`
	} `json:"latest_versions"`
	Tags []modelVersionTag `json:"tags"`
}

func (rm *registeredModelJSON) toPromptInfo() PromptInfo {
	info := PromptInfo{
		Name:        rm.Name,
		Description: rm.Description,
		Tags:        make(map[string]string),
	}

	if rm.CreationTimestamp > 0 {
		info.CreatedAt = time.UnixMilli(rm.CreationTimestamp)
	}
	if rm.LastUpdatedTimestamp > 0 {
		info.UpdatedAt = time.UnixMilli(rm.LastUpdatedTimestamp)
	}

	// Get latest version number
	if len(rm.LatestVersions) > 0 {
		if v, err := strconv.Atoi(rm.LatestVersions[0].Version); err == nil {
			info.LatestVersion = v
		}
	}

	// Process tags (filter out internal ones)
	for _, tag := range rm.Tags {
		switch tag.Key {
		case tagIsPrompt, tagPromptType:
			// Internal tags, don't expose
		default:
			info.Tags[tag.Key] = tag.Value
		}
	}

	return info
}

// toPromptWithoutTemplate converts a model version to a Prompt without loading template.
// Used for listing operations where template content is not needed.
func (mv *modelVersionJSON) toPromptWithoutTemplate() Prompt {
	p := Prompt{
		Name:        mv.Name,
		Template:    "", // Intentionally empty for listings
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

	// Process tags (filter out internal ones including template)
	for _, tag := range mv.Tags {
		switch tag.Key {
		case tagPromptText, tagIsPrompt, tagPromptType, tagDescription:
			// Internal tags, don't expose
			// Also skip template for listing operations
		default:
			p.Tags[tag.Key] = tag.Value
		}
	}

	// Check for description in tags (takes precedence)
	for _, tag := range mv.Tags {
		if tag.Key == tagDescription && tag.Value != "" {
			p.Description = tag.Value
			break
		}
	}

	return p
}

// ListPrompts returns prompts matching the criteria.
// Only prompts (RegisteredModels with is_prompt tag) are returned.
// Returns metadata only; use LoadPrompt for full template content.
func (c *Client) ListPrompts(ctx context.Context, opts ...ListPromptsOption) (*PromptList, error) {
	listOpts := &listPromptsOptions{
		maxResults: 100, // Default page size
	}
	for _, opt := range opts {
		opt(listOpts)
	}

	query := url.Values{}
	query.Set("filter", buildPromptsFilter(listOpts))

	if listOpts.maxResults > 0 {
		query.Set("max_results", strconv.Itoa(listOpts.maxResults))
	}
	if listOpts.pageToken != "" {
		query.Set("page_token", listOpts.pageToken)
	}
	for _, o := range listOpts.orderBy {
		query.Add("order_by", o)
	}

	var resp struct {
		RegisteredModels []registeredModelJSON `json:"registered_models"`
		NextPageToken    string                `json:"next_page_token"`
	}

	err := c.transport.Get(ctx, "/api/2.0/mlflow/registered-models/search", query, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to list prompts: %w", err)
	}

	result := &PromptList{
		Prompts:       make([]PromptInfo, 0, len(resp.RegisteredModels)),
		NextPageToken: resp.NextPageToken,
	}

	for _, rm := range resp.RegisteredModels {
		result.Prompts = append(result.Prompts, rm.toPromptInfo())
	}

	return result, nil
}

// buildPromptsFilter constructs the filter string for listing prompts.
func buildPromptsFilter(opts *listPromptsOptions) string {
	// Base filter: only return prompts
	filters := []string{"tags.`" + tagIsPrompt + "` = 'true'"}

	// Add name pattern if specified
	if opts.nameFilter != "" {
		filters = append(filters, fmt.Sprintf("name LIKE '%s'", opts.nameFilter))
	}

	// Add tag filters
	for k, v := range opts.tagFilter {
		filters = append(filters, fmt.Sprintf("tags.`%s` = '%s'", k, v))
	}

	return joinFilters(filters)
}

// ListPromptVersions returns versions for a specific prompt.
// Returns metadata only; use LoadPrompt with WithVersion for full template content.
func (c *Client) ListPromptVersions(ctx context.Context, name string, opts ...ListVersionsOption) (*PromptVersionList, error) {
	if name == "" {
		return nil, fmt.Errorf("mlflow: prompt name is required")
	}

	listOpts := &listVersionsOptions{
		maxResults: 100, // Default page size
	}
	for _, opt := range opts {
		opt(listOpts)
	}

	query := url.Values{}
	query.Set("filter", buildVersionsFilter(name, listOpts))

	if listOpts.maxResults > 0 {
		query.Set("max_results", strconv.Itoa(listOpts.maxResults))
	}
	if listOpts.pageToken != "" {
		query.Set("page_token", listOpts.pageToken)
	}
	if len(listOpts.orderBy) > 0 {
		for _, o := range listOpts.orderBy {
			query.Add("order_by", o)
		}
	} else {
		// Default: newest versions first
		query.Set("order_by", "version_number DESC")
	}

	var resp struct {
		ModelVersions []modelVersionJSON `json:"model_versions"`
		NextPageToken string             `json:"next_page_token"`
	}

	err := c.transport.Get(ctx, "/api/2.0/mlflow/model-versions/search", query, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to list prompt versions: %w", err)
	}

	result := &PromptVersionList{
		Versions:      make([]Prompt, 0, len(resp.ModelVersions)),
		NextPageToken: resp.NextPageToken,
	}

	for _, mv := range resp.ModelVersions {
		result.Versions = append(result.Versions, mv.toPromptWithoutTemplate())
	}

	return result, nil
}

// buildVersionsFilter constructs the filter string for listing versions.
func buildVersionsFilter(name string, opts *listVersionsOptions) string {
	// Base filter: specific prompt name
	filters := []string{fmt.Sprintf("name='%s'", name)}

	// Add tag filters
	for k, v := range opts.tagFilter {
		filters = append(filters, fmt.Sprintf("tags.`%s` = '%s'", k, v))
	}

	return joinFilters(filters)
}

// joinFilters joins filter conditions with AND.
func joinFilters(filters []string) string {
	if len(filters) == 0 {
		return ""
	}
	if len(filters) == 1 {
		return filters[0]
	}
	result := filters[0]
	for _, f := range filters[1:] {
		result += " AND " + f
	}
	return result
}
