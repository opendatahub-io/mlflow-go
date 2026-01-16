# ADR-0006: Protobuf Strategy

**Status**: Accepted

**Date**: 2025-01-16

**Authors**: @ederign

## Context

MLflow defines its API using Protocol Buffers. The proto files live in the MLflow repository and define:
- Request/response message types for all API endpoints
- Service definitions with HTTP endpoint mappings
- Custom Databricks-specific proto extensions

We need to decide how to handle these proto definitions in the Go SDK:
1. Whether to use generated proto types at all
2. How to obtain and version the proto files
3. How to handle Databricks-internal dependencies

## Decision

### 1. Fetch MLflow protos from source, pin to specific commit

We fetch `model_registry.proto` directly from MLflow's GitHub repository, pinned to a specific commit SHA stored in `tools/proto/PROTO_VERSION`. This ensures reproducible builds and explicit control over when to adopt schema changes.

**Generation command**: `make gen`

**Fetch script**: `tools/proto/fetch-protos.sh` downloads from:
```
https://raw.githubusercontent.com/mlflow/mlflow/${MLFLOW_COMMIT}/mlflow/protos/model_registry.proto
```

### 2. Create a local stub for databricks.proto

MLflow's protos import `databricks.proto` which contains Databricks-internal definitions. Since this file is not publicly available, we create a minimal stub at `tools/proto/stubs/databricks.proto` that defines only the option extensions needed for compilation:

- `Visibility` enum
- `DatabricksRpcOptions`, `HttpEndpoint`, `ApiVersion` messages
- Proto option extensions for fields, methods, messages, services, and enums

### 3. Generate but don't expose proto types publicly

Generated types go to `internal/gen/mlflowpb/` (internal package). The public API uses hand-crafted types (`Prompt`, `PromptInfo`, etc.) as decided in ADR-0004.

### 4. Use JSON structs instead of proto types for API communication

MLflow's REST API uses JSON, not protobuf wire format. We define JSON struct types directly in `client.go` (e.g., `modelVersionJSON`) rather than using proto-generated types with JSON tags. This is simpler and avoids proto runtime dependencies for basic operations.

## Alternatives Considered

### Alternative 1: Vendor proto files directly

Copy proto files into the repository without a fetch mechanism.

**Rejected**: Makes updates manual and error-prone. No clear provenance for which MLflow version the protos came from.

### Alternative 2: Use proto types for JSON serialization

Use proto-generated types with `protojson` for API communication.

**Rejected**: Adds unnecessary complexity. The REST API returns plain JSON, not proto-encoded data. Using protojson would require handling proto field naming conventions (snake_case vs camelCase) and optional field semantics.

### Alternative 3: No proto generation at all

Just use hand-written JSON structs for everything.

**Rejected**: Having the official proto definitions available provides documentation value and enables future features (like proper proto-over-HTTP if MLflow adds support). The generation overhead is minimal.

### Alternative 4: Depend on a databricks-sdk-go for proto definitions

Wait for or depend on an official Databricks SDK that exports these types.

**Rejected**: No such SDK exists for the required types. Creating a minimal stub is simpler and has no external dependencies.

## Consequences

### Positive

- Reproducible builds with pinned proto versions
- Clear upgrade path when MLflow releases new APIs
- Proto definitions serve as authoritative documentation
- No dependency on unavailable Databricks internal code

### Negative

- Must maintain databricks.proto stub manually (low effort, rarely changes)
- Must manually update PROTO_VERSION when adopting new MLflow releases
- Generated code exists but is largely unused (slight repo bloat)

### Neutral

- Proto generation requires `protoc` and `protoc-gen-go` installed locally
- Developers must run `make gen` after changing PROTO_VERSION

## References

- [MLflow protos source](https://github.com/mlflow/mlflow/tree/master/mlflow/protos)
- [ADR-0004: Prompt Type Abstraction](0004-prompt-type-abstraction.md)
- [Protocol Buffers Go Tutorial](https://protobuf.dev/getting-started/gotutorial/)
