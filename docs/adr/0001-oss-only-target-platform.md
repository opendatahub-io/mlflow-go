# ADR-0001: Target Open-Source MLflow Only

**Status**: Accepted
**Date**: 2026-01-14
**Deciders**: Eder, Claude

## Context

MLflow has two distinct implementations for the Prompt Registry:

1. **Open-Source MLflow**: Prompts are stored as Model Registry entities (RegisteredModel + ModelVersion) with special tags like `mlflow.prompt.is_prompt` and `mlflow.prompt.text`. This uses the standard Model Registry REST API endpoints.

2. **Databricks Unity Catalog**: Native prompt entities with dedicated REST endpoints at `/api/2.0/mlflow/unity-catalog/prompts/*`. These endpoints only work with Databricks-managed MLflow.

We needed to decide which platform(s) to support in our Go SDK.

## Decision

**We will target open-source MLflow only. Databricks Unity Catalog integration is explicitly out of scope.**

The Go SDK will:
- Use Model Registry endpoints (`/registered-models/*`, `/model-versions/*`) for prompt operations
- Store prompt templates in the `mlflow.prompt.text` tag on ModelVersion entities
- Mark prompts with `mlflow.prompt.is_prompt=true` tag on RegisteredModel entities
- Test against OSS MLflow server started via `uv run --with mlflow mlflow server`

## Rationale

### 1. Portability and Openness

Open-source MLflow guarantees data portability. Users can:
- Run MLflow anywhere (local, cloud VMs, Kubernetes)
- Export data without vendor lock-in
- Modify and extend the server if needed

Databricks-only features tie users to a specific vendor.

### 2. Simplified Development and Testing

OSS MLflow can be started locally with a single command:
```bash
uv run --with mlflow mlflow server
```

Testing against Databricks would require:
- Active Databricks workspace
- Authentication tokens
- Network access to Databricks APIs
- Potentially costly compute resources

### 3. Broader User Base

Many MLflow users run the open-source version:
- Individual developers and small teams
- Organizations with on-premise requirements
- Academic and research institutions
- Companies evaluating MLflow before committing to a vendor

A Databricks-only SDK would exclude these users.

### 4. API Stability

The Model Registry API is stable and well-documented. The prompt-as-tagged-model pattern is how the Python SDK implements OSS prompt support. By following the same pattern, we:
- Match established behavior
- Benefit from community testing
- Have clear reference implementation in Python

### 5. Scope Clarity

Supporting both platforms would require:
- Dual API implementations
- Backend detection logic
- Separate test suites
- Potential compatibility matrix issues

Focusing on OSS keeps the codebase simple and maintainable.

## Consequences

### Positive

- Simple, focused implementation
- Easy local development and testing
- No vendor dependencies
- Clear API contracts based on Model Registry
- Matches Python SDK's OSS behavior

### Negative

- Cannot use dedicated Unity Catalog prompt endpoints
- Template stored in tags (potential size limits)
- No native prompt search (must filter registered models)
- Users on Databricks must use Python SDK for full feature set

### Neutral

- Prompt aliases will use Model Registry alias mechanism (post-MVP)
- Some Databricks-only features (governance, lineage) are not applicable

## Alternatives Considered

### A. Databricks Unity Catalog Only

**Rejected**: Would exclude the majority of open-source users and require Databricks infrastructure for development.

### B. Support Both Platforms

**Rejected**: Doubles implementation complexity without clear benefit. Users on Databricks already have the Python SDK. The Go SDK's value is enabling OSS MLflow in Go-based systems.

### C. Abstract Backend with Plugins

**Rejected**: Over-engineered for MVP. Could be revisited if there's strong demand for Databricks support, but adds significant complexity.

## References

- [MLflow Prompt Registry Docs](https://mlflow.org/docs/latest/genai/prompt-registry/)
- [OSS vs Managed MLflow](https://docs.databricks.com/aws/en/mlflow3/genai/overview/oss-managed-diff)
- [Python SDK OSS Implementation](https://github.com/mlflow/mlflow/blob/master/mlflow/prompt/registry_utils.py)
- [Constitution v0.4.0 - Target Platform section](../specs/001-prompt-registry-sdk/../../../.specify/memory/constitution.md)
