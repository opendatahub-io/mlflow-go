package artifacts

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"reflect"
	"strings"

	"github.com/opendatahub-io/mlflow-go/internal/artifact"
	"github.com/opendatahub-io/mlflow-go/internal/gen/mlflowpb"
	"github.com/opendatahub-io/mlflow-go/internal/transport"
)

const (
	maxArtifactUploadSize = 100 << 20 // 100 MiB

	// artifactProxyPrefix is the mlflow-artifacts proxy endpoint prefix for
	// path-based artifact upload/download.
	artifactProxyPrefix = "/api/2.0/mlflow-artifacts/artifacts/"
)

// Client provides access to MLflow run artifacts.
// It is safe for concurrent use.
type Client struct {
	transport *transport.Client
	store     *artifact.Store
}

// NewClient creates a new Artifacts client.
// This is typically called internally by the root mlflow.Client.
func NewClient(t *transport.Client) *Client {
	return &Client{
		transport: t,
		store:     artifact.NewStore(t),
	}
}

// ListArtifacts lists artifacts logged for a run.
func (c *Client) ListArtifacts(ctx context.Context, runID string, opts ...ListArtifactsOption) (*ListArtifactsResult, error) {
	if runID == "" {
		return nil, fmt.Errorf("mlflow: run ID is required")
	}

	o := &listArtifactsOptions{}
	for _, opt := range opts {
		opt(o)
	}

	query := url.Values{"run_id": []string{runID}}
	if o.path != "" {
		query.Set("path", o.path)
	}
	if o.pageToken != "" {
		query.Set("page_token", o.pageToken)
	}

	var resp mlflowpb.ListArtifacts_Response
	if err := c.transport.Get(ctx, "/api/2.0/mlflow/artifacts/list", query, &resp); err != nil {
		return nil, fmt.Errorf("failed to list artifacts: %w", err)
	}

	return listArtifactsResultFromProto(&resp), nil
}

// LogArtifact uploads a single artifact file to a run.
func (c *Client) LogArtifact(ctx context.Context, runID, artifactPath string, r io.Reader, opts ...LogArtifactOption) error {
	if runID == "" {
		return fmt.Errorf("mlflow: run ID is required")
	}
	if artifactPath == "" {
		return fmt.Errorf("mlflow: artifact path is required")
	}
	if isNilReader(r) {
		return fmt.Errorf("mlflow: artifact reader is required")
	}

	o := &logArtifactOptions{}
	for _, opt := range opts {
		opt(o)
	}

	artifactURI, err := c.store.ArtifactURI(ctx, runID)
	if err != nil {
		return err
	}

	maxSize := int64(maxArtifactUploadSize)
	if artifact.SupportsTrackingServerArtifacts(artifactURI) {
		// Legacy /ajax-api/2.0/mlflow/upload-artifact rejects bodies over 10 MiB
		// (MLflow v3.12.0 upload_artifact_handler). Cap before buffering.
		maxSize = artifact.MaxTrackingServerUploadSize
	}

	content, err := readArtifactContent(r, maxSize)
	if err != nil {
		return err
	}

	uploadOpts := artifact.UploadOptions{
		ContentType: o.contentType,
		Expiration:  o.expiration,
	}
	if err := c.store.Upload(ctx, runID, artifactURI, artifactPath, content, uploadOpts); err != nil {
		return fmt.Errorf("failed to log artifact: %w", err)
	}

	return nil
}

// DownloadArtifact opens a single artifact file from a run for streaming download.
// The caller must close the returned ReadCloser.
func (c *Client) DownloadArtifact(ctx context.Context, runID, artifactPath string, opts ...DownloadArtifactOption) (io.ReadCloser, error) {
	if runID == "" {
		return nil, fmt.Errorf("mlflow: run ID is required")
	}
	if artifactPath == "" {
		return nil, fmt.Errorf("mlflow: artifact path is required")
	}

	o := &downloadArtifactOptions{}
	for _, opt := range opts {
		opt(o)
	}

	artifactURI, err := c.store.ArtifactURI(ctx, runID)
	if err != nil {
		return nil, err
	}

	downloadOpts := artifact.DownloadOptions{Expiration: o.expiration}
	rc, err := c.store.Download(ctx, runID, artifactURI, artifactPath, downloadOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to download artifact: %w", err)
	}

	return rc, nil
}

