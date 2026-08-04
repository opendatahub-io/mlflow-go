package mcpregistry

// createServerOptions holds configuration for a CreateMCPServer call.
type createServerOptions struct {
	description string
	icons       []map[string]any
}

// CreateMCPServerOption configures a CreateMCPServer call.
type CreateMCPServerOption func(*createServerOptions)

// WithServerDescription sets the description for a new MCP server.
func WithServerDescription(description string) CreateMCPServerOption {
	return func(o *createServerOptions) {
		o.description = description
	}
}

// WithServerIcons sets icon descriptors for a new MCP server.
func WithServerIcons(icons []map[string]any) CreateMCPServerOption {
	return func(o *createServerOptions) {
		o.icons = icons
	}
}

// searchServersOptions holds configuration for a SearchMCPServers call.
type searchServersOptions struct {
	filter     string
	maxResults int
	pageToken  string
	orderBy    []string
}

// SearchMCPServersOption configures a SearchMCPServers call.
type SearchMCPServersOption func(*searchServersOptions)

// WithServersFilter sets the search filter string for servers.
func WithServersFilter(filter string) SearchMCPServersOption {
	return func(o *searchServersOptions) {
		o.filter = filter
	}
}

// WithServersMaxResults sets the maximum number of servers to return per page.
func WithServersMaxResults(n int) SearchMCPServersOption {
	return func(o *searchServersOptions) {
		o.maxResults = n
	}
}

// WithServersPageToken sets the pagination token for fetching the next page.
func WithServersPageToken(token string) SearchMCPServersOption {
	return func(o *searchServersOptions) {
		o.pageToken = token
	}
}

// WithServersOrderBy sets the sort order for the server search results.
func WithServersOrderBy(fields ...string) SearchMCPServersOption {
	return func(o *searchServersOptions) {
		o.orderBy = fields
	}
}

// updateServerOptions holds configuration for an UpdateMCPServer call.
// Pointer fields distinguish "not provided" (nil, left unchanged) from
// "provided" (updated to the given value, even if zero/empty).
type updateServerOptions struct {
	displayName *string
	description *string
	icons       *[]map[string]any
}

// UpdateMCPServerOption configures an UpdateMCPServer call.
type UpdateMCPServerOption func(*updateServerOptions)

// WithUpdatedServerDisplayName updates the display name of an MCP server.
// Pass "" to clear it.
func WithUpdatedServerDisplayName(displayName string) UpdateMCPServerOption {
	return func(o *updateServerOptions) {
		o.displayName = &displayName
	}
}

// WithUpdatedServerDescription updates the description of an MCP server.
// Pass "" to clear it.
func WithUpdatedServerDescription(description string) UpdateMCPServerOption {
	return func(o *updateServerOptions) {
		o.description = &description
	}
}

// WithUpdatedServerIcons updates the icon descriptors of an MCP server.
// Pass an empty slice to clear them.
func WithUpdatedServerIcons(icons []map[string]any) UpdateMCPServerOption {
	return func(o *updateServerOptions) {
		o.icons = &icons
	}
}

// createServerVersionOptions holds configuration for a CreateMCPServerVersion call.
type createServerVersionOptions struct {
	displayName    string
	status         MCPServerVersionStatus
	source         string
	tools          []MCPTool
	connectOptions map[string]ConnectOptionSettings
}

// CreateMCPServerVersionOption configures a CreateMCPServerVersion call.
type CreateMCPServerVersionOption func(*createServerVersionOptions)

// WithVersionDisplayName sets the display name for a new server version.
func WithVersionDisplayName(name string) CreateMCPServerVersionOption {
	return func(o *createServerVersionOptions) {
		o.displayName = name
	}
}

// WithVersionStatus sets the lifecycle status for a new server version.
// Default: MCPServerVersionStatusDraft.
func WithVersionStatus(status MCPServerVersionStatus) CreateMCPServerVersionOption {
	return func(o *createServerVersionOptions) {
		o.status = status
	}
}

// WithVersionSource sets the source metadata for a new server version.
func WithVersionSource(source string) CreateMCPServerVersionOption {
	return func(o *createServerVersionOptions) {
		o.source = source
	}
}

// WithVersionTools explicitly sets the tools for a new server version.
// If not set, MLflow auto-discovers tools from the server.
func WithVersionTools(tools []MCPTool) CreateMCPServerVersionOption {
	return func(o *createServerVersionOptions) {
		o.tools = tools
	}
}

