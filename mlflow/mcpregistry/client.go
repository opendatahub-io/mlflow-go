package mcpregistry

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/opendatahub-io/mlflow-go/internal/transport"
)

// mcpServersBasePath is the base path for the MLflow MCP Server Registry REST API.
// Unlike the Prompt Registry and Tracking APIs, these endpoints are
// Pydantic/JSON based rather than protobuf based.
const mcpServersBasePath = "/ajax-api/3.0/mlflow/mcp-servers"

// defaultSearchMaxResults is the default page size for search operations,
// matching the MLflow server-side default.
const defaultSearchMaxResults = 100

// Client provides access to the MLflow MCP Server Registry.
// It is safe for concurrent use.
type Client struct {
	transport *transport.Client
}

// NewClient creates a new MCP Registry client.
// This is typically called internally by the root mlflow.Client.
func NewClient(t *transport.Client) *Client {
	return &Client{transport: t}
}

// hasPathTraversalSegment reports whether any "/"-separated segment of value
// is "." or "..", which could let a caller-supplied identifier escape the
// intended REST resource path when concatenated into a URL.
func hasPathTraversalSegment(value string) bool {
	for _, segment := range strings.Split(value, "/") {
		if segment == "." || segment == ".." {
			return true
		}
	}
	return false
}

// requirePathParam validates that a caller-supplied identifier is non-empty
// and safe to embed as a REST path segment. MCP server names may legitimately
// contain "/" (e.g. "com.example/my-server"), so "/" itself is allowed, but a
// bare "." or ".." segment is rejected so a crafted identifier can't escape
// the intended resource path.
func requirePathParam(paramName, value string) error {
	if value == "" {
		return fmt.Errorf("mlflow: %s is required", paramName)
	}
	if hasPathTraversalSegment(value) {
		return fmt.Errorf(`mlflow: %s must not contain "." or ".." path segments`, paramName)
	}
	return nil
}

// --- Wire types ---
//
// The MCP Registry REST API exchanges plain JSON (Pydantic models), not
// protobuf messages, so requests and responses are modeled by hand here and
// converted to/from the public domain types defined in server.go.

type mcpToolWire struct {
	Name         string           `json:"name"`
	Title        string           `json:"title,omitempty"`
	Description  string           `json:"description,omitempty"`
	InputSchema  map[string]any   `json:"inputSchema,omitempty"`
	OutputSchema map[string]any   `json:"outputSchema,omitempty"`
	Annotations  map[string]any   `json:"annotations,omitempty"`
	Icons        []map[string]any `json:"icons,omitempty"`
	Execution    map[string]any   `json:"execution,omitempty"`
}

type mcpAliasWire struct {
	Alias   string `json:"alias"`
	Version string `json:"version"`
}

type connectOptionSettingsWire struct {
	Hidden bool `json:"hidden,omitempty"`
}

type mcpAccessEndpointSummaryWire struct {
	ID                   string                `json:"id"`
	ServerName           string                `json:"server_name"`
	URL                  string                `json:"url"`
	TransportType        string                `json:"transport_type,omitempty"`
	Workspace            string                `json:"workspace,omitempty"`
	ServerVersion        string                `json:"server_version,omitempty"`
	ServerAlias          string                `json:"server_alias,omitempty"`
	ResolvedVersion      *mcpServerVersionWire `json:"resolved_version,omitempty"`
	CreatedBy            string                `json:"created_by,omitempty"`
	LastUpdatedBy        string                `json:"last_updated_by,omitempty"`
	CreationTimestamp    *int64                `json:"creation_timestamp,omitempty"`
	LastUpdatedTimestamp *int64                `json:"last_updated_timestamp,omitempty"`
}

type mcpServerWire struct {
	Name                 string                         `json:"name"`
	DisplayName          string                         `json:"display_name,omitempty"`
	Description          string                         `json:"description,omitempty"`
	Icons                []map[string]any               `json:"icons,omitempty"`
	Status               string                         `json:"status,omitempty"`
	Workspace            string                         `json:"workspace,omitempty"`
	AccessEndpoints      []mcpAccessEndpointSummaryWire `json:"access_endpoints,omitempty"`
	LatestVersion        string                         `json:"latest_version,omitempty"`
	Aliases              []mcpAliasWire                 `json:"aliases,omitempty"`
	Tags                 map[string]string              `json:"tags,omitempty"`
	CreatedBy            string                         `json:"created_by,omitempty"`
	LastUpdatedBy        string                         `json:"last_updated_by,omitempty"`
	CreationTimestamp    *int64                         `json:"creation_timestamp,omitempty"`
	LastUpdatedTimestamp *int64                         `json:"last_updated_timestamp,omitempty"`
}

