package mcpregistry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

// --- CreateMCPServer ---

func TestCreateMCPServer_EmptyName(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	_, err := client.CreateMCPServer(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestCreateMCPServer_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != mcpServersBasePath {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "com.example/my-server" {
			t.Errorf("expected body.name=com.example/my-server, got %v", body["name"])
		}
		if body["description"] != "A test server" {
			t.Errorf("expected body.description=A test server, got %v", body["description"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"name":               "com.example/my-server",
			"description":        "A test server",
			"status":             "active",
			"creation_timestamp": 1700000000000,
		})
	}))

	server, err := client.CreateMCPServer(context.Background(), "com.example/my-server",
		WithServerDescription("A test server"))
	if err != nil {
		t.Fatalf("CreateMCPServer() error = %v", err)
	}

	if server.Name != "com.example/my-server" {
		t.Errorf("Name = %q, want %q", server.Name, "com.example/my-server")
	}
	if server.Description != "A test server" {
		t.Errorf("Description = %q, want %q", server.Description, "A test server")
	}
	if server.Status != "active" {
		t.Errorf("Status = %q, want %q", server.Status, "active")
	}
}

// --- GetMCPServer ---

func TestGetMCPServer_EmptyName(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	_, err := client.GetMCPServer(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestGetMCPServer_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != mcpServersBasePath+"/my-server" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"name":           "my-server",
			"latest_version": "1.0.0",
			"aliases": []map[string]string{
				{"alias": "production", "version": "1.0.0"},
			},
			"tags": map[string]string{"team": "ml"},
		})
	}))

	server, err := client.GetMCPServer(context.Background(), "my-server")
	if err != nil {
		t.Fatalf("GetMCPServer() error = %v", err)
	}

	if server.LatestVersion != "1.0.0" {
		t.Errorf("LatestVersion = %q, want %q", server.LatestVersion, "1.0.0")
	}
	if server.Aliases["production"] != "1.0.0" {
		t.Errorf("Aliases[production] = %q, want %q", server.Aliases["production"], "1.0.0")
	}
	if server.Tags["team"] != "ml" {
		t.Errorf("Tags[team] = %q, want %q", server.Tags["team"], "ml")
	}
}

func TestGetMCPServer_NotFound(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error_code": "RESOURCE_DOES_NOT_EXIST",
			"message":    "MCP server not found",
		})
	}))

	_, err := client.GetMCPServer(context.Background(), "unknown")
	if err == nil {
		t.Fatal("expected error for non-existent server")
	}
	if !errors.IsNotFound(err) {
		t.Errorf("expected IsNotFound, got %v", err)
	}
}

// --- SearchMCPServers ---

func TestSearchMCPServers_InvalidMaxResults(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	_, err := client.SearchMCPServers(context.Background(), WithServersMaxResults(0))
	if err == nil {
		t.Error("expected error for non-positive max results")
	}
}

func TestSearchMCPServers_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != mcpServersBasePath {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("filter_string") != "name LIKE 'com.example%'" {
			t.Errorf("unexpected filter_string: %s", r.URL.Query().Get("filter_string"))
		}
		if r.URL.Query().Get("max_results") != "10" {
			t.Errorf("unexpected max_results: %s", r.URL.Query().Get("max_results"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"mcp_servers": []map[string]any{
				{"name": "com.example/server-a"},
				{"name": "com.example/server-b"},
			},
			"next_page_token": "next-token",
		})
	}))

	list, err := client.SearchMCPServers(context.Background(),
		WithServersFilter("name LIKE 'com.example%'"),
		WithServersMaxResults(10),
	)
	if err != nil {
		t.Fatalf("SearchMCPServers() error = %v", err)
	}

	if len(list.Servers) != 2 {
		t.Fatalf("len(Servers) = %d, want 2", len(list.Servers))
	}
	if list.Servers[0].Name != "com.example/server-a" {
		t.Errorf("Servers[0].Name = %q, want %q", list.Servers[0].Name, "com.example/server-a")
	}
	if list.NextPageToken != "next-token" {
		t.Errorf("NextPageToken = %q, want %q", list.NextPageToken, "next-token")
	}
}

// --- SetMCPServerTag ---

