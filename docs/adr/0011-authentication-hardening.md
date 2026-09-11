# ADR-0011: Authentication Hardening

**Status**: Accepted

**Date**: 2026-09-10

**Supersedes**: [ADR-0001](0001-authentication-pattern.md)

## Context

[ADR-0001](0001-authentication-pattern.md) established the SDK's authentication pattern: read `MLFLOW_TRACKING_TOKEN` from the environment, allow explicit override via `WithToken`, and inject a Bearer header on every request. That design was sufficient for simple deployments but left gaps that appeared during production hardening:

1. **Credential leakage on redirects** -- The `Authorization` header was injected unconditionally by the HTTP round tripper, so if the tracking server returned a redirect (e.g. to a pre-signed object-store URL), the credentials were forwarded to the third-party host.
2. **Plaintext transmission** -- Nothing prevented combining `WithInsecure` (which allows plain HTTP or disables TLS verification) with token authentication, risking credential exposure in transit.
3. **Token rotation** -- Kubernetes projected service-account tokens expire and rotate on disk. The original static-token model required a process restart to pick up new credentials.
4. **Explicit empty-value semantics** -- Calling `WithToken("")` was indistinguishable from not calling `WithToken` at all, so the env-var fallback still fired, contradicting the documented precedence that explicit options override environment variables.

This ADR supersedes ADR-0001 and documents the complete authentication design.

## Decision

### 1. Environment-variable fallback with explicit override (carried forward from ADR-0001)

The SDK reads `MLFLOW_TRACKING_TOKEN` from the environment when no explicit token option is provided:

```go
client, err := mlflow.NewClient()                           // uses env vars
client, err := mlflow.NewClient(mlflow.WithToken("tok"))    // overrides env
```

### 2. Explicit empty-value suppression

Calling `WithToken("")` or `WithTokenPath("")` explicitly disables token authentication, even when `MLFLOW_TRACKING_TOKEN` is set. This is implemented via boolean sentinel fields (`tokenSet`, `tokenPathSet`) on the options struct, so the env-var fallback is skipped whenever either option function was invoked regardless of the value passed.

### 3. File-based token authentication with automatic rotation

`WithTokenPath(path)` configures the SDK to re-read a token file on every request. This supports Kubernetes projected service-account volumes where the kubelet rotates the token periodically without restarting the pod. The round tripper reads the file, trims whitespace, and rejects empty files with a clear error.

```go
client, err := mlflow.NewClient(
    mlflow.WithTokenPath("/var/run/secrets/tokens/mlflow"),
)
```

### 4. Origin-scoped credential injection

Both token round trippers (`tokenRoundTripper` and `tokenFileRoundTripper`) store the tracking server's origin (`scheme://host`) at construction time. The `Authorization` header is only set when the request URL's origin matches. For requests to any other host -- including redirects to subdomains or scheme-downgraded URLs -- any pre-existing `Authorization` header is actively stripped before forwarding.

### 5. Insecure + token rejection

`transport.New()` returns an error when `Insecure` is true and either `Token` or `TokenPath` is set. This prevents sending credentials over plain HTTP or TLS-unverified connections. Users who need custom TLS behaviour (e.g. a private CA) should use an HTTPS URI, omit `WithInsecure`, and configure the TLS settings through `WithHTTPClient`.

## Alternatives Considered

### Alternative 1: Redirect-policy hook instead of origin check

Override `http.Client.CheckRedirect` to strip the `Authorization` header on cross-origin redirects.

**Rejected because**:
- Redirect policies are a single function; composing with a user-supplied policy is fragile
- The round tripper approach is transparent -- it works regardless of how the request reaches a foreign host (redirects, manual URL construction, etc.)
- Go's `net/http` already strips sensitive headers on cross-origin redirects in some cases, but only for requests initiated through `Client.Do`, not for custom round tripper chains

### Alternative 2: Insecure + token as a warning instead of a hard error

Log a warning when insecure mode is combined with token authentication, but allow it.

**Rejected because**:
- A warning is easy to miss in production
- Credential exposure in transit is a security incident, not a degraded-service scenario
- Users who understand the risk can opt in via `WithHTTPClient` with a custom transport

### Alternative 3: Pointer fields instead of boolean sentinels for empty-value tracking

Use `*string` for `token` and `tokenPath` in the options struct so `nil` means "not set" and `""` means "explicitly empty".

**Rejected because**:
- Pointer fields complicate every read site with nil checks
- The sentinel booleans (`tokenSet`, `tokenPathSet`) are internal and add no API complexity
- The functional-options pattern already encourages value types

## Consequences

### Positive

- Credentials are never leaked to third-party hosts on redirect
- Credentials are never sent over unverified connections without explicit opt-in
- Kubernetes projected token rotation works without process restart
- `WithToken("")` correctly suppresses the env-var fallback, matching documented precedence
- All behaviour is covered by unit and end-to-end tests

### Negative

- Users who previously relied on `WithInsecure` + `WithToken` for local development against plain HTTP must now remove `WithInsecure` and switch to HTTPS
- The round tripper wrapping adds one origin comparison per request (negligible cost, but adds code)

### Neutral

- OAuth/OIDC and other advanced auth flows remain out of scope; users implement them via custom `http.Client` injection
- The `user:password` colon heuristic for Basic auth (from ADR-0001) is preserved unchanged

## References

- [ADR-0001: Authentication Pattern](0001-authentication-pattern.md) -- superseded predecessor
- [Go `http.Request.Clone` documentation](https://pkg.go.dev/net/http#Request.Clone) -- Clone preserves a nil Header map
- [Kubernetes Projected Volumes](https://kubernetes.io/docs/concepts/storage/projected-volumes/) -- token rotation mechanism
- [Python MLflow SDK environment variables](https://mlflow.org/docs/latest/tracking.html#logging-to-a-tracking-server)
