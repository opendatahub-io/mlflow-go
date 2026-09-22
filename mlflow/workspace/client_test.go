package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/opendatahub-io/mlflow-go/internal/errors"
	"github.com/opendatahub-io/mlflow-go/internal/transport"
)

// isLive reports whether tests should run against a real MLflow service.
// Live mode is opt-in: it requires MLFLOW_RUN_LIVE_TESTS to be set (guarding
// against `go test ./...` accidentally mutating a server) together with
// MLFLOW_TRACKING_URI pointing at the target service.
func isLive() bool {
	return os.Getenv("MLFLOW_RUN_LIVE_TESTS") != "" && os.Getenv("MLFLOW_TRACKING_URI") != ""
}

// skipIfLive skips the current test when a live service is configured.
func skipIfLive(t *testing.T, reason string) {
	t.Helper()
	if isLive() {
		t.Skipf("mock-only: %s", reason)
	}
}

// newTestClient returns a workspace Client. In live mode (see isLive) the mock
// handler is ignored and the client targets MLFLOW_TRACKING_URI; otherwise an
// httptest.Server is started with the given handler.
func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()

	var baseURL string
	if isLive() {
		baseURL = os.Getenv("MLFLOW_TRACKING_URI")
	} else {
		server := httptest.NewServer(handler)
		t.Cleanup(server.Close)
		baseURL = server.URL
	}

	tc, err := transport.New(transport.Config{BaseURL: baseURL})
	if err != nil {
		t.Fatalf("transport.New() error = %v", err)
	}

	return NewClient(tc)
}

// uniqueName returns a workspace name unique to this test run.
// Names conform to the MLflow pattern ^[a-z0-9]([-a-z0-9]*[a-z0-9])?$.
func uniqueName(t *testing.T) string {
	t.Helper()
	slug := strings.ToLower(strings.ReplaceAll(t.Name(), "_", "-"))
	slug = strings.ReplaceAll(slug, "/", "-")
	return fmt.Sprintf("t-%s-%x", slug, time.Now().UnixNano())
}

// cleanupWorkspace registers a t.Cleanup that deletes the named workspace.
func cleanupWorkspace(t *testing.T, client *Client, name string) {
	t.Helper()
	t.Cleanup(func() {
		if err := client.DeleteWorkspace(context.Background(), name); err != nil {
			t.Logf("cleanup: failed to delete workspace %q: %v", name, err)
		}
	})
}

func mustEncodeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("failed to encode response: %v", err)
	}
}

func workspaceJSON(name string) map[string]any {
	return map[string]any{"workspace": map[string]any{"name": name}}
}

// ---------------------------------------------------------------------------
// GetServerInfo
// ---------------------------------------------------------------------------

func TestGetServerInfo_Enabled(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/3.0/mlflow/server-info" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		mustEncodeJSON(t, w, map[string]any{"workspaces_enabled": true})
	}))

	info, err := client.GetServerInfo(context.Background())
	if err != nil {
		t.Fatalf("GetServerInfo() error = %v", err)
	}
	if isLive() {
		t.Logf("live server: WorkspacesEnabled = %v", info.WorkspacesEnabled)
	} else if !info.WorkspacesEnabled {
		t.Error("expected WorkspacesEnabled = true")
	}
}

func TestGetServerInfo_Disabled(t *testing.T) {
	skipIfLive(t, "tests a fabricated server response")

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mustEncodeJSON(t, w, map[string]any{"workspaces_enabled": false})
	}))

	info, err := client.GetServerInfo(context.Background())
	if err != nil {
		t.Fatalf("GetServerInfo() error = %v", err)
	}
	if info.WorkspacesEnabled {
		t.Error("expected WorkspacesEnabled = false")
	}
}

// ---------------------------------------------------------------------------
// GetWorkspace
// ---------------------------------------------------------------------------

func TestGetWorkspace_Success(t *testing.T) {
	name := "my-ws"
	if isLive() {
		name = "default"
	}

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mustEncodeJSON(t, w, workspaceJSON(name))
	}))

	ws, err := client.GetWorkspace(context.Background(), name)
	if err != nil {
		t.Fatalf("GetWorkspace(%q) error = %v", name, err)
	}
	if ws.Name != name {
		t.Errorf("Name = %q, want %q", ws.Name, name)
	}
}

