package tracking

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opendatahub-io/mlflow-go/internal/errors"
	"github.com/opendatahub-io/mlflow-go/internal/transport"
)

func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	tc, err := transport.New(transport.Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("transport.New() error = %v", err)
	}

	return NewClient(tc)
}

func mustDecodeJSON(t *testing.T, r *http.Request, dst any) {
	t.Helper()
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
}

func mustEncodeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("failed to encode response: %v", err)
	}
}

// --- CreateExperiment tests ---

func TestCreateExperiment_Success(t *testing.T) {
	var receivedName string

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path != "/api/2.0/mlflow/experiments/create" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		var req struct {
			Name string `json:"name"`
		}
		mustDecodeJSON(t, r, &req)
		receivedName = req.Name

		mustEncodeJSON(t, w, map[string]any{
			"experiment_id": "123",
		})
	}))

	id, err := client.CreateExperiment(context.Background(), "my-experiment")
	if err != nil {
		t.Fatalf("CreateExperiment() error = %v", err)
	}

	if id != "123" {
		t.Errorf("experiment ID = %q, want %q", id, "123")
	}
	if receivedName != "my-experiment" {
		t.Errorf("name = %q, want %q", receivedName, "my-experiment")
	}
}

func TestCreateExperiment_EmptyName(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	_, err := client.CreateExperiment(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestCreateExperiment_WithTags(t *testing.T) {
	var receivedTags []map[string]string

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req struct {
			Tags []map[string]string `json:"tags"`
		}
		mustDecodeJSON(t, r, &req)
		receivedTags = req.Tags

		mustEncodeJSON(t, w, map[string]any{
			"experiment_id": "456",
		})
	}))

	_, err := client.CreateExperiment(context.Background(), "tagged-exp",
		WithExperimentTags(map[string]string{"team": "ml"}),
	)
	if err != nil {
		t.Fatalf("CreateExperiment() error = %v", err)
	}

	foundTeam := false
	for _, tag := range receivedTags {
		if tag["key"] == "team" && tag["value"] == "ml" {
			foundTeam = true
		}
	}
	if !foundTeam {
		t.Error("expected team tag to be sent")
	}
}

func TestCreateExperiment_AlreadyExists(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		mustEncodeJSON(t, w, map[string]string{
			"error_code": "RESOURCE_ALREADY_EXISTS",
			"message":    "Experiment already exists",
		})
	}))

	_, err := client.CreateExperiment(context.Background(), "existing")
	if err == nil {
		t.Error("expected error for existing experiment")
	}
	if !errors.IsAlreadyExists(err) {
		t.Errorf("expected IsAlreadyExists, got %v", err)
	}
}

// --- GetExperiment tests ---

func TestGetExperiment_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path != "/api/2.0/mlflow/experiments/get" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		if r.URL.Query().Get("experiment_id") != "123" {
			t.Errorf("experiment_id = %q, want %q", r.URL.Query().Get("experiment_id"), "123")
		}

		mustEncodeJSON(t, w, map[string]any{
			"experiment": map[string]any{
				"experiment_id":     "123",
				"name":              "my-experiment",
				"artifact_location": "/artifacts",
				"lifecycle_stage":   "active",
				"creation_time":     1700000000000,
				"last_update_time":  1700000100000,
				"tags": []map[string]string{
					{"key": "team", "value": "ml"},
				},
			},
		})
	}))

	exp, err := client.GetExperiment(context.Background(), "123")
	if err != nil {
		t.Fatalf("GetExperiment() error = %v", err)
	}

	if exp.ID != "123" {
		t.Errorf("ID = %q, want %q", exp.ID, "123")
	}
	if exp.Name != "my-experiment" {
		t.Errorf("Name = %q, want %q", exp.Name, "my-experiment")
	}
	if exp.ArtifactLocation != "/artifacts" {
		t.Errorf("ArtifactLocation = %q, want %q", exp.ArtifactLocation, "/artifacts")
	}
	if exp.Tags["team"] != "ml" {
		t.Errorf("Tags[team] = %q, want %q", exp.Tags["team"], "ml")
	}
	if exp.CreationTime.IsZero() {
		t.Error("CreationTime should not be zero")
	}
}

