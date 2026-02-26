# ADR-0009: Experiment Tracking Client

**Status**: Accepted

**Date**: 2026-02-25

**Authors**: @ederign

## Context

The Go SDK supports the Prompt Registry domain (ADR-0004, ADR-0005). MLflow's other core domain is Experiment Tracking: creating experiments, logging runs with metrics/params/tags, and searching results. Users need this to record and compare ML training runs from Go applications.

The multi-package structure (ADR-0005) was designed to accommodate this expansion via `client.Tracking()`.

## Decision

### 1. Explicit client API — pass identifiers to every call

All tracking methods take `runID` or `experimentID` as explicit parameters. There is no stateful "active run" concept.

```go
err := client.Tracking().LogMetric(ctx, runID, "rmse", 0.85, tracking.WithStep(1))
```

This matches the Python SDK's `MlflowClient` (explicit client), not the `mlflow.log_metric()` fluent API which relies on global state.

### 2. MVP scope

The initial release covers 17 methods across three groups:

**Experiments** (7): `CreateExperiment`, `GetExperiment`, `GetExperimentByName`, `UpdateExperiment`, `DeleteExperiment`, `SearchExperiments`, `SetExperimentTag`

**Runs** (5): `CreateRun`, `GetRun`, `UpdateRun`, `DeleteRun`, `SearchRuns`

**Logging** (5): `LogMetric`, `LogParam`, `SetTag`, `DeleteTag`, `LogBatch`

Excluded from MVP: artifacts, restore experiment/run, metric history, `DeleteExperimentTag`.

### 3. Package structure follows ADR-0005

```
mlflow/tracking/
├── client.go      # 17 API methods
├── types.go       # Domain types + proto conversion
└── options.go     # Functional options per operation
```

Accessed via `client.Tracking()` using the lazy `sync.Once` pattern established by `client.PromptRegistry()`.

### 4. Functional options for optional parameters

Each operation with optional parameters gets its own option type:

```go
run, err := client.Tracking().CreateRun(ctx, expID,
    tracking.WithRunName("training-run"),
    tracking.WithRunTags(map[string]string{"model": "sklearn"}),
)

info, err := client.Tracking().UpdateRun(ctx, runID,
    tracking.WithStatus(tracking.RunStatusFinished),
    tracking.WithEndTime(time.Now()),
)
```

`UpdateRun` makes status optional (via `WithStatus`) to match Python SDK's `update_run(run_id, status=None, name=None)`.

### 5. Typed constants for enums

```go
type RunStatus string   // RUNNING, SCHEDULED, FINISHED, FAILED, KILLED
type ViewType string    // ACTIVE_ONLY, DELETED_ONLY, ALL
```

String-typed constants provide readability and match the Python SDK's string-based status values. Internal maps convert to/from protobuf enum values.

### 6. Proto-to-domain conversion (ADR-0006)

Protobuf types from `service.proto` are generated into `internal/gen/mlflowpb/` and converted to public domain types (`Experiment`, `Run`, `Metric`, etc.) via unexported conversion functions in `types.go`. This follows the same pattern as the Prompt Registry.

### 7. Raw filter strings for search

Search methods accept filter expressions as raw strings:

```go
tracking.WithExperimentsFilter("name LIKE 'my-%'")
tracking.WithRunsFilter("metrics.rmse < 1.0")
```

This passes the filter string directly to MLflow's server-side filter parser, avoiding the need to implement a filter DSL in Go.

## Alternatives Considered

### Alternative 1: Fluent/stateful run API

Provide `mlflow.StartRun()` and `mlflow.LogMetric()` functions that track an "active run" via global state, matching Python's `mlflow.log_metric()` API.

**Rejected**: Global mutable state is not idiomatic Go. The explicit client pattern is safer for concurrent use, easier to test, and matches the Python SDK's `MlflowClient` class which is the recommended approach for production use.

### Alternative 2: Required status in UpdateRun

Make `RunStatus` a required positional parameter in `UpdateRun`.

**Rejected**: Python SDK makes status optional in `update_run(run_id, status=None, name=None)`. Users may want to rename a run without changing its status. Functional options handle this cleanly.

### Alternative 3: Filter builder DSL

Provide a typed filter builder (e.g., `filter.Metric("rmse").Lt(1.0)`) instead of raw filter strings.

**Rejected**: MLflow's filter syntax is well-documented and stable. A Go DSL would add complexity, lag behind server-side capabilities, and force users to learn a Go-specific API instead of the standard MLflow filter syntax.

### Alternative 4: Full API surface from day one

Include artifacts, restore, metric history, and all remaining operations.

**Rejected**: YAGNI. The MVP covers the most common tracking workflows. Additional operations can be added incrementally without breaking changes.

## Consequences

### Positive

- Users can record and compare ML runs from Go applications
- API mirrors Python SDK's `MlflowClient` for discoverability (ADR-0007)
- Functional options allow adding parameters without breaking changes
- Same patterns as Prompt Registry reduce cognitive overhead

### Negative

- MVP excludes artifact management — users needing artifacts must use Python SDK or REST API directly
- Raw filter strings are not type-checked at compile time
- 17 methods is a large surface to test and maintain

### Neutral

- `service.proto` added to proto generation pipeline alongside `model_registry.proto`
- Integration tests require a running MLflow server (same as Prompt Registry)

## References

- [ADR-0005: Multi-Package Structure](0005-flat-package-structure.md)
- [ADR-0006: Protobuf Strategy](0006-protobuf-strategy.md)
- [ADR-0007: Python SDK Naming Alignment](0007-python-sdk-naming-alignment.md)
- [Python MlflowClient tracking API](https://mlflow.org/docs/latest/python_api/mlflow.client.html)
- [MLflow REST API — Tracking](https://mlflow.org/docs/latest/rest-api.html)
