# ADR-0005: Multi-Package Structure with Domain Sub-Clients

**Status**: Accepted (revised 2025-01-16)

**Date**: 2025-01-15 (revised 2025-01-16)

**Authors**: @ederign

## Context

The SDK needs a package structure that can grow to support multiple MLflow domains:
- Prompt Registry (current)
- Tracking (future)
- Model Registry (future)
- Experiments (future)

We initially chose a flat structure for simplicity, but reconsidered to enable clean expansion.

## Decision

Use a multi-package architecture with domain-specific sub-clients accessible via accessor methods:

```
├── mlflow/
│   ├── client.go              # Root Client with PromptRegistry() accessor
│   ├── options.go             # Client-level options (WithTrackingURI, WithToken, etc.)
│   ├── errors.go              # Re-exports error helpers
│   └── promptregistry/        # Prompt Registry domain
│       ├── client.go          # PromptRegistryClient with LoadPrompt, RegisterPrompt
│       ├── prompt.go          # Prompt, PromptInfo, PromptList types
│       └── options.go         # Domain-specific options (WithVersion, WithTags, etc.)
```

### API Usage

```go
import "github.com/opendatahub-io/mlflow-go/mlflow"

client, err := mlflow.NewClient(
    mlflow.WithTrackingURI("https://mlflow.example.com"),
)

// Access prompt registry via sub-client
prompt, err := client.PromptRegistry().LoadPrompt(ctx, "my-prompt")
prompt, err := client.PromptRegistry().RegisterPrompt(ctx, "name", "template")
list, err := client.PromptRegistry().ListPrompts(ctx)

// Future domains follow same pattern
// run, err := client.Tracking().CreateRun(ctx, experimentID)
// model, err := client.ModelRegistry().GetModel(ctx, "name")
```

## Alternatives Considered

### Alternative 1: Flat Package (Previous Decision)

```go
client.LoadPrompt(ctx, "my-prompt")
```

**Why reversed**:
- SDK will expand beyond prompt registry
- Adding methods directly to Client would create a bloated interface
- Refactoring later would be a breaking change for users
- Better to establish the pattern now

### Alternative 2: Separate Top-Level Packages

```go
import "github.com/opendatahub-io/mlflow-go/promptregistry"
prClient := promptregistry.NewClient(baseClient)
```

**Why rejected**:
- Forces users to import multiple packages
- Each domain needs its own initialization
- Accessor pattern is more ergonomic and common in Go SDKs

## Consequences

### Positive

- Clean separation between domains
- Root Client interface stays small as SDK grows
- Consistent pattern for all future domains
- Each domain can have its own options without collision

### Negative

- Slightly more verbose: `client.PromptRegistry().LoadPrompt()` vs `client.LoadPrompt()`
- More packages to maintain
- Internal packages need to share transport layer

### Neutral

- Sub-clients are created lazily on first access (no extra allocation if unused)
- Domain-specific options live in their respective packages

## References

- AWS SDK for Go v2 uses similar pattern: `client.S3()`, `client.DynamoDB()`
- Google Cloud Go SDK: `client.Storage()`, `client.BigQuery()`
- ADR-0004: Prompt Type Abstraction Layer