type mcpServerVersionWire struct {
	Name                 string                               `json:"name"`
	Version              string                               `json:"version"`
	ServerJSON           map[string]any                       `json:"server_json"`
	DisplayName          string                               `json:"display_name,omitempty"`
	Status               string                               `json:"status,omitempty"`
	Workspace            string                               `json:"workspace,omitempty"`
	Tools                []mcpToolWire                        `json:"tools,omitempty"`
	Aliases              []string                             `json:"aliases,omitempty"`
	Tags                 map[string]string                    `json:"tags,omitempty"`
	ConnectOptions       map[string]connectOptionSettingsWire `json:"connect_options,omitempty"`
	Source               string                               `json:"source,omitempty"`
	CreatedBy            string                               `json:"created_by,omitempty"`
	LastUpdatedBy        string                               `json:"last_updated_by,omitempty"`
	CreationTimestamp    *int64                               `json:"creation_timestamp,omitempty"`
	LastUpdatedTimestamp *int64                               `json:"last_updated_timestamp,omitempty"`
}

type mcpAccessEndpointWire struct {
	ID                   string                `json:"id"`
	ServerName           string                `json:"server_name"`
	URL                  string                `json:"url"`
	TransportType        string                `json:"transport_type,omitempty"`
	Workspace            string                `json:"workspace,omitempty"`
	Tools                []mcpToolWire         `json:"tools,omitempty"`
	ServerVersion        string                `json:"server_version,omitempty"`
	ServerAlias          string                `json:"server_alias,omitempty"`
	ResolvedVersion      *mcpServerVersionWire `json:"resolved_version,omitempty"`
	CreatedBy            string                `json:"created_by,omitempty"`
	LastUpdatedBy        string                `json:"last_updated_by,omitempty"`
	CreationTimestamp    *int64                `json:"creation_timestamp,omitempty"`
	LastUpdatedTimestamp *int64                `json:"last_updated_timestamp,omitempty"`
}

type createMCPServerRequest struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Icons       []map[string]any `json:"icons,omitempty"`
}

// updateMCPServerRequest is a partial-update (PATCH) request. Fields left as
// nil are omitted from the request body entirely and left unchanged
// server-side; a non-nil pointer (even to a zero value) is sent explicitly.
type updateMCPServerRequest struct {
	DisplayName *string           `json:"display_name,omitempty"`
	Description *string           `json:"description,omitempty"`
	Icons       *[]map[string]any `json:"icons,omitempty"`
}

type searchMCPServersResponse struct {
	MCPServers    []mcpServerWire `json:"mcp_servers"`
	NextPageToken string          `json:"next_page_token,omitempty"`
}

type createMCPServerVersionRequest struct {
	ServerJSON     map[string]any                       `json:"server_json"`
	DisplayName    string                               `json:"display_name,omitempty"`
	Status         string                               `json:"status,omitempty"`
	Source         string                               `json:"source,omitempty"`
	Tools          []mcpToolWire                        `json:"tools,omitempty"`
	ConnectOptions map[string]connectOptionSettingsWire `json:"connect_options,omitempty"`
}

// updateMCPServerVersionRequest is a partial-update (PATCH) request. Fields
// left as nil are omitted from the request body entirely and left unchanged
// server-side; a non-nil pointer (even to a zero value) is sent explicitly.
type updateMCPServerVersionRequest struct {
	Status         *string                               `json:"status,omitempty"`
	Tools          *[]mcpToolWire                        `json:"tools,omitempty"`
	ConnectOptions *map[string]connectOptionSettingsWire `json:"connect_options,omitempty"`
}

type searchMCPServerVersionsResponse struct {
	MCPServerVersions []mcpServerVersionWire `json:"mcp_server_versions"`
	NextPageToken     string                 `json:"next_page_token,omitempty"`
}

type createMCPAccessEndpointRequest struct {
	ServerVersion string `json:"server_version,omitempty"`
	ServerAlias   string `json:"server_alias,omitempty"`
	URL           string `json:"url"`
	TransportType string `json:"transport_type,omitempty"`
}

