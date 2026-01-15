// ABOUTME: Tests for the main SDK client.
// ABOUTME: Verifies client initialization from options and environment variables.

package mlflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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
	// Clear env var to ensure it's not set
	os.Unsetenv("MLFLOW_TRACKING_URI")

	_, err := NewClient()
	if err == nil {
		t.Error("expected error for missing tracking URI")
	}
}

func TestNewClient_FromEnvVar(t *testing.T) {
	os.Setenv("MLFLOW_TRACKING_URI", "https://mlflow.test.com")
	defer os.Unsetenv("MLFLOW_TRACKING_URI")

	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if client.TrackingURI() != "https://mlflow.test.com" {
		t.Errorf("TrackingURI() = %q, want %q", client.TrackingURI(), "https://mlflow.test.com")
	}
}

func TestNewClient_ExplicitOverridesEnv(t *testing.T) {
	os.Setenv("MLFLOW_TRACKING_URI", "https://env.example.com")
	defer os.Unsetenv("MLFLOW_TRACKING_URI")

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
	os.Setenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY", "true")
	defer os.Unsetenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY")

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
	os.Setenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY", "1")
	defer os.Unsetenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY")

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

func TestNewClient_TokenFromEnv(t *testing.T) {
	os.Setenv("MLFLOW_TRACKING_URI", "https://mlflow.example.com")
	os.Setenv("MLFLOW_TRACKING_TOKEN", "secret-token")
	defer func() {
		os.Unsetenv("MLFLOW_TRACKING_URI")
		os.Unsetenv("MLFLOW_TRACKING_TOKEN")
	}()

	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// We can't directly check the token, but verify client was created
	if client == nil {
		t.Error("client should not be nil")
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

func TestLoadPrompt_EmptyName(t *testing.T) {
	client, err := NewClient(
		WithTrackingURI("https://mlflow.example.com"),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.LoadPrompt(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestRegisterPrompt_EmptyName(t *testing.T) {
	client, err := NewClient(
		WithTrackingURI("https://mlflow.example.com"),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.RegisterPrompt(context.Background(), "", "template")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestRegisterPrompt_EmptyTemplate(t *testing.T) {
	client, err := NewClient(
		WithTrackingURI("https://mlflow.example.com"),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.RegisterPrompt(context.Background(), "my-prompt", "")
	if err == nil {
		t.Error("expected error for empty template")
	}
}

func TestLoadPrompt_Success(t *testing.T) {
	// Create test server that simulates MLflow API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/2.0/mlflow/registered-models/get":
			json.NewEncoder(w).Encode(map[string]any{
				"registered_model": map[string]any{
					"name": "test-prompt",
					"latest_versions": []map[string]any{
						{"version": "2"},
					},
				},
			})
		case "/api/2.0/mlflow/model-versions/get":
			json.NewEncoder(w).Encode(map[string]any{
				"model_version": map[string]any{
					"name":                   "test-prompt",
					"version":                "2",
					"description":            "A test prompt",
					"creation_timestamp":     1700000000000,
					"last_updated_timestamp": 1700000100000,
					"tags": []map[string]string{
						{"key": "mlflow.prompt.is_prompt", "value": "true"},
						{"key": "mlflow.prompt.text", "value": "Hello, {{name}}!"},
						{"key": "team", "value": "ml"},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(
		WithTrackingURI(server.URL),
		WithInsecure(),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	prompt, err := client.LoadPrompt(context.Background(), "test-prompt")
	if err != nil {
		t.Fatalf("LoadPrompt() error = %v", err)
	}

	if prompt.Name != "test-prompt" {
		t.Errorf("Name = %q, want %q", prompt.Name, "test-prompt")
	}
	if prompt.Version != 2 {
		t.Errorf("Version = %d, want %d", prompt.Version, 2)
	}
	if prompt.Template != "Hello, {{name}}!" {
		t.Errorf("Template = %q, want %q", prompt.Template, "Hello, {{name}}!")
	}
	if prompt.Description != "A test prompt" {
		t.Errorf("Description = %q, want %q", prompt.Description, "A test prompt")
	}
	if prompt.Tags["team"] != "ml" {
		t.Errorf("Tags[team] = %q, want %q", prompt.Tags["team"], "ml")
	}
	// Internal tags should not be exposed
	if _, ok := prompt.Tags["mlflow.prompt.is_prompt"]; ok {
		t.Error("internal tag should not be in user tags")
	}
}

func TestLoadPrompt_WithVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path != "/api/2.0/mlflow/model-versions/get" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		version := r.URL.Query().Get("version")
		if version != "3" {
			t.Errorf("version = %q, want %q", version, "3")
		}

		json.NewEncoder(w).Encode(map[string]any{
			"model_version": map[string]any{
				"name":    "test-prompt",
				"version": "3",
				"tags": []map[string]string{
					{"key": "mlflow.prompt.text", "value": "Version 3 template"},
				},
			},
		})
	}))
	defer server.Close()

	client, err := NewClient(
		WithTrackingURI(server.URL),
		WithInsecure(),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	prompt, err := client.LoadPrompt(context.Background(), "test-prompt", WithVersion(3))
	if err != nil {
		t.Fatalf("LoadPrompt() error = %v", err)
	}

	if prompt.Version != 3 {
		t.Errorf("Version = %d, want %d", prompt.Version, 3)
	}
	if prompt.Template != "Version 3 template" {
		t.Errorf("Template = %q, want %q", prompt.Template, "Version 3 template")
	}
}

func TestLoadPrompt_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error_code": "RESOURCE_DOES_NOT_EXIST",
			"message":    "Registered Model with name=unknown not found",
		})
	}))
	defer server.Close()

	client, err := NewClient(
		WithTrackingURI(server.URL),
		WithInsecure(),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.LoadPrompt(context.Background(), "unknown")
	if err == nil {
		t.Error("expected error for non-existent prompt")
	}
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound, got %v", err)
	}
}

func TestLoadPrompt_NoVersions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"registered_model": map[string]any{
				"name":            "empty-prompt",
				"latest_versions": []any{},
			},
		})
	}))
	defer server.Close()

	client, err := NewClient(
		WithTrackingURI(server.URL),
		WithInsecure(),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.LoadPrompt(context.Background(), "empty-prompt")
	if err == nil {
		t.Error("expected error for prompt with no versions")
	}
	if err.Error() != `prompt "empty-prompt" has no versions` {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRegisterPrompt_NewPrompt(t *testing.T) {
	var createModelCalled, createVersionCalled bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/2.0/mlflow/registered-models/create":
			createModelCalled = true
			json.NewEncoder(w).Encode(map[string]any{
				"registered_model": map[string]any{
					"name": "new-prompt",
				},
			})
		case "/api/2.0/mlflow/model-versions/create":
			createVersionCalled = true
			json.NewEncoder(w).Encode(map[string]any{
				"model_version": map[string]any{
					"name":                   "new-prompt",
					"version":                "1",
					"description":            "First version",
					"creation_timestamp":     1700000000000,
					"last_updated_timestamp": 1700000000000,
					"tags": []map[string]string{
						{"key": "mlflow.prompt.text", "value": "Hello, {{name}}!"},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(
		WithTrackingURI(server.URL),
		WithInsecure(),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	prompt, err := client.RegisterPrompt(
		context.Background(),
		"new-prompt",
		"Hello, {{name}}!",
		WithDescription("First version"),
	)
	if err != nil {
		t.Fatalf("RegisterPrompt() error = %v", err)
	}

	if !createModelCalled {
		t.Error("expected registered-models/create to be called")
	}
	if !createVersionCalled {
		t.Error("expected model-versions/create to be called")
	}
	if prompt.Name != "new-prompt" {
		t.Errorf("Name = %q, want %q", prompt.Name, "new-prompt")
	}
	if prompt.Version != 1 {
		t.Errorf("Version = %d, want %d", prompt.Version, 1)
	}
	if prompt.Template != "Hello, {{name}}!" {
		t.Errorf("Template = %q, want %q", prompt.Template, "Hello, {{name}}!")
	}
}

func TestRegisterPrompt_ExistingPrompt(t *testing.T) {
	var createVersionCalled bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/2.0/mlflow/registered-models/create":
			// Prompt already exists - return 409
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{
				"error_code": "RESOURCE_ALREADY_EXISTS",
				"message":    "Registered Model 'existing-prompt' already exists",
			})
		case "/api/2.0/mlflow/model-versions/create":
			createVersionCalled = true
			json.NewEncoder(w).Encode(map[string]any{
				"model_version": map[string]any{
					"name":    "existing-prompt",
					"version": "2",
					"tags": []map[string]string{
						{"key": "mlflow.prompt.text", "value": "Updated template"},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(
		WithTrackingURI(server.URL),
		WithInsecure(),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	prompt, err := client.RegisterPrompt(
		context.Background(),
		"existing-prompt",
		"Updated template",
	)
	if err != nil {
		t.Fatalf("RegisterPrompt() error = %v", err)
	}

	if !createVersionCalled {
		t.Error("expected model-versions/create to be called")
	}
	if prompt.Version != 2 {
		t.Errorf("Version = %d, want %d", prompt.Version, 2)
	}
}

func TestRegisterPrompt_WithTags(t *testing.T) {
	var receivedTags []map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/2.0/mlflow/registered-models/create":
			json.NewEncoder(w).Encode(map[string]any{
				"registered_model": map[string]any{"name": "tagged-prompt"},
			})
		case "/api/2.0/mlflow/model-versions/create":
			var req struct {
				Tags []map[string]string `json:"tags"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			receivedTags = req.Tags

			json.NewEncoder(w).Encode(map[string]any{
				"model_version": map[string]any{
					"name":    "tagged-prompt",
					"version": "1",
					"tags":    req.Tags,
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(
		WithTrackingURI(server.URL),
		WithInsecure(),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.RegisterPrompt(
		context.Background(),
		"tagged-prompt",
		"Template",
		WithTags(map[string]string{"team": "ml", "env": "prod"}),
	)
	if err != nil {
		t.Fatalf("RegisterPrompt() error = %v", err)
	}

	// Check that user tags were included
	foundTeam := false
	foundEnv := false
	for _, tag := range receivedTags {
		if tag["key"] == "team" && tag["value"] == "ml" {
			foundTeam = true
		}
		if tag["key"] == "env" && tag["value"] == "prod" {
			foundEnv = true
		}
	}
	if !foundTeam {
		t.Error("expected team tag to be sent")
	}
	if !foundEnv {
		t.Error("expected env tag to be sent")
	}
}

func TestRegisterPrompt_PermissionDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"error_code": "PERMISSION_DENIED",
			"message":    "User lacks permission to create prompts",
		})
	}))
	defer server.Close()

	client, err := NewClient(
		WithTrackingURI(server.URL),
		WithInsecure(),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.RegisterPrompt(context.Background(), "new-prompt", "Template")
	if err == nil {
		t.Error("expected error for permission denied")
	}
	if !IsPermissionDenied(err) {
		t.Errorf("expected IsPermissionDenied, got %v", err)
	}
}
