package mlflow

import (
	internalerrors "github.com/opendatahub-io/mlflow-go/internal/errors"
)

// APIError represents an error response from the MLflow API.
type APIError = internalerrors.APIError

// IsNotFound reports whether err indicates a resource was not found (404).
func IsNotFound(err error) bool {
	return internalerrors.IsNotFound(err)
}

// IsUnauthorized reports whether err indicates invalid or missing credentials (401).
func IsUnauthorized(err error) bool {
	return internalerrors.IsUnauthorized(err)
}

// IsPermissionDenied reports whether err indicates the caller lacks permission (403).
func IsPermissionDenied(err error) bool {
	return internalerrors.IsPermissionDenied(err)
}

// IsInvalidArgument reports whether err indicates an invalid argument (400).
func IsInvalidArgument(err error) bool {
	return internalerrors.IsInvalidArgument(err)
}

// IsAlreadyExists reports whether err indicates the resource already exists (409).
func IsAlreadyExists(err error) bool {
	return internalerrors.IsAlreadyExists(err)
}

// IsAliasConflict reports whether err indicates the operation failed
// because aliases point to the resource (HTTP 409 Conflict without RESOURCE_ALREADY_EXISTS code).
// Note: MLflow OSS silently removes aliases on version deletion; this only triggers on Databricks.
func IsAliasConflict(err error) bool {
	return internalerrors.IsAliasConflict(err)
}
