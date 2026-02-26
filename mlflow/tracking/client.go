package tracking

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"time"

	"github.com/opendatahub-io/mlflow-go/internal/conv"
	"github.com/opendatahub-io/mlflow-go/internal/gen/mlflowpb"
	"github.com/opendatahub-io/mlflow-go/internal/transport"
)

// defaultSearchMaxResults is the default page size for search operations.
// Matches the MLflow Python SDK default.
const defaultSearchMaxResults = 1000

// Client provides access to MLflow experiment tracking.
// It is safe for concurrent use.
type Client struct {
	transport *transport.Client
}

// NewClient creates a new Tracking client.
// This is typically called internally by the root mlflow.Client.
func NewClient(t *transport.Client) *Client {
	return &Client{transport: t}
}

// --- Experiment operations ---

// CreateExperiment creates a new experiment and returns its ID.
func (c *Client) CreateExperiment(ctx context.Context, name string, opts ...CreateExperimentOption) (string, error) {
	if name == "" {
		return "", fmt.Errorf("mlflow: experiment name is required")
	}

	o := &createExperimentOptions{}
	for _, opt := range opts {
		opt(o)
	}

	req := &mlflowpb.CreateExperiment{
		Name: &name,
	}

	if o.artifactLocation != "" {
		req.ArtifactLocation = &o.artifactLocation
	}

	for k, v := range o.tags {
		req.Tags = append(req.Tags, &mlflowpb.ExperimentTag{Key: conv.Ptr(k), Value: conv.Ptr(v)})
	}

	var resp mlflowpb.CreateExperiment_Response

	err := c.transport.Post(ctx, "/api/2.0/mlflow/experiments/create", req, &resp)
	if err != nil {
		return "", fmt.Errorf("failed to create experiment: %w", err)
	}

	return resp.GetExperimentId(), nil
}

// GetExperiment retrieves an experiment by ID.
func (c *Client) GetExperiment(ctx context.Context, experimentID string) (*Experiment, error) {
	if experimentID == "" {
		return nil, fmt.Errorf("mlflow: experiment ID is required")
	}

	query := url.Values{
		"experiment_id": []string{experimentID},
	}

	var resp mlflowpb.GetExperiment_Response

	err := c.transport.Get(ctx, "/api/2.0/mlflow/experiments/get", query, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get experiment: %w", err)
	}

	exp := experimentFromProto(resp.Experiment)

	return &exp, nil
}

// GetExperimentByName retrieves an experiment by name.
func (c *Client) GetExperimentByName(ctx context.Context, name string) (*Experiment, error) {
	if name == "" {
		return nil, fmt.Errorf("mlflow: experiment name is required")
	}

	query := url.Values{
		"experiment_name": []string{name},
	}

	var resp mlflowpb.GetExperimentByName_Response

	err := c.transport.Get(ctx, "/api/2.0/mlflow/experiments/get-by-name", query, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get experiment by name: %w", err)
	}

	exp := experimentFromProto(resp.Experiment)

	return &exp, nil
}

// DeleteExperiment marks an experiment for deletion.
func (c *Client) DeleteExperiment(ctx context.Context, experimentID string) error {
	if experimentID == "" {
		return fmt.Errorf("mlflow: experiment ID is required")
	}

	req := &mlflowpb.DeleteExperiment{
		ExperimentId: &experimentID,
	}

	var resp mlflowpb.DeleteExperiment_Response

	err := c.transport.Post(ctx, "/api/2.0/mlflow/experiments/delete", req, &resp)
	if err != nil {
		return fmt.Errorf("failed to delete experiment: %w", err)
	}

	return nil
}

// UpdateExperiment renames an experiment.
func (c *Client) UpdateExperiment(ctx context.Context, experimentID, name string) error {
	if experimentID == "" {
		return fmt.Errorf("mlflow: experiment ID is required")
	}
	if name == "" {
		return fmt.Errorf("mlflow: experiment name is required")
	}

	req := &mlflowpb.UpdateExperiment{
		ExperimentId: &experimentID,
		NewName:      &name,
	}

	var resp mlflowpb.UpdateExperiment_Response

	err := c.transport.Post(ctx, "/api/2.0/mlflow/experiments/update", req, &resp)
	if err != nil {
		return fmt.Errorf("failed to update experiment: %w", err)
	}

	return nil
}

