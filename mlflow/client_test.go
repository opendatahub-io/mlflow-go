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
	// Save and restore env var
	saved := os.Getenv("MLFLOW_TRACKING_URI")
	os.Unsetenv("MLFLOW_TRACKING_URI")
	defer func() {
		if saved != "" {
			os.Setenv("MLFLOW_TRACKING_URI", saved)
		}
	}()

	_, err := NewClient()
	if err == nil {
		t.Error("expected error for missing tracking URI")
	}
}

func TestNewClient_FromEnvVar(t *testing.T) {
	saved := os.Getenv("MLFLOW_TRACKING_URI")
	os.Setenv("MLFLOW_TRACKING_URI", "https://mlflow.test.com")
	defer func() {
		if saved != "" {
			os.Setenv("MLFLOW_TRACKING_URI", saved)
		} else {
			os.Unsetenv("MLFLOW_TRACKING_URI")
		}
	}()

	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if client.TrackingURI() != "https://mlflow.test.com" {
		t.Errorf("TrackingURI() = %q, want %q", client.TrackingURI(), "https://mlflow.test.com")
	}
}

func TestNewClient_ExplicitOverridesEnv(t *testing.T) {
	saved := os.Getenv("MLFLOW_TRACKING_URI")
	os.Setenv("MLFLOW_TRACKING_URI", "https://env.example.com")
	defer func() {
		if saved != "" {
			os.Setenv("MLFLOW_TRACKING_URI", saved)
		} else {
			os.Unsetenv("MLFLOW_TRACKING_URI")
		}
	}()

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
	// Save and restore insecure env var
	savedInsecure := os.Getenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY")
	os.Unsetenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY")
	defer func() {
		if savedInsecure != "" {
			os.Setenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY", savedInsecure)
		}
	}()

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
	saved := os.Getenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY")
	os.Setenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY", "true")
	defer func() {
		if saved != "" {
			os.Setenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY", saved)
		} else {
			os.Unsetenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY")
		}
	}()

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
	saved := os.Getenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY")
	os.Setenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY", "1")
	defer func() {
		if saved != "" {
			os.Setenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY", saved)
		} else {
			os.Unsetenv("MLFLOW_INSECURE_SKIP_TLS_VERIFY")
		}
	}()

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

func TestNewClient_InvalidURI(t *testing.T) {
	_, err := NewClient(
		WithTrackingURI("://invalid"),
	)
	if err == nil {
		t.Error("expected error for invalid URI")
	}
}

func TestClient_PromptRegistry_ReturnsSameInstance(t *testing.T) {
	client, err := NewClient(
		WithTrackingURI("https://mlflow.example.com"),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	pr1 := client.PromptRegistry()
	pr2 := client.PromptRegistry()

	if pr1 != pr2 {
		t.Error("PromptRegistry() should return same instance")
	}
}