func TestSetMCPServerTag_EmptyName(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	err := client.SetMCPServerTag(context.Background(), "", "key", "value")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestSetMCPServerTag_EmptyKey(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	err := client.SetMCPServerTag(context.Background(), "my-server", "", "value")
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestSetMCPServerTag_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != mcpServersBasePath+"/my-server/tags" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["key"] != "team" || body["value"] != "ml" {
			t.Errorf("unexpected body: %v", body)
		}

		w.WriteHeader(http.StatusOK)
	}))

	err := client.SetMCPServerTag(context.Background(), "my-server", "team", "ml")
	if err != nil {
		t.Fatalf("SetMCPServerTag() error = %v", err)
	}
}

// --- CreateMCPServerVersion ---

func TestCreateMCPServerVersion_EmptyName(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	_, err := client.CreateMCPServerVersion(context.Background(), "", map[string]any{"name": "x"})
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestCreateMCPServerVersion_EmptyServerJSON(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	_, err := client.CreateMCPServerVersion(context.Background(), "my-server", nil)
	if err == nil {
		t.Error("expected error for empty server JSON")
	}
}

func TestCreateMCPServerVersion_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != mcpServersBasePath+"/my-server/versions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["status"] != "active" {
			t.Errorf("expected body.status=active, got %v", body["status"])
		}
		serverJSON, ok := body["server_json"].(map[string]any)
		if !ok || serverJSON["name"] != "my-server" {
			t.Errorf("unexpected server_json: %v", body["server_json"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"name":    "my-server",
			"version": "1.0.0",
			"status":  "active",
			"server_json": map[string]any{
				"name":    "my-server",
				"version": "1.0.0",
			},
			"tools": []map[string]any{
				{"name": "search", "description": "Search for things"},
			},
		})
	}))

	version, err := client.CreateMCPServerVersion(context.Background(), "my-server",
		map[string]any{"name": "my-server", "version": "1.0.0"},
		WithVersionStatus(MCPServerVersionStatusActive),
	)
	if err != nil {
		t.Fatalf("CreateMCPServerVersion() error = %v", err)
	}

	if version.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", version.Version, "1.0.0")
	}
	if version.Status != MCPServerVersionStatusActive {
		t.Errorf("Status = %q, want %q", version.Status, MCPServerVersionStatusActive)
	}
	if len(version.Tools) != 1 || version.Tools[0].Name != "search" {
		t.Errorf("Tools = %+v, want a single 'search' tool", version.Tools)
	}
}

// --- GetMCPServerVersion ---

func TestGetMCPServerVersion_EmptyArgs(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	if _, err := client.GetMCPServerVersion(context.Background(), "", "1.0.0"); err == nil {
		t.Error("expected error for empty name")
	}
	if _, err := client.GetMCPServerVersion(context.Background(), "my-server", ""); err == nil {
		t.Error("expected error for empty version")
	}
}

func TestGetMCPServerVersion_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != mcpServersBasePath+"/my-server/versions/2.0.0" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"name":    "my-server",
			"version": "2.0.0",
			"status":  "draft",
			"server_json": map[string]any{
				"name":    "my-server",
				"version": "2.0.0",
			},
		})
	}))

	version, err := client.GetMCPServerVersion(context.Background(), "my-server", "2.0.0")
	if err != nil {
		t.Fatalf("GetMCPServerVersion() error = %v", err)
	}

	if version.Version != "2.0.0" {
		t.Errorf("Version = %q, want %q", version.Version, "2.0.0")
	}
	if version.Status != MCPServerVersionStatusDraft {
		t.Errorf("Status = %q, want %q", version.Status, MCPServerVersionStatusDraft)
	}
}

// --- SearchMCPServerVersions ---

func TestSearchMCPServerVersions_EmptyName(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	_, err := client.SearchMCPServerVersions(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestSearchMCPServerVersions_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != mcpServersBasePath+"/my-server/versions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"mcp_server_versions": []map[string]any{
				{"name": "my-server", "version": "1.0.0", "server_json": map[string]any{}},
				{"name": "my-server", "version": "2.0.0", "server_json": map[string]any{}},
			},
		})
	}))

	list, err := client.SearchMCPServerVersions(context.Background(), "my-server")
	if err != nil {
		t.Fatalf("SearchMCPServerVersions() error = %v", err)
	}

	if len(list.Versions) != 2 {
		t.Fatalf("len(Versions) = %d, want 2", len(list.Versions))
	}
}

// --- CreateMCPAccessEndpoint ---

func TestCreateMCPAccessEndpoint_EmptyArgs(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	if _, err := client.CreateMCPAccessEndpoint(context.Background(), "", "https://example.com"); err == nil {
		t.Error("expected error for empty server name")
	}
	if _, err := client.CreateMCPAccessEndpoint(context.Background(), "my-server", ""); err == nil {
		t.Error("expected error for empty endpoint URL")
	}
}

