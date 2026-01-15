# ADR-0005: Flat Package Structure

**Status**: Accepted

**Date**: 2026-01-15

**Authors**: @ederign

## Context

The original plan.md specified a multi-package architecture:

```
├── client.go                 # Root Client with PromptRegistry() accessor
├── promptregistry/           # Subpackage for prompt operations
│   ├── client.go             # LoadPrompt, RegisterPrompt
│   └── prompt.go             # Prompt type
```

This design anticipated future expansion (tracking/, modelregistry/ subpackages) and followed a pattern where the root Client would expose domain-specific sub-clients via accessor methods like `PromptRegistry()`.

During implementation, we needed to decide whether to follow this multi-package structure or simplify.

## Decision

Use a flat `mlflow/` package structure with all prompt registry functionality directly on the Client:

```
├── mlflow/
│   ├── client.go             # Client with LoadPrompt, RegisterPrompt directly
│   ├── prompt.go             # Prompt type
│   ├── options.go            # All functional options
│   └── errors.go             # Error types and helpers
```

Users call `client.LoadPrompt()` and `client.RegisterPrompt()` directly instead of `client.PromptRegistry().LoadPrompt()`.

## Alternatives Considered

### Alternative 1: PromptRegistry() Accessor Pattern

```go
client.PromptRegistry().LoadPrompt(ctx, "my-prompt")
```

**Why rejected**:
- Adds indirection without benefit for a single-domain SDK
- More verbose API for users
- Premature optimization for hypothetical future domains

### Alternative 2: Separate promptregistry Package

```go
import "github.com/ederign/mlflow-go/promptregistry"
prClient := promptregistry.NewClient(client)
prClient.LoadPrompt(ctx, "my-prompt")
```

**Why rejected**:
- Forces users to import multiple packages
- Awkward initialization flow
- No clear benefit until we actually have multiple domains

## Consequences

### Positive

- Simpler API: `client.LoadPrompt()` instead of `client.PromptRegistry().LoadPrompt()`
- Single import: `import "github.com/ederign/mlflow-go/mlflow"`
- Fewer files and packages to maintain
- Easier to understand for new contributors

### Negative

- If we add tracking/ or modelregistry/ domains later, we'll need to either:
  - Add methods directly to Client (growing the interface)
  - Refactor to subpackages (breaking change)
- Less namespace separation between domains

### Neutral

- The Prompt type abstraction (ADR-0004) still allows future compatibility with Databricks APIs
- Can still add subpackages later if needed, treating this as the "prompt registry focused" SDK

## References

- Original plan.md project structure
- ADR-0004: Prompt Type Abstraction Layer (related forward-compatibility decision)
- Go standard library patterns (e.g., `http.Client` has methods directly rather than sub-clients)
