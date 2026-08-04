// Package mcpregistry provides types and operations for the MLflow MCP
// (Model Context Protocol) Server Registry.
//
// The MCP Registry is a centralized catalog for registering, versioning, and
// sharing Model Context Protocol servers across a team. Unlike the Prompt
// Registry and Tracking APIs, the MCP Registry REST endpoints are
// Pydantic/JSON based rather than protobuf based, so this package talks to
// the server using hand-written request/response types instead of generated
// protobuf messages.
//
// Reference: https://mlflow.org/docs/latest/genai/mcp/ (MCP Server Registry)
package mcpregistry

import "time"

// MCPServerVersionStatus represents the lifecycle status of an MCP server version.
type MCPServerVersionStatus string

const (
	MCPServerVersionStatusDraft      MCPServerVersionStatus = "draft"
	MCPServerVersionStatusActive     MCPServerVersionStatus = "active"
	MCPServerVersionStatusDeprecated MCPServerVersionStatus = "deprecated"
	MCPServerVersionStatusDeleted    MCPServerVersionStatus = "deleted"
)

// MCPTransportType represents how an MCP access endpoint is reached.
type MCPTransportType string

const (
	// MCPTransportStreamableHTTP is the default transport type used by the
	// MLflow MCP Registry when none is specified.
	MCPTransportStreamableHTTP MCPTransportType = "streamable-http"

	// MCPTransportSSE reaches the MCP server over Server-Sent Events.
	MCPTransportSSE MCPTransportType = "sse"
)

// MCPTool describes a single tool exposed by an MCP server version.
// MLflow auto-discovers tools when a server version is created.
type MCPTool struct {
	// Name is the tool identifier.
	Name string `json:"name"`

	// Title is an optional human-readable title.
	Title string `json:"title,omitempty"`

	// Description explains what the tool does.
	Description string `json:"description,omitempty"`

	// InputSchema is the JSON Schema describing the tool's input parameters.
	InputSchema map[string]any `json:"inputSchema,omitempty"`

	// OutputSchema is the JSON Schema describing the tool's output, if any.
	OutputSchema map[string]any `json:"outputSchema,omitempty"`

	// Annotations contains additional MCP tool annotations (e.g. hints).
	Annotations map[string]any `json:"annotations,omitempty"`

	// Icons contains optional icon descriptors for the tool.
	Icons []map[string]any `json:"icons,omitempty"`

	// Execution contains optional MCP execution metadata for the tool.
	Execution map[string]any `json:"execution,omitempty"`
}

// ConnectOptionSettings configures how a single connection mode (e.g. a
// package registry type or remote transport) is presented to clients when
// they connect to an MCP server version.
type ConnectOptionSettings struct {
	// Hidden, when true, hides this connection option from the default
	// client-facing connection instructions.
	Hidden bool
}

// MCPAccessEndpointSummary is a lightweight view of an access endpoint as
// returned embedded within an MCPServer.
type MCPAccessEndpointSummary struct {
	// ID uniquely identifies the access endpoint.
	ID string

	// ServerName is the name of the MCP server this endpoint belongs to.
	ServerName string

	// EndpointURL is the URL clients use to reach the MCP server.
	EndpointURL string

	// TransportType describes how the endpoint is reached (e.g. "streamable-http").
	TransportType MCPTransportType

	// Workspace is the workspace this endpoint belongs to, if workspace
	// isolation is enabled on the server.
	Workspace string

	// ServerVersion pins the endpoint to a specific version, if set.
	ServerVersion string

	// ServerAlias pins the endpoint to an alias (e.g. "production"), if set.
	ServerAlias string

	// ResolvedVersion is the version currently resolved by ServerVersion/ServerAlias.
	ResolvedVersion *MCPServerVersion

	// CreatedBy is the user who created the endpoint.
	CreatedBy string

	// LastUpdatedBy is the user who last updated the endpoint.
	LastUpdatedBy string

	// CreationTimestamp is when the endpoint was created.
	CreationTimestamp time.Time

	// LastUpdatedTimestamp is when the endpoint was last updated.
	LastUpdatedTimestamp time.Time
}