// updateMCPAccessEndpointRequest is a partial-update (PATCH) request. Fields
// left as nil are omitted from the request body entirely and left unchanged
// server-side; a non-nil pointer (even to a zero value) is sent explicitly.
type updateMCPAccessEndpointRequest struct {
	ServerVersion *string `json:"server_version,omitempty"`
	ServerAlias   *string `json:"server_alias,omitempty"`
	URL           *string `json:"url,omitempty"`
	TransportType *string `json:"transport_type,omitempty"`
}

type searchMCPAccessEndpointsResponse struct {
	MCPAccessEndpoints []mcpAccessEndpointWire `json:"mcp_access_endpoints"`
	NextPageToken      string                  `json:"next_page_token,omitempty"`
}

type setTagRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type setAliasRequest struct {
	Alias   string `json:"alias"`
	Version string `json:"version"`
}

// --- Wire -> domain conversion helpers ---

func timestampFromWire(ms *int64) time.Time {
	if ms == nil {
		return time.Time{}
	}
	return time.UnixMilli(*ms)
}

func toolsFromWire(tools []mcpToolWire) []MCPTool {
	if tools == nil {
		return nil
	}
	out := make([]MCPTool, 0, len(tools))
	for _, t := range tools {
		out = append(out, MCPTool(t))
	}
	return out
}

func toolsToWire(tools []MCPTool) []mcpToolWire {
	if tools == nil {
		return nil
	}
	out := make([]mcpToolWire, 0, len(tools))
	for _, t := range tools {
		out = append(out, mcpToolWire(t))
	}
	return out
}

func connectOptionsFromWire(opts map[string]connectOptionSettingsWire) map[string]ConnectOptionSettings {
	if opts == nil {
		return nil
	}
	out := make(map[string]ConnectOptionSettings, len(opts))
	for k, v := range opts {
		out[k] = ConnectOptionSettings(v)
	}
	return out
}

func connectOptionsToWire(opts map[string]ConnectOptionSettings) map[string]connectOptionSettingsWire {
	if opts == nil {
		return nil
	}
	out := make(map[string]connectOptionSettingsWire, len(opts))
	for k, v := range opts {
		out[k] = connectOptionSettingsWire(v)
	}
	return out
}

func serverVersionFromWire(v *mcpServerVersionWire) *MCPServerVersion {
	if v == nil {
		return nil
	}

	aliases := make([]string, len(v.Aliases))
	copy(aliases, v.Aliases)

	tags := v.Tags
	if tags == nil {
		tags = map[string]string{}
	}

	connectOptions := connectOptionsFromWire(v.ConnectOptions)
	if connectOptions == nil {
		connectOptions = map[string]ConnectOptionSettings{}
	}

	return &MCPServerVersion{
		Name:                 v.Name,
		Version:              v.Version,
		ServerJSON:           v.ServerJSON,
		DisplayName:          v.DisplayName,
		Status:               MCPServerVersionStatus(v.Status),
		Workspace:            v.Workspace,
		Tools:                toolsFromWire(v.Tools),
		Aliases:              aliases,
		Tags:                 tags,
		ConnectOptions:       connectOptions,
		Source:               v.Source,
		CreatedBy:            v.CreatedBy,
		LastUpdatedBy:        v.LastUpdatedBy,
		CreationTimestamp:    timestampFromWire(v.CreationTimestamp),
		LastUpdatedTimestamp: timestampFromWire(v.LastUpdatedTimestamp),
	}
}

func accessEndpointSummaryFromWire(b mcpAccessEndpointSummaryWire) MCPAccessEndpointSummary {
	return MCPAccessEndpointSummary{
		ID:                   b.ID,
		ServerName:           b.ServerName,
		EndpointURL:          b.URL,
		TransportType:        MCPTransportType(b.TransportType),
		Workspace:            b.Workspace,
		ServerVersion:        b.ServerVersion,
		ServerAlias:          b.ServerAlias,
		ResolvedVersion:      serverVersionFromWire(b.ResolvedVersion),
		CreatedBy:            b.CreatedBy,
		LastUpdatedBy:        b.LastUpdatedBy,
		CreationTimestamp:    timestampFromWire(b.CreationTimestamp),
		LastUpdatedTimestamp: timestampFromWire(b.LastUpdatedTimestamp),
	}
}

