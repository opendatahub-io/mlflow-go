# ADR-0002: Error Type Design

**Status**: Accepted

**Date**: 2026-01-14

## Context

The SDK needs to surface errors from MLflow API calls in a way that:
- Follows Go conventions (errors.Is, errors.As)
- Provides enough detail for callers to handle specific error cases
- Never leaks sensitive information (tokens, prompt content)
- Maps cleanly to MLflow's error responses

MLflow returns errors as JSON with status code, error code, and message.

## Decision

Implement a structured `APIError` type with helper functions:

```go
type APIError struct {
    StatusCode int    // HTTP status code (e.g., 404, 500)
    Code       string // MLflow error code (e.g., "RESOURCE_NOT_FOUND")
    Message    string // Human-readable message from server
    RequestID  string // Request ID for debugging (if available)
}

func (e *APIError) Error() string {
    return fmt.Sprintf("mlflow: %s (status %d, code %s)", e.Message, e.StatusCode, e.Code)
}

// Helper functions for common checks
func IsNotFound(err error) bool
func IsAlreadyExists(err error) bool
func IsPermissionDenied(err error) bool
func IsInvalidArgument(err error) bool
```

Usage:
```go
prompt, err := client.PromptRegistry().GetPrompt(ctx, "my-prompt")
if mlflow.IsNotFound(err) {
    // Handle missing prompt
}
var apiErr *mlflow.APIError
if errors.As(err, &apiErr) {
    log.Printf("Request %s failed: %s", apiErr.RequestID, apiErr.Message)
}
```

## Alternatives Considered

### Alternative 1: Return raw HTTP errors

Just return the error from http.Client without wrapping.

**Rejected because**:
- Callers can't distinguish "prompt not found" from "network timeout"
- No access to MLflow-specific error codes
- Poor developer experience

### Alternative 2: Error codes as constants with switch statements

Define error code constants and have users switch on them.

```go
if err != nil {
    switch mlflow.ErrorCode(err) {
    case mlflow.ErrNotFound:
        // ...
    }
}
```

**Rejected because**:
- Doesn't compose well with errors.Is/errors.As
- Forces exhaustive switches or default cases
- Less idiomatic Go

### Alternative 3: Separate error types per category

Define `NotFoundError`, `PermissionError`, etc. as distinct types.

**Rejected because**:
- Explosion of types for diminishing returns
- Most callers only care about 2-3 error categories
- Helper functions achieve same goal with less API surface

## Consequences

### Positive

- Works with standard Go error handling (errors.Is, errors.As)
- Helper functions cover common cases without type assertions
- Structured error provides debugging info (RequestID)
- Single error type keeps API surface small

### Negative

- Less granular than separate error types (can't match on specific type)
- Helper functions must be maintained as MLflow adds error codes
- RequestID only available if MLflow server provides it

### Neutral

- Error messages from server are passed through; SDK doesn't translate them

## References

- Go error handling best practices: https://go.dev/blog/go1.13-errors
- Google Cloud Go error handling: https://pkg.go.dev/google.golang.org/api/googleapi#Error
- AWS SDK for Go v2 error handling: https://aws.github.io/aws-sdk-go-v2/docs/handling-errors/
