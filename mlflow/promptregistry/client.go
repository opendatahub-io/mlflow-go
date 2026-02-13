package promptregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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
	tagModelConfig = "_mlflow_prompt_model_config"
	promptTypeText = "text"
	promptTypeChat = "chat"
	aliasTagPrefix = "mlflow.prompt.alias."
	aliasLatest    = "latest"
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
// If no version is specified via WithVersion or WithAlias, loads the latest version.
func (c *Client) LoadPrompt(ctx context.Context, name string, opts ...LoadOption) (*PromptVersion, error) {
	if name == "" {
		return nil, fmt.Errorf("mlflow: prompt name is required")
	}

	loadOpts := &loadOptions{}
	for _, opt := range opts {
		opt(loadOpts)
	}

	// If alias is specified, use the alias endpoint directly
	if loadOpts.alias != "" {
		return c.loadPromptByAlias(ctx, name, loadOpts.alias)
	}

	if loadOpts.version > 0 {
		return c.loadPromptVersionByNumber(ctx, name, loadOpts.version)
	}

	// No version specified - use the special "latest" alias
	return c.loadPromptByAlias(ctx, name, aliasLatest)
}

// loadPromptByAlias loads a prompt version by alias using the MLflow alias endpoint.
// MLflow supports a special "latest" alias that returns the highest version number.
func (c *Client) loadPromptByAlias(ctx context.Context, name, alias string) (*PromptVersion, error) {
	var resp mlflowpb.GetModelVersionByAlias_Response

	query := url.Values{
		"name":  []string{name},
		"alias": []string{alias},
	}

	err := c.transport.Get(ctx, "/api/2.0/mlflow/registered-models/alias", query, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt by alias %q: %w", alias, err)
	}

	return modelVersionToPromptVersion(resp.ModelVersion), nil
}

// loadPromptVersionByNumber loads a specific version of a prompt by version number.
func (c *Client) loadPromptVersionByNumber(ctx context.Context, name string, version int) (*PromptVersion, error) {
	var resp mlflowpb.GetModelVersion_Response

	query := url.Values{
		"name":    []string{name},
		"version": []string{strconv.Itoa(version)},
	}

	err := c.transport.Get(ctx, "/api/2.0/mlflow/model-versions/get", query, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt version: %w", err)
	}

	return modelVersionToPromptVersion(resp.ModelVersion), nil
}

func modelVersionToPromptVersion(mv *mlflowpb.ModelVersion) *PromptVersion {
	if mv == nil {
		return nil
	}

	pv := &PromptVersion{
		Name:          mv.GetName(),
		CommitMessage: mv.GetDescription(),
		Tags:          make(map[string]string),
	}

	// Parse version
	if v, err := strconv.Atoi(mv.GetVersion()); err == nil {
		pv.Version = v
	}

	// Convert timestamps
	if mv.CreationTimestamp != nil && *mv.CreationTimestamp > 0 {
		pv.CreatedAt = time.UnixMilli(*mv.CreationTimestamp)
	}
	if mv.LastUpdatedTimestamp != nil && *mv.LastUpdatedTimestamp > 0 {
		pv.UpdatedAt = time.UnixMilli(*mv.LastUpdatedTimestamp)
	}

	var promptType string
	var promptText string
	var modelConfigJSON string

	// Process tags
	for _, tag := range mv.Tags {
		key := tag.GetKey()
		value := tag.GetValue()
		switch key {
		case tagPromptText:
			promptText = value
		case tagPromptType:
			promptType = value
		case tagModelConfig:
			modelConfigJSON = value
		case tagDescription:
			if value != "" {
				pv.CommitMessage = value
			}
		case tagIsPrompt:
			// Internal tag, don't expose
		default:
			// Check for alias tags
			if strings.HasPrefix(key, aliasTagPrefix) {
				// Skip alias tags in user tags
			} else {
				pv.Tags[key] = value
			}
		}
	}

	// Parse template based on type
	if promptType == promptTypeChat && promptText != "" {
		var messages []ChatMessage
		if err := json.Unmarshal([]byte(promptText), &messages); err == nil {
			pv.Messages = messages
		}
	} else {
		pv.Template = promptText
	}

	// Parse model config
	if modelConfigJSON != "" {
		var config PromptModelConfig
		if err := json.Unmarshal([]byte(modelConfigJSON), &config); err == nil {
			pv.ModelConfig = &config
		}
	}

	return pv
}