func TestGetWorkspace_EmptyName(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	if _, err := client.GetWorkspace(context.Background(), ""); err == nil {
		t.Error("expected error for empty name")
	}
}

// TestGetWorkspace_EscapesNameSegment verifies the client encodes the name as a
// single {workspace_name} path segment: reserved characters are escaped and a
// "/" does not split the path. Mock-only: it drives a fabricated request path.
func TestGetWorkspace_EscapesNameSegment(t *testing.T) {
	skipIfLive(t, "asserts the client-constructed request path")

	var gotPath string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		mustEncodeJSON(t, w, workspaceJSON("team/a b"))
	}))

	if _, err := client.GetWorkspace(context.Background(), "team/a b"); err != nil {
		t.Fatalf("GetWorkspace() error = %v", err)
	}
	if want := "/api/3.0/mlflow/workspaces/team%2Fa%20b"; gotPath != want {
		t.Errorf("request path = %q, want %q", gotPath, want)
	}
}

// ---------------------------------------------------------------------------
// CreateWorkspace
// ---------------------------------------------------------------------------

func TestCreateWorkspace_Success(t *testing.T) {
	if isLive() {
		client := newTestClient(t, nil)
		name := uniqueName(t)
		cleanupWorkspace(t, client, name)

		ws, err := client.CreateWorkspace(context.Background(), name)
		if err != nil {
			t.Fatalf("CreateWorkspace(%q) error = %v", name, err)
		}
		if ws.Name != name {
			t.Errorf("Name = %q, want %q", ws.Name, name)
		}
		return
	}

	var receivedName string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)
		receivedName = req["name"]
		mustEncodeJSON(t, w, workspaceJSON(req["name"]))
	}))

	ws, err := client.CreateWorkspace(context.Background(), "new-ws")
	if err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}
	if ws.Name != "new-ws" {
		t.Errorf("Name = %q, want %q", ws.Name, "new-ws")
	}
	if receivedName != "new-ws" {
		t.Errorf("received name = %q, want %q", receivedName, "new-ws")
	}
}

func TestCreateWorkspace_EmptyName(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	if _, err := client.CreateWorkspace(context.Background(), ""); err == nil {
		t.Error("expected error for empty name")
	}
}

func TestCreateWorkspace_InvalidName(t *testing.T) {
	if isLive() {
		client := newTestClient(t, nil)
		_, err := client.CreateWorkspace(context.Background(), "UPPER_CASE!")
		if err == nil {
			t.Fatal("expected error for invalid workspace name")
		}
		t.Logf("got expected error: %v", err)
		return
	}

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		mustEncodeJSON(t, w, map[string]string{
			"error_code": "INVALID_PARAMETER_VALUE",
			"message":    "Workspace name must match the pattern ^[a-z0-9]([-a-z0-9]*[a-z0-9])?$",
		})
	}))

	_, err := client.CreateWorkspace(context.Background(), "UPPER_CASE!")
	if err == nil {
		t.Fatal("expected error for invalid workspace name")
	}
}

// ---------------------------------------------------------------------------
// DeleteWorkspace
// ---------------------------------------------------------------------------

func TestDeleteWorkspace_Success(t *testing.T) {
	if isLive() {
		client := newTestClient(t, nil)
		name := uniqueName(t)

		_, err := client.CreateWorkspace(context.Background(), name)
		if err != nil {
			t.Fatalf("setup: CreateWorkspace(%q) error = %v", name, err)
		}

		if err := client.DeleteWorkspace(context.Background(), name); err != nil {
			t.Fatalf("DeleteWorkspace(%q) error = %v", name, err)
		}
		return
	}

	deleted := false
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deleted = true
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))

	if err := client.DeleteWorkspace(context.Background(), "doomed-ws"); err != nil {
		t.Fatalf("DeleteWorkspace() error = %v", err)
	}
	if !deleted {
		t.Error("expected DELETE request")
	}
}

func TestDeleteWorkspace_EmptyName(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	if err := client.DeleteWorkspace(context.Background(), ""); err == nil {
		t.Error("expected error for empty name")
	}
}

