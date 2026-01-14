// ABOUTME: Tests for functional options.
// ABOUTME: Verifies option constructors correctly set configuration values.

package mlflow

import (
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestWithTrackingURI(t *testing.T) {
	opts := &options{}
	WithTrackingURI("https://mlflow.example.com")(opts)

	if opts.trackingURI != "https://mlflow.example.com" {
		t.Errorf("trackingURI = %q, want %q", opts.trackingURI, "https://mlflow.example.com")
	}
}

func TestWithToken(t *testing.T) {
	opts := &options{}
	WithToken("my-secret-token")(opts)

	if opts.token != "my-secret-token" {
		t.Errorf("token = %q, want %q", opts.token, "my-secret-token")
	}
}

func TestWithHTTPClient(t *testing.T) {
	customClient := &http.Client{Timeout: 5 * time.Second}
	opts := &options{}
	WithHTTPClient(customClient)(opts)

	if opts.httpClient != customClient {
		t.Error("httpClient not set correctly")
	}
}

func TestWithLogger(t *testing.T) {
	handler := slog.NewTextHandler(os.Stdout, nil)
	opts := &options{}
	WithLogger(handler)(opts)

	if opts.logger == nil {
		t.Error("logger should not be nil")
	}
}

func TestWithLogger_Nil(t *testing.T) {
	opts := &options{}
	WithLogger(nil)(opts)

	if opts.logger != nil {
		t.Error("logger should be nil when nil handler provided")
	}
}

func TestWithInsecure(t *testing.T) {
	opts := &options{}
	if opts.insecure {
		t.Error("insecure should be false by default")
	}

	WithInsecure()(opts)

	if !opts.insecure {
		t.Error("insecure should be true after WithInsecure()")
	}
}

func TestWithTimeout(t *testing.T) {
	opts := &options{}
	WithTimeout(60 * time.Second)(opts)

	if opts.timeout != 60*time.Second {
		t.Errorf("timeout = %v, want %v", opts.timeout, 60*time.Second)
	}
}

func TestWithVersion(t *testing.T) {
	opts := &loadOptions{}
	WithVersion(5)(opts)

	if opts.version != 5 {
		t.Errorf("version = %d, want %d", opts.version, 5)
	}
}

func TestWithDescription(t *testing.T) {
	opts := &registerOptions{}
	WithDescription("Initial version")(opts)

	if opts.description != "Initial version" {
		t.Errorf("description = %q, want %q", opts.description, "Initial version")
	}
}

func TestWithTags(t *testing.T) {
	opts := &registerOptions{}
	tags := map[string]string{"team": "ml", "env": "prod"}
	WithTags(tags)(opts)

	if len(opts.tags) != 2 {
		t.Errorf("tags length = %d, want %d", len(opts.tags), 2)
	}
	if opts.tags["team"] != "ml" {
		t.Errorf("tags[team] = %q, want %q", opts.tags["team"], "ml")
	}
}

func TestMultipleOptions(t *testing.T) {
	opts := &options{}

	// Apply multiple options
	for _, opt := range []Option{
		WithTrackingURI("https://example.com"),
		WithToken("token123"),
		WithInsecure(),
		WithTimeout(45 * time.Second),
	} {
		opt(opts)
	}

	if opts.trackingURI != "https://example.com" {
		t.Errorf("trackingURI = %q", opts.trackingURI)
	}
	if opts.token != "token123" {
		t.Errorf("token = %q", opts.token)
	}
	if !opts.insecure {
		t.Error("insecure should be true")
	}
	if opts.timeout != 45*time.Second {
		t.Errorf("timeout = %v", opts.timeout)
	}
}
