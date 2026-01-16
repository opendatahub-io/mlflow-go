# ADR-0003: Resilience Strategy

**Status**: Accepted

**Date**: 2026-01-14

## Context

Network calls fail. The SDK needs a strategy for handling transient failures:
- Connection resets
- Timeouts
- 5xx server errors
- Rate limiting (429)

Options range from "SDK handles everything" to "SDK does nothing, caller handles."

Key considerations:
- MLflow API idempotency is not well-documented
- Retry with backoff adds complexity and hidden latency
- Different callers have different tolerance for latency vs. reliability
- Go ecosystem generally favors explicit over implicit behavior

## Decision

**The SDK does NOT retry.** All errors are surfaced immediately to the caller.

```go
prompt, err := client.PromptRegistry().GetPrompt(ctx, "my-prompt")
if err != nil {
    // Caller decides whether to retry, with what backoff, etc.
    return err
}
```

Callers who want retry behavior can:
1. Wrap calls in their own retry loop
2. Use a retry library (e.g., `github.com/avast/retry-go`)
3. Inject a custom `http.RoundTripper` that handles retries

The SDK provides enough error information (via `APIError`) for callers to make retry decisions:
- `StatusCode` indicates retriable (5xx, 429) vs. non-retriable (4xx)
- `Code` provides MLflow-specific error categorization

## Alternatives Considered

### Alternative 1: Automatic retry with exponential backoff

SDK retries transient failures (5xx, connection errors) with exponential backoff and jitter.

**Rejected because**:
- Hides latency from caller (request that "takes 30s" is actually retrying)
- Retry policy is one-size-fits-all; callers have different needs
- Idempotency not guaranteed for all MLflow operations
- Adds complexity (max retries, backoff config, which errors to retry)

### Alternative 2: Configurable retry policy

Allow callers to pass a `RetryPolicy` option to configure retry behavior.

```go
client, err := mlflow.NewClient(
    mlflow.WithRetryPolicy(mlflow.ExponentialBackoff(3, time.Second)),
)
```

**Rejected because**:
- Still hides retry behavior inside SDK
- API surface grows (retry options, policies, callbacks)
- YAGNI - callers who need retry can implement it themselves
- Retry libraries already exist and are battle-tested

### Alternative 3: Retry only for safe/idempotent operations

Retry GET requests but not POST/PUT/DELETE.

**Rejected because**:
- MLflow's REST API doesn't cleanly map HTTP verbs to idempotency
- Partial retry (some methods, not others) is confusing
- Still has the latency-hiding problem

## Consequences

### Positive

- Simple mental model: one request, one response (or error)
- No hidden latency or retry storms
- Caller has full control over retry policy
- SDK stays small and focused

### Negative

- Callers must implement retry logic for production use
- More boilerplate for common "just retry 3 times" case
- SDK appears "less robust" compared to SDKs with built-in retry

### Neutral

- Matches philosophy of Go standard library (net/http doesn't retry)
- Consistent with "no magic" principle

## References

- [AWS SDK for Go v2 retry](https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/retries-timeouts/)
- [Google Cloud Go retry](https://pkg.go.dev/cloud.google.com/go#hdr-Retrying_Errors)
- [retry-go library](https://github.com/avast/retry-go)
