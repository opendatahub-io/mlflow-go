package errors

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode"
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
		return fmt.Sprintf("mlflow: %s: %s (status %d)", escapeControls(e.Code), escapeControls(e.Message), e.StatusCode)
	}
	return fmt.Sprintf("mlflow: %s (status %d)", escapeControls(e.Message), e.StatusCode)
}

func escapeControls(value string) string {
	var result strings.Builder
	start := 0
	for i, r := range value {
		if unicode.IsControl(r) || unicode.Is(unicode.Bidi_Control, r) || r == '\u2028' || r == '\u2029' {
			result.WriteString(value[start:i])
			quoted := strconv.Quote(string(r))
			result.WriteString(quoted[1 : len(quoted)-1])
			start = i + len(string(r))
		}
	}
	result.WriteString(value[start:])
	return result.String()
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

// IsAlreadyExists reports whether err indicates the resource already exists.
// Checks for HTTP 409 Conflict or MLflow's RESOURCE_ALREADY_EXISTS error code.
func IsAlreadyExists(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusConflict ||
			apiErr.Code == "RESOURCE_ALREADY_EXISTS"
	}
	return false
}

// IsAliasConflict reports whether err indicates the operation failed
// because aliases point to the resource (HTTP 409 Conflict without a
// RESOURCE_ALREADY_EXISTS code).
// Note: MLflow OSS silently removes aliases on version deletion; this only triggers on Databricks.
func IsAliasConflict(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusConflict &&
			apiErr.Code != "RESOURCE_ALREADY_EXISTS"
	}
	return false
}
