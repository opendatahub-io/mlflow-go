package promptregistry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/opendatahub-io/mlflow-go/internal/errors"
	"github.com/opendatahub-io/mlflow-go/internal/transport"
)

func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	tc, err := transport.New(transport.Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("transport.New() error = %v", err)
	}

	return NewClient(tc)
}

func TestLoadPrompt_EmptyName(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	_, err := client.LoadPrompt(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestRegisterPrompt_EmptyName(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	_, err := client.RegisterPrompt(context.Background(), "", "template")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestRegisterPrompt_EmptyTemplate(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	_, err := client.RegisterPrompt(context.Background(), "my-prompt", "")
	if err == nil {
		t.Error("expected error for empty template")
	}
}

func TestLoadPrompt_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/2.0/mlflow/registered-models/alias":
			// Verify the "latest" alias is being requested
			if r.URL.Query().Get("alias") != "latest" {
				t.Errorf("expected alias=latest, got %s", r.URL.Query().Get("alias"))
			}
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
	if prompt.CommitMessage != "A test prompt" {
		t.Errorf("CommitMessage = %q, want %q", prompt.CommitMessage, "A test prompt")
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
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error_code": "RESOURCE_DOES_NOT_EXIST",
			"message":    "Registered Model with name=unknown not found",
		})
	}))

	_, err := client.LoadPrompt(context.Background(), "unknown")
	if err == nil {
		t.Error("expected error for non-existent prompt")
	}
	if !errors.IsNotFound(err) {
		t.Errorf("expected IsNotFound, got %v", err)
	}
}

func TestLoadPrompt_NoVersions(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error_code": "RESOURCE_DOES_NOT_EXIST",
			"message":    "Latest version not found for model empty-prompt.",
		})
	}))

	_, err := client.LoadPrompt(context.Background(), "empty-prompt")
	if err == nil {
		t.Error("expected error for prompt with no versions")
	}
}

func TestRegisterPrompt_NewPrompt(t *testing.T) {
	var createModelCalled, createVersionCalled bool

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	prompt, err := client.RegisterPrompt(
		context.Background(),
		"new-prompt",
		"Hello, {{name}}!",
		WithCommitMessage("First version"),
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

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	_, err := client.RegisterPrompt(
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
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"error_code": "PERMISSION_DENIED",
			"message":    "User lacks permission to create prompts",
		})
	}))

	_, err := client.RegisterPrompt(context.Background(), "new-prompt", "Template")
	if err == nil {
		t.Error("expected error for permission denied")
	}
	if !errors.IsPermissionDenied(err) {
		t.Errorf("expected IsPermissionDenied, got %v", err)
	}
}

func TestListPrompts_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path != "/api/2.0/mlflow/registered-models/search" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		// Verify filter includes is_prompt tag
		filter := r.URL.Query().Get("filter")
		if filter == "" || !strings.Contains(filter, "mlflow.prompt.is_prompt") {
			t.Errorf("filter should include is_prompt tag, got: %s", filter)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"registered_models": []map[string]any{
				{
					"name":                   "greeting-prompt",
					"description":            "A greeting prompt",
					"creation_timestamp":     1700000000000,
					"last_updated_timestamp": 1700000100000,
					"latest_versions": []map[string]any{
						{"version": "3"},
					},
					"tags": []map[string]string{
						{"key": "mlflow.prompt.is_prompt", "value": "true"},
						{"key": "team", "value": "ml"},
					},
				},
				{
					"name":        "qa-prompt",
					"description": "A QA prompt",
					"latest_versions": []map[string]any{
						{"version": "1"},
					},
					"tags": []map[string]string{
						{"key": "mlflow.prompt.is_prompt", "value": "true"},
					},
				},
			},
			"next_page_token": "token123",
		})
	}))

	result, err := client.ListPrompts(context.Background())
	if err != nil {
		t.Fatalf("ListPrompts() error = %v", err)
	}

	if len(result.Prompts) != 2 {
		t.Errorf("got %d prompts, want 2", len(result.Prompts))
	}
	if result.NextPageToken != "token123" {
		t.Errorf("NextPageToken = %q, want %q", result.NextPageToken, "token123")
	}

	// Verify first prompt
	p := result.Prompts[0]
	if p.Name != "greeting-prompt" {
		t.Errorf("Name = %q, want %q", p.Name, "greeting-prompt")
	}
	if p.LatestVersion != 3 {
		t.Errorf("LatestVersion = %d, want %d", p.LatestVersion, 3)
	}
	if p.Tags["team"] != "ml" {
		t.Errorf("Tags[team] = %q, want %q", p.Tags["team"], "ml")
	}
	// Internal tag should not be exposed
	if _, ok := p.Tags["mlflow.prompt.is_prompt"]; ok {
		t.Error("internal tag should not be in user tags")
	}
}