func TestGetExperiment_EmptyID(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	_, err := client.GetExperiment(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty ID")
	}
}

func TestGetExperiment_NotFound(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		mustEncodeJSON(t, w, map[string]string{
			"error_code": "RESOURCE_DOES_NOT_EXIST",
			"message":    "Experiment not found",
		})
	}))

	_, err := client.GetExperiment(context.Background(), "999")
	if err == nil {
		t.Error("expected error for non-existent experiment")
	}
	if !errors.IsNotFound(err) {
		t.Errorf("expected IsNotFound, got %v", err)
	}
}

// --- GetExperimentByName tests ---

func TestGetExperimentByName_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path != "/api/2.0/mlflow/experiments/get-by-name" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		if r.URL.Query().Get("experiment_name") != "my-experiment" {
			t.Errorf("experiment_name = %q, want %q", r.URL.Query().Get("experiment_name"), "my-experiment")
		}

		mustEncodeJSON(t, w, map[string]any{
			"experiment": map[string]any{
				"experiment_id": "123",
				"name":          "my-experiment",
			},
		})
	}))

	exp, err := client.GetExperimentByName(context.Background(), "my-experiment")
	if err != nil {
		t.Fatalf("GetExperimentByName() error = %v", err)
	}

	if exp.Name != "my-experiment" {
		t.Errorf("Name = %q, want %q", exp.Name, "my-experiment")
	}
}

func TestGetExperimentByName_EmptyName(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	_, err := client.GetExperimentByName(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

// --- DeleteExperiment tests ---

func TestDeleteExperiment_Success(t *testing.T) {
	var deleteCalled bool

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path != "/api/2.0/mlflow/experiments/delete" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		deleteCalled = true
		mustEncodeJSON(t, w, map[string]any{})
	}))

	err := client.DeleteExperiment(context.Background(), "123")
	if err != nil {
		t.Fatalf("DeleteExperiment() error = %v", err)
	}

	if !deleteCalled {
		t.Error("expected delete endpoint to be called")
	}
}

func TestDeleteExperiment_EmptyID(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	err := client.DeleteExperiment(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty ID")
	}
}

// --- UpdateExperiment tests ---

func TestUpdateExperiment_Success(t *testing.T) {
	var receivedID, receivedName string

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path != "/api/2.0/mlflow/experiments/update" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		var req struct {
			ExperimentID string `json:"experiment_id"`
			NewName      string `json:"new_name"`
		}
		mustDecodeJSON(t, r, &req)
		receivedID = req.ExperimentID
		receivedName = req.NewName

		mustEncodeJSON(t, w, map[string]any{})
	}))

	err := client.UpdateExperiment(context.Background(), "123", "renamed")
	if err != nil {
		t.Fatalf("UpdateExperiment() error = %v", err)
	}

	if receivedID != "123" {
		t.Errorf("experiment_id = %q, want %q", receivedID, "123")
	}
	if receivedName != "renamed" {
		t.Errorf("new_name = %q, want %q", receivedName, "renamed")
	}
}

func TestUpdateExperiment_EmptyID(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	err := client.UpdateExperiment(context.Background(), "", "name")
	if err == nil {
		t.Error("expected error for empty ID")
	}
}

