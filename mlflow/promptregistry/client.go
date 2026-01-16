package promptregistry

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/opendatahub-io/mlflow-go/internal/errors"
	"github.com/opendatahub-io/mlflow-go/internal/gen/mlflowpb"
	"github.com/opendatahub-io/mlflow-go/internal/transport"
)

// Prompt tag keys used by MLflow to store prompt metadata.
const (
	tagPromptText  = "mlflow.prompt.text"
	tagIsPrompt    = "mlflow.prompt.is_prompt"
	tagPromptType  = "_mlflow_prompt_type"
	tagDescription = "mlflow.prompt.description"
)

// Client provides access to the MLflow Prompt Registry.
// It is safe for concurrent use.
type Client struct {
	transport *transport.Client
}

// NewClient creates a new Prompt Registry client.
// This is typically called internally by the root mlflow.Client.
func NewClient(t *transport.Client) *Client {
	return &Client{transport: t}
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
	var resp mlflowpb.GetRegisteredModel_Response

	query := url.Values{"name": []string{name}}
	err := c.transport.Get(ctx, "/api/2.0/mlflow/registered-models/get", query, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt: %w", err)
	}

	// Try to find the latest version from the registered model response
	latestVersion := 0
	if resp.RegisteredModel != nil && len(resp.RegisteredModel.LatestVersions) > 0 {
		v := resp.RegisteredModel.LatestVersions[0].GetVersion()
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
	var resp mlflowpb.SearchModelVersions_Response

	query := url.Values{
		"filter":      []string{fmt.Sprintf("name='%s'", escapeFilterValue(name))},
		"order_by":    []string{"version_number DESC"},
		"max_results": []string{"1"},
	}

	err := c.transport.Get(ctx, "/api/2.0/mlflow/model-versions/search", query, &resp)
	if err != nil {
		return 0, fmt.Errorf("failed to search versions: %w", err)
	}

	if len(resp.ModelVersions) == 0 {
		return 0, fmt.Errorf("prompt %q has no versions", name)
	}

	version, err := strconv.Atoi(resp.ModelVersions[0].GetVersion())
	if err != nil || version <= 0 {
		return 0, fmt.Errorf("invalid version number for prompt %q", name)
	}

	return version, nil
}

// loadPromptVersion loads a specific version of a prompt.
func (c *Client) loadPromptVersion(ctx context.Context, name string, version int) (*Prompt, error) {
	var resp mlflowpb.GetModelVersion_Response

	query := url.Values{
		"name":    []string{name},
		"version": []string{strconv.Itoa(version)},
	}

	err := c.transport.Get(ctx, "/api/2.0/mlflow/model-versions/get", query, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt version: %w", err)
	}

	return modelVersionToPrompt(resp.ModelVersion), nil
}

func modelVersionToPrompt(mv *mlflowpb.ModelVersion) *Prompt {
	if mv == nil {
		return nil
	}

	p := &Prompt{
		Name:        mv.GetName(),
		Template:    "",
		Description: mv.GetDescription(),
		Tags:        make(map[string]string),
	}

	// Parse version
	if v, err := strconv.Atoi(mv.GetVersion()); err == nil {
		p.Version = v
	}

	// Convert timestamps
	if mv.CreationTimestamp != nil && *mv.CreationTimestamp > 0 {
		p.CreatedAt = time.UnixMilli(*mv.CreationTimestamp)
	}
	if mv.LastUpdatedTimestamp != nil && *mv.LastUpdatedTimestamp > 0 {
		p.UpdatedAt = time.UnixMilli(*mv.LastUpdatedTimestamp)
	}

	// Process tags
	for _, tag := range mv.Tags {
		key := tag.GetKey()
		value := tag.GetValue()
		switch key {
		case tagPromptText:
			p.Template = value
		case tagDescription:
			if value != "" {
				p.Description = value
			}
		case tagIsPrompt, tagPromptType:
			// Internal tags, don't expose
		default:
			p.Tags[key] = value
		}
	}

	return p
}

// modelVersionToPromptWithoutTemplate converts a model version to a Prompt without loading template.
// Used for listing operations where template content is not needed.
func modelVersionToPromptWithoutTemplate(mv *mlflowpb.ModelVersion) Prompt {
	if mv == nil {
		return Prompt{}
	}

	p := Prompt{
		Name:        mv.GetName(),
		Template:    "", // Intentionally empty for listings
		Description: mv.GetDescription(),
		Tags:        make(map[string]string),
	}

	// Parse version
	if v, err := strconv.Atoi(mv.GetVersion()); err == nil {
		p.Version = v
	}

	// Convert timestamps
	if mv.CreationTimestamp != nil && *mv.CreationTimestamp > 0 {
		p.CreatedAt = time.UnixMilli(*mv.CreationTimestamp)
	}
	if mv.LastUpdatedTimestamp != nil && *mv.LastUpdatedTimestamp > 0 {
		p.UpdatedAt = time.UnixMilli(*mv.LastUpdatedTimestamp)
	}

	// Process tags (filter out internal ones including template)
	for _, tag := range mv.Tags {
		key := tag.GetKey()
		value := tag.GetValue()
		switch key {
		case tagPromptText, tagIsPrompt, tagPromptType, tagDescription:
			// Internal tags, don't expose
		default:
			p.Tags[key] = value
		}
	}

	// Check for description in tags (takes precedence)
	for _, tag := range mv.Tags {
		if tag.GetKey() == tagDescription && tag.GetValue() != "" {
			p.Description = tag.GetValue()
			break
		}
	}

	return p
}

func registeredModelToPromptInfo(rm *mlflowpb.RegisteredModel) PromptInfo {
	if rm == nil {
		return PromptInfo{}
	}

	info := PromptInfo{
		Name:        rm.GetName(),
		Description: rm.GetDescription(),
		Tags:        make(map[string]string),
	}

	if rm.CreationTimestamp != nil && *rm.CreationTimestamp > 0 {
		info.CreatedAt = time.UnixMilli(*rm.CreationTimestamp)
	}
	if rm.LastUpdatedTimestamp != nil && *rm.LastUpdatedTimestamp > 0 {
		info.UpdatedAt = time.UnixMilli(*rm.LastUpdatedTimestamp)
	}

	// Get latest version number
	if len(rm.LatestVersions) > 0 {
		if v, err := strconv.Atoi(rm.LatestVersions[0].GetVersion()); err == nil {
			info.LatestVersion = v
		}
	}

	// Process tags (filter out internal ones)
	for _, tag := range rm.Tags {
		key := tag.GetKey()
		value := tag.GetValue()
		switch key {
		case tagIsPrompt, tagPromptType:
			// Internal tags, don't expose
		default:
			info.Tags[key] = value
		}
	}

	return info
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
	req := &mlflowpb.CreateRegisteredModel{
		Name: &name,
		Tags: []*mlflowpb.RegisteredModelTag{
			{Key: ptr(tagIsPrompt), Value: ptr("true")},
		},
	}

	var resp mlflowpb.CreateRegisteredModel_Response

	err := c.transport.Post(ctx, "/api/2.0/mlflow/registered-models/create", req, &resp)
	if err != nil {
		// Ignore 409 (already exists) - that's expected for existing prompts
		if errors.IsAlreadyExists(err) {
			return nil
		}
		return fmt.Errorf("failed to create prompt: %w", err)
	}

	return nil
}

// createModelVersion creates a new version of the prompt with the template.
func (c *Client) createModelVersion(ctx context.Context, name, template string, opts *registerOptions) (*Prompt, error) {
	// Build tags for the version
	tags := []*mlflowpb.ModelVersionTag{
		{Key: ptr(tagPromptText), Value: ptr(template)},
		{Key: ptr(tagPromptType), Value: ptr("text")},
		{Key: ptr(tagIsPrompt), Value: ptr("true")},
	}

	// Add user-provided tags
	for k, v := range opts.tags {
		tags = append(tags, &mlflowpb.ModelVersionTag{Key: ptr(k), Value: ptr(v)})
	}

	source := "mlflow-artifacts:/" + name
	req := &mlflowpb.CreateModelVersion{
		Name:        &name,
		Source:      &source,
		Description: &opts.description,
		Tags:        tags,
	}

	var resp mlflowpb.CreateModelVersion_Response

	err := c.transport.Post(ctx, "/api/2.0/mlflow/model-versions/create", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt version: %w", err)
	}

	return modelVersionToPrompt(resp.ModelVersion), nil
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

	var resp mlflowpb.SearchRegisteredModels_Response

	err := c.transport.Get(ctx, "/api/2.0/mlflow/registered-models/search", query, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to list prompts: %w", err)
	}

	result := &PromptList{
		Prompts:       make([]PromptInfo, 0, len(resp.RegisteredModels)),
		NextPageToken: resp.GetNextPageToken(),
	}

	for _, rm := range resp.RegisteredModels {
		result.Prompts = append(result.Prompts, registeredModelToPromptInfo(rm))
	}

	return result, nil
}

// buildPromptsFilter constructs the filter string for listing prompts.
func buildPromptsFilter(opts *listPromptsOptions) string {
	// Base filter: only return prompts
	filters := []string{"tags.`" + tagIsPrompt + "` = 'true'"}

	// Add name pattern if specified
	if opts.nameFilter != "" {
		filters = append(filters, fmt.Sprintf("name LIKE '%s'", escapeFilterValue(opts.nameFilter)))
	}

	// Add tag filters
	for k, v := range opts.tagFilter {
		filters = append(filters, fmt.Sprintf("tags.`%s` = '%s'", escapeFilterKey(k), escapeFilterValue(v)))
	}

	return joinFilters(filters)
}

// ListPromptVersions returns versions for a specific prompt.
// Returns metadata only; use LoadPrompt with WithVersion for full template content.
//
// Note: Due to a limitation in MLflow OSS model-versions/search endpoint,
// this method fetches versions individually. Pagination options are ignored.
func (c *Client) ListPromptVersions(ctx context.Context, name string, opts ...ListVersionsOption) (*PromptVersionList, error) {
	if name == "" {
		return nil, fmt.Errorf("mlflow: prompt name is required")
	}

	// Apply options (maxResults used to limit results)
	listOpts := &listVersionsOptions{
		maxResults: 100,
	}
	for _, opt := range opts {
		opt(listOpts)
	}

	// Get the registered model to find the latest version number
	latestVersion, err := c.findLatestVersion(ctx, name)
	if err != nil {
		// If findLatestVersion fails, try getting the model directly
		var getModelResp mlflowpb.GetRegisteredModel_Response

		query := url.Values{"name": []string{name}}
		if getErr := c.transport.Get(ctx, "/api/2.0/mlflow/registered-models/get", query, &getModelResp); getErr != nil {
			return nil, fmt.Errorf("failed to get prompt: %w", getErr)
		}

		if getModelResp.RegisteredModel != nil && len(getModelResp.RegisteredModel.LatestVersions) > 0 {
			if v, parseErr := strconv.Atoi(getModelResp.RegisteredModel.LatestVersions[0].GetVersion()); parseErr == nil {
				latestVersion = v
			}
		}

		if latestVersion == 0 {
			return &PromptVersionList{Versions: []Prompt{}}, nil
		}
	}

	// Fetch each version individually (workaround for broken search endpoint)
	result := &PromptVersionList{
		Versions: make([]Prompt, 0, latestVersion),
	}

	for v := latestVersion; v >= 1; v-- {
		if listOpts.maxResults > 0 && len(result.Versions) >= listOpts.maxResults {
			break
		}

		var resp mlflowpb.GetModelVersion_Response

		query := url.Values{
			"name":    []string{name},
			"version": []string{strconv.Itoa(v)},
		}

		err := c.transport.Get(ctx, "/api/2.0/mlflow/model-versions/get", query, &resp)
		if err != nil {
			if errors.IsNotFound(err) {
				continue // Version might have been deleted
			}
			return nil, fmt.Errorf("failed to get version %d: %w", v, err)
		}

		result.Versions = append(result.Versions, modelVersionToPromptWithoutTemplate(resp.ModelVersion))
	}

	return result, nil
}

// escapeFilterKey escapes backticks in filter keys to prevent injection.
func escapeFilterKey(s string) string {
	return strings.ReplaceAll(s, "`", "``")
}

// escapeFilterValue escapes single quotes in filter values to prevent injection.
func escapeFilterValue(s string) string {
	return strings.ReplaceAll(s, "'", "''")
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

// ptr returns a pointer to the given value.
func ptr[T any](v T) *T {
	return &v
}
