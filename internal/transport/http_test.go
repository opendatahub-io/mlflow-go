// ABOUTME: Tests for HTTP transport layer.
// ABOUTME: Uses httptest.Server to verify request/response handling.

package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/ederign/mlflow-go"
)

func TestClient_Get_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("name") != "test-prompt" {
			t.Errorf("expected query param name=test-prompt, got %s", r.URL.Query().Get("name"))
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Authorization header, got %s", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL: server.URL,
		Token:   "test-token",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	var result map[string]string
	query := url.Values{"name": []string{"test-prompt"}}
	err = client.Get(context.Background(), "/api/test", query, &result)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if result["status"] != "ok" {
		t.Errorf("result = %v, want status=ok", result)
	}
}

func TestClient_Post_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "my-prompt" {
			t.Errorf("expected body.name=my-prompt, got %s", body["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"version": "1"})
	}))
	defer server.Close()

	client, err := New(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	var result map[string]string
	body := map[string]string{"name": "my-prompt"}
	err = client.Post(context.Background(), "/api/create", body, &result)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}

	if result["version"] != "1" {
		t.Errorf("result = %v, want version=1", result)
	}
}

func TestClient_Error_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error_code": "RESOURCE_DOES_NOT_EXIST",
			"message":    "Model not found",
		})
	}))
	defer server.Close()

	client, err := New(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = client.Get(context.Background(), "/api/test", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !mlflow.IsNotFound(err) {
		t.Errorf("expected IsNotFound, got %v", err)
	}

	apiErr, ok := err.(*mlflow.APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Code != "RESOURCE_DOES_NOT_EXIST" {
		t.Errorf("Code = %q, want RESOURCE_DOES_NOT_EXIST", apiErr.Code)
	}
}

func TestClient_Error_BadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error_code": "INVALID_PARAMETER_VALUE",
			"message":    "Invalid name",
		})
	}))
	defer server.Close()

	client, err := New(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = client.Post(context.Background(), "/api/test", nil, nil)
	if !mlflow.IsInvalidArgument(err) {
		t.Errorf("expected IsInvalidArgument, got %v", err)
	}
}

func TestClient_Error_Conflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"error_code": "RESOURCE_ALREADY_EXISTS",
			"message":    "Model already exists",
		})
	}))
	defer server.Close()

	client, err := New(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = client.Post(context.Background(), "/api/test", nil, nil)
	if !mlflow.IsAlreadyExists(err) {
		t.Errorf("expected IsAlreadyExists, got %v", err)
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := New(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = client.Get(ctx, "/api/test", nil, nil)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestNew_InvalidURL(t *testing.T) {
	_, err := New(Config{BaseURL: "://invalid"})
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestNew_DefaultTimeout(t *testing.T) {
	client, err := New(Config{BaseURL: "http://localhost"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("timeout = %v, want 30s", client.httpClient.Timeout)
	}
}

func TestNew_CustomTimeout(t *testing.T) {
	client, err := New(Config{
		BaseURL: "http://localhost",
		Timeout: 60 * time.Second,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if client.httpClient.Timeout != 60*time.Second {
		t.Errorf("timeout = %v, want 60s", client.httpClient.Timeout)
	}
}

func TestClient_NoAuthHeader_WhenNoToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("expected no Authorization header, got %s", auth)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := New(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = client.Get(context.Background(), "/api/test", nil, nil)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
}