func TestListPrompts_WithNameFilter(t *testing.T) {
	var receivedFilter string

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		receivedFilter = r.URL.Query().Get("filter")
		json.NewEncoder(w).Encode(map[string]any{
			"registered_models": []map[string]any{},
		})
	}))

	_, err := client.ListPrompts(context.Background(), WithNameFilter("greeting%"))
	if err != nil {
		t.Fatalf("ListPrompts() error = %v", err)
	}

	if !strings.Contains(receivedFilter, "name LIKE 'greeting%'") {
		t.Errorf("filter should include name pattern, got: %s", receivedFilter)
	}
}

func TestListPrompts_WithPagination(t *testing.T) {
	var receivedPageToken string
	var receivedMaxResults string

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		receivedPageToken = r.URL.Query().Get("page_token")
		receivedMaxResults = r.URL.Query().Get("max_results")
		json.NewEncoder(w).Encode(map[string]any{
			"registered_models": []map[string]any{},
		})
	}))

	_, err := client.ListPrompts(context.Background(),
		WithPageToken("abc123"),
		WithMaxResults(50),
	)
	if err != nil {
		t.Fatalf("ListPrompts() error = %v", err)
	}

	if receivedPageToken != "abc123" {
		t.Errorf("page_token = %q, want %q", receivedPageToken, "abc123")
	}
	if receivedMaxResults != "50" {
		t.Errorf("max_results = %q, want %q", receivedMaxResults, "50")
	}
}

func TestListPrompts_Empty(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"registered_models": []map[string]any{},
		})
	}))

	result, err := client.ListPrompts(context.Background())
	if err != nil {
		t.Fatalf("ListPrompts() error = %v", err)
	}

	if result.Prompts == nil {
		t.Error("Prompts should not be nil, should be empty slice")
	}
	if len(result.Prompts) != 0 {
		t.Errorf("got %d prompts, want 0", len(result.Prompts))
	}
}

func TestListPrompts_WithTagFilter(t *testing.T) {
	var receivedFilter string

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		receivedFilter = r.URL.Query().Get("filter")
		json.NewEncoder(w).Encode(map[string]any{
			"registered_models": []map[string]any{},
		})
	}))

	_, err := client.ListPrompts(context.Background(),
		WithTagFilter(map[string]string{"team": "ml", "env": "prod"}),
	)
	if err != nil {
		t.Fatalf("ListPrompts() error = %v", err)
	}

	// Should include tag filters in the filter string
	if !strings.Contains(receivedFilter, "tags.`team` = 'ml'") {
		t.Errorf("filter should include team tag, got: %s", receivedFilter)
	}
	if !strings.Contains(receivedFilter, "tags.`env` = 'prod'") {
		t.Errorf("filter should include env tag, got: %s", receivedFilter)
	}
}

func TestListPrompts_WithOrderBy(t *testing.T) {
	var receivedOrderBy string

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		receivedOrderBy = r.URL.Query().Get("order_by")
		json.NewEncoder(w).Encode(map[string]any{
			"registered_models": []map[string]any{},
		})
	}))

	_, err := client.ListPrompts(context.Background(),
		WithOrderBy("name ASC", "timestamp DESC"),
	)
	if err != nil {
		t.Fatalf("ListPrompts() error = %v", err)
	}

	if !strings.Contains(receivedOrderBy, "name ASC") {
		t.Errorf("order_by should include 'name ASC', got: %s", receivedOrderBy)
	}
}

func TestListPromptVersions_WithMaxResults(t *testing.T) {
	var receivedMaxResults string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/2.0/mlflow/model-versions/search":
			receivedMaxResults = r.URL.Query().Get("max_results")
			// Return 2 versions via search (server respects max_results)
			json.NewEncoder(w).Encode(map[string]any{
				"model_versions": []map[string]any{
					{
						"name":        "test-prompt",
						"version":     "5",
						"description": "Version 5",
						"tags": []map[string]string{
							{"key": "mlflow.prompt.text", "value": "Template"},
						},
					},
					{
						"name":        "test-prompt",
						"version":     "4",
						"description": "Version 4",
						"tags": []map[string]string{
							{"key": "mlflow.prompt.text", "value": "Template"},
						},
					},
				},
			})

		default:
			http.NotFound(w, r)
		}
	}))

	// Request only 2 versions
	result, err := client.ListPromptVersions(context.Background(), "test-prompt",
		WithVersionsMaxResults(2),
	)
	if err != nil {
		t.Fatalf("ListPromptVersions() error = %v", err)
	}

	if receivedMaxResults != "2" {
		t.Errorf("max_results = %q, want %q", receivedMaxResults, "2")
	}

	if len(result.Versions) != 2 {
		t.Errorf("got %d versions, want 2", len(result.Versions))
	}
}