func TestCreateMCPAccessEndpoint_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != mcpServersBasePath+"/my-server/endpoints" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["url"] != "https://example.com/mcp" {
			t.Errorf("unexpected url: %v", body["url"])
		}
		if body["transport_type"] != "streamable-http" {
			t.Errorf("expected default transport_type=streamable-http, got %v", body["transport_type"])
		}
		if body["server_alias"] != "production" {
			t.Errorf("expected server_alias=production, got %v", body["server_alias"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":             "1",
			"server_name":    "my-server",
			"url":            "https://example.com/mcp",
			"transport_type": "streamable-http",
			"server_alias":   "production",
		})
	}))

	endpoint, err := client.CreateMCPAccessEndpoint(context.Background(), "my-server", "https://example.com/mcp",
		WithAccessEndpointServerAlias("production"),
	)
	if err != nil {
		t.Fatalf("CreateMCPAccessEndpoint() error = %v", err)
	}

	if endpoint.ID != "1" {
		t.Errorf("ID = %q, want %q", endpoint.ID, "1")
	}
	if endpoint.TransportType != MCPTransportStreamableHTTP {
		t.Errorf("TransportType = %q, want %q", endpoint.TransportType, MCPTransportStreamableHTTP)
	}
	if endpoint.ServerAlias != "production" {
		t.Errorf("ServerAlias = %q, want %q", endpoint.ServerAlias, "production")
	}
}

// --- SearchMCPAccessEndpoints ---

func TestSearchMCPAccessEndpoints_AllServers(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != mcpServersBasePath+"/endpoints" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"mcp_access_endpoints": []map[string]any{
				{"id": "1", "server_name": "server-a", "url": "https://a.example.com"},
				{"id": "2", "server_name": "server-b", "url": "https://b.example.com"},
			},
		})
	}))

	list, err := client.SearchMCPAccessEndpoints(context.Background())
	if err != nil {
		t.Fatalf("SearchMCPAccessEndpoints() error = %v", err)
	}

	if len(list.Endpoints) != 2 {
		t.Fatalf("len(Endpoints) = %d, want 2", len(list.Endpoints))
	}
}

func TestSearchMCPAccessEndpoints_ScopedToServer(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != mcpServersBasePath+"/my-server/endpoints" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("server_alias") != "production" {
			t.Errorf("unexpected server_alias: %s", r.URL.Query().Get("server_alias"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"mcp_access_endpoints": []map[string]any{
				{"id": "1", "server_name": "my-server", "url": "https://a.example.com"},
			},
		})
	}))

	list, err := client.SearchMCPAccessEndpoints(context.Background(),
		WithAccessEndpointsServerName("my-server"),
		WithAccessEndpointsServerAlias("production"),
	)
	if err != nil {
		t.Fatalf("SearchMCPAccessEndpoints() error = %v", err)
	}

	if len(list.Endpoints) != 1 {
		t.Fatalf("len(Endpoints) = %d, want 1", len(list.Endpoints))
	}
	if list.Endpoints[0].ServerName != "my-server" {
		t.Errorf("ServerName = %q, want %q", list.Endpoints[0].ServerName, "my-server")
	}
}

// --- DeleteMCPAccessEndpoint ---

func TestDeleteMCPAccessEndpoint_InvalidArgs(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	if err := client.DeleteMCPAccessEndpoint(context.Background(), "", "1"); err == nil {
		t.Error("expected error for empty server name")
	}
	if err := client.DeleteMCPAccessEndpoint(context.Background(), "my-server", ""); err == nil {
		t.Error("expected error for empty endpoint ID")
	}
}

func TestDeleteMCPAccessEndpoint_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != mcpServersBasePath+"/my-server/endpoints/42" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	}))

	err := client.DeleteMCPAccessEndpoint(context.Background(), "my-server", "42")
	if err != nil {
		t.Fatalf("DeleteMCPAccessEndpoint() error = %v", err)
	}
}

func TestDeleteMCPAccessEndpoint_NotFound(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error_code": "RESOURCE_DOES_NOT_EXIST",
			"message":    "Access endpoint not found",
		})
	}))

	err := client.DeleteMCPAccessEndpoint(context.Background(), "my-server", "42")
	if !errors.IsNotFound(err) {
		t.Errorf("expected IsNotFound, got %v", err)
	}
}

// --- GetMCPAccessEndpoint ---