// modelVersionToPromptVersionWithoutTemplate converts a model version to a PromptVersion without loading template.
// Used for listing operations where template content is not needed.
func modelVersionToPromptVersionWithoutTemplate(mv *mlflowpb.ModelVersion) PromptVersion {
	if mv == nil {
		return PromptVersion{}
	}

	pv := PromptVersion{
		Name:          mv.GetName(),
		CommitMessage: mv.GetDescription(),
		Tags:          make(map[string]string),
	}

	// Parse version
	if v, err := strconv.Atoi(mv.GetVersion()); err == nil {
		pv.Version = v
	}

	// Convert timestamps
	if mv.CreationTimestamp != nil && *mv.CreationTimestamp > 0 {
		pv.CreatedAt = time.UnixMilli(*mv.CreationTimestamp)
	}
	if mv.LastUpdatedTimestamp != nil && *mv.LastUpdatedTimestamp > 0 {
		pv.UpdatedAt = time.UnixMilli(*mv.LastUpdatedTimestamp)
	}

	// Process tags (filter out internal ones including template)
	for _, tag := range mv.Tags {
		key := tag.GetKey()
		value := tag.GetValue()
		switch key {
		case tagPromptText, tagIsPrompt, tagPromptType, tagDescription, tagModelConfig:
			// Internal tags, don't expose
		default:
			if !strings.HasPrefix(key, aliasTagPrefix) {
				pv.Tags[key] = value
			}
		}
	}

	// Check for commit message in tags (takes precedence)
	for _, tag := range mv.Tags {
		if tag.GetKey() == tagDescription && tag.GetValue() != "" {
			pv.CommitMessage = tag.GetValue()
			break
		}
	}

	return pv
}

func registeredModelToPrompt(rm *mlflowpb.RegisteredModel) Prompt {
	if rm == nil {
		return Prompt{}
	}

	p := Prompt{
		Name:        rm.GetName(),
		Description: rm.GetDescription(),
		Tags:        make(map[string]string),
	}

	if rm.CreationTimestamp != nil && *rm.CreationTimestamp > 0 {
		p.CreationTimestamp = time.UnixMilli(*rm.CreationTimestamp)
	}

	// Get latest version number
	if len(rm.LatestVersions) > 0 {
		if v, err := strconv.Atoi(rm.LatestVersions[0].GetVersion()); err == nil {
			p.LatestVersion = v
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
			if !strings.HasPrefix(key, aliasTagPrefix) {
				p.Tags[key] = value
			}
		}
	}

	return p
}

// RegisterPrompt registers a text prompt in the registry.
// If the prompt doesn't exist, it creates a new one with version 1.
// If the prompt exists, it creates a new version.
func (c *Client) RegisterPrompt(ctx context.Context, name, template string, opts ...RegisterOption) (*PromptVersion, error) {
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
	return c.createTextPromptVersion(ctx, name, template, regOpts)
}