// UploadArtifact uploads an artifact to the mlflow-artifacts proxy using a
// storage path (no run ID required).
// Uses PUT /api/2.0/mlflow-artifacts/artifacts/{artifactPath}.
//
// The artifact is streamed from r with no size cap and without being buffered in
// memory, so arbitrarily large objects can be uploaded. A leading slash on
// artifactPath is trimmed so that both "foo/bar" and "/foo/bar" address the same
// object.
func (c *Client) UploadArtifact(ctx context.Context, artifactPath string, r io.Reader, opts ...UploadArtifactOption) error {
	if artifactPath == "" {
		return fmt.Errorf("mlflow: artifact path is required")
	}
	if isNilReader(r) {
		return fmt.Errorf("mlflow: artifact reader is required")
	}

	o := &uploadArtifactOptions{}
	for _, opt := range opts {
		opt(o)
	}

	contentType := o.contentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	proxyPath, err := proxyArtifactPath(artifactPath)
	if err != nil {
		return err
	}
	if err := c.transport.PutReader(ctx, proxyPath, r, contentType); err != nil {
		return fmt.Errorf("failed to upload artifact: %w", err)
	}
	return nil
}

// DownloadArtifactByPath downloads an artifact from the mlflow-artifacts proxy
// using a storage path (no run ID required).
// Uses GET /api/2.0/mlflow-artifacts/artifacts/{artifactPath}.
// The caller must close the returned ReadCloser.
//
// The artifact is streamed with no size cap, so arbitrarily large objects can be
// read incrementally. A leading slash on artifactPath is trimmed so that both
// "foo/bar" and "/foo/bar" address the same object.
func (c *Client) DownloadArtifactByPath(ctx context.Context, artifactPath string) (io.ReadCloser, error) {
	if artifactPath == "" {
		return nil, fmt.Errorf("mlflow: artifact path is required")
	}

	proxyPath, err := proxyArtifactPath(artifactPath)
	if err != nil {
		return nil, err
	}
	rc, err := c.transport.GetBodyStream(ctx, proxyPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download artifact: %w", err)
	}
	return rc, nil
}

// proxyArtifactPath builds the mlflow-artifacts proxy request path for a storage
// path. All leading slashes are trimmed so an "absolute" path does not produce a
// doubled slash after the prefix, which some artifact backends reject or resolve
// to a different key. It rejects a path that is empty (or only slashes) or that
// contains a "." or ".." segment: the transport sends storage paths verbatim
// without collapsing such segments, so they could traverse outside the intended
// key.
func proxyArtifactPath(artifactPath string) (string, error) {
	trimmed := strings.TrimLeft(artifactPath, "/")
	if trimmed == "" {
		return "", fmt.Errorf("mlflow: artifact path is required")
	}
	for _, seg := range strings.Split(trimmed, "/") {
		if seg == "." || seg == ".." {
			return "", fmt.Errorf("mlflow: artifact path must not contain %q or %q segments", ".", "..")
		}
	}
	return artifactProxyPrefix + trimmed, nil
}

// isNilReader reports whether r is either an untyped nil interface or a typed
// nil pointer/interface value. The plain r == nil check misses a typed nil
// (e.g. a (*bytes.Buffer)(nil) stored in an io.Reader), which would otherwise
// panic when read.
func isNilReader(r io.Reader) bool {
	if r == nil {
		return true
	}
	switch v := reflect.ValueOf(r); v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		return v.IsNil()
	default:
		return false
	}
}

func readArtifactContent(r io.Reader, maxSize int64) ([]byte, error) {
	limited := io.LimitReader(r, maxSize+1)
	content, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("failed to read artifact content: %w", err)
	}
	if int64(len(content)) > maxSize {
		return nil, fmt.Errorf("artifact content exceeds maximum upload size of %d bytes", maxSize)
	}
	return content, nil
}
