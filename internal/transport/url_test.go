package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// captureServer starts an httptest server that records the escaped path and raw
// query of the first request it receives.
func captureServer(t *testing.T, gotPath, gotQuery *string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*gotPath = r.URL.EscapedPath()
		*gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestBuildURL_WireForm covers the literal-path builder used by Get/Post/Delete/
// Patch and the raw byte/stream helpers. The path argument is treated as a
// literal value: reserved characters are percent-encoded exactly once, "." and
// ".." segments are preserved (no RFC 3986 normalization), and any encoding in
// the base URL prefix survives.
func TestBuildURL_WireForm(t *testing.T) {
	tests := []struct {
		name      string
		basePath  string // appended to the test server's base URL
		path      string
		query     url.Values
		wantPath  string
		wantQuery string
	}{
		{
			name:     "simple path",
			path:     "/api/2.0/mlflow/experiments/get",
			wantPath: "/api/2.0/mlflow/experiments/get",
		},
		{
			name:     "plain base path prefix",
			basePath: "/mlflow",
			path:     "/api/2.0/mlflow/runs/get",
			wantPath: "/mlflow/api/2.0/mlflow/runs/get",
		},
		{
			name:     "escaped base path prefix is preserved",
			basePath: "/api%2Fv1",
			path:     "/api/2.0/mlflow/runs/get",
			wantPath: "/api%2Fv1/api/2.0/mlflow/runs/get",
		},
		{
			name:     "dot segments are not collapsed",
			path:     "/api/2.0/mlflow-artifacts/artifacts/a/../b/./c",
			wantPath: "/api/2.0/mlflow-artifacts/artifacts/a/../b/./c",
		},
		{
			name:     "literal percent is encoded once",
			path:     "/api/2.0/mlflow-artifacts/artifacts/data%2F2026.csv",
			wantPath: "/api/2.0/mlflow-artifacts/artifacts/data%252F2026.csv",
		},
		{
			name:     "space in path is encoded",
			path:     "/api/2.0/mlflow-artifacts/artifacts/my file.txt",
			wantPath: "/api/2.0/mlflow-artifacts/artifacts/my%20file.txt",
		},
		{
			name:      "query parameters are encoded",
			path:      "/api/2.0/mlflow/experiments/get-by-name",
			query:     url.Values{"experiment_name": []string{"a/b c"}},
			wantPath:  "/api/2.0/mlflow/experiments/get-by-name",
			wantQuery: "experiment_name=a%2Fb+c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath, gotQuery string
			srv := captureServer(t, &gotPath, &gotQuery)

			c, err := New(Config{BaseURL: srv.URL + tt.basePath})
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			if err := c.Get(context.Background(), tt.path, tt.query, nil); err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			if gotPath != tt.wantPath {
				t.Errorf("wire path = %q, want %q", gotPath, tt.wantPath)
			}
			if gotQuery != tt.wantQuery {
				t.Errorf("wire query = %q, want %q", gotQuery, tt.wantQuery)
			}
		})
	}
}

// TestBuildEscapedURL_WireForm covers the escaped-path builder used by
// GetEscaped/DeleteEscaped. The caller supplies an already percent-encoded path
// (e.g. a single segment via url.PathEscape); the encoding is preserved so the
// segment is neither re-encoded nor split, and any escaped base prefix survives.
func TestBuildEscapedURL_WireForm(t *testing.T) {
	tests := []struct {
		name        string
		basePath    string
		escapedPath string
		wantPath    string
	}{
		{
			name:        "encoded slash in segment is preserved",
			escapedPath: "/api/3.0/mlflow/workspaces/" + url.PathEscape("team/a"),
			wantPath:    "/api/3.0/mlflow/workspaces/team%2Fa",
		},
		{
			name:        "encoded percent and space in segment are preserved",
			escapedPath: "/api/3.0/mlflow/workspaces/" + url.PathEscape("a b%c"),
			wantPath:    "/api/3.0/mlflow/workspaces/a%20b%25c",
		},
		{
			name:        "escaped base prefix is preserved",
			basePath:    "/api%2Fv1",
			escapedPath: "/api/3.0/mlflow/workspaces/" + url.PathEscape("ws"),
			wantPath:    "/api%2Fv1/api/3.0/mlflow/workspaces/ws",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath, gotQuery string
			srv := captureServer(t, &gotPath, &gotQuery)

			c, err := New(Config{BaseURL: srv.URL + tt.basePath})
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			if err := c.GetEscaped(context.Background(), tt.escapedPath, nil, nil); err != nil {
				t.Fatalf("GetEscaped() error = %v", err)
			}
			if gotPath != tt.wantPath {
				t.Errorf("wire path = %q, want %q", gotPath, tt.wantPath)
			}
		})
	}
}

// TestGetEscaped_InvalidEscaping ensures a malformed percent-escape is reported
// as an error rather than silently producing a wrong URL.
func TestGetEscaped_InvalidEscaping(t *testing.T) {
	c, err := New(Config{BaseURL: "http://example.test"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := c.GetEscaped(context.Background(), "/api/3.0/mlflow/workspaces/%zz", nil, nil); err == nil {
		t.Error("expected error for invalid percent-escape, got nil")
	}
}

// TestDeleteEscaped_WireForm confirms the DELETE escaped-path variant preserves
// an encoded single segment.
func TestDeleteEscaped_WireForm(t *testing.T) {
	var gotPath, gotQuery string
	srv := captureServer(t, &gotPath, &gotQuery)

	c, err := New(Config{BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	endpoint := "/api/3.0/mlflow/workspaces/" + url.PathEscape("team/a")
	if err := c.DeleteEscaped(context.Background(), endpoint, nil, nil); err != nil {
		t.Fatalf("DeleteEscaped() error = %v", err)
	}
	if want := "/api/3.0/mlflow/workspaces/team%2Fa"; gotPath != want {
		t.Errorf("wire path = %q, want %q", gotPath, want)
	}
}
