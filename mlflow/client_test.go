package mlflow

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNewClient_WithTrackingURI(t *testing.T) {
	client, err := NewClient(
		WithTrackingURI("https://mlflow.example.com"),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if client.TrackingURI() != "https://mlflow.example.com" {
		t.Errorf("TrackingURI() = %q, want %q", client.TrackingURI(), "https://mlflow.example.com")
	}
}

func TestNewClient_MissingTrackingURI(t *testing.T) {
	// Save and restore env var
	saved := os.Getenv("MLFLOW_TRACKING_URI")
	os.Unsetenv("MLFLOW_TRACKING_URI")
	defer func() {
		if saved != "" {
			os.Setenv("MLFLOW_TRACKING_URI", saved)
		}
	}()

	_, err := NewClient()
	if err == nil {
		t.Error("expected error for missing tracking URI")
	}
}

func TestNewClient_FromEnvVar(t *testing.T) {
	saved := os.Getenv("MLFLOW_TRACKING_URI")
	os.Setenv("MLFLOW_TRACKING_URI", "https://mlflow.test.com")
	defer func() {
		if saved != "" {
			os.Setenv("MLFLOW_TRACKING_URI", saved)
		} else {
			os.Unsetenv("MLFLOW_TRACKING_URI")
		}
	}()

	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if client.TrackingURI() != "https://mlflow.test.com" {
		t.Errorf("TrackingURI() = %q, want %q", client.TrackingURI(), "https://mlflow.test.com")
	}
}

func TestNewClient_ExplicitOverridesEnv(t *testing.T) {
	saved := os.Getenv("MLFLOW_TRACKING_URI")
	os.Setenv("MLFLOW_TRACKING_URI", "https://env.example.com")
	defer func() {
		if saved != "" {
			os.Setenv("MLFLOW_TRACKING_URI", saved)
		} else {
			os.Unsetenv("MLFLOW_TRACKING_URI")
		}
	}()

	client, err := NewClient(
		WithTrackingURI("https://explicit.example.com"),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Explicit option should take precedence over env var
	if client.TrackingURI() != "https://explicit.example.com" {
		t.Errorf("TrackingURI() = %q, want %q", client.TrackingURI(), "https://explicit.example.com")
	}
}

func TestNewClient_HTTPRejectedByDefault(t *testing.T) {
	// Save and restore insecure env var
	savedInsecure := os.Getenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY")
	os.Unsetenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY")
	defer func() {
		if savedInsecure != "" {
			os.Setenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY", savedInsecure)
		}
	}()

	_, err := NewClient(
		WithTrackingURI("http://mlflow.example.com"),
	)
	if err == nil {
		t.Error("expected error for HTTP URI without insecure mode")
	}
}

func TestNewClient_HTTPAllowedWithInsecure(t *testing.T) {
	client, err := NewClient(
		WithTrackingURI("http://localhost:5000"),
		WithInsecure(),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if !client.IsInsecure() {
		t.Error("IsInsecure() should be true")
	}
}

func TestNewClient_HTTPAllowedWithEnvVar(t *testing.T) {
	saved := os.Getenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY")
	os.Setenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY", "true")
	defer func() {
		if saved != "" {
			os.Setenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY", saved)
		} else {
			os.Unsetenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY")
		}
	}()

	client, err := NewClient(
		WithTrackingURI("http://localhost:5000"),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if !client.IsInsecure() {
		t.Error("IsInsecure() should be true")
	}
}

func TestNewClient_InsecureEnvVar_One(t *testing.T) {
	saved := os.Getenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY")
	os.Setenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY", "1")
	defer func() {
		if saved != "" {
			os.Setenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY", saved)
		} else {
			os.Unsetenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY")
		}
	}()

	client, err := NewClient(
		WithTrackingURI("http://localhost:5000"),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if !client.IsInsecure() {
		t.Error("IsInsecure() should be true for '1'")
	}
}

func TestNewClient_BareHostPort(t *testing.T) {
	client, err := NewClient(
		WithTrackingURI("mlflow.example.com:5000"),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if client.TrackingURI() != "https://mlflow.example.com:5000" {
		t.Errorf("TrackingURI() = %q, want %q", client.TrackingURI(), "https://mlflow.example.com:5000")
	}
}

func TestNewClient_BareHostOnly(t *testing.T) {
	client, err := NewClient(
		WithTrackingURI("mlflow.example.com"),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if client.TrackingURI() != "https://mlflow.example.com" {
		t.Errorf("TrackingURI() = %q, want %q", client.TrackingURI(), "https://mlflow.example.com")
	}
}

func TestNewClient_InvalidURI(t *testing.T) {
	_, err := NewClient(
		WithTrackingURI("://invalid"),
	)
	if err == nil {
		t.Error("expected error for invalid URI")
	}
}

func TestClient_PromptRegistry_ReturnsSameInstance(t *testing.T) {
	client, err := NewClient(
		WithTrackingURI("https://mlflow.example.com"),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	pr1 := client.PromptRegistry()
	pr2 := client.PromptRegistry()

	if pr1 != pr2 {
		t.Error("PromptRegistry() should return same instance")
	}
}

func TestClient_Artifacts_ReturnsSameInstance(t *testing.T) {
	client, err := NewClient(
		WithTrackingURI("https://mlflow.example.com"),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	a1 := client.Artifacts()
	a2 := client.Artifacts()

	if a1 != a2 {
		t.Error("Artifacts() should return same instance")
	}
}

func TestClient_MCPRegistry_ReturnsSameInstance(t *testing.T) {
	client, err := NewClient(
		WithTrackingURI("https://mlflow.example.com"),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	r1 := client.MCPRegistry()
	r2 := client.MCPRegistry()

	if r1 != r2 {
		t.Error("MCPRegistry() should return same instance")
	}
}

func TestNewClient_WithToken(t *testing.T) {
	_, err := NewClient(
		WithTrackingURI("https://mlflow.example.com"),
		WithToken("my-secret"),
	)
	if err != nil {
		t.Fatalf("NewClient(WithToken) error = %v", err)
	}
}

func TestNewClient_WithTokenPath(t *testing.T) {
	tokenFile := t.TempDir() + "/token"
	if err := os.WriteFile(tokenFile, []byte("sa-token"), 0600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	_, err := NewClient(
		WithTrackingURI("https://mlflow.example.com"),
		WithTokenPath(tokenFile),
	)
	if err != nil {
		t.Fatalf("NewClient(WithTokenPath) error = %v", err)
	}
}

func TestNewClient_TokenFromEnvVar(t *testing.T) {
	saved := os.Getenv("MLFLOW_TRACKING_TOKEN")
	os.Setenv("MLFLOW_TRACKING_TOKEN", "env-token")
	defer func() {
		if saved != "" {
			os.Setenv("MLFLOW_TRACKING_TOKEN", saved)
		} else {
			os.Unsetenv("MLFLOW_TRACKING_TOKEN")
		}
	}()

	_, err := NewClient(
		WithTrackingURI("https://mlflow.example.com"),
	)
	if err != nil {
		t.Fatalf("NewClient(token from env) error = %v", err)
	}
}

func TestNewClient_ExplicitTokenOverridesEnv(t *testing.T) {
	saved := os.Getenv("MLFLOW_TRACKING_TOKEN")
	os.Setenv("MLFLOW_TRACKING_TOKEN", "env-token")
	defer func() {
		if saved != "" {
			os.Setenv("MLFLOW_TRACKING_TOKEN", saved)
		} else {
			os.Unsetenv("MLFLOW_TRACKING_TOKEN")
		}
	}()

	// Explicit WithToken should prevent env var from being used.
	client, err := NewClient(
		WithTrackingURI("https://mlflow.example.com"),
		WithToken("explicit-token"),
	)
	if err != nil {
		t.Fatalf("NewClient(explicit token) error = %v", err)
	}
	if client.opts.token != "explicit-token" {
		t.Errorf("opts.token = %q, want %q", client.opts.token, "explicit-token")
	}
}

func TestNewClient_EmptyTokenSuppressesEnv(t *testing.T) {
	saved := os.Getenv("MLFLOW_TRACKING_TOKEN")
	os.Setenv("MLFLOW_TRACKING_TOKEN", "env-token")
	defer func() {
		if saved != "" {
			os.Setenv("MLFLOW_TRACKING_TOKEN", saved)
		} else {
			os.Unsetenv("MLFLOW_TRACKING_TOKEN")
		}
	}()

	client, err := NewClient(
		WithTrackingURI("https://mlflow.example.com"),
		WithToken(""),
	)
	if err != nil {
		t.Fatalf("NewClient(WithToken empty) error = %v", err)
	}
	if client.opts.token != "" {
		t.Errorf("opts.token = %q, want empty (env should be suppressed by explicit empty WithToken)", client.opts.token)
	}
}

func TestNewClient_EmptyTokenPathSuppressesEnv(t *testing.T) {
	saved := os.Getenv("MLFLOW_TRACKING_TOKEN")
	os.Setenv("MLFLOW_TRACKING_TOKEN", "env-token")
	defer func() {
		if saved != "" {
			os.Setenv("MLFLOW_TRACKING_TOKEN", saved)
		} else {
			os.Unsetenv("MLFLOW_TRACKING_TOKEN")
		}
	}()

	client, err := NewClient(
		WithTrackingURI("https://mlflow.example.com"),
		WithTokenPath(""),
	)
	if err != nil {
		t.Fatalf("NewClient(WithTokenPath empty) error = %v", err)
	}
	if client.opts.token != "" {
		t.Errorf("opts.token = %q, want empty (env should be suppressed by explicit empty WithTokenPath)", client.opts.token)
	}
}

// versionHandler returns a handler that records the Authorization header
// and responds with a valid /version body.
func versionHandler(t *testing.T, gotAuth *string) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("2.18.0"))
	})
}

func TestClient_TokenEnvSentOnRequest(t *testing.T) {
	var gotAuth string
	srv := httptest.NewTLSServer(versionHandler(t, &gotAuth))
	defer srv.Close()

	saved := os.Getenv("MLFLOW_TRACKING_TOKEN")
	os.Setenv("MLFLOW_TRACKING_TOKEN", "env-token")
	defer func() {
		if saved != "" {
			os.Setenv("MLFLOW_TRACKING_TOKEN", saved)
		} else {
			os.Unsetenv("MLFLOW_TRACKING_TOKEN")
		}
	}()

	client, err := NewClient(
		WithTrackingURI(srv.URL),
		WithHTTPClient(srv.Client()),
	)
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	if _, err := client.Tracking().GetVersion(context.Background()); err != nil {
		t.Fatalf("GetVersion error: %v", err)
	}
	if gotAuth != "Bearer env-token" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer env-token")
	}
}

func TestClient_ExplicitTokenOverridesEnvOnRequest(t *testing.T) {
	var gotAuth string
	srv := httptest.NewTLSServer(versionHandler(t, &gotAuth))
	defer srv.Close()

	saved := os.Getenv("MLFLOW_TRACKING_TOKEN")
	os.Setenv("MLFLOW_TRACKING_TOKEN", "env-token")
	defer func() {
		if saved != "" {
			os.Setenv("MLFLOW_TRACKING_TOKEN", saved)
		} else {
			os.Unsetenv("MLFLOW_TRACKING_TOKEN")
		}
	}()

	client, err := NewClient(
		WithTrackingURI(srv.URL),
		WithHTTPClient(srv.Client()),
		WithToken("explicit-token"),
	)
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	if _, err := client.Tracking().GetVersion(context.Background()); err != nil {
		t.Fatalf("GetVersion error: %v", err)
	}
	if gotAuth != "Bearer explicit-token" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer explicit-token")
	}
}

func TestClient_TokenPathSentOnRequest(t *testing.T) {
	var gotAuth string
	srv := httptest.NewTLSServer(versionHandler(t, &gotAuth))
	defer srv.Close()

	tokenFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenFile, []byte("file-token-v1"), 0600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	client, err := NewClient(
		WithTrackingURI(srv.URL),
		WithHTTPClient(srv.Client()),
		WithTokenPath(tokenFile),
	)
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	if _, err := client.Tracking().GetVersion(context.Background()); err != nil {
		t.Fatalf("GetVersion error: %v", err)
	}
	if gotAuth != "Bearer file-token-v1" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer file-token-v1")
	}

	// Rotate the token and verify the next request picks it up.
	if err := os.WriteFile(tokenFile, []byte("file-token-v2"), 0600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	if _, err := client.Tracking().GetVersion(context.Background()); err != nil {
		t.Fatalf("GetVersion (rotated) error: %v", err)
	}
	if gotAuth != "Bearer file-token-v2" {
		t.Errorf("Authorization after rotation = %q, want %q", gotAuth, "Bearer file-token-v2")
	}
}

func TestClient_EmptyTokenSuppressesEnvOnRequest(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(versionHandler(t, &gotAuth))
	defer srv.Close()

	saved := os.Getenv("MLFLOW_TRACKING_TOKEN")
	os.Setenv("MLFLOW_TRACKING_TOKEN", "env-token")
	defer func() {
		if saved != "" {
			os.Setenv("MLFLOW_TRACKING_TOKEN", saved)
		} else {
			os.Unsetenv("MLFLOW_TRACKING_TOKEN")
		}
	}()

	client, err := NewClient(
		WithTrackingURI(srv.URL),
		WithInsecure(),
		WithToken(""),
	)
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	if _, err := client.Tracking().GetVersion(context.Background()); err != nil {
		t.Fatalf("GetVersion error: %v", err)
	}
	if gotAuth != "" {
		t.Errorf("Authorization = %q, want empty (env should be suppressed)", gotAuth)
	}
}

func TestClient_ExplicitAuthHeaderNotClobberedByEnvToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("2.18.0"))
	}))
	defer srv.Close()

	t.Setenv("MLFLOW_TRACKING_TOKEN", "AMBIENT-SA-TOKEN")

	client, err := NewClient(
		WithTrackingURI(srv.URL),
		WithHTTPClient(srv.Client()),
		WithHeaders(map[string]string{"Authorization": "Bearer APP-SCOPED-TOKEN"}),
	)
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	if _, err := client.Tracking().GetVersion(context.Background()); err != nil {
		t.Fatalf("GetVersion error: %v", err)
	}
	if gotAuth != "Bearer APP-SCOPED-TOKEN" {
		t.Errorf("Authorization = %q, want %q (explicit header clobbered by ambient env token)", gotAuth, "Bearer APP-SCOPED-TOKEN")
	}
}
