# ADR-0001: Authentication Pattern

**Status**: Accepted

**Date**: 2026-01-14

## Context

The SDK needs to authenticate requests to the MLflow server. MLflow supports multiple authentication methods depending on deployment (basic auth, tokens, OAuth, etc.). We need to decide how the Go SDK handles authentication.

Key considerations:
- Python SDK uses `MLFLOW_TRACKING_TOKEN` and `MLFLOW_TRACKING_URI` environment variables
- Users expect familiar patterns from other Go SDKs (AWS, GCP, Stripe)
- SDK should not become a credential management system
- Local development often uses no auth or basic auth

## Decision

Authentication via environment variable with explicit constructor override:

1. **Primary method**: SDK reads `MLFLOW_TRACKING_TOKEN` from environment
2. **Server URL**: SDK reads `MLFLOW_TRACKING_URI` from environment (matching Python SDK)
3. **Override**: Constructor parameters override environment variables when provided
4. **Transport**: Token passed via `Authorization: Bearer <token>` header

```go
// Reads from MLFLOW_TRACKING_URI and MLFLOW_TRACKING_TOKEN
client, err := mlflow.NewClient()

// Explicit override
client, err := mlflow.NewClient(
    mlflow.WithTrackingURI("https://my-mlflow.example.com"),
    mlflow.WithToken("my-token"),
)
```

## Alternatives Considered

### Alternative 1: Explicit credentials only (no env vars)

Require users to always pass credentials explicitly to the constructor.

**Rejected because**:
- Breaks Python SDK parity (users expect env vars to work)
- Forces credentials into code or config files
- Makes 12-factor app patterns harder

### Alternative 2: http.Client injection only (SDK is auth-agnostic)

Let users configure authentication entirely on their own http.Client.

**Rejected because**:
- Poor developer experience for common case
- Every user reimplements the same auth logic
- Harder to document and support

### Alternative 3: Support all methods (env, explicit, http.Client)

Support environment variables, explicit constructor params, AND custom http.Client with auth already configured.

**Rejected because**:
- Precedence rules become confusing
- Testing which auth method is in effect is complex
- YAGNI - env var + explicit covers 99% of use cases

## Consequences

### Positive

- Matches Python SDK behavior - users can reuse existing env vars
- Simple mental model: env var is default, constructor overrides
- No credential files or config systems to maintain
- Works well with container orchestration (K8s secrets as env vars)

### Negative

- Users who want OAuth/OIDC flows must implement custom http.Client RoundTripper
- No built-in credential refresh for expiring tokens
- Environment variable approach doesn't work well for multi-tenant scenarios

### Neutral

- SDK is opinionated about auth pattern but still allows escape hatch via http.Client injection

## References

- Python MLflow SDK environment variables: https://mlflow.org/docs/latest/tracking.html#logging-to-a-tracking-server
- AWS SDK for Go v2 credential chain: https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials
- 12-factor app config: https://12factor.net/config