func serverFromWire(s *mcpServerWire) *MCPServer {
	if s == nil {
		return nil
	}

	aliases := make(map[string]string, len(s.Aliases))
	for _, a := range s.Aliases {
		aliases[a.Alias] = a.Version
	}

	endpoints := make([]MCPAccessEndpointSummary, 0, len(s.AccessEndpoints))
	for _, b := range s.AccessEndpoints {
		endpoints = append(endpoints, accessEndpointSummaryFromWire(b))
	}

	tags := s.Tags
	if tags == nil {
		tags = map[string]string{}
	}

	return &MCPServer{
		Name:                 s.Name,
		DisplayName:          s.DisplayName,
		Description:          s.Description,
		Icons:                s.Icons,
		Status:               s.Status,
		Workspace:            s.Workspace,
		AccessEndpoints:      endpoints,
		LatestVersion:        s.LatestVersion,
		Aliases:              aliases,
		Tags:                 tags,
		CreatedBy:            s.CreatedBy,
		LastUpdatedBy:        s.LastUpdatedBy,
		CreationTimestamp:    timestampFromWire(s.CreationTimestamp),
		LastUpdatedTimestamp: timestampFromWire(s.LastUpdatedTimestamp),
	}
}

func accessEndpointFromWire(b *mcpAccessEndpointWire) *MCPAccessEndpoint {
	if b == nil {
		return nil
	}

	return &MCPAccessEndpoint{
		ID:                   b.ID,
		ServerName:           b.ServerName,
		EndpointURL:          b.URL,
		TransportType:        MCPTransportType(b.TransportType),
		Workspace:            b.Workspace,
		Tools:                toolsFromWire(b.Tools),
		ServerVersion:        b.ServerVersion,
		ServerAlias:          b.ServerAlias,
		ResolvedVersion:      serverVersionFromWire(b.ResolvedVersion),
		CreatedBy:            b.CreatedBy,
		LastUpdatedBy:        b.LastUpdatedBy,
		CreationTimestamp:    timestampFromWire(b.CreationTimestamp),
		LastUpdatedTimestamp: timestampFromWire(b.LastUpdatedTimestamp),
	}
}

// --- Server operations ---

// CreateMCPServer registers a new MCP server in the registry.
func (c *Client) CreateMCPServer(ctx context.Context, name string, opts ...CreateMCPServerOption) (*MCPServer, error) {
	if err := requirePathParam("server name", name); err != nil {
		return nil, err
	}

	o := &createServerOptions{}
	for _, opt := range opts {
		opt(o)
	}

	req := &createMCPServerRequest{
		Name:        name,
		Description: o.description,
		Icons:       o.icons,
	}

	var resp mcpServerWire

	err := c.transport.Post(ctx, mcpServersBasePath, req, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}

	return serverFromWire(&resp), nil
}

// GetMCPServer retrieves an MCP server by name.
func (c *Client) GetMCPServer(ctx context.Context, name string) (*MCPServer, error) {
	if err := requirePathParam("server name", name); err != nil {
		return nil, err
	}

	var resp mcpServerWire

	err := c.transport.Get(ctx, mcpServersBasePath+"/"+name, nil, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP server: %w", err)
	}

	return serverFromWire(&resp), nil
}

// SearchMCPServers searches for MCP servers matching the given criteria.
func (c *Client) SearchMCPServers(ctx context.Context, opts ...SearchMCPServersOption) (*MCPServerList, error) {
	o := &searchServersOptions{maxResults: defaultSearchMaxResults}
	for _, opt := range opts {
		opt(o)
	}

	if o.maxResults <= 0 {
		return nil, fmt.Errorf("mlflow: max results must be positive")
	}

	query := url.Values{"max_results": []string{strconv.Itoa(o.maxResults)}}
	if o.filter != "" {
		query.Set("filter_string", o.filter)
	}
	if o.pageToken != "" {
		query.Set("page_token", o.pageToken)
	}
	if len(o.orderBy) > 0 {
		query["order_by"] = o.orderBy
	}

	var resp searchMCPServersResponse

	err := c.transport.Get(ctx, mcpServersBasePath, query, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to search MCP servers: %w", err)
	}

	result := &MCPServerList{
		Servers:       make([]MCPServer, 0, len(resp.MCPServers)),
		NextPageToken: resp.NextPageToken,
	}
	for _, s := range resp.MCPServers {
		wire := s
		result.Servers = append(result.Servers, *serverFromWire(&wire))
	}

	return result, nil
}

