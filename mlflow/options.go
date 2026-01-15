// ABOUTME: Defines functional options for configuring the SDK client and operations.
// ABOUTME: Follows the functional options pattern used by AWS, Google Cloud, and Stripe Go SDKs.

package mlflow

import (
	"log/slog"
	"net/http"
	"time"
)

// options holds the configuration for a Client.
type options struct {
	trackingURI string
	token       string
	httpClient  *http.Client
	logger      *slog.Logger
	insecure    bool
	timeout     time.Duration
}

// Option configures a Client.
type Option func(*options)

// WithTrackingURI sets the MLflow server URL.
// Overrides MLFLOW_TRACKING_URI environment variable.
func WithTrackingURI(uri string) Option {
	return func(o *options) {
		o.trackingURI = uri
	}
}

// WithToken sets the authentication token.
// Overrides MLFLOW_TRACKING_TOKEN environment variable.
func WithToken(token string) Option {
	return func(o *options) {
		o.token = token
	}
}

// WithHTTPClient sets a custom HTTP client.
// Use this to configure timeouts, TLS, or proxies.
// When a custom client is provided, WithTimeout is ignored;
// configure the timeout directly on the provided client.
func WithHTTPClient(client *http.Client) Option {
	return func(o *options) {
		o.httpClient = client
	}
}

// WithLogger sets a structured logger for debug output.
// If not set, the SDK is silent.
func WithLogger(handler slog.Handler) Option {
	return func(o *options) {
		if handler != nil {
			o.logger = slog.New(handler)
		}
	}
}

// WithInsecure allows HTTP connections (not recommended for production).
// Overrides MLFLOW_INSECURE_SKIP_TLS_VERIFY environment variable.
func WithInsecure() Option {
	return func(o *options) {
		o.insecure = true
	}
}

// WithTimeout sets the default timeout for API operations.
// Default: 30 seconds.
func WithTimeout(d time.Duration) Option {
	return func(o *options) {
		o.timeout = d
	}
}

// loadOptions holds the configuration for a LoadPrompt call.
type loadOptions struct {
	version int
}

// LoadOption configures a LoadPrompt call.
type LoadOption func(*loadOptions)

// WithVersion specifies the version to load.
// If not set, loads the latest version.
func WithVersion(version int) LoadOption {
	return func(o *loadOptions) {
		o.version = version
	}
}

// registerOptions holds the configuration for a RegisterPrompt call.
type registerOptions struct {
	description string
	tags        map[string]string
}

// RegisterOption configures a RegisterPrompt call.
type RegisterOption func(*registerOptions)

// WithDescription sets the version description.
func WithDescription(description string) RegisterOption {
	return func(o *registerOptions) {
		o.description = description
	}
}

// WithTags sets metadata tags for the version.
func WithTags(tags map[string]string) RegisterOption {
	return func(o *registerOptions) {
		o.tags = tags
	}
}

// listPromptsOptions holds the configuration for a ListPrompts call.
type listPromptsOptions struct {
	maxResults int
	pageToken  string
	nameFilter string
	tagFilter  map[string]string
	orderBy    []string
}

// ListPromptsOption configures a ListPrompts call.
type ListPromptsOption func(*listPromptsOptions)

// WithMaxResults sets the maximum number of prompts to return per page.
// Default: 100. Maximum: 1000.
func WithMaxResults(n int) ListPromptsOption {
	return func(o *listPromptsOptions) {
		o.maxResults = n
	}
}

// WithPageToken sets the pagination token for fetching the next page.
func WithPageToken(token string) ListPromptsOption {
	return func(o *listPromptsOptions) {
		o.pageToken = token
	}
}

// WithNameFilter filters prompts by name pattern.
// Uses SQL LIKE syntax (e.g., "greeting%" matches names starting with "greeting").
func WithNameFilter(pattern string) ListPromptsOption {
	return func(o *listPromptsOptions) {
		o.nameFilter = pattern
	}
}

// WithTagFilter filters prompts by tag values.
// All specified tags must match (AND logic).
func WithTagFilter(tags map[string]string) ListPromptsOption {
	return func(o *listPromptsOptions) {
		o.tagFilter = tags
	}
}

// WithOrderBy sets the sort order for results.
// Examples: "name ASC", "creation_timestamp DESC".
func WithOrderBy(fields ...string) ListPromptsOption {
	return func(o *listPromptsOptions) {
		o.orderBy = fields
	}
}

// listVersionsOptions holds the configuration for a ListPromptVersions call.
type listVersionsOptions struct {
	maxResults int
	pageToken  string
	tagFilter  map[string]string
	orderBy    []string
}

// ListVersionsOption configures a ListPromptVersions call.
type ListVersionsOption func(*listVersionsOptions)

// WithVersionsMaxResults sets the maximum number of versions to return per page.
// Default: 100. Maximum: 1000.
func WithVersionsMaxResults(n int) ListVersionsOption {
	return func(o *listVersionsOptions) {
		o.maxResults = n
	}
}

// WithVersionsPageToken sets the pagination token for fetching the next page.
func WithVersionsPageToken(token string) ListVersionsOption {
	return func(o *listVersionsOptions) {
		o.pageToken = token
	}
}

// WithVersionsTagFilter filters versions by tag values.
// All specified tags must match (AND logic).
func WithVersionsTagFilter(tags map[string]string) ListVersionsOption {
	return func(o *listVersionsOptions) {
		o.tagFilter = tags
	}
}

// WithVersionsOrderBy sets the sort order for results.
// Default: "version_number DESC" (newest first).
func WithVersionsOrderBy(fields ...string) ListVersionsOption {
	return func(o *listVersionsOptions) {
		o.orderBy = fields
	}
}
