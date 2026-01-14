// ABOUTME: Tests for the main SDK client.
// ABOUTME: Verifies client initialization from options and environment variables.

package mlflow

import (
	"os"
	"testing"
)

func TestNewClient_WithTrackingURI(t *testing.T) {
	client, err := NewClient(
		WithTrackingURI("https://mlflow.example.com"),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if client.TrackingURI() != "https://mlflow.example.com" {
		t.Errorf("TrackingURI() = %q, want %q", client.TrackingURI(), "https://mlflow.example.com")
	}
}

func TestNewClient_MissingTrackingURI(t *testing.T) {
	// Clear env var to ensure it's not set
	os.Unsetenv("MLFLOW_TRACKING_URI")

	_, err := NewClient()
	if err == nil {
		t.Error("expected error for missing tracking URI")
	}
}

func TestNewClient_FromEnvVar(t *testing.T) {
	os.Setenv("MLFLOW_TRACKING_URI", "https://mlflow.test.com")
	defer os.Unsetenv("MLFLOW_TRACKING_URI")

	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if client.TrackingURI() != "https://mlflow.test.com" {
		t.Errorf("TrackingURI() = %q, want %q", client.TrackingURI(), "https://mlflow.test.com")
	}
}

func TestNewClient_ExplicitOverridesEnv(t *testing.T) {
	os.Setenv("MLFLOW_TRACKING_URI", "https://env.example.com")
	defer os.Unsetenv("MLFLOW_TRACKING_URI")

	client, err := NewClient(
		WithTrackingURI("https://explicit.example.com"),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Explicit option should take precedence over env var
	if client.TrackingURI() != "https://explicit.example.com" {
		t.Errorf("TrackingURI() = %q, want %q", client.TrackingURI(), "https://explicit.example.com")
	}
}

func TestNewClient_HTTPRejectedByDefault(t *testing.T) {
	_, err := NewClient(
		WithTrackingURI("http://mlflow.example.com"),
	)
	if err == nil {
		t.Error("expected error for HTTP URI without insecure mode")
	}
}

func TestNewClient_HTTPAllowedWithInsecure(t *testing.T) {
	client, err := NewClient(
		WithTrackingURI("http://localhost:5000"),
		WithInsecure(),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if !client.IsInsecure() {
		t.Error("IsInsecure() should be true")
	}
}

func TestNewClient_HTTPAllowedWithEnvVar(t *testing.T) {
	os.Setenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY", "true")
	defer os.Unsetenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY")

	client, err := NewClient(
		WithTrackingURI("http://localhost:5000"),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if !client.IsInsecure() {
		t.Error("IsInsecure() should be true")
	}
}

func TestNewClient_InsecureEnvVar_One(t *testing.T) {
	os.Setenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY", "1")
	defer os.Unsetenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY")

	client, err := NewClient(
		WithTrackingURI("http://localhost:5000"),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if !client.IsInsecure() {
		t.Error("IsInsecure() should be true for '1'")
	}
}

func TestNewClient_TokenFromEnv(t *testing.T) {
	os.Setenv("MLFLOW_TRACKING_URI", "https://mlflow.example.com")
	os.Setenv("MLFLOW_TRACKING_TOKEN", "secret-token")
	defer func() {
		os.Unsetenv("MLFLOW_TRACKING_URI")
		os.Unsetenv("MLFLOW_TRACKING_TOKEN")
	}()

	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// We can't directly check the token, but verify client was created
	if client == nil {
		t.Error("client should not be nil")
	}
}

func TestNewClient_InvalidURI(t *testing.T) {
	_, err := NewClient(
		WithTrackingURI("://invalid"),
	)
	if err == nil {
		t.Error("expected error for invalid URI")
	}
}
