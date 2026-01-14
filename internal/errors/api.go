// ABOUTME: API error type for MLflow API failures.
// ABOUTME: Provides structured error information and helper functions.

package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// APIError represents an error response from the MLflow API.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	RequestID  string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("mlflow: %s: %s (status %d)", e.Code, e.Message, e.StatusCode)
	}
	return fmt.Sprintf("mlflow: %s (status %d)", e.Message, e.StatusCode)
}

// IsNotFound reports whether err indicates a resource was not found (404).
func IsNotFound(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// IsUnauthorized reports whether err indicates invalid or missing credentials (401).
func IsUnauthorized(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusUnauthorized
	}
	return false
}

// IsPermissionDenied reports whether err indicates the caller lacks permission (403).
func IsPermissionDenied(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusForbidden
	}
	return false
}

// IsInvalidArgument reports whether err indicates an invalid argument (400).
func IsInvalidArgument(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusBadRequest
	}
	return false
}

// IsAlreadyExists reports whether err indicates the resource already exists (409).
func IsAlreadyExists(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusConflict
	}
	return false
}
