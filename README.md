# mlflow-go

A Go SDK for the [MLflow](https://mlflow.org) Prompt Registry.

## Features

- Load prompts by name (latest or specific version)
- Register new prompts and versions
- Modify prompts locally with immutable operations
- Full context support for cancellation and timeouts
- Structured logging with `slog.Handler`
- Type-safe error handling

## Installation

```bash
go get github.com/ederign/mlflow-go
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/ederign/mlflow-go/mlflow"
)

func main() {
    // Create client (reads MLFLOW_TRACKING_URI from environment)
    client, err := mlflow.NewClient()
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Load a prompt
    prompt, err := client.LoadPrompt(ctx, "my-prompt")
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

## Usage Examples

### Load a Specific Version

```go
prompt, err := client.LoadPrompt(ctx, "my-prompt", mlflow.WithVersion(2))
```

### Register a New Prompt

```go
prompt, err := client.RegisterPrompt(ctx, "greeting-prompt",
    "Hello {{name}}, welcome to {{company}}!",
    mlflow.WithDescription("Initial greeting template"),
    mlflow.WithTags(map[string]string{
        "team": "ml-platform",
        "category": "onboarding",
    }),
)
fmt.Printf("Created: %s v%d\n", prompt.Name, prompt.Version)
```

### Modify and Create New Version

```go
// Load existing prompt
prompt, err := client.LoadPrompt(ctx, "greeting-prompt")
if err != nil {
    log.Fatal(err)
}

// Modify locally (original unchanged)
modified := prompt.
    WithTemplate("Hello {{name}}! Welcome on {{day}}!").
    WithDescription("Added day variable")

// Register as new version
newVersion, err := client.RegisterPrompt(ctx, modified.Name, modified.Template,
    mlflow.WithDescription(modified.Description),
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
prompt, err := client.LoadPrompt(ctx, "my-prompt")
if err != nil {
    switch {
    case mlflow.IsNotFound(err):
        fmt.Println("Prompt does not exist")
    case mlflow.IsUnauthorized(err):
        fmt.Println("Invalid token")
    case mlflow.IsPermissionDenied(err):
        fmt.Println("Access denied")
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

## Development

### Prerequisites

- Go 1.21+
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

# Run integration tests
make test/integration

# Stop local MLflow server
make dev/down
```

### Project Structure

```
mlflow-go/
├── mlflow/              # Public SDK package
│   ├── client.go        # Client and API methods
│   ├── prompt.go        # Prompt type and methods
│   ├── options.go       # Functional options
│   └── errors.go        # Error types and helpers
├── internal/            # Internal packages
│   ├── errors/          # APIError implementation
│   └── transport/       # HTTP client
├── examples/            # Usage examples
└── specs/               # Design documentation
```

## License

Apache 2.0 - see [LICENSE](LICENSE)