// WithVersionConnectOptions sets per-connection-mode display settings (e.g.
// hiding a package registry or remote transport) for a new server version.
func WithVersionConnectOptions(connectOptions map[string]ConnectOptionSettings) CreateMCPServerVersionOption {
	return func(o *createServerVersionOptions) {
		o.connectOptions = connectOptions
	}
}

// updateServerVersionOptions holds configuration for an
// UpdateMCPServerVersion call. Pointer fields distinguish "not provided"
// (nil, left unchanged) from "provided" (updated to the given value, even
// if zero/empty).
type updateServerVersionOptions struct {
	status         *MCPServerVersionStatus
	tools          *[]MCPTool
	connectOptions *map[string]ConnectOptionSettings
}

// UpdateMCPServerVersionOption configures an UpdateMCPServerVersion call.
type UpdateMCPServerVersionOption func(*updateServerVersionOptions)

// WithUpdatedVersionStatus transitions a server version to a new lifecycle status.
func WithUpdatedVersionStatus(status MCPServerVersionStatus) UpdateMCPServerVersionOption {
	return func(o *updateServerVersionOptions) {
		o.status = &status
	}
}

// WithUpdatedVersionTools replaces the tools recorded for a server version.
// Pass an empty slice to clear them.
func WithUpdatedVersionTools(tools []MCPTool) UpdateMCPServerVersionOption {
	return func(o *updateServerVersionOptions) {
		o.tools = &tools
	}
}

// WithUpdatedVersionConnectOptions replaces the per-connection-mode display
// settings recorded for a server version. Pass an empty map to clear them.
func WithUpdatedVersionConnectOptions(connectOptions map[string]ConnectOptionSettings) UpdateMCPServerVersionOption {
	return func(o *updateServerVersionOptions) {
		o.connectOptions = &connectOptions
	}
}

// searchServerVersionsOptions holds configuration for a SearchMCPServerVersions call.
type searchServerVersionsOptions struct {
	filter     string
	maxResults int
	pageToken  string
	orderBy    []string
}

// SearchMCPServerVersionsOption configures a SearchMCPServerVersions call.
type SearchMCPServerVersionsOption func(*searchServerVersionsOptions)

// WithVersionsFilter sets the search filter string for server versions.
func WithVersionsFilter(filter string) SearchMCPServerVersionsOption {
	return func(o *searchServerVersionsOptions) {
		o.filter = filter
	}
}

// WithVersionsMaxResults sets the maximum number of versions to return per page.
func WithVersionsMaxResults(n int) SearchMCPServerVersionsOption {
	return func(o *searchServerVersionsOptions) {
		o.maxResults = n
	}
}

// WithVersionsPageToken sets the pagination token for fetching the next page.
func WithVersionsPageToken(token string) SearchMCPServerVersionsOption {
	return func(o *searchServerVersionsOptions) {
		o.pageToken = token
	}
}

// WithVersionsOrderBy sets the sort order for the version search results.
func WithVersionsOrderBy(fields ...string) SearchMCPServerVersionsOption {
	return func(o *searchServerVersionsOptions) {
		o.orderBy = fields
	}
}

// createAccessEndpointOptions holds configuration for a CreateMCPAccessEndpoint call.
type createAccessEndpointOptions struct {
	transportType MCPTransportType
	serverVersion string
	serverAlias   string
}

// CreateMCPAccessEndpointOption configures a CreateMCPAccessEndpoint call.
type CreateMCPAccessEndpointOption func(*createAccessEndpointOptions)

// WithAccessEndpointTransportType sets the transport type for a new access endpoint.
// Default: MCPTransportStreamableHTTP.
func WithAccessEndpointTransportType(transportType MCPTransportType) CreateMCPAccessEndpointOption {
	return func(o *createAccessEndpointOptions) {
		o.transportType = transportType
	}
}

// WithAccessEndpointServerVersion pins a new access endpoint to a specific server version.
// Mutually exclusive with WithAccessEndpointServerAlias.
func WithAccessEndpointServerVersion(version string) CreateMCPAccessEndpointOption {
	return func(o *createAccessEndpointOptions) {
		o.serverVersion = version
	}
}

// WithAccessEndpointServerAlias pins a new access endpoint to a server alias (e.g. "production").
// Mutually exclusive with WithAccessEndpointServerVersion.
func WithAccessEndpointServerAlias(alias string) CreateMCPAccessEndpointOption {
	return func(o *createAccessEndpointOptions) {
		o.serverAlias = alias
	}
}

