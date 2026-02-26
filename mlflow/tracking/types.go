// Package tracking provides types and operations for MLflow experiment tracking.
//
// Experiment tracking records runs of ML code, logging parameters, metrics,
// and tags for later comparison. This package provides a Go client for the
// MLflow tracking REST API.
package tracking

import (
	"time"

	"github.com/opendatahub-io/mlflow-go/internal/gen/mlflowpb"
)

// RunStatus represents the status of a run.
type RunStatus string

const (
	RunStatusRunning   RunStatus = "RUNNING"
	RunStatusScheduled RunStatus = "SCHEDULED"
	RunStatusFinished  RunStatus = "FINISHED"
	RunStatusFailed    RunStatus = "FAILED"
	RunStatusKilled    RunStatus = "KILLED"
)

// runStatusToProto maps domain RunStatus to protobuf RunStatus enum values.
var runStatusToProto = map[RunStatus]mlflowpb.RunStatus{
	RunStatusRunning:   mlflowpb.RunStatus_RUNNING,
	RunStatusScheduled: mlflowpb.RunStatus_SCHEDULED,
	RunStatusFinished:  mlflowpb.RunStatus_FINISHED,
	RunStatusFailed:    mlflowpb.RunStatus_FAILED,
	RunStatusKilled:    mlflowpb.RunStatus_KILLED,
}

// protoToRunStatus maps protobuf RunStatus enum values to domain RunStatus.
var protoToRunStatus = map[mlflowpb.RunStatus]RunStatus{
	mlflowpb.RunStatus_RUNNING:   RunStatusRunning,
	mlflowpb.RunStatus_SCHEDULED: RunStatusScheduled,
	mlflowpb.RunStatus_FINISHED:  RunStatusFinished,
	mlflowpb.RunStatus_FAILED:    RunStatusFailed,
	mlflowpb.RunStatus_KILLED:    RunStatusKilled,
}

// ExperimentKind classifies the type of work an experiment tracks.
// The MLflow UI uses this to customize the experiment view.
// Set via WithExperimentKind when creating an experiment.
type ExperimentKind string

const (
	ExperimentKindMLDevelopment    ExperimentKind = "custom_model_development"
	ExperimentKindGenAIDevelopment ExperimentKind = "genai_development"
	ExperimentKindFineTuning       ExperimentKind = "finetuning"
	ExperimentKindForecasting      ExperimentKind = "forecasting"
	ExperimentKindClassification   ExperimentKind = "classification"
	ExperimentKindRegression       ExperimentKind = "regression"
	ExperimentKindAutoML           ExperimentKind = "automl"
)

// ViewType controls which lifecycle stages are returned in search results.
type ViewType string

const (
	ViewTypeActiveOnly  ViewType = "ACTIVE_ONLY"
	ViewTypeDeletedOnly ViewType = "DELETED_ONLY"
	ViewTypeAll         ViewType = "ALL"
)

// viewTypeToProto maps domain ViewType to protobuf ViewType enum values.
var viewTypeToProto = map[ViewType]mlflowpb.ViewType{
	ViewTypeActiveOnly:  mlflowpb.ViewType_ACTIVE_ONLY,
	ViewTypeDeletedOnly: mlflowpb.ViewType_DELETED_ONLY,
	ViewTypeAll:         mlflowpb.ViewType_ALL,
}

// Experiment represents an MLflow experiment.
type Experiment struct {
	ID               string
	Name             string
	ArtifactLocation string
	LifecycleStage   string
	Tags             map[string]string
	CreationTime     time.Time
	LastUpdateTime   time.Time
}

// ExperimentList contains experiments and a pagination token.
type ExperimentList struct {
	Experiments   []Experiment
	NextPageToken string
}

// Run represents an MLflow run with its info and data.
type Run struct {
	Info RunInfo
	Data RunData
}

// RunInfo contains metadata about a run.
type RunInfo struct {
	RunID          string
	ExperimentID   string
	RunName        string
	UserID         string
	Status         RunStatus
	StartTime      time.Time
	EndTime        time.Time
	ArtifactURI    string
	LifecycleStage string
}

// RunData contains the metrics, params, and tags for a run.
type RunData struct {
	Metrics []Metric
	Params  []Param
	Tags    map[string]string
}

// Metric represents a metric logged to a run.
type Metric struct {
	Key       string
	Value     float64
	Timestamp time.Time
	Step      int64
}

// Param represents a parameter logged to a run.
type Param struct {
	Key   string
	Value string
}

// RunList contains runs and a pagination token.
type RunList struct {
	Runs          []Run
	NextPageToken string
}

// experimentFromProto converts a protobuf Experiment to a domain Experiment.
func experimentFromProto(exp *mlflowpb.Experiment) Experiment {
	if exp == nil {
		return Experiment{}
	}

	e := Experiment{
		ID:               exp.GetExperimentId(),
		Name:             exp.GetName(),
		ArtifactLocation: exp.GetArtifactLocation(),
		LifecycleStage:   exp.GetLifecycleStage(),
		Tags:             make(map[string]string),
	}

	if exp.CreationTime != nil {
		e.CreationTime = time.UnixMilli(*exp.CreationTime)
	}
	if exp.LastUpdateTime != nil {
		e.LastUpdateTime = time.UnixMilli(*exp.LastUpdateTime)
	}

	for _, tag := range exp.Tags {
		e.Tags[tag.GetKey()] = tag.GetValue()
	}

	return e
}

// runFromProto converts a protobuf Run to a domain Run.
func runFromProto(r *mlflowpb.Run) Run {
	if r == nil {
		return Run{}
	}

	return Run{
		Info: runInfoFromProto(r.Info),
		Data: runDataFromProto(r.Data),
	}
}

// runInfoFromProto converts a protobuf RunInfo to a domain RunInfo.
func runInfoFromProto(ri *mlflowpb.RunInfo) RunInfo {
	if ri == nil {
		return RunInfo{}
	}

	info := RunInfo{
		RunID:          ri.GetRunId(),
		ExperimentID:   ri.GetExperimentId(),
		RunName:        ri.GetRunName(),
		UserID:         ri.GetUserId(),
		ArtifactURI:    ri.GetArtifactUri(),
		LifecycleStage: ri.GetLifecycleStage(),
	}

	if ri.Status != nil {
		if s, ok := protoToRunStatus[*ri.Status]; ok {
			info.Status = s
		}
	}

	if ri.StartTime != nil {
		info.StartTime = time.UnixMilli(*ri.StartTime)
	}
	if ri.EndTime != nil {
		info.EndTime = time.UnixMilli(*ri.EndTime)
	}

	return info
}

// runDataFromProto converts a protobuf RunData to a domain RunData.
func runDataFromProto(rd *mlflowpb.RunData) RunData {
	if rd == nil {
		return RunData{}
	}

	data := RunData{
		Metrics: make([]Metric, 0, len(rd.Metrics)),
		Params:  make([]Param, 0, len(rd.Params)),
		Tags:    make(map[string]string),
	}

	for _, m := range rd.Metrics {
		metric := Metric{
			Key:   m.GetKey(),
			Value: m.GetValue(),
			Step:  m.GetStep(),
		}
		if m.Timestamp != nil {
			metric.Timestamp = time.UnixMilli(*m.Timestamp)
		}
		data.Metrics = append(data.Metrics, metric)
	}

	for _, p := range rd.Params {
		data.Params = append(data.Params, Param{
			Key:   p.GetKey(),
			Value: p.GetValue(),
		})
	}

	for _, tag := range rd.Tags {
		data.Tags[tag.GetKey()] = tag.GetValue()
	}

	return data
}