func TestUpdateExperiment_EmptyName(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	err := client.UpdateExperiment(context.Background(), "123", "")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

// --- SearchExperiments tests ---

func TestSearchExperiments_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path != "/api/2.0/mlflow/experiments/search" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		mustEncodeJSON(t, w, map[string]any{
			"experiments": []map[string]any{
				{
					"experiment_id": "1",
					"name":          "exp-1",
				},
				{
					"experiment_id": "2",
					"name":          "exp-2",
				},
			},
			"next_page_token": "token123",
		})
	}))

	result, err := client.SearchExperiments(context.Background())
	if err != nil {
		t.Fatalf("SearchExperiments() error = %v", err)
	}

	if len(result.Experiments) != 2 {
		t.Errorf("got %d experiments, want 2", len(result.Experiments))
	}
	if result.NextPageToken != "token123" {
		t.Errorf("NextPageToken = %q, want %q", result.NextPageToken, "token123")
	}
	if result.Experiments[0].Name != "exp-1" {
		t.Errorf("first experiment name = %q, want %q", result.Experiments[0].Name, "exp-1")
	}
}

func TestSearchExperiments_Empty(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mustEncodeJSON(t, w, map[string]any{
			"experiments": []map[string]any{},
		})
	}))

	result, err := client.SearchExperiments(context.Background())
	if err != nil {
		t.Fatalf("SearchExperiments() error = %v", err)
	}

	if result.Experiments == nil {
		t.Error("Experiments should not be nil, should be empty slice")
	}
	if len(result.Experiments) != 0 {
		t.Errorf("got %d experiments, want 0", len(result.Experiments))
	}
}

func TestSearchExperiments_InvalidViewType(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	_, err := client.SearchExperiments(context.Background(),
		WithExperimentsViewType("INVALID"),
	)
	if err == nil {
		t.Error("expected error for invalid view type")
	}
}

// --- SetExperimentTag tests ---

func TestSetExperimentTag_Success(t *testing.T) {
	var receivedID, receivedKey, receivedValue string

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req struct {
			ExperimentID string `json:"experiment_id"`
			Key          string `json:"key"`
			Value        string `json:"value"`
		}
		mustDecodeJSON(t, r, &req)
		receivedID = req.ExperimentID
		receivedKey = req.Key
		receivedValue = req.Value

		mustEncodeJSON(t, w, map[string]any{})
	}))

	err := client.SetExperimentTag(context.Background(), "123", "env", "prod")
	if err != nil {
		t.Fatalf("SetExperimentTag() error = %v", err)
	}

	if receivedID != "123" {
		t.Errorf("experiment_id = %q, want %q", receivedID, "123")
	}
	if receivedKey != "env" {
		t.Errorf("key = %q, want %q", receivedKey, "env")
	}
	if receivedValue != "prod" {
		t.Errorf("value = %q, want %q", receivedValue, "prod")
	}
}

func TestSetExperimentTag_EmptyID(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	err := client.SetExperimentTag(context.Background(), "", "key", "value")
	if err == nil {
		t.Error("expected error for empty ID")
	}
}

func TestSetExperimentTag_EmptyKey(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	err := client.SetExperimentTag(context.Background(), "123", "", "value")
	if err == nil {
		t.Error("expected error for empty key")
	}
}

// --- CreateRun tests ---

func TestCreateRun_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path != "/api/2.0/mlflow/runs/create" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		mustEncodeJSON(t, w, map[string]any{
			"run": map[string]any{
				"info": map[string]any{
					"run_id":        "abc-123",
					"experiment_id": "1",
					"status":        "RUNNING",
					"start_time":    1700000000000,
					"artifact_uri":  "/artifacts/abc-123",
				},
				"data": map[string]any{
					"metrics": []any{},
					"params":  []any{},
					"tags":    []any{},
				},
			},
		})
	}))

	run, err := client.CreateRun(context.Background(), "1",
		WithRunName("test-run"),
	)
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	if run.Info.RunID != "abc-123" {
		t.Errorf("RunID = %q, want %q", run.Info.RunID, "abc-123")
	}
	if run.Info.Status != RunStatusRunning {
		t.Errorf("Status = %q, want %q", run.Info.Status, RunStatusRunning)
	}
	if run.Info.ExperimentID != "1" {
		t.Errorf("ExperimentID = %q, want %q", run.Info.ExperimentID, "1")
	}
}