// SearchExperiments searches for experiments matching the given criteria.
func (c *Client) SearchExperiments(ctx context.Context, opts ...SearchExperimentsOption) (*ExperimentList, error) {
	o := &searchExperimentsOptions{
		maxResults: defaultSearchMaxResults,
	}
	for _, opt := range opts {
		opt(o)
	}

	if o.maxResults <= 0 {
		return nil, fmt.Errorf("mlflow: max results must be positive")
	}

	req := &mlflowpb.SearchExperiments{}

	if o.filter != "" {
		req.Filter = &o.filter
	}
	maxResults := int64(o.maxResults) // int→int64 widening: always safe
	req.MaxResults = &maxResults
	if o.pageToken != "" {
		req.PageToken = &o.pageToken
	}
	if len(o.orderBy) > 0 {
		req.OrderBy = o.orderBy
	}
	if o.viewType != "" {
		vt, ok := viewTypeToProto[o.viewType]
		if !ok {
			return nil, fmt.Errorf("mlflow: invalid view type: %s", o.viewType)
		}
		req.ViewType = &vt
	}

	var resp mlflowpb.SearchExperiments_Response

	err := c.transport.Post(ctx, "/api/2.0/mlflow/experiments/search", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to search experiments: %w", err)
	}

	result := &ExperimentList{
		Experiments:   make([]Experiment, 0, len(resp.Experiments)),
		NextPageToken: resp.GetNextPageToken(),
	}

	for _, exp := range resp.Experiments {
		result.Experiments = append(result.Experiments, experimentFromProto(exp))
	}

	return result, nil
}

// SetExperimentTag sets a tag on an experiment.
func (c *Client) SetExperimentTag(ctx context.Context, experimentID, key, value string) error {
	if experimentID == "" {
		return fmt.Errorf("mlflow: experiment ID is required")
	}
	if key == "" {
		return fmt.Errorf("mlflow: tag key is required")
	}

	req := &mlflowpb.SetExperimentTag{
		ExperimentId: &experimentID,
		Key:          &key,
		Value:        &value,
	}

	var resp mlflowpb.SetExperimentTag_Response

	err := c.transport.Post(ctx, "/api/2.0/mlflow/experiments/set-experiment-tag", req, &resp)
	if err != nil {
		return fmt.Errorf("failed to set experiment tag: %w", err)
	}

	return nil
}

// --- Run operations ---

// CreateRun creates a new run in the specified experiment.
func (c *Client) CreateRun(ctx context.Context, experimentID string, opts ...CreateRunOption) (*Run, error) {
	if experimentID == "" {
		return nil, fmt.Errorf("mlflow: experiment ID is required")
	}

	o := &createRunOptions{}
	for _, opt := range opts {
		opt(o)
	}

	req := &mlflowpb.CreateRun{
		ExperimentId: &experimentID,
	}

	if o.runName != "" {
		req.RunName = &o.runName
	}
	if o.startTime != nil {
		ms := o.startTime.UnixMilli()
		req.StartTime = &ms
	}

	for k, v := range o.tags {
		req.Tags = append(req.Tags, &mlflowpb.RunTag{Key: conv.Ptr(k), Value: conv.Ptr(v)})
	}

	var resp mlflowpb.CreateRun_Response

	err := c.transport.Post(ctx, "/api/2.0/mlflow/runs/create", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to create run: %w", err)
	}

	run := runFromProto(resp.Run)

	return &run, nil
}

// GetRun retrieves a run by ID.
func (c *Client) GetRun(ctx context.Context, runID string) (*Run, error) {
	if runID == "" {
		return nil, fmt.Errorf("mlflow: run ID is required")
	}

	query := url.Values{
		"run_id": []string{runID},
	}

	var resp mlflowpb.GetRun_Response

	err := c.transport.Get(ctx, "/api/2.0/mlflow/runs/get", query, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get run: %w", err)
	}

	run := runFromProto(resp.Run)

	return &run, nil
}

