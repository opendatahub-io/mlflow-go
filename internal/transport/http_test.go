package transport

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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

func TestClient_Patch_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["status"] != "active" {
			t.Errorf("expected body.status=active, got %s", body["status"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "active"})
	}))
	defer server.Close()

	client, err := New(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	var result map[string]string
	body := map[string]string{"status": "active"}
	err = client.Patch(context.Background(), "/api/update", body, &result)
	if err != nil {
		t.Fatalf("Patch() error = %v", err)
	}

	if result["status"] != "active" {
		t.Errorf("result = %v, want status=active", result)
	}
}

func TestClient_Patch_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error_code": "INVALID_PARAMETER_VALUE",
			"message":    "Invalid status",
		})
	}))
	defer server.Close()

	client, err := New(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = client.Patch(context.Background(), "/api/update", map[string]string{"status": "bogus"}, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*errors.APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusBadRequest)
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

func TestClient_GetBytes_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("Accept") == "application/json" {
			t.Error("GetBytes should not set Accept: application/json")
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte("artifact-data"))
	}))
	defer server.Close()

	client, err := New(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	data, contentType, err := client.GetBytes(context.Background(), "/api/artifacts/file", nil)
	if err != nil {
		t.Fatalf("GetBytes() error = %v", err)
	}
	if string(data) != "artifact-data" {
		t.Errorf("data = %q, want artifact-data", string(data))
	}
	if contentType != "application/octet-stream" {
		t.Errorf("contentType = %q, want application/octet-stream", contentType)
	}
}