func TestCreateRun_EmptyExperimentID(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	_, err := client.CreateRun(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty experiment ID")
	}
}

// --- GetRun tests ---

func TestGetRun_Success(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path != "/api/2.0/mlflow/runs/get" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		if r.URL.Query().Get("run_id") != "abc-123" {
			t.Errorf("run_id = %q, want %q", r.URL.Query().Get("run_id"), "abc-123")
		}

		mustEncodeJSON(t, w, map[string]any{
			"run": map[string]any{
				"info": map[string]any{
					"run_id":        "abc-123",
					"experiment_id": "1",
					"run_name":      "test-run",
					"status":        "FINISHED",
				},
				"data": map[string]any{
					"metrics": []map[string]any{
						{"key": "rmse", "value": 0.5, "timestamp": 1700000000000, "step": 1},
					},
					"params": []map[string]any{
						{"key": "lr", "value": "0.01"},
					},
					"tags": []map[string]any{
						{"key": "model", "value": "sklearn"},
					},
				},
			},
		})
	}))

	run, err := client.GetRun(context.Background(), "abc-123")
	if err != nil {
		t.Fatalf("GetRun() error = %v", err)
	}

	if run.Info.RunID != "abc-123" {
		t.Errorf("RunID = %q, want %q", run.Info.RunID, "abc-123")
	}
	if run.Info.RunName != "test-run" {
		t.Errorf("RunName = %q, want %q", run.Info.RunName, "test-run")
	}
	if run.Info.Status != RunStatusFinished {
		t.Errorf("Status = %q, want %q", run.Info.Status, RunStatusFinished)
	}

	if len(run.Data.Metrics) != 1 {
		t.Fatalf("got %d metrics, want 1", len(run.Data.Metrics))
	}
	if run.Data.Metrics[0].Key != "rmse" {
		t.Errorf("metric key = %q, want %q", run.Data.Metrics[0].Key, "rmse")
	}
	if run.Data.Metrics[0].Value != 0.5 {
		t.Errorf("metric value = %f, want %f", run.Data.Metrics[0].Value, 0.5)
	}

	if len(run.Data.Params) != 1 {
		t.Fatalf("got %d params, want 1", len(run.Data.Params))
	}
	if run.Data.Params[0].Key != "lr" || run.Data.Params[0].Value != "0.01" {
		t.Errorf("param = %+v", run.Data.Params[0])
	}

	if run.Data.Tags["model"] != "sklearn" {
		t.Errorf("Tags[model] = %q, want %q", run.Data.Tags["model"], "sklearn")
	}
}

func TestGetRun_EmptyID(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	_, err := client.GetRun(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty run ID")
	}
}

func TestGetRun_NotFound(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		mustEncodeJSON(t, w, map[string]string{
			"error_code": "RESOURCE_DOES_NOT_EXIST",
			"message":    "Run not found",
		})
	}))

	_, err := client.GetRun(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent run")
	}
	if !errors.IsNotFound(err) {
		t.Errorf("expected IsNotFound, got %v", err)
	}
}

// --- UpdateRun tests ---

func TestUpdateRun_Success(t *testing.T) {
	var receivedRunID string
	var receivedStatus int

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req struct {
			RunID  string `json:"run_id"`
			Status int    `json:"status"`
		}
		mustDecodeJSON(t, r, &req)
		receivedRunID = req.RunID
		receivedStatus = req.Status

		mustEncodeJSON(t, w, map[string]any{
			"run_info": map[string]any{
				"run_id":        "abc-123",
				"experiment_id": "1",
				"status":        "FINISHED",
				"end_time":      1700000100000,
			},
		})
	}))

	info, err := client.UpdateRun(context.Background(), "abc-123", WithStatus(RunStatusFinished))
	if err != nil {
		t.Fatalf("UpdateRun() error = %v", err)
	}

	if receivedRunID != "abc-123" {
		t.Errorf("run_id = %q, want %q", receivedRunID, "abc-123")
	}
	// RunStatus_FINISHED = 3 in protobuf enum
	if receivedStatus != 3 {
		t.Errorf("status = %d, want 3 (FINISHED)", receivedStatus)
	}
	if info.Status != RunStatusFinished {
		t.Errorf("info.Status = %q, want %q", info.Status, RunStatusFinished)
	}
}