// UpdateRun updates a run's status, name, and/or end time.
// All fields are optional — only provided options are sent to the server.
func (c *Client) UpdateRun(ctx context.Context, runID string, opts ...UpdateRunOption) (*RunInfo, error) {
	if runID == "" {
		return nil, fmt.Errorf("mlflow: run ID is required")
	}

	o := &updateRunOptions{}
	for _, opt := range opts {
		opt(o)
	}

	req := &mlflowpb.UpdateRun{
		RunId: &runID,
	}

	if o.status != nil {
		protoStatus, ok := runStatusToProto[*o.status]
		if !ok {
			return nil, fmt.Errorf("mlflow: invalid run status: %s", *o.status)
		}
		req.Status = &protoStatus
	}
	if o.endTime != nil {
		ms := o.endTime.UnixMilli()
		req.EndTime = &ms
	}
	if o.runName != "" {
		req.RunName = &o.runName
	}

	var resp mlflowpb.UpdateRun_Response

	err := c.transport.Post(ctx, "/api/2.0/mlflow/runs/update", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to update run: %w", err)
	}

	info := runInfoFromProto(resp.RunInfo)

	return &info, nil
}

// DeleteRun marks a run for deletion.
func (c *Client) DeleteRun(ctx context.Context, runID string) error {
	if runID == "" {
		return fmt.Errorf("mlflow: run ID is required")
	}

	req := &mlflowpb.DeleteRun{
		RunId: &runID,
	}

	var resp mlflowpb.DeleteRun_Response

	err := c.transport.Post(ctx, "/api/2.0/mlflow/runs/delete", req, &resp)
	if err != nil {
		return fmt.Errorf("failed to delete run: %w", err)
	}

	return nil
}

// SearchRuns searches for runs in the specified experiments.
func (c *Client) SearchRuns(ctx context.Context, experimentIDs []string, opts ...SearchRunsOption) (*RunList, error) {
	if len(experimentIDs) == 0 {
		return nil, fmt.Errorf("mlflow: at least one experiment ID is required")
	}

	o := &searchRunsOptions{
		maxResults: defaultSearchMaxResults,
	}
	for _, opt := range opts {
		opt(o)
	}

	if o.maxResults <= 0 {
		return nil, fmt.Errorf("mlflow: max results must be positive")
	}

	req := &mlflowpb.SearchRuns{
		ExperimentIds: experimentIDs,
	}

	if o.filter != "" {
		req.Filter = &o.filter
	}
	n := o.maxResults
	if n > math.MaxInt32 {
		n = math.MaxInt32
	}
	maxResults := int32(n) //nolint:gosec // bounds checked above
	req.MaxResults = &maxResults
	if o.pageToken != "" {
		req.PageToken = &o.pageToken
	}
	if len(o.orderBy) > 0 {
		req.OrderBy = o.orderBy
	}
	if o.viewType != "" {
		vt, ok := viewTypeToProto[o.viewType]
		if !ok {
			return nil, fmt.Errorf("mlflow: invalid view type: %s", o.viewType)
		}
		req.RunViewType = &vt
	}

	var resp mlflowpb.SearchRuns_Response

	err := c.transport.Post(ctx, "/api/2.0/mlflow/runs/search", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to search runs: %w", err)
	}

	result := &RunList{
		Runs:          make([]Run, 0, len(resp.Runs)),
		NextPageToken: resp.GetNextPageToken(),
	}

	for _, r := range resp.Runs {
		result.Runs = append(result.Runs, runFromProto(r))
	}

	return result, nil
}

// --- Logging operations ---