func TestGetMCPAccessEndpoint_EmptyArgs(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	if _, err := client.GetMCPAccessEndpoint(context.Background(), "", "1"); err == nil {
		t.Error("expected error for empty server name")
	}
	if _, err := client.GetMCPAccessEndpoint(context.Background(), "my-server", ""); err == nil {
		t.Error("expected error for empty endpoint ID")
	}
}

func TestGetMCPAccessEndpoint_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != mcpServersBasePath+"/my-server/endpoints/42" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":             "42",
			"server_name":    "my-server",
			"url":            "https://example.com/mcp",
			"transport_type": "streamable-http",
		})
	}))

	endpoint, err := client.GetMCPAccessEndpoint(context.Background(), "my-server", "42")
	if err != nil {
		t.Fatalf("GetMCPAccessEndpoint() error = %v", err)
	}
	if endpoint.ID != "42" {
		t.Errorf("ID = %q, want %q", endpoint.ID, "42")
	}
}

// --- UpdateMCPAccessEndpoint ---

func TestUpdateMCPAccessEndpoint_EmptyArgs(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	if _, err := client.UpdateMCPAccessEndpoint(context.Background(), "", "1"); err == nil {
		t.Error("expected error for empty server name")
	}
	if _, err := client.UpdateMCPAccessEndpoint(context.Background(), "my-server", ""); err == nil {
		t.Error("expected error for empty endpoint ID")
	}
}

func TestUpdateMCPAccessEndpoint_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != mcpServersBasePath+"/my-server/endpoints/42" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["url"] != "https://updated.example.com/mcp" {
			t.Errorf("unexpected url: %v", body["url"])
		}
		if _, ok := body["server_alias"]; ok {
			t.Errorf("expected server_alias to be omitted, got %v", body["server_alias"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":             "42",
			"server_name":    "my-server",
			"url":            "https://updated.example.com/mcp",
			"transport_type": "sse",
		})
	}))

	endpoint, err := client.UpdateMCPAccessEndpoint(context.Background(), "my-server", "42",
		WithUpdatedEndpointURL("https://updated.example.com/mcp"),
		WithUpdatedEndpointTransportType(MCPTransportSSE),
	)
	if err != nil {
		t.Fatalf("UpdateMCPAccessEndpoint() error = %v", err)
	}
	if endpoint.EndpointURL != "https://updated.example.com/mcp" {
		t.Errorf("EndpointURL = %q, want %q", endpoint.EndpointURL, "https://updated.example.com/mcp")
	}
	if endpoint.TransportType != MCPTransportSSE {
		t.Errorf("TransportType = %q, want %q", endpoint.TransportType, MCPTransportSSE)
	}
}

// --- DeleteMCPServerTag ---

func TestDeleteMCPServerTag_EmptyArgs(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	if err := client.DeleteMCPServerTag(context.Background(), "", "key"); err == nil {
		t.Error("expected error for empty name")
	}
	if err := client.DeleteMCPServerTag(context.Background(), "my-server", ""); err == nil {
		t.Error("expected error for empty key")
	}
}

func TestDeleteMCPServerTag_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != mcpServersBasePath+"/my-server/tags/team" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	}))

	err := client.DeleteMCPServerTag(context.Background(), "my-server", "team")
	if err != nil {
		t.Fatalf("DeleteMCPServerTag() error = %v", err)
	}
}

// --- UpdateMCPServer ---

func TestUpdateMCPServer_EmptyName(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	_, err := client.UpdateMCPServer(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestUpdateMCPServer_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != mcpServersBasePath+"/my-server" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["display_name"] != "My Server" {
			t.Errorf("unexpected display_name: %v", body["display_name"])
		}
		if _, ok := body["description"]; ok {
			t.Errorf("expected description to be omitted, got %v", body["description"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"name":         "my-server",
			"display_name": "My Server",
		})
	}))

	server, err := client.UpdateMCPServer(context.Background(), "my-server",
		WithUpdatedServerDisplayName("My Server"),
	)
	if err != nil {
		t.Fatalf("UpdateMCPServer() error = %v", err)
	}
	if server.DisplayName != "My Server" {
		t.Errorf("DisplayName = %q, want %q", server.DisplayName, "My Server")
	}
}

// --- DeleteMCPServer ---

func TestDeleteMCPServer_EmptyName(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	err := client.DeleteMCPServer(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestDeleteMCPServer_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != mcpServersBasePath+"/my-server" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	}))

	err := client.DeleteMCPServer(context.Background(), "my-server")
	if err != nil {
		t.Fatalf("DeleteMCPServer() error = %v", err)
	}
}

