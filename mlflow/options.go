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