func TestClient_GetBody_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Write([]byte("stream-data"))
	}))
	defer server.Close()

	client, err := New(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	rc, err := client.GetBody(context.Background(), "/api/artifacts/file", nil)
	if err != nil {
		t.Fatalf("GetBody() error = %v", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(data) != "stream-data" {
		t.Errorf("data = %q, want stream-data", string(data))
	}
}

func TestClient_PutBytes_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/octet-stream" {
			t.Errorf("Content-Type = %q, want application/octet-stream", ct)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "upload-data" {
			t.Errorf("body = %q, want upload-data", string(body))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := New(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = client.PutBytes(context.Background(), "/api/artifacts/file", []byte("upload-data"), "application/octet-stream")
	if err != nil {
		t.Fatalf("PutBytes() error = %v", err)
	}
}

func TestClient_PostBytes_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Query().Get("run_uuid") != "run-1" {
			t.Errorf("run_uuid = %q, want run-1", r.URL.Query().Get("run_uuid"))
		}
		if ct := r.Header.Get("Content-Type"); ct != "text/plain" {
			t.Errorf("Content-Type = %q, want text/plain", ct)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "post-data" {
			t.Errorf("body = %q, want post-data", string(body))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := New(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	query := url.Values{"run_uuid": []string{"run-1"}}
	err = client.PostBytes(context.Background(), "/upload", query, []byte("post-data"), "text/plain")
	if err != nil {
		t.Fatalf("PostBytes() error = %v", err)
	}
}

func TestClient_GetBytes_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error_code": "RESOURCE_DOES_NOT_EXIST",
			"message":    "Artifact not found",
		})
	}))
	defer server.Close()

	client, err := New(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, _, err = client.GetBytes(context.Background(), "/api/artifacts/missing", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.IsNotFound(err) {
		t.Errorf("expected IsNotFound, got %v", err)
	}
}

func TestClient_DoAbsolute_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.Header.Get("X-Amz-Signature") != "abc123" {
			t.Errorf("expected X-Amz-Signature header")
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "presigned-upload" {
			t.Errorf("body = %q, want presigned-upload", string(body))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := New(Config{BaseURL: "http://localhost:9999"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	data, _, err := client.DoAbsolute(
		context.Background(),
		http.MethodPut,
		server.URL+"/presigned",
		map[string]string{"X-Amz-Signature": "abc123", "Content-Type": "application/octet-stream"},
		[]byte("presigned-upload"),
	)
	if err != nil {
		t.Fatalf("DoAbsolute() error = %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty response body, got %q", string(data))
	}
}

func TestReadResponseBody_ExceedsLimit(t *testing.T) {
	_, err := readResponseBody(strings.NewReader(strings.Repeat("a", maxResponseBodySize+1)))
	if err == nil {
		t.Fatal("expected error for oversized body, got nil")
	}
}

func TestLimitedReadCloser_CopyStopsAtLimit(t *testing.T) {
	const limit int64 = 8
	body := io.NopCloser(strings.NewReader(strings.Repeat("x", int(limit)+4)))
	rc := newLimitedReadCloser(body, limit)

	var buf strings.Builder
	n, err := io.Copy(&buf, rc)
	if err == nil {
		t.Fatal("expected size error, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Fatalf("error = %v, want exceeds maximum size", err)
	}
	if n != limit {
		t.Fatalf("copied %d bytes, want %d", n, limit)
	}
	if int64(buf.Len()) != limit {
		t.Fatalf("buffer len = %d, want %d", buf.Len(), limit)
	}

	// Subsequent reads must keep reporting the exceeded error with no extra bytes.
	extra := make([]byte, 4)
	n2, err2 := rc.Read(extra)
	if n2 != 0 {
		t.Fatalf("subsequent Read returned %d bytes, want 0", n2)
	}
	if err2 == nil || !strings.Contains(err2.Error(), "exceeds maximum size") {
		t.Fatalf("subsequent Read error = %v, want exceeds maximum size", err2)
	}
}

func TestRedactAbsoluteURLForLog(t *testing.T) {
	got := redactAbsoluteURLForLog("https://storage.example.com/object?X-Amz-Signature=secret&X-Amz-Credential=abc")
	want := "https://storage.example.com/object"
	if got != want {
		t.Errorf("redactAbsoluteURLForLog() = %q, want %q", got, want)
	}
}

func TestClient_DoAbsoluteGetBody_RedactsPresignedURLInLogs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery == "" {
			t.Error("request must keep presigned query params")
		}
		w.Write([]byte("artifact-bytes"))
	}))
	defer server.Close()

	handler := &testLogHandler{}
	logger := slog.New(handler)

	client, err := New(Config{
		BaseURL: "http://localhost:9999",
		Logger:  logger,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	presignedURL := server.URL + "/object?X-Amz-Signature=secret&X-Amz-Credential=abc"
	rc, err := client.DoAbsoluteGetBody(context.Background(), presignedURL, nil)
	if err != nil {
		t.Fatalf("DoAbsoluteGetBody() error = %v", err)
	}
	defer rc.Close()

	if _, err := io.ReadAll(rc); err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	var requestLogs int
	for _, record := range handler.records {
		if record.Message != "request" {
			continue
		}
		requestLogs++
		urlAttr, _ := record.Attrs["url"].(string)
		if strings.Contains(urlAttr, "X-Amz-Signature") || strings.Contains(urlAttr, "secret") {
			t.Errorf("request log url leaked presigned query params: %q", urlAttr)
		}
		if !strings.HasSuffix(urlAttr, "/object") {
			t.Errorf("request log url = %q, want host/path without query", urlAttr)
		}
	}
	if requestLogs != 1 {
		t.Fatalf("expected 1 request log, got %d", requestLogs)
	}
}

func TestNew_RejectsInsecureWithToken(t *testing.T) {
	_, err := New(Config{
		BaseURL:  "https://mlflow.example.com",
		Insecure: true,
		Token:    "secret",
	})
	if err == nil {
		t.Fatal("expected error when Insecure and Token are both set")
	}
	if !strings.Contains(err.Error(), "insecure") {
		t.Errorf("error = %v, want mention of insecure", err)
	}
}

func TestNew_RejectsInsecureWithTokenPath(t *testing.T) {
	_, err := New(Config{
		BaseURL:   "https://mlflow.example.com",
		Insecure:  true,
		TokenPath: "/var/run/secrets/token",
	})
	if err == nil {
		t.Fatal("expected error when Insecure and TokenPath are both set")
	}
}

func TestNew_AllowsInsecureWithoutToken(t *testing.T) {
	_, err := New(Config{
		BaseURL:  "https://mlflow.example.com",
		Insecure: true,
	})
	if err != nil {
		t.Fatalf("New() error = %v; Insecure without token should be allowed", err)
	}
}