// --- SetMCPServerAlias / GetMCPServerVersionByAlias / DeleteMCPServerAlias ---

func TestSetMCPServerAlias_EmptyArgs(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	if err := client.SetMCPServerAlias(context.Background(), "", "production", "1.0.0"); err == nil {
		t.Error("expected error for empty name")
	}
	if err := client.SetMCPServerAlias(context.Background(), "my-server", "", "1.0.0"); err == nil {
		t.Error("expected error for empty alias")
	}
	if err := client.SetMCPServerAlias(context.Background(), "my-server", "production", ""); err == nil {
		t.Error("expected error for empty version")
	}
}

func TestSetMCPServerAlias_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != mcpServersBasePath+"/my-server/aliases" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["alias"] != "production" || body["version"] != "1.0.0" {
			t.Errorf("unexpected body: %v", body)
		}

		w.WriteHeader(http.StatusOK)
	}))

	err := client.SetMCPServerAlias(context.Background(), "my-server", "production", "1.0.0")
	if err != nil {
		t.Fatalf("SetMCPServerAlias() error = %v", err)
	}
}

func TestGetMCPServerVersionByAlias_EmptyArgs(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	if _, err := client.GetMCPServerVersionByAlias(context.Background(), "", "production"); err == nil {
		t.Error("expected error for empty name")
	}
	if _, err := client.GetMCPServerVersionByAlias(context.Background(), "my-server", ""); err == nil {
		t.Error("expected error for empty alias")
	}
}

func TestGetMCPServerVersionByAlias_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != mcpServersBasePath+"/my-server/aliases/production" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"name":    "my-server",
			"version": "1.0.0",
			"status":  "active",
			"server_json": map[string]any{
				"name":    "my-server",
				"version": "1.0.0",
			},
		})
	}))

	version, err := client.GetMCPServerVersionByAlias(context.Background(), "my-server", "production")
	if err != nil {
		t.Fatalf("GetMCPServerVersionByAlias() error = %v", err)
	}
	if version.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", version.Version, "1.0.0")
	}
}

func TestDeleteMCPServerAlias_EmptyArgs(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	if err := client.DeleteMCPServerAlias(context.Background(), "", "production"); err == nil {
		t.Error("expected error for empty name")
	}
	if err := client.DeleteMCPServerAlias(context.Background(), "my-server", ""); err == nil {
		t.Error("expected error for empty alias")
	}
}

func TestDeleteMCPServerAlias_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != mcpServersBasePath+"/my-server/aliases/production" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	}))

	err := client.DeleteMCPServerAlias(context.Background(), "my-server", "production")
	if err != nil {
		t.Fatalf("DeleteMCPServerAlias() error = %v", err)
	}
}

// --- UpdateMCPServerVersion ---

func TestUpdateMCPServerVersion_EmptyArgs(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	if _, err := client.UpdateMCPServerVersion(context.Background(), "", "1.0.0"); err == nil {
		t.Error("expected error for empty name")
	}
	if _, err := client.UpdateMCPServerVersion(context.Background(), "my-server", ""); err == nil {
		t.Error("expected error for empty version")
	}
}

func TestUpdateMCPServerVersion_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != mcpServersBasePath+"/my-server/versions/1.0.0" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["status"] != "active" {
			t.Errorf("unexpected status: %v", body["status"])
		}
		connectOptions, ok := body["connect_options"].(map[string]any)
		if !ok {
			t.Fatalf("expected connect_options in body, got %v", body["connect_options"])
		}
		npm, ok := connectOptions["npm"].(map[string]any)
		if !ok || npm["hidden"] != true {
			t.Errorf("unexpected connect_options.npm: %v", connectOptions["npm"])
		}
		if _, ok := body["tools"]; ok {
			t.Errorf("expected tools to be omitted, got %v", body["tools"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"name":    "my-server",
			"version": "1.0.0",
			"status":  "active",
			"server_json": map[string]any{
				"name":    "my-server",
				"version": "1.0.0",
			},
			"connect_options": map[string]any{
				"npm": map[string]any{"hidden": true},
			},
		})
	}))

	version, err := client.UpdateMCPServerVersion(context.Background(), "my-server", "1.0.0",
		WithUpdatedVersionStatus(MCPServerVersionStatusActive),
		WithUpdatedVersionConnectOptions(map[string]ConnectOptionSettings{
			"npm": {Hidden: true},
		}),
	)
	if err != nil {
		t.Fatalf("UpdateMCPServerVersion() error = %v", err)
	}
	if version.Status != MCPServerVersionStatusActive {
		t.Errorf("Status = %q, want %q", version.Status, MCPServerVersionStatusActive)
	}
	if !version.ConnectOptions["npm"].Hidden {
		t.Errorf("ConnectOptions[npm].Hidden = false, want true")
	}
}

