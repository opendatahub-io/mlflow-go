# ADR-0007: Python SDK Naming Alignment

**Status**: Accepted

**Date**: 2026-01-23

**Authors**: @ederign

## Context

The Go SDK needs to choose names for types and fields that represent MLflow concepts. Two options exist:
1. Invent Go-idiomatic names independently
2. Align with Python SDK naming for discoverability

The constitution (Principle II) states: "Methods map 1:1 to MLflow's public API semantics and naming, aligned to the Python API reference for discoverability."

A specific example: Python SDK uses `PromptVersion.commit_message` for version descriptions, while our initial implementation used `Description`.

## Decision

Align Go SDK type and field names with Python SDK naming conventions:

- `PromptVersion.CommitMessage` (not `Description`) - matches Python's `commit_message`
- `Prompt.Description` (unchanged) - matches Python's prompt-level `description`
- `WithCommitMessage()` option (not `WithDescription()`)

JSON serialization uses snake_case to match Python: `"commit_message"`.

## Alternatives Considered

### Alternative 1: Independent Go Naming

Use names that feel natural in Go without checking Python SDK.

Rejected: Users familiar with MLflow Python SDK would find the Go SDK less discoverable. Constitution explicitly requires Python alignment.

### Alternative 2: Generic "Description" Everywhere

Use `Description` for both prompt-level and version-level descriptions.

Rejected: Python SDK distinguishes between `Prompt.description` (prompt metadata) and `PromptVersion.commit_message` (version commit message). Using the same name conflates different concepts.

## Consequences

### Positive

- Users can transfer Python SDK knowledge to Go SDK
- Documentation and examples are easier to cross-reference
- Follows constitution Principle II

### Negative

- Some names may feel less Go-idiomatic (e.g., `CommitMessage` vs `Description`)
- Requires checking Python SDK when adding new types/fields

### Neutral

- JSON field names use snake_case (Python convention) rather than camelCase

## References

- Python SDK PromptVersion: https://github.com/mlflow/mlflow/blob/master/mlflow/entities/model_registry/prompt_version.py
- Python SDK Prompt: https://github.com/mlflow/mlflow/blob/master/mlflow/entities/model_registry/prompt.py
- Constitution Principle II: MLflow 1:1 API Mapping