func TestListPromptVersions_Success(t *testing.T) {
	var receivedFilter, receivedOrderBy string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/2.0/mlflow/model-versions/search":
			receivedFilter = r.URL.Query().Get("filter")
			receivedOrderBy = r.URL.Query().Get("order_by")
			// Return versions via search endpoint
			json.NewEncoder(w).Encode(map[string]any{
				"model_versions": []map[string]any{
					{
						"name":                   "test-prompt",
						"version":                "3",
						"description":            "Version 3",
						"creation_timestamp":     1700000300000,
						"last_updated_timestamp": 1700000300000,
						"tags": []map[string]string{
							{"key": "mlflow.prompt.text", "value": "Template v3"},
							{"key": "author", "value": "alice"},
						},
					},
					{
						"name":               "test-prompt",
						"version":            "2",
						"description":        "Version 2",
						"creation_timestamp": 1700000200000,
						"tags": []map[string]string{
							{"key": "mlflow.prompt.text", "value": "Template v2"},
						},
					},
					{
						"name":               "test-prompt",
						"version":            "1",
						"description":        "Version 1",
						"creation_timestamp": 1700000100000,
						"tags": []map[string]string{
							{"key": "mlflow.prompt.text", "value": "Template v1"},
						},
					},
				},
			})

		default:
			http.NotFound(w, r)
		}
	}))

	result, err := client.ListPromptVersions(context.Background(), "test-prompt")
	if err != nil {
		t.Fatalf("ListPromptVersions() error = %v", err)
	}

	// Verify correct filter was sent
	if receivedFilter != "name='test-prompt'" {
		t.Errorf("filter = %q, want %q", receivedFilter, "name='test-prompt'")
	}

	// Verify correct ordering was requested
	if receivedOrderBy != "version_number DESC" {
		t.Errorf("order_by = %q, want %q", receivedOrderBy, "version_number DESC")
	}

	if len(result.Versions) != 3 {
		t.Errorf("got %d versions, want 3", len(result.Versions))
	}

	// Verify versions are returned newest first
	if result.Versions[0].Version != 3 {
		t.Errorf("first version = %d, want 3", result.Versions[0].Version)
	}
	if result.Versions[2].Version != 1 {
		t.Errorf("last version = %d, want 1", result.Versions[2].Version)
	}

	// Template should be empty in listing results
	if result.Versions[0].Template != "" {
		t.Error("Template should be empty in listing results")
	}

	// User tags should be present
	if result.Versions[0].Tags["author"] != "alice" {
		t.Errorf("Tags[author] = %q, want %q", result.Versions[0].Tags["author"], "alice")
	}
}

func TestListPromptVersions_FallbackWhenSearchEmpty(t *testing.T) {
	// Tests the fallback path when search returns empty (MLflow OSS eventual consistency)
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/2.0/mlflow/model-versions/search":
			// Return empty results (simulates MLflow OSS eventual consistency issue)
			json.NewEncoder(w).Encode(map[string]any{
				"model_versions": []map[string]any{},
			})

		case "/api/2.0/mlflow/registered-models/alias":
			// Fallback: return latest version (3)
			json.NewEncoder(w).Encode(map[string]any{
				"model_version": map[string]any{
					"name":    "test-prompt",
					"version": "3",
					"tags": []map[string]string{
						{"key": "mlflow.prompt.text", "value": "Template v3"},
					},
				},
			})

		case "/api/2.0/mlflow/model-versions/get":
			// Fallback: individual fetches
			version := r.URL.Query().Get("version")
			versionData := map[string]map[string]any{
				"3": {"name": "test-prompt", "version": "3", "description": "Version 3"},
				"2": {"name": "test-prompt", "version": "2", "description": "Version 2"},
				"1": {"name": "test-prompt", "version": "1", "description": "Version 1"},
			}
			if data, ok := versionData[version]; ok {
				json.NewEncoder(w).Encode(map[string]any{"model_version": data})
			} else {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST"})
			}

		default:
			http.NotFound(w, r)
		}
	}))

	result, err := client.ListPromptVersions(context.Background(), "test-prompt")
	if err != nil {
		t.Fatalf("ListPromptVersions() error = %v", err)
	}

	// Should get all 3 versions via fallback
	if len(result.Versions) != 3 {
		t.Errorf("got %d versions, want 3", len(result.Versions))
	}

	// Verify versions are returned newest first
	if len(result.Versions) >= 3 {
		if result.Versions[0].Version != 3 {
			t.Errorf("first version = %d, want 3", result.Versions[0].Version)
		}
		if result.Versions[2].Version != 1 {
			t.Errorf("last version = %d, want 1", result.Versions[2].Version)
		}
	}
}

