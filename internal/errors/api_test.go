package errors

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *APIError
		expected string
	}{
		{
			name: "with code",
			err: &APIError{
				StatusCode: 404,
				Code:       "RESOURCE_DOES_NOT_EXIST",
				Message:    "Registered Model with name=foo not found",
			},
			expected: "mlflow: RESOURCE_DOES_NOT_EXIST: Registered Model with name=foo not found (status 404)",
		},
		{
			name: "without code",
			err: &APIError{
				StatusCode: 500,
				Message:    "Internal server error",
			},
			expected: "mlflow: Internal server error (status 500)",
		},
		{
			name: "with request ID",
			err: &APIError{
				StatusCode: 401,
				Code:       "UNAUTHENTICATED",
				Message:    "Invalid token",
				RequestID:  "req-123",
			},
			expected: "mlflow: UNAUTHENTICATED: Invalid token (status 401)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestAPIError_ImplementsError(t *testing.T) {
	var _ error = &APIError{}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "APIError with 404",
			err:      &APIError{StatusCode: http.StatusNotFound, Message: "not found"},
			expected: true,
		},
		{
			name:     "APIError with 500",
			err:      &APIError{StatusCode: http.StatusInternalServerError, Message: "error"},
			expected: false,
		},
		{
			name:     "wrapped APIError with 404",
			err:      fmt.Errorf("wrapped: %w", &APIError{StatusCode: http.StatusNotFound}),
			expected: true,
		},
		{
			name:     "non-APIError",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNotFound(tt.err)
			if got != tt.expected {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsUnauthorized(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "APIError with 401",
			err:      &APIError{StatusCode: http.StatusUnauthorized},
			expected: true,
		},
		{
			name:     "APIError with 403",
			err:      &APIError{StatusCode: http.StatusForbidden},
			expected: false,
		},
		{
			name:     "wrapped APIError with 401",
			err:      fmt.Errorf("wrapped: %w", &APIError{StatusCode: http.StatusUnauthorized}),
			expected: true,
		},
		{
			name:     "non-APIError",
			err:      errors.New("unauthorized"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsUnauthorized(tt.err)
			if got != tt.expected {
				t.Errorf("IsUnauthorized() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsPermissionDenied(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "APIError with 403",
			err:      &APIError{StatusCode: http.StatusForbidden},
			expected: true,
		},
		{
			name:     "APIError with 401",
			err:      &APIError{StatusCode: http.StatusUnauthorized},
			expected: false,
		},
		{
			name:     "wrapped APIError with 403",
			err:      fmt.Errorf("wrapped: %w", &APIError{StatusCode: http.StatusForbidden}),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPermissionDenied(tt.err)
			if got != tt.expected {
				t.Errorf("IsPermissionDenied() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsInvalidArgument(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "APIError with 400",
			err:      &APIError{StatusCode: http.StatusBadRequest},
			expected: true,
		},
		{
			name:     "APIError with 404",
			err:      &APIError{StatusCode: http.StatusNotFound},
			expected: false,
		},
		{
			name:     "wrapped APIError with 400",
			err:      fmt.Errorf("wrapped: %w", &APIError{StatusCode: http.StatusBadRequest}),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsInvalidArgument(tt.err)
			if got != tt.expected {
				t.Errorf("IsInvalidArgument() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsAlreadyExists(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "APIError with 409",
			err:      &APIError{StatusCode: http.StatusConflict},
			expected: true,
		},
		{
			name:     "APIError with RESOURCE_ALREADY_EXISTS code",
			err:      &APIError{StatusCode: http.StatusBadRequest, Code: "RESOURCE_ALREADY_EXISTS"},
			expected: true,
		},
		{
			name:     "APIError with 400",
			err:      &APIError{StatusCode: http.StatusBadRequest},
			expected: false,
		},
		{
			name:     "wrapped APIError with 409",
			err:      fmt.Errorf("wrapped: %w", &APIError{StatusCode: http.StatusConflict}),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAlreadyExists(tt.err)
			if got != tt.expected {
				t.Errorf("IsAlreadyExists() = %v, want %v", got, tt.expected)
			}
		})
	}
}