func TestUpdateRun_EmptyID(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	_, err := client.UpdateRun(context.Background(), "", WithStatus(RunStatusFinished))
	if err == nil {
		t.Error("expected error for empty run ID")
	}
}

func TestUpdateRun_InvalidStatus(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	_, err := client.UpdateRun(context.Background(), "abc-123", WithStatus(RunStatus("INVALID")))
	if err == nil {
		t.Error("expected error for invalid status")
	}
}

func TestUpdateRun_NameOnly(t *testing.T) {
	var receivedRunName string
	var receivedStatus int

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req struct {
			RunName string `json:"run_name"`
			Status  int    `json:"status"`
		}
		mustDecodeJSON(t, r, &req)
		receivedRunName = req.RunName
		receivedStatus = req.Status

		mustEncodeJSON(t, w, map[string]any{
			"run_info": map[string]any{
				"run_id":   "abc-123",
				"run_name": "renamed-run",
				"status":   "RUNNING",
			},
		})
	}))

	info, err := client.UpdateRun(context.Background(), "abc-123", WithRunNameUpdate("renamed-run"))
	if err != nil {
		t.Fatalf("UpdateRun() error = %v", err)
	}

	if receivedRunName != "renamed-run" {
		t.Errorf("run_name = %q, want %q", receivedRunName, "renamed-run")
	}
	// Status should be 0 (not set) when not provided
	if receivedStatus != 0 {
		t.Errorf("status = %d, want 0 (not set)", receivedStatus)
	}
	if info.RunName != "renamed-run" {
		t.Errorf("info.RunName = %q, want %q", info.RunName, "renamed-run")
	}
}

// --- DeleteRun tests ---

func TestDeleteRun_Success(t *testing.T) {
	var deleteCalled bool

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path != "/api/2.0/mlflow/runs/delete" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		deleteCalled = true
		mustEncodeJSON(t, w, map[string]any{})
	}))

	err := client.DeleteRun(context.Background(), "abc-123")
	if err != nil {
		t.Fatalf("DeleteRun() error = %v", err)
	}

	if !deleteCalled {
		t.Error("expected delete endpoint to be called")
	}
}

func TestDeleteRun_EmptyID(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	err := client.DeleteRun(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty run ID")
	}
}

// --- SearchRuns tests ---

func TestSearchRuns_Success(t *testing.T) {
	var receivedExperimentIDs []string

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req struct {
			ExperimentIDs []string `json:"experiment_ids"`
		}
		mustDecodeJSON(t, r, &req)
		receivedExperimentIDs = req.ExperimentIDs

		mustEncodeJSON(t, w, map[string]any{
			"runs": []map[string]any{
				{
					"info": map[string]any{
						"run_id":        "run-1",
						"experiment_id": "1",
						"status":        "FINISHED",
					},
					"data": map[string]any{},
				},
			},
			"next_page_token": "page2",
		})
	}))

	result, err := client.SearchRuns(context.Background(), []string{"1", "2"})
	if err != nil {
		t.Fatalf("SearchRuns() error = %v", err)
	}

	if len(receivedExperimentIDs) != 2 {
		t.Errorf("experiment_ids count = %d, want 2", len(receivedExperimentIDs))
	}
	if len(result.Runs) != 1 {
		t.Errorf("got %d runs, want 1", len(result.Runs))
	}
	if result.NextPageToken != "page2" {
		t.Errorf("NextPageToken = %q, want %q", result.NextPageToken, "page2")
	}
}

func TestSearchRuns_EmptyExperimentIDs(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	_, err := client.SearchRuns(context.Background(), []string{})
	if err == nil {
		t.Error("expected error for empty experiment IDs")
	}
}