// SetMCPServerTag sets a tag on an MCP server.
func (c *Client) SetMCPServerTag(ctx context.Context, name, key, value string) error {
	if err := requirePathParam("server name", name); err != nil {
		return err
	}
	if err := requirePathParam("tag key", key); err != nil {
		return err
	}

	req := &setTagRequest{Key: key, Value: value}

	err := c.transport.Post(ctx, mcpServersBasePath+"/"+name+"/tags", req, nil)
	if err != nil {
		return fmt.Errorf("failed to set MCP server tag: %w", err)
	}

	return nil
}

// DeleteMCPServerTag removes a tag from an MCP server.
func (c *Client) DeleteMCPServerTag(ctx context.Context, name, key string) error {
	if err := requirePathParam("server name", name); err != nil {
		return err
	}
	if err := requirePathParam("tag key", key); err != nil {
		return err
	}

	err := c.transport.Delete(ctx, mcpServersBasePath+"/"+name+"/tags/"+key, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete MCP server tag: %w", err)
	}

	return nil
}

// UpdateMCPServer updates mutable fields (display name, description, icons)
// on an existing MCP server. Only fields configured via the supplied options
// are modified; omitted fields are left unchanged.
func (c *Client) UpdateMCPServer(ctx context.Context, name string, opts ...UpdateMCPServerOption) (*MCPServer, error) {
	if err := requirePathParam("server name", name); err != nil {
		return nil, err
	}

	o := &updateServerOptions{}
	for _, opt := range opts {
		opt(o)
	}

	req := &updateMCPServerRequest{
		DisplayName: o.displayName,
		Description: o.description,
		Icons:       o.icons,
	}

	var resp mcpServerWire

	err := c.transport.Patch(ctx, mcpServersBasePath+"/"+name, req, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to update MCP server: %w", err)
	}

	return serverFromWire(&resp), nil
}

// DeleteMCPServer removes an MCP server and all of its versions, access
// endpoints, aliases, and tags.
func (c *Client) DeleteMCPServer(ctx context.Context, name string) error {
	if err := requirePathParam("server name", name); err != nil {
		return err
	}

	err := c.transport.Delete(ctx, mcpServersBasePath+"/"+name, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete MCP server: %w", err)
	}

	return nil
}

// SetMCPServerAlias points an alias (e.g. "production") at a specific
// version of an MCP server, creating or overwriting the alias.
func (c *Client) SetMCPServerAlias(ctx context.Context, name, alias, version string) error {
	if err := requirePathParam("server name", name); err != nil {
		return err
	}
	if err := requirePathParam("alias", alias); err != nil {
		return err
	}
	if err := requirePathParam("version", version); err != nil {
		return err
	}

	req := &setAliasRequest{Alias: alias, Version: version}

	err := c.transport.Post(ctx, mcpServersBasePath+"/"+name+"/aliases", req, nil)
	if err != nil {
		return fmt.Errorf("failed to set MCP server alias: %w", err)
	}

	return nil
}

// GetMCPServerVersionByAlias resolves an alias (e.g. "production") to the
// MCP server version it currently points to.
func (c *Client) GetMCPServerVersionByAlias(ctx context.Context, name, alias string) (*MCPServerVersion, error) {
	if err := requirePathParam("server name", name); err != nil {
		return nil, err
	}
	if err := requirePathParam("alias", alias); err != nil {
		return nil, err
	}

	var resp mcpServerVersionWire

	err := c.transport.Get(ctx, mcpServersBasePath+"/"+name+"/aliases/"+alias, nil, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP server version by alias: %w", err)
	}

	return serverVersionFromWire(&resp), nil
}

// DeleteMCPServerAlias removes an alias from an MCP server.
func (c *Client) DeleteMCPServerAlias(ctx context.Context, name, alias string) error {
	if err := requirePathParam("server name", name); err != nil {
		return err
	}
	if err := requirePathParam("alias", alias); err != nil {
		return err
	}

	err := c.transport.Delete(ctx, mcpServersBasePath+"/"+name+"/aliases/"+alias, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete MCP server alias: %w", err)
	}

	return nil
}