// LogMetric logs a metric value for a run.
func (c *Client) LogMetric(ctx context.Context, runID, key string, value float64, opts ...LogMetricOption) error {
	if runID == "" {
		return fmt.Errorf("mlflow: run ID is required")
	}
	if key == "" {
		return fmt.Errorf("mlflow: metric key is required")
	}

	o := &logMetricOptions{}
	for _, opt := range opts {
		opt(o)
	}

	ts := time.Now()
	if o.timestamp != nil {
		ts = *o.timestamp
	}
	tsMs := ts.UnixMilli()

	req := &mlflowpb.LogMetric{
		RunId:     &runID,
		Key:       &key,
		Value:     &value,
		Timestamp: &tsMs,
	}

	if o.step != nil {
		req.Step = o.step
	}

	var resp mlflowpb.LogMetric_Response

	err := c.transport.Post(ctx, "/api/2.0/mlflow/runs/log-metric", req, &resp)
	if err != nil {
		return fmt.Errorf("failed to log metric: %w", err)
	}

	return nil
}

// LogParam logs a parameter for a run.
func (c *Client) LogParam(ctx context.Context, runID, key, value string) error {
	if runID == "" {
		return fmt.Errorf("mlflow: run ID is required")
	}
	if key == "" {
		return fmt.Errorf("mlflow: param key is required")
	}

	req := &mlflowpb.LogParam{
		RunId: &runID,
		Key:   &key,
		Value: &value,
	}

	var resp mlflowpb.LogParam_Response

	err := c.transport.Post(ctx, "/api/2.0/mlflow/runs/log-parameter", req, &resp)
	if err != nil {
		return fmt.Errorf("failed to log param: %w", err)
	}

	return nil
}

// SetTag sets a tag on a run.
func (c *Client) SetTag(ctx context.Context, runID, key, value string) error {
	if runID == "" {
		return fmt.Errorf("mlflow: run ID is required")
	}
	if key == "" {
		return fmt.Errorf("mlflow: tag key is required")
	}

	req := &mlflowpb.SetTag{
		RunId: &runID,
		Key:   &key,
		Value: &value,
	}

	var resp mlflowpb.SetTag_Response

	err := c.transport.Post(ctx, "/api/2.0/mlflow/runs/set-tag", req, &resp)
	if err != nil {
		return fmt.Errorf("failed to set tag: %w", err)
	}

	return nil
}

// DeleteTag removes a tag from a run.
func (c *Client) DeleteTag(ctx context.Context, runID, key string) error {
	if runID == "" {
		return fmt.Errorf("mlflow: run ID is required")
	}
	if key == "" {
		return fmt.Errorf("mlflow: tag key is required")
	}

	req := &mlflowpb.DeleteTag{
		RunId: &runID,
		Key:   &key,
	}

	var resp mlflowpb.DeleteTag_Response

	err := c.transport.Post(ctx, "/api/2.0/mlflow/runs/delete-tag", req, &resp)
	if err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}

	return nil
}

// LogBatch logs a batch of metrics, params, and tags for a run.
func (c *Client) LogBatch(ctx context.Context, runID string, metrics []Metric, params []Param, tags map[string]string) error {
	if runID == "" {
		return fmt.Errorf("mlflow: run ID is required")
	}

	req := &mlflowpb.LogBatch{
		RunId: &runID,
	}

	now := time.Now()
	for _, m := range metrics {
		ts := now
		if !m.Timestamp.IsZero() {
			ts = m.Timestamp
		}
		tsMs := ts.UnixMilli()
		pbMetric := &mlflowpb.Metric{
			Key:       conv.Ptr(m.Key),
			Value:     conv.Ptr(m.Value),
			Step:      conv.Ptr(m.Step),
			Timestamp: &tsMs,
		}
		req.Metrics = append(req.Metrics, pbMetric)
	}

	for _, p := range params {
		req.Params = append(req.Params, &mlflowpb.Param{
			Key:   conv.Ptr(p.Key),
			Value: conv.Ptr(p.Value),
		})
	}

	for k, v := range tags {
		req.Tags = append(req.Tags, &mlflowpb.RunTag{Key: conv.Ptr(k), Value: conv.Ptr(v)})
	}

	var resp mlflowpb.LogBatch_Response

	err := c.transport.Post(ctx, "/api/2.0/mlflow/runs/log-batch", req, &resp)
	if err != nil {
		return fmt.Errorf("failed to log batch: %w", err)
	}

	return nil
}