func TestSearchRuns_InvalidViewType(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	_, err := client.SearchRuns(context.Background(), []string{"1"},
		WithRunsViewType("INVALID"),
	)
	if err == nil {
		t.Error("expected error for invalid view type")
	}
}

// --- LogMetric tests ---

func TestLogMetric_Success(t *testing.T) {
	var receivedRunID, receivedKey string
	var receivedValue float64

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path != "/api/2.0/mlflow/runs/log-metric" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		var req struct {
			RunID string  `json:"run_id"`
			Key   string  `json:"key"`
			Value float64 `json:"value"`
		}
		mustDecodeJSON(t, r, &req)
		receivedRunID = req.RunID
		receivedKey = req.Key
		receivedValue = req.Value

		mustEncodeJSON(t, w, map[string]any{})
	}))

	err := client.LogMetric(context.Background(), "abc-123", "rmse", 0.42)
	if err != nil {
		t.Fatalf("LogMetric() error = %v", err)
	}

	if receivedRunID != "abc-123" {
		t.Errorf("run_id = %q, want %q", receivedRunID, "abc-123")
	}
	if receivedKey != "rmse" {
		t.Errorf("key = %q, want %q", receivedKey, "rmse")
	}
	if receivedValue != 0.42 {
		t.Errorf("value = %f, want %f", receivedValue, 0.42)
	}
}

func TestLogMetric_EmptyRunID(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	err := client.LogMetric(context.Background(), "", "rmse", 0.42)
	if err == nil {
		t.Error("expected error for empty run ID")
	}
}

func TestLogMetric_EmptyKey(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	err := client.LogMetric(context.Background(), "abc-123", "", 0.42)
	if err == nil {
		t.Error("expected error for empty key")
	}
}

// --- LogParam tests ---

func TestLogParam_Success(t *testing.T) {
	var receivedRunID, receivedKey, receivedValue string

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path != "/api/2.0/mlflow/runs/log-parameter" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		var req struct {
			RunID string `json:"run_id"`
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		mustDecodeJSON(t, r, &req)
		receivedRunID = req.RunID
		receivedKey = req.Key
		receivedValue = req.Value

		mustEncodeJSON(t, w, map[string]any{})
	}))

	err := client.LogParam(context.Background(), "abc-123", "lr", "0.01")
	if err != nil {
		t.Fatalf("LogParam() error = %v", err)
	}

	if receivedRunID != "abc-123" {
		t.Errorf("run_id = %q, want %q", receivedRunID, "abc-123")
	}
	if receivedKey != "lr" {
		t.Errorf("key = %q, want %q", receivedKey, "lr")
	}
	if receivedValue != "0.01" {
		t.Errorf("value = %q, want %q", receivedValue, "0.01")
	}
}

func TestLogParam_EmptyRunID(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	err := client.LogParam(context.Background(), "", "lr", "0.01")
	if err == nil {
		t.Error("expected error for empty run ID")
	}
}

func TestLogParam_EmptyKey(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	err := client.LogParam(context.Background(), "abc-123", "", "0.01")
	if err == nil {
		t.Error("expected error for empty key")
	}
}

// --- SetTag tests ---

func TestSetTag_Success(t *testing.T) {
	var receivedRunID, receivedKey, receivedValue string

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path != "/api/2.0/mlflow/runs/set-tag" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		var req struct {
			RunID string `json:"run_id"`
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		mustDecodeJSON(t, r, &req)
		receivedRunID = req.RunID
		receivedKey = req.Key
		receivedValue = req.Value

		mustEncodeJSON(t, w, map[string]any{})
	}))

	err := client.SetTag(context.Background(), "abc-123", "model", "sklearn")
	if err != nil {
		t.Fatalf("SetTag() error = %v", err)
	}

	if receivedRunID != "abc-123" {
		t.Errorf("run_id = %q, want %q", receivedRunID, "abc-123")
	}
	if receivedKey != "model" {
		t.Errorf("key = %q, want %q", receivedKey, "model")
	}
	if receivedValue != "sklearn" {
		t.Errorf("value = %q, want %q", receivedValue, "sklearn")
	}
}

