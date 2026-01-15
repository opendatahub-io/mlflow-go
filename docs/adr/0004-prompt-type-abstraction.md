<!-- ABOUTME: ADR documenting the decision to wrap proto types in a public Prompt abstraction. -->
<!-- ABOUTME: Explains forward compatibility with Databricks Unity Catalog native prompts. -->

# ADR-0004: Prompt Type Abstraction Layer

**Status**: Accepted

**Date**: 2026-01-15

## Context

The SDK needs to represent prompts loaded from or registered to MLflow. There are multiple ways to model this:

1. **Expose generated proto types directly** - Use `*mlflowpb.ModelVersion` (OSS) or `*mlflowpb.PromptVersion` (Databricks) as the public API
2. **Wrap proto types in a public abstraction** - Define our own `Prompt` type that hides the underlying storage model

Additionally, MLflow has two different storage backends:

| Backend | Storage Model | Template Location |
|---------|---------------|-------------------|
| OSS MLflow | `ModelVersion` with tags | Hidden in `mlflow.prompt.text` tag |
| Databricks Unity Catalog | Native `PromptVersion` | First-class `template` field |

Our SDK currently targets OSS-only, but may support Databricks in the future.

## Decision

**Define a public `Prompt` type that abstracts the underlying storage model.**

```go
type Prompt struct {
    Name        string
    Version     int
    Template    string            // First-class field
    Description string
    Tags        map[string]string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

This type is populated differently depending on the backend:
- **OSS**: Extract `Template` from `ModelVersion.Tags["mlflow.prompt.text"]`
- **Databricks** (future): Read `Template` directly from `PromptVersion.Template`

The conversion happens in an internal layer, keeping the public API stable.

## Rationale

### 1. Forward Compatibility with Databricks

Our `Prompt` type aligns with Databricks' native `PromptVersion` structure:

| Our `Prompt` | Databricks `PromptVersion` | OSS `ModelVersion` |
|--------------|----------------------------|-------------------|
| `Name string` | `name string` ✓ | `name string` ✓ |
| `Version int` | `version string` ✓ | `version string` ✓ |
| `Template string` | `template string` ✓ | Tag extraction required |
| `Description string` | `description string` ✓ | `description string` ✓ |
| `Tags map[string]string` | `[]PromptVersionTag` ✓ | `[]ModelVersionTag` ✓ |
| `CreatedAt time.Time` | `creation_timestamp` ✓ | `creation_timestamp` ✓ |

When Databricks support is added, only the internal conversion layer changes. The public `Prompt` type stays the same.

### 2. Proto Types Are Awkward for Users

Generated proto types have:
- All pointer fields (`*string`, `*int64`) requiring nil checks
- Proto-specific methods (`ProtoMessage()`, `ProtoReflect()`, `Reset()`)
- Internal fields visible in IDE (`state`, `unknownFields`, `sizeCache`)
- 18+ fields when users only need 7

### 3. Semantic Improvements

| Proto Reality | Our Abstraction |
|---------------|-----------------|
| `Version` is `*string` | `Version` is `int` (parsed) |
| Timestamps are `*int64` (ms) | `time.Time` (idiomatic Go) |
| Tags are `[]*Tag` | `map[string]string` (simpler) |
| Template hidden in tags (OSS) | `Template` is a field |

### 4. Constitution Compliance

> "Generated packages are internal/ to prevent consumers from depending on unstable generated names."
> "The stable, user-facing SDK wraps generated types behind an idiomatic Go package boundary."

## Alternatives Considered

### Alternative 1: Expose Proto Types Directly

Just export `*mlflowpb.ModelVersion` as the public type.

**Rejected because**:
- Ties public API to OSS storage model
- Would break if we add Databricks support
- Poor developer experience (pointers, extra fields)
- Violates constitution's internal package principle

### Alternative 2: Different Types per Backend

Define `OSSPrompt` and `DatabricksPrompt` with different structures.

**Rejected because**:
- Users would need to handle both types
- Makes switching backends a breaking change
- Over-engineering for current scope

### Alternative 3: Interface-Based Abstraction

Define a `Prompt` interface implemented by backend-specific types.

**Rejected because**:
- Adds complexity without benefit
- Users still need type assertions for fields
- A simple struct is sufficient

## Consequences

### Positive

- Public API is stable regardless of backend changes
- Clean developer experience with idiomatic Go types
- Forward-compatible with Databricks Unity Catalog
- Aligns with constitution principles

### Negative

- Requires conversion layer (~30 lines of code)
- Proto types generated but not directly exposed (could seem wasteful)
- Two representations of "prompt" in codebase (proto internal, Prompt public)

### Neutral

- Conversion overhead is negligible (one allocation per load/register)

## References

- Constitution: "Protobuf-Driven Data Model" section
- `mlflow/prompt/registry_utils.py` - Python SDK's OSS implementation
- `unity_catalog_prompt_messages.proto` - Databricks prompt structure
- `concerns.md` - OSS vs Databricks comparison