// MCPServer represents an MLflow MCP Registry server entry.
type MCPServer struct {
	// Name is the server identifier (e.g. "com.example/my-server").
	Name string

	// DisplayName is an optional human-friendly name for the server.
	DisplayName string

	// Description describes the server.
	Description string

	// Icons contains optional icon descriptors for the server.
	Icons []map[string]any

	// Status is the server's overall status.
	Status string

	// Workspace is the workspace this server belongs to, if workspace
	// isolation is enabled on the server.
	Workspace string

	// AccessEndpoints lists the access endpoints registered for this server.
	AccessEndpoints []MCPAccessEndpointSummary

	// LatestVersion is the most recent version identifier, empty if no versions exist.
	LatestVersion string

	// Aliases maps alias names to version identifiers.
	Aliases map[string]string

	// Tags are key-value metadata pairs.
	Tags map[string]string

	// CreatedBy is the user who created the server.
	CreatedBy string

	// LastUpdatedBy is the user who last updated the server.
	LastUpdatedBy string

	// CreationTimestamp is when the server was created.
	CreationTimestamp time.Time

	// LastUpdatedTimestamp is when the server was last updated.
	LastUpdatedTimestamp time.Time
}

// MCPServerList contains servers and a pagination token for the next page.
type MCPServerList struct {
	Servers       []MCPServer
	NextPageToken string
}

// MCPServerVersion represents a specific version of an MCP server.
type MCPServerVersion struct {
	// Name is the server identifier this version belongs to.
	Name string

	// Version is the semantic version string (e.g. "1.0.0").
	Version string

	// ServerJSON is the raw MCP server.json document describing how to run
	// the server (packages, remotes, etc.), per the MCP registry schema.
	ServerJSON map[string]any

	// DisplayName is an optional human-friendly name for the version.
	DisplayName string

	// Status is the version's lifecycle status (draft, active, deprecated, deleted).
	Status MCPServerVersionStatus

	// Workspace is the workspace this version belongs to, if workspace
	// isolation is enabled on the server.
	Workspace string

	// Tools lists the tools MLflow discovered for this version.
	Tools []MCPTool

	// Aliases lists the alias names currently pointing at this version.
	Aliases []string

	// Tags are key-value metadata pairs.
	Tags map[string]string

	// ConnectOptions configures per-connection-mode display settings (e.g.
	// hiding a specific package registry or remote transport from the
	// default connection instructions), keyed by connection mode name.
	ConnectOptions map[string]ConnectOptionSettings

	// Source describes where this version's definition came from.
	Source string

	// CreatedBy is the user who created the version.
	CreatedBy string

	// LastUpdatedBy is the user who last updated the version.
	LastUpdatedBy string

	// CreationTimestamp is when the version was created.
	CreationTimestamp time.Time

	// LastUpdatedTimestamp is when the version was last updated.
	LastUpdatedTimestamp time.Time
}

// MCPServerVersionList contains versions and a pagination token for the next page.
type MCPServerVersionList struct {
	Versions      []MCPServerVersion
	NextPageToken string
}

// MCPAccessEndpoint represents a concrete, reachable URL bound to an MCP
// server (optionally pinned to a version or alias).
type MCPAccessEndpoint struct {
	// ID uniquely identifies the access endpoint.
	ID string

	// ServerName is the name of the MCP server this endpoint belongs to.
	ServerName string

	// EndpointURL is the URL clients use to reach the MCP server.
	EndpointURL string

	// TransportType describes how the endpoint is reached (e.g. "streamable-http").
	TransportType MCPTransportType

	// Workspace is the workspace this endpoint belongs to, if workspace
	// isolation is enabled on the server.
	Workspace string

	// Tools lists the tools exposed by the resolved version, if known.
	Tools []MCPTool

	// ServerVersion pins the endpoint to a specific version, if set.
	ServerVersion string

	// ServerAlias pins the endpoint to an alias (e.g. "production"), if set.
	ServerAlias string

	// ResolvedVersion is the version currently resolved by ServerVersion/ServerAlias.
	ResolvedVersion *MCPServerVersion

	// CreatedBy is the user who created the endpoint.
	CreatedBy string

	// LastUpdatedBy is the user who last updated the endpoint.
	LastUpdatedBy string

	// CreationTimestamp is when the endpoint was created.
	CreationTimestamp time.Time

	// LastUpdatedTimestamp is when the endpoint was last updated.
	LastUpdatedTimestamp time.Time
}

// MCPAccessEndpointList contains access endpoints and a pagination token for the next page.
type MCPAccessEndpointList struct {
	Endpoints     []MCPAccessEndpoint
	NextPageToken string
}