func TestSetTag_EmptyRunID(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	err := client.SetTag(context.Background(), "", "key", "value")
	if err == nil {
		t.Error("expected error for empty run ID")
	}
}

func TestSetTag_EmptyKey(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	err := client.SetTag(context.Background(), "abc-123", "", "value")
	if err == nil {
		t.Error("expected error for empty key")
	}
}

// --- DeleteTag tests ---

func TestDeleteTag_Success(t *testing.T) {
	var receivedRunID, receivedKey string

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path != "/api/2.0/mlflow/runs/delete-tag" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		var req struct {
			RunID string `json:"run_id"`
			Key   string `json:"key"`
		}
		mustDecodeJSON(t, r, &req)
		receivedRunID = req.RunID
		receivedKey = req.Key

		mustEncodeJSON(t, w, map[string]any{})
	}))

	err := client.DeleteTag(context.Background(), "abc-123", "model")
	if err != nil {
		t.Fatalf("DeleteTag() error = %v", err)
	}

	if receivedRunID != "abc-123" {
		t.Errorf("run_id = %q, want %q", receivedRunID, "abc-123")
	}
	if receivedKey != "model" {
		t.Errorf("key = %q, want %q", receivedKey, "model")
	}
}

func TestDeleteTag_EmptyRunID(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	err := client.DeleteTag(context.Background(), "", "key")
	if err == nil {
		t.Error("expected error for empty run ID")
	}
}

func TestDeleteTag_EmptyKey(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	err := client.DeleteTag(context.Background(), "abc-123", "")
	if err == nil {
		t.Error("expected error for empty key")
	}
}

// --- LogBatch tests ---

func TestLogBatch_Success(t *testing.T) {
	var receivedRunID string
	var metricsCount, paramsCount, tagsCount int

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path != "/api/2.0/mlflow/runs/log-batch" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		var req struct {
			RunID   string           `json:"run_id"`
			Metrics []map[string]any `json:"metrics"`
			Params  []map[string]any `json:"params"`
			Tags    []map[string]any `json:"tags"`
		}
		mustDecodeJSON(t, r, &req)
		receivedRunID = req.RunID
		metricsCount = len(req.Metrics)
		paramsCount = len(req.Params)
		tagsCount = len(req.Tags)

		mustEncodeJSON(t, w, map[string]any{})
	}))

	err := client.LogBatch(context.Background(), "abc-123",
		[]Metric{
			{Key: "rmse", Value: 0.5, Step: 1},
			{Key: "mae", Value: 0.3, Step: 1},
		},
		[]Param{
			{Key: "lr", Value: "0.01"},
		},
		map[string]string{
			"model": "sklearn",
		},
	)
	if err != nil {
		t.Fatalf("LogBatch() error = %v", err)
	}

	if receivedRunID != "abc-123" {
		t.Errorf("run_id = %q, want %q", receivedRunID, "abc-123")
	}
	if metricsCount != 2 {
		t.Errorf("metrics count = %d, want 2", metricsCount)
	}
	if paramsCount != 1 {
		t.Errorf("params count = %d, want 1", paramsCount)
	}
	if tagsCount != 1 {
		t.Errorf("tags count = %d, want 1", tagsCount)
	}
}

func TestLogBatch_EmptyRunID(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	err := client.LogBatch(context.Background(), "", nil, nil, nil)
	if err == nil {
		t.Error("expected error for empty run ID")
	}
}

func TestLogBatch_EmptyBatch(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mustEncodeJSON(t, w, map[string]any{})
	}))

	// Empty batch should succeed (server handles it)
	err := client.LogBatch(context.Background(), "abc-123", nil, nil, nil)
	if err != nil {
		t.Fatalf("LogBatch() with empty batch error = %v", err)
	}
}
