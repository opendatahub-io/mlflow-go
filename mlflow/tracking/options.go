package tracking

import "time"

// createExperimentOptions holds configuration for a CreateExperiment call.
type createExperimentOptions struct {
	artifactLocation string
	tags             map[string]string
}

// CreateExperimentOption configures a CreateExperiment call.
type CreateExperimentOption func(*createExperimentOptions)

// WithArtifactLocation sets the artifact storage location for the experiment.
func WithArtifactLocation(loc string) CreateExperimentOption {
	return func(o *createExperimentOptions) {
		o.artifactLocation = loc
	}
}

// WithExperimentTags sets tags on the experiment. Tags are merged with any
// previously set tags (e.g., from WithExperimentKind).
func WithExperimentTags(tags map[string]string) CreateExperimentOption {
	return func(o *createExperimentOptions) {
		if len(tags) == 0 {
			return
		}
		if o.tags == nil {
			o.tags = make(map[string]string, len(tags))
		}
		for k, v := range tags {
			o.tags[k] = v
		}
	}
}

// WithExperimentKind sets the experiment kind, which controls how the MLflow UI
// displays the experiment. If not set, the UI will prompt users to select a type.
func WithExperimentKind(kind ExperimentKind) CreateExperimentOption {
	return func(o *createExperimentOptions) {
		if o.tags == nil {
			o.tags = make(map[string]string)
		}
		o.tags["mlflow.experimentKind"] = string(kind)
	}
}

// createRunOptions holds configuration for a CreateRun call.
type createRunOptions struct {
	runName   string
	startTime *time.Time
	tags      map[string]string
}

// CreateRunOption configures a CreateRun call.
type CreateRunOption func(*createRunOptions)

// WithRunName sets the name for the run.
func WithRunName(name string) CreateRunOption {
	return func(o *createRunOptions) {
		o.runName = name
	}
}

// WithStartTime sets the start time for the run.
// If not set, the server uses the current time.
func WithStartTime(t time.Time) CreateRunOption {
	return func(o *createRunOptions) {
		o.startTime = &t
	}
}

// WithRunTags sets tags on the run at creation time.
func WithRunTags(tags map[string]string) CreateRunOption {
	return func(o *createRunOptions) {
		o.tags = tags
	}
}

// searchExperimentsOptions holds configuration for a SearchExperiments call.
type searchExperimentsOptions struct {
	filter     string
	maxResults int
	pageToken  string
	orderBy    []string
	viewType   ViewType
}

// SearchExperimentsOption configures a SearchExperiments call.
type SearchExperimentsOption func(*searchExperimentsOptions)

// WithExperimentsFilter sets the search filter string for experiments.
func WithExperimentsFilter(filter string) SearchExperimentsOption {
	return func(o *searchExperimentsOptions) {
		o.filter = filter
	}
}

// WithExperimentsMaxResults sets the maximum number of experiments to return.
func WithExperimentsMaxResults(n int) SearchExperimentsOption {
	return func(o *searchExperimentsOptions) {
		o.maxResults = n
	}
}

// WithExperimentsPageToken sets the pagination token for experiments.
func WithExperimentsPageToken(token string) SearchExperimentsOption {
	return func(o *searchExperimentsOptions) {
		o.pageToken = token
	}
}

// WithExperimentsOrderBy sets the sort order for experiments.
// Examples: "name ASC", "creation_time DESC".
func WithExperimentsOrderBy(fields ...string) SearchExperimentsOption {
	return func(o *searchExperimentsOptions) {
		o.orderBy = fields
	}
}

// WithExperimentsViewType sets the view type filter for experiments.
func WithExperimentsViewType(viewType ViewType) SearchExperimentsOption {
	return func(o *searchExperimentsOptions) {
		o.viewType = viewType
	}
}

// searchRunsOptions holds configuration for a SearchRuns call.
type searchRunsOptions struct {
	filter     string
	maxResults int
	pageToken  string
	orderBy    []string
	viewType   ViewType
}

// SearchRunsOption configures a SearchRuns call.
type SearchRunsOption func(*searchRunsOptions)

// WithRunsFilter sets the search filter string for runs.
// Uses MLflow filter syntax (e.g., "metrics.rmse < 1" or "params.model = 'sklearn'").
func WithRunsFilter(filter string) SearchRunsOption {
	return func(o *searchRunsOptions) {
		o.filter = filter
	}
}

// WithRunsMaxResults sets the maximum number of runs to return.
func WithRunsMaxResults(n int) SearchRunsOption {
	return func(o *searchRunsOptions) {
		o.maxResults = n
	}
}

// WithRunsPageToken sets the pagination token for runs.
func WithRunsPageToken(token string) SearchRunsOption {
	return func(o *searchRunsOptions) {
		o.pageToken = token
	}
}

// WithRunsOrderBy sets the sort order for runs.
// Examples: "start_time DESC", "metrics.rmse ASC".
func WithRunsOrderBy(fields ...string) SearchRunsOption {
	return func(o *searchRunsOptions) {
		o.orderBy = fields
	}
}

// WithRunsViewType sets the view type filter for runs.
func WithRunsViewType(viewType ViewType) SearchRunsOption {
	return func(o *searchRunsOptions) {
		o.viewType = viewType
	}
}

// logMetricOptions holds configuration for a LogMetric call.
type logMetricOptions struct {
	step      *int64
	timestamp *time.Time
}

// LogMetricOption configures a LogMetric call.
type LogMetricOption func(*logMetricOptions)

// WithStep sets the step number for the metric.
func WithStep(step int64) LogMetricOption {
	return func(o *logMetricOptions) {
		o.step = &step
	}
}

// WithTimestamp sets the timestamp for the metric.
// If not set, the server uses the current time.
func WithTimestamp(t time.Time) LogMetricOption {
	return func(o *logMetricOptions) {
		o.timestamp = &t
	}
}

// updateRunOptions holds configuration for an UpdateRun call.
type updateRunOptions struct {
	status  *RunStatus
	endTime *time.Time
	runName string
}

// UpdateRunOption configures an UpdateRun call.
type UpdateRunOption func(*updateRunOptions)

// WithStatus sets the run status.
func WithStatus(status RunStatus) UpdateRunOption {
	return func(o *updateRunOptions) {
		o.status = &status
	}
}

// WithEndTime sets the end time for the run.
func WithEndTime(t time.Time) UpdateRunOption {
	return func(o *updateRunOptions) {
		o.endTime = &t
	}
}

// WithRunNameUpdate sets a new name for the run.
func WithRunNameUpdate(name string) UpdateRunOption {
	return func(o *updateRunOptions) {
		o.runName = name
	}
}