func TestListPromptVersions_EmptyName(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	_, err := client.ListPromptVersions(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestDeletePromptVersion_Success(t *testing.T) {
	var deleteCalled bool
	var receivedName, receivedVersion string

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path != "/api/2.0/mlflow/model-versions/delete" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodDelete {
			t.Errorf("unexpected method: %s", r.Method)
		}

		deleteCalled = true
		var req struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		receivedName = req.Name
		receivedVersion = req.Version

		json.NewEncoder(w).Encode(map[string]any{})
	}))

	err := client.DeletePromptVersion(context.Background(), "test-prompt", 2)
	if err != nil {
		t.Fatalf("DeletePromptVersion() error = %v", err)
	}

	if !deleteCalled {
		t.Error("expected delete endpoint to be called")
	}
	if receivedName != "test-prompt" {
		t.Errorf("name = %q, want %q", receivedName, "test-prompt")
	}
	if receivedVersion != "2" {
		t.Errorf("version = %q, want %q", receivedVersion, "2")
	}
}

func TestDeletePromptVersion_NotFound(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error_code": "RESOURCE_DOES_NOT_EXIST",
			"message":    "Model version not found",
		})
	}))

	err := client.DeletePromptVersion(context.Background(), "test-prompt", 99)
	if err == nil {
		t.Error("expected error for non-existent version")
	}
	if !errors.IsNotFound(err) {
		t.Errorf("expected IsNotFound, got %v", err)
	}
}

func TestDeletePromptVersion_EmptyName(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	err := client.DeletePromptVersion(context.Background(), "", 1)
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestDeletePromptVersion_InvalidVersion(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	err := client.DeletePromptVersion(context.Background(), "test-prompt", 0)
	if err == nil {
		t.Error("expected error for zero version")
	}

	err = client.DeletePromptVersion(context.Background(), "test-prompt", -1)
	if err == nil {
		t.Error("expected error for negative version")
	}
}

func TestDeletePromptVersion_AliasConflict(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"error_code": "ALIAS_EXISTS",
			"message":    "Cannot delete version with alias",
		})
	}))

	err := client.DeletePromptVersion(context.Background(), "test-prompt", 1)
	if err == nil {
		t.Error("expected error for alias conflict")
	}
	if !errors.IsAliasConflict(err) {
		t.Errorf("expected IsAliasConflict, got %v", err)
	}
}

