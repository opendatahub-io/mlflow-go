# mlflow-go

A Go SDK for [MLflow](https://mlflow.org). Currently supports the Prompt Registry, with more capabilities planned.

## Features

### Prompt Registry

- Load prompts by name (latest or specific version)
- List prompts and versions with filtering and pagination
- Register text prompts and chat prompts (with model configuration)
- Delete prompts, versions, and tags
- Format prompts with variable substitution
- Modify prompts locally with immutable operations

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
| `MLFLOW_TRACKING_TOKEN` | Authentication token | No |
| `MLFLOW_INSECURE_SKIP_TLS_VERIFY` | Allow HTTP (set to `true` or `1`) | No |

### Explicit Configuration

```go
client, err := mlflow.NewClient(
    mlflow.WithTrackingURI("https://mlflow.example.com"),
    mlflow.WithToken("my-token"),
    mlflow.WithTimeout(60 * time.Second),
)
```

### Local Development

```go
// Allow HTTP for local development
client, err := mlflow.NewClient(
    mlflow.WithTrackingURI("http://localhost:5000"),
    mlflow.WithInsecure(),
)
```

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

This Go SDK covers the core Prompt Registry functionality. Some advanced features from the [Python SDK](https://mlflow.org/docs/latest/genai/prompt-registry/) are not yet implemented:

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

# Seed sample prompts (featuring Bella and Dora!)
make dev/seed

# Run the sample app
make run-sample

# Run integration tests
make test/integration

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
│   └── promptregistry/         # Prompt Registry sub-client
│       ├── client.go           # PromptRegistry API methods
│       ├── prompt.go           # Prompt, PromptInfo types
│       └── options.go          # Domain-specific options
├── internal/                   # Internal packages
│   ├── errors/                 # APIError implementation
│   └── transport/              # HTTP client
├── sample-app/                 # Demo application
└── specs/                      # Design documentation
```

## License

Apache 2.0 - see [LICENSE](LICENSE)