// --- Server version operations ---

// CreateMCPServerVersion adds a new version to an existing MCP server.
// serverJSON must follow the MCP registry server.json schema and must include
// a "name" matching the server's name and a "version" for the new version.
func (c *Client) CreateMCPServerVersion(ctx context.Context, name string, serverJSON map[string]any, opts ...CreateMCPServerVersionOption) (*MCPServerVersion, error) {
	if err := requirePathParam("server name", name); err != nil {
		return nil, err
	}
	if len(serverJSON) == 0 {
		return nil, fmt.Errorf("mlflow: server JSON is required")
	}
	serverJSONName, ok := serverJSON["name"].(string)
	if !ok || serverJSONName == "" {
		return nil, fmt.Errorf(`mlflow: server JSON "name" is required`)
	}
	if serverJSONName != name {
		return nil, fmt.Errorf("mlflow: server JSON name %q must match server name %q", serverJSONName, name)
	}
	if serverJSONVersion, ok := serverJSON["version"].(string); !ok || serverJSONVersion == "" {
		return nil, fmt.Errorf(`mlflow: server JSON "version" is required`)
	}

	o := &createServerVersionOptions{status: MCPServerVersionStatusDraft}
	for _, opt := range opts {
		opt(o)
	}

	req := &createMCPServerVersionRequest{
		ServerJSON:     serverJSON,
		DisplayName:    o.displayName,
		Status:         string(o.status),
		Source:         o.source,
		Tools:          toolsToWire(o.tools),
		ConnectOptions: connectOptionsToWire(o.connectOptions),
	}

	var resp mcpServerVersionWire

	err := c.transport.Post(ctx, mcpServersBasePath+"/"+name+"/versions", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server version: %w", err)
	}

	return serverVersionFromWire(&resp), nil
}

// GetMCPServerVersion retrieves a specific version of an MCP server.
func (c *Client) GetMCPServerVersion(ctx context.Context, name, version string) (*MCPServerVersion, error) {
	if err := requirePathParam("server name", name); err != nil {
		return nil, err
	}
	if err := requirePathParam("version", version); err != nil {
		return nil, err
	}

	var resp mcpServerVersionWire

	err := c.transport.Get(ctx, mcpServersBasePath+"/"+name+"/versions/"+version, nil, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP server version: %w", err)
	}

	return serverVersionFromWire(&resp), nil
}

// UpdateMCPServerVersion updates mutable fields (status, tools,
// connect options) on an existing MCP server version. Only fields
// configured via the supplied options are modified; omitted fields are left
// unchanged.
func (c *Client) UpdateMCPServerVersion(ctx context.Context, name, version string, opts ...UpdateMCPServerVersionOption) (*MCPServerVersion, error) {
	if err := requirePathParam("server name", name); err != nil {
		return nil, err
	}
	if err := requirePathParam("version", version); err != nil {
		return nil, err
	}

	o := &updateServerVersionOptions{}
	for _, opt := range opts {
		opt(o)
	}

	req := &updateMCPServerVersionRequest{}
	if o.status != nil {
		status := string(*o.status)
		req.Status = &status
	}
	if o.tools != nil {
		tools := toolsToWire(*o.tools)
		req.Tools = &tools
	}
	if o.connectOptions != nil {
		connectOptions := connectOptionsToWire(*o.connectOptions)
		req.ConnectOptions = &connectOptions
	}

	var resp mcpServerVersionWire

	err := c.transport.Patch(ctx, mcpServersBasePath+"/"+name+"/versions/"+version, req, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to update MCP server version: %w", err)
	}

	return serverVersionFromWire(&resp), nil
}

// DeleteMCPServerVersion removes a specific version of an MCP server.
func (c *Client) DeleteMCPServerVersion(ctx context.Context, name, version string) error {
	if err := requirePathParam("server name", name); err != nil {
		return err
	}
	if err := requirePathParam("version", version); err != nil {
		return err
	}

	err := c.transport.Delete(ctx, mcpServersBasePath+"/"+name+"/versions/"+version, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete MCP server version: %w", err)
	}

	return nil
}