// TestDeleteWorkspace_EscapesNameSegment mirrors TestGetWorkspace_EscapesNameSegment
// for the DELETE path. Mock-only: it drives a fabricated request path.
func TestDeleteWorkspace_EscapesNameSegment(t *testing.T) {
	skipIfLive(t, "asserts the client-constructed request path")

	var gotPath string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusOK)
	}))

	if err := client.DeleteWorkspace(context.Background(), "team/a b"); err != nil {
		t.Fatalf("DeleteWorkspace() error = %v", err)
	}
	if want := "/api/3.0/mlflow/workspaces/team%2Fa%20b"; gotPath != want {
		t.Errorf("request path = %q, want %q", gotPath, want)
	}
}

// ---------------------------------------------------------------------------
// EnsureWorkspace
// ---------------------------------------------------------------------------

func TestEnsureWorkspace_CreatesNew(t *testing.T) {
	if isLive() {
		client := newTestClient(t, nil)
		name := uniqueName(t)
		cleanupWorkspace(t, client, name)

		ws, err := client.EnsureWorkspace(context.Background(), name)
		if err != nil {
			t.Fatalf("EnsureWorkspace(%q) error = %v", name, err)
		}
		if ws.Name != name {
			t.Errorf("Name = %q, want %q", ws.Name, name)
		}
		return
	}

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost && r.URL.Path == "/api/3.0/mlflow/workspaces" {
			mustEncodeJSON(t, w, workspaceJSON("new-ws"))
			return
		}
		http.NotFound(w, r)
	}))

	ws, err := client.EnsureWorkspace(context.Background(), "new-ws")
	if err != nil {
		t.Fatalf("EnsureWorkspace() error = %v", err)
	}
	if ws.Name != "new-ws" {
		t.Errorf("Name = %q, want %q", ws.Name, "new-ws")
	}
}

func TestEnsureWorkspace_AlreadyExists(t *testing.T) {
	if isLive() {
		client := newTestClient(t, nil)
		name := uniqueName(t)
		cleanupWorkspace(t, client, name)

		ws1, err := client.CreateWorkspace(context.Background(), name)
		if err != nil {
			t.Fatalf("setup: CreateWorkspace(%q) error = %v", name, err)
		}

		ws2, err := client.EnsureWorkspace(context.Background(), name)
		if err != nil {
			t.Fatalf("EnsureWorkspace(%q) error = %v", name, err)
		}
		if ws2.Name != ws1.Name {
			t.Errorf("EnsureWorkspace returned %q, want %q", ws2.Name, ws1.Name)
		}
		return
	}

	calls := 0
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		calls++
		if r.Method == http.MethodPost && r.URL.Path == "/api/3.0/mlflow/workspaces" {
			w.WriteHeader(http.StatusConflict)
			mustEncodeJSON(t, w, map[string]string{
				"error_code": "RESOURCE_ALREADY_EXISTS",
				"message":    "workspace exists",
			})
			return
		}
		mustEncodeJSON(t, w, workspaceJSON("existing-ws"))
	}))

	ws, err := client.EnsureWorkspace(context.Background(), "existing-ws")
	if err != nil {
		t.Fatalf("EnsureWorkspace() error = %v", err)
	}
	if ws.Name != "existing-ws" {
		t.Errorf("Name = %q, want %q", ws.Name, "existing-ws")
	}
	if calls != 2 {
		t.Errorf("expected 2 calls (create + get), got %d", calls)
	}
}

func TestEnsureWorkspace_Default(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet {
			t.Errorf("expected GET for default workspace, got %s", r.Method)
		}
		mustEncodeJSON(t, w, workspaceJSON("default"))
	}))

	ws, err := client.EnsureWorkspace(context.Background(), "default")
	if err != nil {
		t.Fatalf("EnsureWorkspace(default) error = %v", err)
	}
	if ws.Name != "default" {
		t.Errorf("Name = %q, want %q", ws.Name, "default")
	}
}

func TestEnsureWorkspace_EmptyName(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	if _, err := client.EnsureWorkspace(context.Background(), ""); err == nil {
		t.Error("expected error for empty name")
	}
}

func TestEnsureWorkspace_CreateFails(t *testing.T) {
	skipIfLive(t, "tests server-error handling")

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		mustEncodeJSON(t, w, map[string]string{
			"error_code": "INTERNAL_ERROR",
			"message":    "server failure",
		})
	}))

	if _, err := client.EnsureWorkspace(context.Background(), "fail-ws"); err == nil {
		t.Error("expected error for server failure")
	}
}

// Ensure errors import is used by referencing it.
var _ = errors.IsAlreadyExists