func TestDeletePrompt_Success(t *testing.T) {
	var deleteCalled bool
	var receivedName string

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path != "/api/2.0/mlflow/registered-models/delete" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodDelete {
			t.Errorf("unexpected method: %s", r.Method)
		}

		deleteCalled = true
		var req struct {
			Name string `json:"name"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		receivedName = req.Name

		json.NewEncoder(w).Encode(map[string]any{})
	}))

	err := client.DeletePrompt(context.Background(), "test-prompt")
	if err != nil {
		t.Fatalf("DeletePrompt() error = %v", err)
	}

	if !deleteCalled {
		t.Error("expected delete endpoint to be called")
	}
	if receivedName != "test-prompt" {
		t.Errorf("name = %q, want %q", receivedName, "test-prompt")
	}
}

func TestDeletePrompt_NotFound(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error_code": "RESOURCE_DOES_NOT_EXIST",
			"message":    "Registered Model not found",
		})
	}))

	err := client.DeletePrompt(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent prompt")
	}
	if !errors.IsNotFound(err) {
		t.Errorf("expected IsNotFound, got %v", err)
	}
}

func TestDeletePrompt_EmptyName(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	err := client.DeletePrompt(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestDeletePrompt_PermissionDenied(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"error_code": "PERMISSION_DENIED",
			"message":    "User lacks permission to delete",
		})
	}))

	err := client.DeletePrompt(context.Background(), "test-prompt")
	if err == nil {
		t.Error("expected error for permission denied")
	}
	if !errors.IsPermissionDenied(err) {
		t.Errorf("expected IsPermissionDenied, got %v", err)
	}
}

func TestDeletePromptTag_Success(t *testing.T) {
	var deleteCalled bool
	var receivedName, receivedKey string

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path != "/api/2.0/mlflow/registered-models/delete-tag" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodDelete {
			t.Errorf("unexpected method: %s", r.Method)
		}

		deleteCalled = true
		var req struct {
			Name string `json:"name"`
			Key  string `json:"key"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		receivedName = req.Name
		receivedKey = req.Key

		json.NewEncoder(w).Encode(map[string]any{})
	}))

	err := client.DeletePromptTag(context.Background(), "test-prompt", "environment")
	if err != nil {
		t.Fatalf("DeletePromptTag() error = %v", err)
	}

	if !deleteCalled {
		t.Error("expected delete endpoint to be called")
	}
	if receivedName != "test-prompt" {
		t.Errorf("name = %q, want %q", receivedName, "test-prompt")
	}
	if receivedKey != "environment" {
		t.Errorf("key = %q, want %q", receivedKey, "environment")
	}
}

func TestDeletePromptTag_NotFound(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error_code": "RESOURCE_DOES_NOT_EXIST",
			"message":    "Tag not found",
		})
	}))

	err := client.DeletePromptTag(context.Background(), "test-prompt", "missing")
	if err == nil {
		t.Error("expected error for non-existent tag")
	}
	if !errors.IsNotFound(err) {
		t.Errorf("expected IsNotFound, got %v", err)
	}
}

func TestDeletePromptTag_EmptyName(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	err := client.DeletePromptTag(context.Background(), "", "key")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestDeletePromptTag_EmptyKey(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	err := client.DeletePromptTag(context.Background(), "test-prompt", "")
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestDeletePromptVersionTag_Success(t *testing.T) {
	var deleteCalled bool
	var receivedName, receivedVersion, receivedKey string

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path != "/api/2.0/mlflow/model-versions/delete-tag" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodDelete {
			t.Errorf("unexpected method: %s", r.Method)
		}

		deleteCalled = true
		var req struct {
			Name    string `json:"name"`
			Version string `json:"version"`
			Key     string `json:"key"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		receivedName = req.Name
		receivedVersion = req.Version
		receivedKey = req.Key

		json.NewEncoder(w).Encode(map[string]any{})
	}))

	err := client.DeletePromptVersionTag(context.Background(), "test-prompt", 1, "reviewed")
	if err != nil {
		t.Fatalf("DeletePromptVersionTag() error = %v", err)
	}

	if !deleteCalled {
		t.Error("expected delete endpoint to be called")
	}
	if receivedName != "test-prompt" {
		t.Errorf("name = %q, want %q", receivedName, "test-prompt")
	}
	if receivedVersion != "1" {
		t.Errorf("version = %q, want %q", receivedVersion, "1")
	}
	if receivedKey != "reviewed" {
		t.Errorf("key = %q, want %q", receivedKey, "reviewed")
	}
}

func TestDeletePromptVersionTag_NotFound(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error_code": "RESOURCE_DOES_NOT_EXIST",
			"message":    "Tag not found",
		})
	}))

	err := client.DeletePromptVersionTag(context.Background(), "test-prompt", 1, "missing")
	if err == nil {
		t.Error("expected error for non-existent tag")
	}
	if !errors.IsNotFound(err) {
		t.Errorf("expected IsNotFound, got %v", err)
	}
}

func TestDeletePromptVersionTag_EmptyName(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	err := client.DeletePromptVersionTag(context.Background(), "", 1, "key")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestDeletePromptVersionTag_InvalidVersion(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	err := client.DeletePromptVersionTag(context.Background(), "test-prompt", 0, "key")
	if err == nil {
		t.Error("expected error for zero version")
	}

	err = client.DeletePromptVersionTag(context.Background(), "test-prompt", -1, "key")
	if err == nil {
		t.Error("expected error for negative version")
	}
}

func TestDeletePromptVersionTag_EmptyKey(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	err := client.DeletePromptVersionTag(context.Background(), "test-prompt", 1, "")
	if err == nil {
		t.Error("expected error for empty key")
	}
}
