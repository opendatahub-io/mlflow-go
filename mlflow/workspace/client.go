package workspace

import (
	"context"
	"fmt"
	"net/url"

	internalerrors "github.com/opendatahub-io/mlflow-go/internal/errors"
	"github.com/opendatahub-io/mlflow-go/internal/transport"
)

// Client provides access to MLflow workspace management.
// It is safe for concurrent use.
type Client struct {
	transport *transport.Client
}

// NewClient creates a new Workspace client.
// This is typically called internally by the root mlflow.Client.
func NewClient(t *transport.Client) *Client {
	return &Client{transport: t}
}

// serverInfoResponse is the JSON shape of GET /api/3.0/mlflow/server-info.
type serverInfoResponse struct {
	WorkspacesEnabled bool `json:"workspaces_enabled"`
}

// GetServerInfo retrieves server feature flags, including workspace support.
func (c *Client) GetServerInfo(ctx context.Context) (*ServerInfo, error) {
	var resp serverInfoResponse
	if err := c.transport.Get(ctx, "/api/3.0/mlflow/server-info", nil, &resp); err != nil {
		return nil, fmt.Errorf("failed to get server info: %w", err)
	}
	return &ServerInfo{
		WorkspacesEnabled: resp.WorkspacesEnabled,
	}, nil
}

// workspaceResponse is the JSON shape returned by workspace CRUD endpoints.
type workspaceResponse struct {
	Workspace struct {
		Name string `json:"name"`
	} `json:"workspace"`
}

// GetWorkspace retrieves a workspace by name.
func (c *Client) GetWorkspace(ctx context.Context, name string) (*Workspace, error) {
	if name == "" {
		return nil, fmt.Errorf("mlflow: workspace name is required")
	}

	var resp workspaceResponse
	// The name is a single path segment ({workspace_name}); escape it here so
	// reserved characters are encoded once and never split the path.
	endpoint := "/api/3.0/mlflow/workspaces/" + url.PathEscape(name)
	if err := c.transport.GetEscaped(ctx, endpoint, nil, &resp); err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}
	return workspaceFromResponse(&resp), nil
}

// CreateWorkspace creates a new workspace.
func (c *Client) CreateWorkspace(ctx context.Context, name string) (*Workspace, error) {
	if name == "" {
		return nil, fmt.Errorf("mlflow: workspace name is required")
	}

	req := map[string]string{"name": name}
	var resp workspaceResponse
	if err := c.transport.Post(ctx, "/api/3.0/mlflow/workspaces", req, &resp); err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}
	return workspaceFromResponse(&resp), nil
}

// DeleteWorkspace deletes a workspace by name.
func (c *Client) DeleteWorkspace(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("mlflow: workspace name is required")
	}

	// See GetWorkspace: escape the single {workspace_name} path segment.
	endpoint := "/api/3.0/mlflow/workspaces/" + url.PathEscape(name)
	if err := c.transport.DeleteEscaped(ctx, endpoint, nil, nil); err != nil {
		return fmt.Errorf("failed to delete workspace: %w", err)
	}

	return nil
}

// EnsureWorkspace idempotently creates a workspace.
// It skips the "default" workspace (always exists) and handles
// RESOURCE_ALREADY_EXISTS races from concurrent callers.
func (c *Client) EnsureWorkspace(ctx context.Context, name string) (*Workspace, error) {
	if name == "" {
		return nil, fmt.Errorf("mlflow: workspace name is required")
	}
	if name == "default" {
		return c.GetWorkspace(ctx, name)
	}

	ws, err := c.CreateWorkspace(ctx, name)
	if err == nil {
		return ws, nil
	}
	if !internalerrors.IsAlreadyExists(err) {
		return nil, err
	}

	return c.GetWorkspace(ctx, name)
}

func workspaceFromResponse(resp *workspaceResponse) *Workspace {
	return &Workspace{
		Name: resp.Workspace.Name,
	}
}