// RegisterChatPrompt registers a chat prompt in the registry.
// If the prompt doesn't exist, it creates a new one with version 1.
// If the prompt exists, it creates a new version.
func (c *Client) RegisterChatPrompt(ctx context.Context, name string, messages []ChatMessage, opts ...RegisterOption) (*PromptVersion, error) {
	if name == "" {
		return nil, fmt.Errorf("mlflow: prompt name is required")
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("mlflow: at least one message is required for chat prompts")
	}

	regOpts := &registerOptions{}
	for _, opt := range opts {
		opt(regOpts)
	}

	// Step 1: Ensure the RegisteredModel exists
	if err := c.ensureRegisteredModel(ctx, name); err != nil {
		return nil, err
	}

	// Step 2: Create a new ModelVersion with the chat messages
	return c.createChatPromptVersion(ctx, name, messages, regOpts)
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

// createTextPromptVersion creates a new version of the prompt with a text template.
func (c *Client) createTextPromptVersion(ctx context.Context, name, template string, opts *registerOptions) (*PromptVersion, error) {
	// Build tags for the version
	tags := []*mlflowpb.ModelVersionTag{
		{Key: ptr(tagPromptText), Value: ptr(template)},
		{Key: ptr(tagPromptType), Value: ptr(promptTypeText)},
		{Key: ptr(tagIsPrompt), Value: ptr("true")},
	}

	// Add model config if provided
	if opts.modelConfig != nil {
		configJSON, err := json.Marshal(opts.modelConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize model config: %w", err)
		}
		tags = append(tags, &mlflowpb.ModelVersionTag{Key: ptr(tagModelConfig), Value: ptr(string(configJSON))})
	}

	// Add user-provided tags
	for k, v := range opts.tags {
		tags = append(tags, &mlflowpb.ModelVersionTag{Key: ptr(k), Value: ptr(v)})
	}

	source := "mlflow-artifacts:/" + name
	req := &mlflowpb.CreateModelVersion{
		Name:        &name,
		Source:      &source,
		Description: &opts.commitMessage,
		Tags:        tags,
	}

	var resp mlflowpb.CreateModelVersion_Response

	err := c.transport.Post(ctx, "/api/2.0/mlflow/model-versions/create", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt version: %w", err)
	}

	return modelVersionToPromptVersion(resp.ModelVersion), nil
}

// createChatPromptVersion creates a new version of the prompt with chat messages.
func (c *Client) createChatPromptVersion(ctx context.Context, name string, messages []ChatMessage, opts *registerOptions) (*PromptVersion, error) {
	// Serialize messages to JSON
	messagesJSON, err := json.Marshal(messages)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize chat messages: %w", err)
	}

	// Build tags for the version
	tags := []*mlflowpb.ModelVersionTag{
		{Key: ptr(tagPromptText), Value: ptr(string(messagesJSON))},
		{Key: ptr(tagPromptType), Value: ptr(promptTypeChat)},
		{Key: ptr(tagIsPrompt), Value: ptr("true")},
	}

	// Add model config if provided
	if opts.modelConfig != nil {
		var configJSON []byte
		configJSON, err = json.Marshal(opts.modelConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize model config: %w", err)
		}
		tags = append(tags, &mlflowpb.ModelVersionTag{Key: ptr(tagModelConfig), Value: ptr(string(configJSON))})
	}

	// Add user-provided tags
	for k, v := range opts.tags {
		tags = append(tags, &mlflowpb.ModelVersionTag{Key: ptr(k), Value: ptr(v)})
	}

	source := "mlflow-artifacts:/" + name
	req := &mlflowpb.CreateModelVersion{
		Name:        &name,
		Source:      &source,
		Description: &opts.commitMessage,
		Tags:        tags,
	}

	var resp mlflowpb.CreateModelVersion_Response

	err = c.transport.Post(ctx, "/api/2.0/mlflow/model-versions/create", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt version: %w", err)
	}

	return modelVersionToPromptVersion(resp.ModelVersion), nil
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
		Prompts:       make([]Prompt, 0, len(resp.RegisteredModels)),
		NextPageToken: resp.GetNextPageToken(),
	}

	for _, rm := range resp.RegisteredModels {
		result.Prompts = append(result.Prompts, registeredModelToPrompt(rm))
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
// MLflow OSS has a search indexing bug where model-versions/search permanently
// returns empty results for models whose versions were created in rapid succession
// (<50ms between calls). Direct GET by version number works fine. See test.md
// for detailed reproduction steps and analysis.
//
// To work around this, ListPromptVersions tries the search endpoint first,
// and falls back to individual version fetches if search returns empty.
func (c *Client) ListPromptVersions(ctx context.Context, name string, opts ...ListVersionsOption) (*PromptVersionList, error) {
	if name == "" {
		return nil, fmt.Errorf("mlflow: prompt name is required")
	}

	listOpts := &listVersionsOptions{
		maxResults: 100,
	}
	for _, opt := range opts {
		opt(listOpts)
	}

	// Try efficient search endpoint first
	result, err := c.listVersionsViaSearch(ctx, name, listOpts.maxResults)
	if err != nil {
		return nil, err
	}

	// If search returned results, we're done
	if len(result.Versions) > 0 {
		return result, nil
	}

	// Search returned empty — fall back to individual fetches.
	// See ListPromptVersions godoc for why this is needed.
	return c.listVersionsViaIndividualFetch(ctx, name, listOpts.maxResults)
}

// listVersionsViaSearch uses the model-versions/search endpoint.
// Returns empty list if no versions found (caller should try fallback).
func (c *Client) listVersionsViaSearch(ctx context.Context, name string, maxResults int) (*PromptVersionList, error) {
	var resp mlflowpb.SearchModelVersions_Response

	query := url.Values{
		"filter":      []string{fmt.Sprintf("name='%s'", escapeFilterValue(name))},
		"order_by":    []string{"version_number DESC"},
		"max_results": []string{strconv.Itoa(maxResults)},
	}

	err := c.transport.Get(ctx, "/api/2.0/mlflow/model-versions/search", query, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to search versions: %w", err)
	}

	result := &PromptVersionList{
		Versions: make([]PromptVersion, 0, len(resp.ModelVersions)),
	}

	for _, mv := range resp.ModelVersions {
		result.Versions = append(result.Versions, modelVersionToPromptVersionWithoutTemplate(mv))
	}

	return result, nil
}

// listVersionsViaIndividualFetch fetches versions one by one.
// Used as fallback when the search endpoint returns empty due to the MLflow OSS
// search indexing bug (see ListPromptVersions godoc).
func (c *Client) listVersionsViaIndividualFetch(ctx context.Context, name string, maxResults int) (*PromptVersionList, error) {
	// Get the latest version number using the "latest" alias
	latestPrompt, err := c.loadPromptByAlias(ctx, name, aliasLatest)
	if err != nil {
		if errors.IsNotFound(err) {
			return &PromptVersionList{Versions: []PromptVersion{}}, nil
		}
		return nil, err
	}
	latestVersion := latestPrompt.Version

	slog.Warn("MLflow search returned empty, falling back to individual fetches",
		"prompt", name,
		"latest_version", latestVersion)

	result := &PromptVersionList{
		Versions: make([]PromptVersion, 0, latestVersion),
	}

	for v := latestVersion; v >= 1; v-- {
		if maxResults > 0 && len(result.Versions) >= maxResults {
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

		result.Versions = append(result.Versions, modelVersionToPromptVersionWithoutTemplate(resp.ModelVersion))
	}

	return result, nil
}

// SetPromptAlias sets an alias for a specific version of a prompt.
func (c *Client) SetPromptAlias(ctx context.Context, name, alias string, version int) error {
	if name == "" {
		return fmt.Errorf("mlflow: prompt name is required")
	}
	if alias == "" {
		return fmt.Errorf("mlflow: alias is required")
	}
	if version <= 0 {
		return fmt.Errorf("mlflow: version must be positive")
	}

	tagKey := aliasTagPrefix + alias
	tagValue := strconv.Itoa(version)

	req := &mlflowpb.SetRegisteredModelTag{
		Name:  &name,
		Key:   &tagKey,
		Value: &tagValue,
	}

	var resp mlflowpb.SetRegisteredModelTag_Response
	err := c.transport.Post(ctx, "/api/2.0/mlflow/registered-models/set-tag", req, &resp)
	if err != nil {
		return fmt.Errorf("failed to set alias: %w", err)
	}

	return nil
}

// DeletePromptAlias removes an alias from a prompt.
func (c *Client) DeletePromptAlias(ctx context.Context, name, alias string) error {
	if name == "" {
		return fmt.Errorf("mlflow: prompt name is required")
	}
	if alias == "" {
		return fmt.Errorf("mlflow: alias is required")
	}

	tagKey := aliasTagPrefix + alias

	req := &mlflowpb.DeleteRegisteredModelTag{
		Name: &name,
		Key:  &tagKey,
	}

	var resp mlflowpb.DeleteRegisteredModelTag_Response
	err := c.transport.Delete(ctx, "/api/2.0/mlflow/registered-models/delete-tag", req, &resp)
	if err != nil {
		return fmt.Errorf("failed to delete alias: %w", err)
	}

	return nil
}

// DeletePromptVersion deletes a specific version of a prompt from the registry.
func (c *Client) DeletePromptVersion(ctx context.Context, name string, version int) error {
	if name == "" {
		return fmt.Errorf("mlflow: prompt name is required")
	}
	if version <= 0 {
		return fmt.Errorf("mlflow: version must be positive")
	}

	versionStr := strconv.Itoa(version)
	req := &mlflowpb.DeleteModelVersion{
		Name:    &name,
		Version: &versionStr,
	}

	var resp mlflowpb.DeleteModelVersion_Response
	err := c.transport.Delete(ctx, "/api/2.0/mlflow/model-versions/delete", req, &resp)
	if err != nil {
		return fmt.Errorf("failed to delete prompt version: %w", err)
	}

	return nil
}

// DeletePrompt deletes a prompt from the registry.
// On MLflow OSS, this cascades to delete all versions and aliases automatically.
// On Databricks, versions must be deleted first.
func (c *Client) DeletePrompt(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("mlflow: prompt name is required")
	}

	req := &mlflowpb.DeleteRegisteredModel{
		Name: &name,
	}

	var resp mlflowpb.DeleteRegisteredModel_Response
	err := c.transport.Delete(ctx, "/api/2.0/mlflow/registered-models/delete", req, &resp)
	if err != nil {
		return fmt.Errorf("failed to delete prompt: %w", err)
	}

	return nil
}

// DeletePromptTag removes a tag from a prompt.
func (c *Client) DeletePromptTag(ctx context.Context, name, key string) error {
	if name == "" {
		return fmt.Errorf("mlflow: prompt name is required")
	}
	if key == "" {
		return fmt.Errorf("mlflow: tag key is required")
	}

	req := &mlflowpb.DeleteRegisteredModelTag{
		Name: &name,
		Key:  &key,
	}

	var resp mlflowpb.DeleteRegisteredModelTag_Response
	err := c.transport.Delete(ctx, "/api/2.0/mlflow/registered-models/delete-tag", req, &resp)
	if err != nil {
		return fmt.Errorf("failed to delete prompt tag: %w", err)
	}

	return nil
}

// DeletePromptVersionTag removes a tag from a specific prompt version.
func (c *Client) DeletePromptVersionTag(ctx context.Context, name string, version int, key string) error {
	if name == "" {
		return fmt.Errorf("mlflow: prompt name is required")
	}
	if version <= 0 {
		return fmt.Errorf("mlflow: version must be positive")
	}
	if key == "" {
		return fmt.Errorf("mlflow: tag key is required")
	}

	versionStr := strconv.Itoa(version)
	req := &mlflowpb.DeleteModelVersionTag{
		Name:    &name,
		Version: &versionStr,
		Key:     &key,
	}

	var resp mlflowpb.DeleteModelVersionTag_Response
	err := c.transport.Delete(ctx, "/api/2.0/mlflow/model-versions/delete-tag", req, &resp)
	if err != nil {
		return fmt.Errorf("failed to delete prompt version tag: %w", err)
	}

	return nil
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
	return strings.Join(filters, " AND ")
}

// ptr returns a pointer to the given value.
func ptr[T any](v T) *T {
	return &v
}