// --- DeleteMCPServerVersion ---

func TestDeleteMCPServerVersion_EmptyArgs(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	if err := client.DeleteMCPServerVersion(context.Background(), "", "1.0.0"); err == nil {
		t.Error("expected error for empty name")
	}
	if err := client.DeleteMCPServerVersion(context.Background(), "my-server", ""); err == nil {
		t.Error("expected error for empty version")
	}
}

func TestDeleteMCPServerVersion_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != mcpServersBasePath+"/my-server/versions/1.0.0" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	}))

	err := client.DeleteMCPServerVersion(context.Background(), "my-server", "1.0.0")
	if err != nil {
		t.Fatalf("DeleteMCPServerVersion() error = %v", err)
	}
}

// --- SetMCPServerVersionTag / DeleteMCPServerVersionTag ---

func TestSetMCPServerVersionTag_EmptyArgs(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	if err := client.SetMCPServerVersionTag(context.Background(), "", "1.0.0", "key", "value"); err == nil {
		t.Error("expected error for empty name")
	}
	if err := client.SetMCPServerVersionTag(context.Background(), "my-server", "", "key", "value"); err == nil {
		t.Error("expected error for empty version")
	}
	if err := client.SetMCPServerVersionTag(context.Background(), "my-server", "1.0.0", "", "value"); err == nil {
		t.Error("expected error for empty key")
	}
}

func TestSetMCPServerVersionTag_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != mcpServersBasePath+"/my-server/versions/1.0.0/tags" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["key"] != "stage" || body["value"] != "prod" {
			t.Errorf("unexpected body: %v", body)
		}

		w.WriteHeader(http.StatusOK)
	}))

	err := client.SetMCPServerVersionTag(context.Background(), "my-server", "1.0.0", "stage", "prod")
	if err != nil {
		t.Fatalf("SetMCPServerVersionTag() error = %v", err)
	}
}

func TestDeleteMCPServerVersionTag_EmptyArgs(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	if err := client.DeleteMCPServerVersionTag(context.Background(), "", "1.0.0", "key"); err == nil {
		t.Error("expected error for empty name")
	}
	if err := client.DeleteMCPServerVersionTag(context.Background(), "my-server", "", "key"); err == nil {
		t.Error("expected error for empty version")
	}
	if err := client.DeleteMCPServerVersionTag(context.Background(), "my-server", "1.0.0", ""); err == nil {
		t.Error("expected error for empty key")
	}
}

func TestDeleteMCPServerVersionTag_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != mcpServersBasePath+"/my-server/versions/1.0.0/tags/stage" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	}))

	err := client.DeleteMCPServerVersionTag(context.Background(), "my-server", "1.0.0", "stage")
	if err != nil {
		t.Fatalf("DeleteMCPServerVersionTag() error = %v", err)
	}
}

// --- CreateMCPServerVersion with connect options ---

func TestCreateMCPServerVersion_WithConnectOptions(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		connectOptions, ok := body["connect_options"].(map[string]any)
		if !ok {
			t.Fatalf("expected connect_options in body, got %v", body["connect_options"])
		}
		npm, ok := connectOptions["npm"].(map[string]any)
		if !ok || npm["hidden"] != true {
			t.Errorf("unexpected connect_options.npm: %v", connectOptions["npm"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"name":    "my-server",
			"version": "1.0.0",
			"status":  "draft",
			"server_json": map[string]any{
				"name":    "my-server",
				"version": "1.0.0",
			},
			"connect_options": map[string]any{
				"npm": map[string]any{"hidden": true},
			},
		})
	}))

	version, err := client.CreateMCPServerVersion(context.Background(), "my-server",
		map[string]any{"name": "my-server", "version": "1.0.0"},
		WithVersionConnectOptions(map[string]ConnectOptionSettings{
			"npm": {Hidden: true},
		}),
	)
	if err != nil {
		t.Fatalf("CreateMCPServerVersion() error = %v", err)
	}
	if !version.ConnectOptions["npm"].Hidden {
		t.Errorf("ConnectOptions[npm].Hidden = false, want true")
	}
}
