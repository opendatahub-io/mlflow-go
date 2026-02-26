# mlflow-go

A Go SDK for [MLflow](https://mlflow.org). Supports Experiment Tracking and the Prompt Registry.

## Features

### Experiment Tracking

- Create, get, update, and delete experiments
- Create, get, update, and delete runs
- Log metrics (single and batch), parameters, and tags
- Search experiments and runs with filter expressions
- Typed run status constants and view type filters

### Prompt Registry

- Load prompts by name (latest or specific version)
- List prompts and versions with filtering and pagination
- Register text prompts and chat prompts (with model configuration)
- Delete prompts, versions, and tags
- Format prompts with variable substitution
- Modify prompts locally with immutable operations

### Workspace Isolation (Midstream)

- Forward custom headers on every request via `WithHeaders`
- Tenant isolation with `X-MLFLOW-WORKSPACE` header
- Compatible with the [Red Hat midstream fork](https://github.com/opendatahub-io/mlflow) (opendatahub-io/mlflow)

### General

- Full context support for cancellation and timeouts
- Structured logging with `slog.Handler`
- Type-safe error handling

## Installation

```bash
go get github.com/opendatahub-io/mlflow-go
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/opendatahub-io/mlflow-go/mlflow"
)

func main() {
    // Create client (reads MLFLOW_TRACKING_URI from environment)
    client, err := mlflow.NewClient()
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Load a prompt
    prompt, err := client.PromptRegistry().LoadPrompt(ctx, "my-prompt")
    if err != nil {
        if mlflow.IsNotFound(err) {
            log.Fatal("Prompt not found")
        }
        log.Fatal(err)
    }

    fmt.Printf("Loaded: %s v%d\n", prompt.Name, prompt.Version)
    fmt.Printf("Template: %s\n", prompt.Template)
}
```

## Configuration

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `MLFLOW_TRACKING_URI` | MLflow server URL | Yes |
| `MLFLOW_INSECURE_SKIP_TLS_VERIFY` | Allow HTTP (set to `true` or `1`) | No |

### Explicit Configuration

```go
client, err := mlflow.NewClient(
    mlflow.WithTrackingURI("https://mlflow.example.com"),
    mlflow.WithHeaders(map[string]string{
        "Authorization": "Bearer my-token",
    }),
    mlflow.WithTimeout(60 * time.Second),
)
```

### Custom Headers and Workspace Isolation

`WithHeaders` forwards custom HTTP headers on every API request. This is primarily used for workspace-based tenant isolation with the [Red Hat midstream fork](https://github.com/opendatahub-io/mlflow) (opendatahub-io/mlflow), but can also carry additional auth headers or routing metadata.

The midstream fork adds multi-tenant workspace support to MLflow. Each workspace acts as an isolated namespace — prompts, models, and versions created in one workspace are completely invisible from another. The server routes requests to the correct workspace based on the `X-MLFLOW-WORKSPACE` header.

> **Note:** Workspace isolation is currently available in the Red Hat midstream fork. This feature is planned to be upstreamed to MLflow in the near future.

```go
// Create a client scoped to a workspace
clientA, err := mlflow.NewClient(
    mlflow.WithTrackingURI("http://127.0.0.1:5000"),
    mlflow.WithInsecure(),
    mlflow.WithHeaders(map[string]string{
        "X-MLFLOW-WORKSPACE": "team-bella",
    }),
)

// Register a prompt in team-bella
clientA.PromptRegistry().RegisterPrompt(ctx, "my-prompt", "Hello {{name}}!")

// Create a second client scoped to team-dora
clientB, err := mlflow.NewClient(
    mlflow.WithTrackingURI("http://127.0.0.1:5000"),
    mlflow.WithInsecure(),
    mlflow.WithHeaders(map[string]string{
        "X-MLFLOW-WORKSPACE": "team-dora",
    }),
)

// This returns NotFound — the prompt exists only in team-bella
_, err = clientB.PromptRegistry().LoadPrompt(ctx, "my-prompt")
// mlflow.IsNotFound(err) == true
```

Workspaces must be pre-created on the server before use. If you reference a workspace that doesn't exist, the server returns a `RESOURCE_DOES_NOT_EXIST` error. For local development:

```bash
# Start midstream server with workspaces enabled
make dev/up-midstream

# Create workspaces
make dev/seed-workspaces
```

### Local Development

```go
// Allow HTTP for local development
client, err := mlflow.NewClient(
    mlflow.WithTrackingURI("http://localhost:5000"),
    mlflow.WithInsecure(),
)
```

## Experiment Tracking

### Create an Experiment and Log a Run

```go
import "github.com/opendatahub-io/mlflow-go/mlflow/tracking"

ctx := context.Background()

// Create an experiment
expID, err := client.Tracking().CreateExperiment(ctx, "my-experiment")

// Create a run in the experiment
run, err := client.Tracking().CreateRun(ctx, expID,
    tracking.WithRunName("training-run-1"),
    tracking.WithRunTags(map[string]string{"model": "sklearn"}),
)
runID := run.Info.RunID

// Log metrics, params, and tags
err = client.Tracking().LogMetric(ctx, runID, "rmse", 0.85, tracking.WithStep(1))
err = client.Tracking().LogParam(ctx, runID, "learning_rate", "0.01")
err = client.Tracking().SetTag(ctx, runID, "status", "training")

// Mark run as finished
info, err := client.Tracking().UpdateRun(ctx, runID,
    tracking.WithStatus(tracking.RunStatusFinished),
    tracking.WithEndTime(time.Now()),
)
```

### Batch Logging

```go
err := client.Tracking().LogBatch(ctx, runID,
    []tracking.Metric{
        {Key: "loss", Value: 0.5, Step: 1},
        {Key: "loss", Value: 0.3, Step: 2},
        {Key: "accuracy", Value: 0.92, Step: 1},
    },
    []tracking.Param{
        {Key: "optimizer", Value: "adam"},
        {Key: "batch_size", Value: "32"},
    },
    map[string]string{
        "framework": "pytorch",
    },
)
```

### List All Experiments

```go
// List experiments (default: active only; use WithExperimentsViewType for deleted/all)
experiments, err := client.Tracking().SearchExperiments(ctx)
for _, e := range experiments.Experiments {
    fmt.Printf("[%s] %s (lifecycle: %s)\n", e.ID, e.Name, e.LifecycleStage)
}
```

### Search Experiments and Runs

```go
// Search experiments by name
experiments, err := client.Tracking().SearchExperiments(ctx,
    tracking.WithExperimentsFilter("name = 'my-experiment'"),
)

// Search runs with metric filter
runs, err := client.Tracking().SearchRuns(ctx, []string{expID},
    tracking.WithRunsFilter("metrics.rmse < 1"),
    tracking.WithRunsOrderBy("metrics.rmse ASC"),
    tracking.WithRunsMaxResults(10),
)
for _, r := range runs.Runs {
    fmt.Printf("Run %s: status=%s\n", r.Info.RunID, r.Info.Status)
}
```

### Get and Delete

```go
// Get experiment by ID or name
exp, err := client.Tracking().GetExperiment(ctx, expID)
exp, err = client.Tracking().GetExperimentByName(ctx, "my-experiment")

// Get run
run, err := client.Tracking().GetRun(ctx, runID)

// Delete tag, run, experiment
err = client.Tracking().DeleteTag(ctx, runID, "status")
err = client.Tracking().DeleteRun(ctx, runID)
err = client.Tracking().DeleteExperiment(ctx, expID)
```

### View Types

Use typed constants to filter by lifecycle stage:

```go
// Search only deleted experiments
experiments, err := client.Tracking().SearchExperiments(ctx,
    tracking.WithExperimentsViewType(tracking.ViewTypeDeletedOnly),
)

// Search all runs (active + deleted)
runs, err := client.Tracking().SearchRuns(ctx, []string{expID},
    tracking.WithRunsViewType(tracking.ViewTypeAll),
)
```

## Prompt Registry

## Core Types

The SDK uses two main types for prompts:

- **`Prompt`** – Lightweight metadata returned by `ListPrompts()`. Contains name, description, latest version number, and tags. Does not include template content.
- **`PromptVersion`** – Full prompt content returned by `LoadPrompt()`. Contains the template (or messages for chat prompts), version number, aliases, model config, and timestamps. This is what you use to format prompts with variables.

Use `ListPrompts()` to browse available prompts, then `LoadPrompt()` to fetch the full content when you need it.

## Usage Examples

### Load a Specific Version

```go
import "github.com/opendatahub-io/mlflow-go/mlflow/promptregistry"

prompt, err := client.PromptRegistry().LoadPrompt(ctx, "my-prompt", promptregistry.WithVersion(2))
```

### Load by Alias

```go
// Load the version pointed to by an alias (e.g., "production", "staging")
prompt, err := client.PromptRegistry().LoadPrompt(ctx, "my-prompt", promptregistry.WithAlias("production"))
```

### Manage Aliases

```go
// Set an alias to point to a specific version
err := client.PromptRegistry().SetPromptAlias(ctx, "my-prompt", "production", 3)

// Delete an alias
err := client.PromptRegistry().DeletePromptAlias(ctx, "my-prompt", "staging")
```

### Delete Prompts and Versions

```go
// Delete a prompt (cascades to delete all versions and aliases on MLflow OSS)
err := client.PromptRegistry().DeletePrompt(ctx, "my-prompt")

// Delete a specific version
err = client.PromptRegistry().DeletePromptVersion(ctx, "my-prompt", 2)
if mlflow.IsAliasConflict(err) {
    // Databricks only: must remove aliases pointing to this version first
    // (MLflow OSS silently removes aliases on version deletion)
    _ = client.PromptRegistry().DeletePromptAlias(ctx, "my-prompt", "production")
    err = client.PromptRegistry().DeletePromptVersion(ctx, "my-prompt", 2)
}

// Delete tags
err = client.PromptRegistry().DeletePromptTag(ctx, "my-prompt", "environment")
err = client.PromptRegistry().DeletePromptVersionTag(ctx, "my-prompt", 1, "reviewed")
```

### List All Prompts

```go
list, err := client.PromptRegistry().ListPrompts(ctx)
if err != nil {
    log.Fatal(err)
}

for _, info := range list.Prompts {
    fmt.Printf("%s (latest: v%d)\n", info.Name, info.LatestVersion)
}

// Pagination: fetch next page if available
if list.NextPageToken != "" {
    nextPage, err := client.PromptRegistry().ListPrompts(ctx, promptregistry.WithPageToken(list.NextPageToken))
    // ...
}
```

### List Prompts with Filters

```go
// Filter by name pattern and tags
list, err := client.PromptRegistry().ListPrompts(ctx,
    promptregistry.WithNameFilter("dog-%"),  // SQL LIKE syntax
    promptregistry.WithTagFilter(map[string]string{"category": "pets"}),
    promptregistry.WithMaxResults(10),
)
```

### List Prompt Versions

```go
versions, err := client.PromptRegistry().ListPromptVersions(ctx, "my-prompt")
if err != nil {
    log.Fatal(err)
}

for _, v := range versions.Versions {
    fmt.Printf("v%d: %s\n", v.Version, v.CommitMessage)
}

// Limit results
versions, err = client.PromptRegistry().ListPromptVersions(ctx, "my-prompt",
    promptregistry.WithVersionsMaxResults(10),
)
```

> **MLflow OSS Known Issue**: MLflow OSS has a bug where the `/model-versions/search` endpoint
> permanently returns empty results for model versions created in rapid succession (<50ms between
> calls). The data is written correctly (direct GET by version number works), but the search
> endpoint never indexes those versions. This affects both SQLite and PostgreSQL backends, pointing
> to a stale SQLAlchemy session issue in MLflow's multi-worker Uvicorn setup rather than a
> database-level problem.
>
> `ListPromptVersions` works around this by trying the search endpoint first, and falling back to
> fetching versions individually (via the `@latest` alias + direct GET per version) when search
> returns empty. We plan to report this issue upstream to MLflow.

### Register a Text Prompt

```go
prompt, err := client.PromptRegistry().RegisterPrompt(ctx, "dog-walker-prompt",
    "Time to walk Bella and Dora! Meeting at {{location}} at {{time}}.",
    promptregistry.WithCommitMessage("Walk reminder for Bella and Dora"),
    promptregistry.WithTags(map[string]string{
        "dogs": "bella,dora",
        "category": "scheduling",
    }),
)
fmt.Printf("Created: %s v%d\n", prompt.Name, prompt.Version)
```

### Register a Chat Prompt

```go
messages := []promptregistry.ChatMessage{
    {Role: "system", Content: "You are a helpful dog walking assistant for {{owner}}."},
    {Role: "user", Content: "When should I walk {{dog_name}} today?"},
}

// Optional: include model configuration
temp := 0.7
modelConfig := &promptregistry.PromptModelConfig{
    Provider:    "openai",
    ModelName:   "gpt-4",
    Temperature: &temp,
}

prompt, err := client.PromptRegistry().RegisterChatPrompt(ctx, "dog-assistant",
    messages,
    promptregistry.WithCommitMessage("Chat assistant for dog walking"),
    promptregistry.WithModelConfig(modelConfig),
)
fmt.Printf("Created chat prompt: %s v%d\n", prompt.Name, prompt.Version)
```

### Format Prompts with Variables

```go
// Text prompts - get formatted string directly
prompt, _ := client.PromptRegistry().LoadPrompt(ctx, "dog-walker-prompt")
text, err := prompt.FormatAsText(map[string]string{
    "location": "Central Park",
    "time":     "3pm",
})
// Result: "Time to walk Bella and Dora! Meeting at Central Park at 3pm."

// Chat prompts - get formatted messages
chatPrompt, _ := client.PromptRegistry().LoadPrompt(ctx, "dog-assistant")
messages, err := chatPrompt.FormatAsMessages(map[string]string{
    "owner":    "Alice",
    "dog_name": "Bella",
})
// Returns []ChatMessage with variables substituted

// Generic format - returns a new PromptVersion with variables substituted
formatted, err := prompt.Format(map[string]string{"location": "Park", "time": "3pm"})
// formatted.Template or formatted.Messages contains the result
```

### Modify and Create New Version

```go
// Load existing prompt
prompt, err := client.PromptRegistry().LoadPrompt(ctx, "dog-walker-prompt")
if err != nil {
    log.Fatal(err)
}

// Modify locally (original unchanged)
modified := prompt.
    WithTemplate("Hey {{owner}}! Bella and Dora are ready for their walk. Don't forget the treats!").
    WithCommitMessage("Added owner and treats reminder")

// Register as new version
newVersion, err := client.PromptRegistry().RegisterPrompt(ctx, modified.Name, modified.Template,
    promptregistry.WithCommitMessage(modified.CommitMessage),
)
fmt.Printf("Created version %d\n", newVersion.Version)
```

### Debug Logging

```go
handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelDebug,
})

client, err := mlflow.NewClient(
    mlflow.WithLogger(handler),
)
```

## Error Handling

The SDK provides type-safe error checking:

```go
prompt, err := client.PromptRegistry().LoadPrompt(ctx, "my-prompt")
if err != nil {
    switch {
    case mlflow.IsNotFound(err):
        fmt.Println("Prompt does not exist")
    case mlflow.IsUnauthorized(err):
        fmt.Println("Invalid token")
    case mlflow.IsPermissionDenied(err):
        fmt.Println("Access denied")
    case mlflow.IsAliasConflict(err):
        fmt.Println("Cannot delete: aliases point to this version (Databricks only)")
    default:
        var apiErr *mlflow.APIError
        if errors.As(err, &apiErr) {
            fmt.Printf("API error: %s (code: %s)\n",
                apiErr.Message, apiErr.Code)
        }
    }
    return
}
```

## Feature Comparison with Python SDK

### Experiment Tracking

| Feature | Status |
|---------|--------|
| Create/get/update/delete experiments | ✅ Supported |
| Create/get/update/delete runs | ✅ Supported |
| Log metrics (single + batch) | ✅ Supported |
| Log params (single + batch) | ✅ Supported |
| Set/delete tags (single + batch) | ✅ Supported |
| Search experiments and runs | ✅ Supported |
| Typed run status and view type | ✅ Supported |
| Set experiment tags | ✅ Supported |
| Restore experiments/runs | ❌ Not yet |
| Metric history | ❌ Not yet |
| Artifact management | ❌ Not yet |

### Prompt Registry

| Feature | Status |
|---------|--------|
| Load/register prompts | ✅ Supported |
| Text and chat prompts | ✅ Supported |
| Variable substitution (`{{var}}`) | ✅ Supported |
| Model configuration | ✅ Supported |
| Version and alias management | ✅ Supported |
| List/search with filters | ✅ Supported |
| Tags on registration | ✅ Supported |
| Delete prompts and versions | ✅ Supported |
| Delete tags | ✅ Supported |
| Custom headers (`WithHeaders`) | ✅ Supported |
| Workspace isolation (midstream) | ✅ Supported |
| Set/update tags after creation | ❌ Not yet |
| Update model config after creation | ❌ Not yet |
| Jinja2 templates (conditionals, loops) | ❌ Not yet |
| Response format specification | ❌ Not yet |
| Cache TTL configuration | ❌ Not yet |

## Development

### Prerequisites

- Go 1.23+
- MLflow server (for integration tests)

### Commands

```bash
# Run unit tests
make test/unit

# Run linter
make lint

# Run all checks (lint, vet, tests)
make check

# Start local MLflow server (requires uv)
make dev/up

# Start MLflow from opendatahub-io/mlflow fork (with workspaces enabled)
make dev/up-midstream

# Create test workspaces (team-bella and team-dora)
make dev/seed-workspaces

# Seed sample prompts (featuring Bella and Dora!)
make dev/seed

# Run the sample app
make run-sample

# Run workspace isolation demo (requires midstream server)
make dev/up-midstream   # in one terminal
make dev/seed-workspaces
make run-sample-workspaces

# Run integration tests
make test/integration

# Run integration tests against midstream
make test/integration-ci-midstream

# Stop local MLflow server
make dev/down

# Reset MLflow (nuke data)
make dev/reset
```

### Project Structure

```
mlflow-go/
├── mlflow/                     # Public SDK package
│   ├── client.go               # Root client with domain accessors
│   ├── options.go              # Client-level options
│   ├── errors.go               # Error types and helpers
│   ├── tracking/               # Experiment Tracking sub-client
│   │   ├── client.go           # Tracking API methods
│   │   ├── types.go            # Experiment, Run, Metric, Param types
│   │   └── options.go          # Domain-specific options
│   └── promptregistry/         # Prompt Registry sub-client
│       ├── client.go           # PromptRegistry API methods
│       ├── prompt.go           # Prompt, PromptInfo types
│       └── options.go          # Domain-specific options
├── internal/                   # Internal packages
│   ├── conv/                   # Shared type-conversion helpers
│   ├── errors/                 # APIError implementation
│   └── transport/              # HTTP client
├── sample-app/                 # Demo application
└── specs/                      # Design documentation
```

## License

Apache 2.0 - see [LICENSE](LICENSE)
