package transport

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/opendatahub-io/mlflow-go/internal/errors"
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
		Headers: map[string]string{"Authorization": "Bearer test-token"},
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

	if !errors.IsNotFound(err) {
		t.Errorf("expected IsNotFound, got %v", err)
	}

	apiErr, ok := err.(*errors.APIError)
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
	if !errors.IsInvalidArgument(err) {
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
	if !errors.IsAlreadyExists(err) {
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

func TestClient_TimeoutExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL: server.URL,
		Timeout: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = client.Get(context.Background(), "/api/test", nil, nil)
	if err == nil {
		t.Error("expected timeout error")
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

// testLogHandler captures log records for testing.
type testLogHandler struct {
	records []testLogRecord
}

type testLogRecord struct {
	Level   string
	Message string
	Attrs   map[string]any
}

func (h *testLogHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *testLogHandler) Handle(_ context.Context, r slog.Record) error {
	record := testLogRecord{
		Level:   r.Level.String(),
		Message: r.Message,
		Attrs:   make(map[string]any),
	}
	r.Attrs(func(a slog.Attr) bool {
		record.Attrs[a.Key] = a.Value.Any()
		return true
	})
	h.records = append(h.records, record)
	return nil
}

func (h *testLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *testLogHandler) WithGroup(name string) slog.Handler {
	return h
}

func TestClient_LogsRequestAndResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	handler := &testLogHandler{}
	logger := slog.New(handler)

	client, err := New(Config{
		BaseURL: server.URL,
		Headers: map[string]string{"Authorization": "Bearer secret-token"},
		Logger:  logger,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	var result map[string]string
	err = client.Get(context.Background(), "/api/test", nil, &result)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// Should have 2 log records: request and response
	if len(handler.records) != 2 {
		t.Fatalf("expected 2 log records, got %d", len(handler.records))
	}

	// Check request log
	reqLog := handler.records[0]
	if reqLog.Message != "request" {
		t.Errorf("request log message = %q, want %q", reqLog.Message, "request")
	}
	if reqLog.Attrs["method"] != "GET" {
		t.Errorf("request log method = %v, want GET", reqLog.Attrs["method"])
	}
	if reqLog.Attrs["url"] == nil {
		t.Error("request log should have url")
	}

	// Check response log
	respLog := handler.records[1]
	if respLog.Message != "response" {
		t.Errorf("response log message = %q, want %q", respLog.Message, "response")
	}
	if respLog.Attrs["status"] != int64(200) {
		t.Errorf("response log status = %v, want 200", respLog.Attrs["status"])
	}
	if respLog.Attrs["duration_ms"] == nil {
		t.Error("response log should have duration_ms")
	}
}

func TestClient_NoLogsWithoutLogger(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// No logger provided
	client, err := New(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// This should not panic or fail even without a logger
	err = client.Get(context.Background(), "/api/test", nil, nil)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
}

func TestClient_LogsNeverIncludeSecrets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"secret": "should-not-be-logged"}`))
	}))
	defer server.Close()

	handler := &testLogHandler{}
	logger := slog.New(handler)

	client, err := New(Config{
		BaseURL: server.URL,
		Headers: map[string]string{"Authorization": "Bearer super-secret-token"},
		Logger:  logger,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	body := map[string]string{"password": "secret123", "template": "Hello {{name}}"}
	err = client.Post(context.Background(), "/api/test", body, nil)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}

	// Verify no secrets in logs
	for _, record := range handler.records {
		for key, val := range record.Attrs {
			strVal, ok := val.(string)
			if !ok {
				continue
			}
			// Check that sensitive data is not logged
			if key == "token" || key == "password" || key == "secret" {
				t.Errorf("sensitive key %q should not be logged", key)
			}
			if strVal == "super-secret-token" || strVal == "secret123" {
				t.Errorf("sensitive value should not be logged: %s=%s", key, strVal)
			}
			// Body content should not be logged
			if strVal == "Hello {{name}}" {
				t.Errorf("request body content should not be logged")
			}
			if strVal == "should-not-be-logged" {
				t.Errorf("response body content should not be logged")
			}
		}
	}
}

func TestCustomHeadersSentOnRequest(t *testing.T) {
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL: server.URL,
		Headers: map[string]string{
			"X-MLFLOW-WORKSPACE": "team-bella",
			"X-Custom":           "value-123",
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = client.Get(context.Background(), "/test", nil, nil)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got := receivedHeaders.Get("X-MLFLOW-WORKSPACE"); got != "team-bella" {
		t.Errorf("X-MLFLOW-WORKSPACE = %q, want %q", got, "team-bella")
	}
	if got := receivedHeaders.Get("X-Custom"); got != "value-123" {
		t.Errorf("X-Custom = %q, want %q", got, "value-123")
	}
	// Standard headers should still be present
	if got := receivedHeaders.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}
}

func TestCustomHeadersWithToken(t *testing.T) {
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL: server.URL,
		Headers: map[string]string{
			"Authorization":      "Bearer my-token",
			"X-MLFLOW-WORKSPACE": "team-dora",
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = client.Get(context.Background(), "/test", nil, nil)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got := receivedHeaders.Get("Authorization"); got != "Bearer my-token" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer my-token")
	}
	if got := receivedHeaders.Get("X-MLFLOW-WORKSPACE"); got != "team-dora" {
		t.Errorf("X-MLFLOW-WORKSPACE = %q, want %q", got, "team-dora")
	}
}