// updateAccessEndpointOptions holds configuration for an
// UpdateMCPAccessEndpoint call. Pointer fields distinguish "not provided"
// (nil, left unchanged) from "provided" (updated to the given value, even
// if zero/empty).
type updateAccessEndpointOptions struct {
	serverVersion *string
	serverAlias   *string
	url           *string
	transportType *MCPTransportType
}

// UpdateMCPAccessEndpointOption configures an UpdateMCPAccessEndpoint call.
type UpdateMCPAccessEndpointOption func(*updateAccessEndpointOptions)

// WithUpdatedEndpointURL updates the URL of an access endpoint.
func WithUpdatedEndpointURL(url string) UpdateMCPAccessEndpointOption {
	return func(o *updateAccessEndpointOptions) {
		o.url = &url
	}
}

// WithUpdatedEndpointTransportType updates the transport type of an access endpoint.
func WithUpdatedEndpointTransportType(transportType MCPTransportType) UpdateMCPAccessEndpointOption {
	return func(o *updateAccessEndpointOptions) {
		o.transportType = &transportType
	}
}

// WithUpdatedEndpointServerVersion re-pins an access endpoint to a specific server version.
// Mutually exclusive with WithUpdatedEndpointServerAlias.
func WithUpdatedEndpointServerVersion(version string) UpdateMCPAccessEndpointOption {
	return func(o *updateAccessEndpointOptions) {
		o.serverVersion = &version
	}
}

// WithUpdatedEndpointServerAlias re-pins an access endpoint to a server alias (e.g. "production").
// Mutually exclusive with WithUpdatedEndpointServerVersion.
func WithUpdatedEndpointServerAlias(alias string) UpdateMCPAccessEndpointOption {
	return func(o *updateAccessEndpointOptions) {
		o.serverAlias = &alias
	}
}

// searchAccessEndpointsOptions holds configuration for a SearchMCPAccessEndpoints call.
type searchAccessEndpointsOptions struct {
	serverName    string
	filter        string
	maxResults    int
	pageToken     string
	orderBy       []string
	serverVersion string
	serverAlias   string
}

// SearchMCPAccessEndpointsOption configures a SearchMCPAccessEndpoints call.
type SearchMCPAccessEndpointsOption func(*searchAccessEndpointsOptions)

// WithAccessEndpointsServerName scopes the search to access endpoints belonging
// to a single server. If not set, the search spans all servers.
func WithAccessEndpointsServerName(name string) SearchMCPAccessEndpointsOption {
	return func(o *searchAccessEndpointsOptions) {
		o.serverName = name
	}
}

// WithAccessEndpointsFilter sets the search filter string for access endpoints.
func WithAccessEndpointsFilter(filter string) SearchMCPAccessEndpointsOption {
	return func(o *searchAccessEndpointsOptions) {
		o.filter = filter
	}
}

// WithAccessEndpointsMaxResults sets the maximum number of endpoints to return per page.
func WithAccessEndpointsMaxResults(n int) SearchMCPAccessEndpointsOption {
	return func(o *searchAccessEndpointsOptions) {
		o.maxResults = n
	}
}

// WithAccessEndpointsPageToken sets the pagination token for fetching the next page.
func WithAccessEndpointsPageToken(token string) SearchMCPAccessEndpointsOption {
	return func(o *searchAccessEndpointsOptions) {
		o.pageToken = token
	}
}

// WithAccessEndpointsOrderBy sets the sort order for the access endpoint search results.
func WithAccessEndpointsOrderBy(fields ...string) SearchMCPAccessEndpointsOption {
	return func(o *searchAccessEndpointsOptions) {
		o.orderBy = fields
	}
}

// WithAccessEndpointsServerVersion filters results to endpoints pinned to a specific version.
func WithAccessEndpointsServerVersion(version string) SearchMCPAccessEndpointsOption {
	return func(o *searchAccessEndpointsOptions) {
		o.serverVersion = version
	}
}

// WithAccessEndpointsServerAlias filters results to endpoints pinned to a specific alias.
func WithAccessEndpointsServerAlias(alias string) SearchMCPAccessEndpointsOption {
	return func(o *searchAccessEndpointsOptions) {
		o.serverAlias = alias
	}
}