// SetMCPServerVersionTag sets a tag on a specific version of an MCP server.
func (c *Client) SetMCPServerVersionTag(ctx context.Context, name, version, key, value string) error {
	if err := requirePathParam("server name", name); err != nil {
		return err
	}
	if err := requirePathParam("version", version); err != nil {
		return err
	}
	if err := requirePathParam("tag key", key); err != nil {
		return err
	}

	req := &setTagRequest{Key: key, Value: value}

	err := c.transport.Post(ctx, mcpServersBasePath+"/"+name+"/versions/"+version+"/tags", req, nil)
	if err != nil {
		return fmt.Errorf("failed to set MCP server version tag: %w", err)
	}

	return nil
}

// DeleteMCPServerVersionTag removes a tag from a specific version of an MCP server.
func (c *Client) DeleteMCPServerVersionTag(ctx context.Context, name, version, key string) error {
	if err := requirePathParam("server name", name); err != nil {
		return err
	}
	if err := requirePathParam("version", version); err != nil {
		return err
	}
	if err := requirePathParam("tag key", key); err != nil {
		return err
	}

	path := mcpServersBasePath + "/" + name + "/versions/" + version + "/tags/" + key

	err := c.transport.Delete(ctx, path, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete MCP server version tag: %w", err)
	}

	return nil
}

// SearchMCPServerVersions returns versions for a specific MCP server.
func (c *Client) SearchMCPServerVersions(ctx context.Context, name string, opts ...SearchMCPServerVersionsOption) (*MCPServerVersionList, error) {
	if err := requirePathParam("server name", name); err != nil {
		return nil, err
	}

	o := &searchServerVersionsOptions{maxResults: defaultSearchMaxResults}
	for _, opt := range opts {
		opt(o)
	}

	if o.maxResults <= 0 {
		return nil, fmt.Errorf("mlflow: max results must be positive")
	}

	query := url.Values{"max_results": []string{strconv.Itoa(o.maxResults)}}
	if o.filter != "" {
		query.Set("filter_string", o.filter)
	}
	if o.pageToken != "" {
		query.Set("page_token", o.pageToken)
	}
	if len(o.orderBy) > 0 {
		query["order_by"] = o.orderBy
	}

	var resp searchMCPServerVersionsResponse

	err := c.transport.Get(ctx, mcpServersBasePath+"/"+name+"/versions", query, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to search MCP server versions: %w", err)
	}

	result := &MCPServerVersionList{
		Versions:      make([]MCPServerVersion, 0, len(resp.MCPServerVersions)),
		NextPageToken: resp.NextPageToken,
	}
	for _, v := range resp.MCPServerVersions {
		wire := v
		result.Versions = append(result.Versions, *serverVersionFromWire(&wire))
	}

	return result, nil
}

// --- Access endpoint operations ---

// CreateMCPAccessEndpoint registers a new access endpoint for an MCP
// server, optionally pinned to a specific version or alias.
func (c *Client) CreateMCPAccessEndpoint(ctx context.Context, serverName, endpointURL string, opts ...CreateMCPAccessEndpointOption) (*MCPAccessEndpoint, error) {
	if err := requirePathParam("server name", serverName); err != nil {
		return nil, err
	}
	if endpointURL == "" {
		return nil, fmt.Errorf("mlflow: endpoint URL is required")
	}

	o := &createAccessEndpointOptions{transportType: MCPTransportStreamableHTTP}
	for _, opt := range opts {
		opt(o)
	}
	if o.serverVersion != "" && o.serverAlias != "" {
		return nil, fmt.Errorf("mlflow: server version and server alias are mutually exclusive")
	}

	req := &createMCPAccessEndpointRequest{
		ServerVersion: o.serverVersion,
		ServerAlias:   o.serverAlias,
		URL:           endpointURL,
		TransportType: string(o.transportType),
	}

	var resp mcpAccessEndpointWire

	err := c.transport.Post(ctx, mcpServersBasePath+"/"+serverName+"/endpoints", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP access endpoint: %w", err)
	}

	return accessEndpointFromWire(&resp), nil
}

// GetMCPAccessEndpoint retrieves a single access endpoint by ID.
func (c *Client) GetMCPAccessEndpoint(ctx context.Context, serverName, endpointID string) (*MCPAccessEndpoint, error) {
	if err := requirePathParam("server name", serverName); err != nil {
		return nil, err
	}
	if err := requirePathParam("endpoint ID", endpointID); err != nil {
		return nil, err
	}

	var resp mcpAccessEndpointWire

	err := c.transport.Get(ctx, mcpServersBasePath+"/"+serverName+"/endpoints/"+endpointID, nil, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP access endpoint: %w", err)
	}

	return accessEndpointFromWire(&resp), nil
}

// UpdateMCPAccessEndpoint updates mutable fields (URL, transport type,
// version/alias pin) on an existing access endpoint. Only fields configured
// via the supplied options are modified; omitted fields are left unchanged.
func (c *Client) UpdateMCPAccessEndpoint(ctx context.Context, serverName, endpointID string, opts ...UpdateMCPAccessEndpointOption) (*MCPAccessEndpoint, error) {
	if err := requirePathParam("server name", serverName); err != nil {
		return nil, err
	}
	if err := requirePathParam("endpoint ID", endpointID); err != nil {
		return nil, err
	}

	o := &updateAccessEndpointOptions{}
	for _, opt := range opts {
		opt(o)
	}
	if o.serverVersion != nil && o.serverAlias != nil && *o.serverVersion != "" && *o.serverAlias != "" {
		return nil, fmt.Errorf("mlflow: server version and server alias are mutually exclusive")
	}

	req := &updateMCPAccessEndpointRequest{
		ServerVersion: o.serverVersion,
		ServerAlias:   o.serverAlias,
		URL:           o.url,
	}
	if o.transportType != nil {
		transportType := string(*o.transportType)
		req.TransportType = &transportType
	}

	var resp mcpAccessEndpointWire

	err := c.transport.Patch(ctx, mcpServersBasePath+"/"+serverName+"/endpoints/"+endpointID, req, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to update MCP access endpoint: %w", err)
	}

	return accessEndpointFromWire(&resp), nil
}

// SearchMCPAccessEndpoints searches for MCP access endpoints matching the given
// criteria. By default the search spans all servers; use
// WithAccessEndpointsServerName to scope the search to a single server.
func (c *Client) SearchMCPAccessEndpoints(ctx context.Context, opts ...SearchMCPAccessEndpointsOption) (*MCPAccessEndpointList, error) {
	o := &searchAccessEndpointsOptions{maxResults: defaultSearchMaxResults}
	for _, opt := range opts {
		opt(o)
	}

	if o.maxResults <= 0 {
		return nil, fmt.Errorf("mlflow: max results must be positive")
	}
	if o.serverName != "" && hasPathTraversalSegment(o.serverName) {
		return nil, fmt.Errorf(`mlflow: server name must not contain "." or ".." path segments`)
	}

	query := url.Values{"max_results": []string{strconv.Itoa(o.maxResults)}}
	if o.filter != "" {
		query.Set("filter_string", o.filter)
	}
	if o.pageToken != "" {
		query.Set("page_token", o.pageToken)
	}
	if len(o.orderBy) > 0 {
		query["order_by"] = o.orderBy
	}
	if o.serverVersion != "" {
		query.Set("server_version", o.serverVersion)
	}
	if o.serverAlias != "" {
		query.Set("server_alias", o.serverAlias)
	}

	path := mcpServersBasePath + "/endpoints"
	if o.serverName != "" {
		path = mcpServersBasePath + "/" + o.serverName + "/endpoints"
	}

	var resp searchMCPAccessEndpointsResponse

	err := c.transport.Get(ctx, path, query, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to search MCP access endpoints: %w", err)
	}

	result := &MCPAccessEndpointList{
		Endpoints:     make([]MCPAccessEndpoint, 0, len(resp.MCPAccessEndpoints)),
		NextPageToken: resp.NextPageToken,
	}
	for _, b := range resp.MCPAccessEndpoints {
		wire := b
		result.Endpoints = append(result.Endpoints, *accessEndpointFromWire(&wire))
	}

	return result, nil
}

// DeleteMCPAccessEndpoint removes an access endpoint from an MCP server.
func (c *Client) DeleteMCPAccessEndpoint(ctx context.Context, serverName, endpointID string) error {
	if err := requirePathParam("server name", serverName); err != nil {
		return err
	}
	if err := requirePathParam("endpoint ID", endpointID); err != nil {
		return err
	}

	path := mcpServersBasePath + "/" + serverName + "/endpoints/" + endpointID

	err := c.transport.Delete(ctx, path, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete MCP access endpoint: %w", err)
	}

	return nil
}
