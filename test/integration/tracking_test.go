//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/opendatahub-io/mlflow-go/mlflow"
	"github.com/opendatahub-io/mlflow-go/mlflow/tracking"
)

// TestExperimentLifecycle tests the full experiment lifecycle:
// create, get, get-by-name, update, set tag, search, delete.
func TestExperimentLifecycle(t *testing.T) {
	client, err := mlflow.NewClient(mlflow.WithInsecure())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	expName := fmt.Sprintf("e2e-tracking-exp-%d", time.Now().UnixNano())

	// Step 1: Create experiment
	t.Log("Step 1: Creating experiment")
	expID, err := client.Tracking().CreateExperiment(ctx, expName)
	if err != nil {
		t.Fatalf("CreateExperiment() error = %v", err)
	}
	t.Cleanup(func() { _ = client.Tracking().DeleteExperiment(ctx, expID) })
	if expID == "" {
		t.Fatal("Expected non-empty experiment ID")
	}
	t.Logf("Created experiment %s with ID %s", expName, expID)

	// Step 2: Get experiment by ID
	t.Log("Step 2: Getting experiment by ID")
	exp, err := client.Tracking().GetExperiment(ctx, expID)
	if err != nil {
		t.Fatalf("GetExperiment() error = %v", err)
	}
	if exp.ID != expID {
		t.Errorf("ID = %q, want %q", exp.ID, expID)
	}
	if exp.Name != expName {
		t.Errorf("Name = %q, want %q", exp.Name, expName)
	}
	if exp.LifecycleStage != "active" {
		t.Errorf("LifecycleStage = %q, want %q", exp.LifecycleStage, "active")
	}

	// Step 3: Get experiment by name
	t.Log("Step 3: Getting experiment by name")
	expByName, err := client.Tracking().GetExperimentByName(ctx, expName)
	if err != nil {
		t.Fatalf("GetExperimentByName() error = %v", err)
	}
	if expByName.ID != expID {
		t.Errorf("GetByName ID = %q, want %q", expByName.ID, expID)
	}

	// Step 4: Update experiment name
	t.Log("Step 4: Updating experiment name")
	updatedName := expName + "-updated"
	err = client.Tracking().UpdateExperiment(ctx, expID, updatedName)
	if err != nil {
		t.Fatalf("UpdateExperiment() error = %v", err)
	}

	expAfterUpdate, err := client.Tracking().GetExperiment(ctx, expID)
	if err != nil {
		t.Fatalf("GetExperiment() after update error = %v", err)
	}
	if expAfterUpdate.Name != updatedName {
		t.Errorf("Name after update = %q, want %q", expAfterUpdate.Name, updatedName)
	}

	// Step 5: Set experiment tag
	t.Log("Step 5: Setting experiment tag")
	err = client.Tracking().SetExperimentTag(ctx, expID, "team", "ml-platform")
	if err != nil {
		t.Fatalf("SetExperimentTag() error = %v", err)
	}

	expAfterTag, err := client.Tracking().GetExperiment(ctx, expID)
	if err != nil {
		t.Fatalf("GetExperiment() after tag error = %v", err)
	}
	if expAfterTag.Tags["team"] != "ml-platform" {
		t.Errorf("Tags[team] = %q, want %q", expAfterTag.Tags["team"], "ml-platform")
	}

	// Step 6: Search experiments
	t.Log("Step 6: Searching experiments")
	searchResult, err := client.Tracking().SearchExperiments(ctx,
		tracking.WithExperimentsFilter(fmt.Sprintf("name = '%s'", updatedName)),
	)
	if err != nil {
		t.Fatalf("SearchExperiments() error = %v", err)
	}
	if len(searchResult.Experiments) == 0 {
		t.Error("Expected at least one experiment in search results")
	}

	// Step 7: Delete experiment
	t.Log("Step 7: Deleting experiment")
	err = client.Tracking().DeleteExperiment(ctx, expID)
	if err != nil {
		t.Fatalf("DeleteExperiment() error = %v", err)
	}

	t.Log("Experiment lifecycle test passed")
}

// TestRunLifecycle tests the full run lifecycle:
// create experiment, create run, log metrics/params/tags, update run, get run.
func TestRunLifecycle(t *testing.T) {
	client, err := mlflow.NewClient(mlflow.WithInsecure())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Setup: create experiment
	expName := fmt.Sprintf("e2e-tracking-run-%d", time.Now().UnixNano())
	expID, err := client.Tracking().CreateExperiment(ctx, expName)
	if err != nil {
		t.Fatalf("CreateExperiment() error = %v", err)
	}
	t.Cleanup(func() { _ = client.Tracking().DeleteExperiment(ctx, expID) })

	// Step 1: Create run
	t.Log("Step 1: Creating run")
	run, err := client.Tracking().CreateRun(ctx, expID,
		tracking.WithRunName("test-run"),
		tracking.WithRunTags(map[string]string{"model": "sklearn"}),
	)
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	if run.Info.RunID == "" {
		t.Fatal("Expected non-empty run ID")
	}
	if run.Info.ExperimentID != expID {
		t.Errorf("ExperimentID = %q, want %q", run.Info.ExperimentID, expID)
	}
	if run.Info.Status != tracking.RunStatusRunning {
		t.Errorf("Status = %q, want %q", run.Info.Status, tracking.RunStatusRunning)
	}
	runID := run.Info.RunID
	t.Cleanup(func() { _ = client.Tracking().DeleteRun(ctx, runID) })
	t.Logf("Created run %s", runID)

	// Step 2: Log metrics
	t.Log("Step 2: Logging metrics")
	err = client.Tracking().LogMetric(ctx, runID, "rmse", 0.85, tracking.WithStep(1))
	if err != nil {
		t.Fatalf("LogMetric() step 1 error = %v", err)
	}
	err = client.Tracking().LogMetric(ctx, runID, "rmse", 0.72, tracking.WithStep(2))
	if err != nil {
		t.Fatalf("LogMetric() step 2 error = %v", err)
	}

	// Step 3: Log params
	t.Log("Step 3: Logging params")
	err = client.Tracking().LogParam(ctx, runID, "learning_rate", "0.01")
	if err != nil {
		t.Fatalf("LogParam() error = %v", err)
	}
	err = client.Tracking().LogParam(ctx, runID, "epochs", "100")
	if err != nil {
		t.Fatalf("LogParam() error = %v", err)
	}

	// Step 4: Set tag
	t.Log("Step 4: Setting tag")
	err = client.Tracking().SetTag(ctx, runID, "status_note", "looking good")
	if err != nil {
		t.Fatalf("SetTag() error = %v", err)
	}

	// Step 5: Get run and verify data
	t.Log("Step 5: Getting run and verifying data")
	loaded, err := client.Tracking().GetRun(ctx, runID)
	if err != nil {
		t.Fatalf("GetRun() error = %v", err)
	}

	if loaded.Info.RunID != runID {
		t.Errorf("RunID = %q, want %q", loaded.Info.RunID, runID)
	}

	// Verify params
	paramMap := make(map[string]string)
	for _, p := range loaded.Data.Params {
		paramMap[p.Key] = p.Value
	}
	if paramMap["learning_rate"] != "0.01" {
		t.Errorf("param learning_rate = %q, want %q", paramMap["learning_rate"], "0.01")
	}
	if paramMap["epochs"] != "100" {
		t.Errorf("param epochs = %q, want %q", paramMap["epochs"], "100")
	}

	// Verify metrics
	metricMap := make(map[string]float64)
	for _, m := range loaded.Data.Metrics {
		metricMap[m.Key] = m.Value
	}
	if _, ok := metricMap["rmse"]; !ok {
		t.Error("expected metric 'rmse' to be present")
	}

	// Verify tags
	if loaded.Data.Tags["status_note"] != "looking good" {
		t.Errorf("tag status_note = %q, want %q", loaded.Data.Tags["status_note"], "looking good")
	}

	// Step 6: Update run status to FINISHED
	t.Log("Step 6: Updating run status")
	endTime := time.Now()
	info, err := client.Tracking().UpdateRun(ctx, runID,
		tracking.WithStatus(tracking.RunStatusFinished),
		tracking.WithEndTime(endTime),
	)
	if err != nil {
		t.Fatalf("UpdateRun() error = %v", err)
	}
	if info.Status != tracking.RunStatusFinished {
		t.Errorf("Status after update = %q, want %q", info.Status, tracking.RunStatusFinished)
	}

	// Verify final state
	final, err := client.Tracking().GetRun(ctx, runID)
	if err != nil {
		t.Fatalf("GetRun() final error = %v", err)
	}
	if final.Info.Status != tracking.RunStatusFinished {
		t.Errorf("Final status = %q, want %q", final.Info.Status, tracking.RunStatusFinished)
	}

	t.Log("Run lifecycle test passed")
}

// TestLogBatch tests batch logging of metrics, params, and tags.
func TestLogBatch(t *testing.T) {
	client, err := mlflow.NewClient(mlflow.WithInsecure())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Setup
	expName := fmt.Sprintf("e2e-tracking-batch-%d", time.Now().UnixNano())
	expID, err := client.Tracking().CreateExperiment(ctx, expName)
	if err != nil {
		t.Fatalf("CreateExperiment() error = %v", err)
	}
	t.Cleanup(func() { _ = client.Tracking().DeleteExperiment(ctx, expID) })

	run, err := client.Tracking().CreateRun(ctx, expID)
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}
	runID := run.Info.RunID
	t.Cleanup(func() { _ = client.Tracking().DeleteRun(ctx, runID) })

	// Log batch
	t.Log("Logging batch of metrics, params, and tags")
	err = client.Tracking().LogBatch(ctx, runID,
		[]tracking.Metric{
			{Key: "loss", Value: 0.5, Step: 1},
			{Key: "loss", Value: 0.3, Step: 2},
			{Key: "accuracy", Value: 0.92, Step: 1},
		},
		[]tracking.Param{
			{Key: "optimizer", Value: "adam"},
			{Key: "batch_size", Value: "32"},
		},
		map[string]string{
			"framework": "pytorch",
			"version":   "2.0",
		},
	)
	if err != nil {
		t.Fatalf("LogBatch() error = %v", err)
	}

	// Verify
	loaded, err := client.Tracking().GetRun(ctx, runID)
	if err != nil {
		t.Fatalf("GetRun() error = %v", err)
	}

	// Verify metrics
	metricMap := make(map[string]float64)
	for _, m := range loaded.Data.Metrics {
		metricMap[m.Key] = m.Value
	}
	if _, ok := metricMap["loss"]; !ok {
		t.Error("expected metric 'loss' to be present")
	}
	if v, ok := metricMap["accuracy"]; !ok || v != 0.92 {
		t.Errorf("metric accuracy = %v, want 0.92", v)
	}

	// Verify params
	paramMap := make(map[string]string)
	for _, p := range loaded.Data.Params {
		paramMap[p.Key] = p.Value
	}
	if paramMap["optimizer"] != "adam" {
		t.Errorf("param optimizer = %q, want %q", paramMap["optimizer"], "adam")
	}
	if paramMap["batch_size"] != "32" {
		t.Errorf("param batch_size = %q, want %q", paramMap["batch_size"], "32")
	}

	// Verify tags
	if loaded.Data.Tags["framework"] != "pytorch" {
		t.Errorf("tag framework = %q, want %q", loaded.Data.Tags["framework"], "pytorch")
	}
	if loaded.Data.Tags["version"] != "2.0" {
		t.Errorf("tag version = %q, want %q", loaded.Data.Tags["version"], "2.0")
	}

	t.Log("LogBatch test passed")
}

// TestSearchRuns tests searching for runs across experiments.
func TestSearchRuns(t *testing.T) {
	client, err := mlflow.NewClient(mlflow.WithInsecure())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Setup: create experiment with runs
	expName := fmt.Sprintf("e2e-tracking-search-%d", time.Now().UnixNano())
	expID, err := client.Tracking().CreateExperiment(ctx, expName)
	if err != nil {
		t.Fatalf("CreateExperiment() error = %v", err)
	}
	t.Cleanup(func() { _ = client.Tracking().DeleteExperiment(ctx, expID) })

	// Create runs with different params
	run1, err := client.Tracking().CreateRun(ctx, expID,
		tracking.WithRunName("run-sklearn"),
	)
	if err != nil {
		t.Fatalf("CreateRun() 1 error = %v", err)
	}
	t.Cleanup(func() { _ = client.Tracking().DeleteRun(ctx, run1.Info.RunID) })
	err = client.Tracking().LogParam(ctx, run1.Info.RunID, "model", "sklearn")
	if err != nil {
		t.Fatalf("LogParam() error = %v", err)
	}

	run2, err := client.Tracking().CreateRun(ctx, expID,
		tracking.WithRunName("run-pytorch"),
	)
	if err != nil {
		t.Fatalf("CreateRun() 2 error = %v", err)
	}
	t.Cleanup(func() { _ = client.Tracking().DeleteRun(ctx, run2.Info.RunID) })
	err = client.Tracking().LogParam(ctx, run2.Info.RunID, "model", "pytorch")
	if err != nil {
		t.Fatalf("LogParam() error = %v", err)
	}

	// Search all runs in experiment
	t.Log("Searching runs")
	result, err := client.Tracking().SearchRuns(ctx, []string{expID})
	if err != nil {
		t.Fatalf("SearchRuns() error = %v", err)
	}

	if len(result.Runs) < 2 {
		t.Errorf("Expected at least 2 runs, got %d", len(result.Runs))
	}

	// Search with filter
	t.Log("Searching runs with filter")
	filtered, err := client.Tracking().SearchRuns(ctx, []string{expID},
		tracking.WithRunsFilter("params.model = 'sklearn'"),
	)
	if err != nil {
		t.Fatalf("SearchRuns() with filter error = %v", err)
	}

	if len(filtered.Runs) != 1 {
		t.Errorf("Expected 1 filtered run, got %d", len(filtered.Runs))
	}

	t.Log("SearchRuns test passed")
}

// TestTrackingNotFound tests that mlflow.IsNotFound works for tracking resources.
func TestTrackingNotFound(t *testing.T) {
	client, err := mlflow.NewClient(mlflow.WithInsecure())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Non-existent experiment
	_, err = client.Tracking().GetExperiment(ctx, "999999999")
	if err == nil {
		t.Fatal("Expected error for non-existent experiment")
	}
	if !mlflow.IsNotFound(err) {
		t.Errorf("Expected IsNotFound for experiment, got: %v", err)
	}

	// Non-existent run
	_, err = client.Tracking().GetRun(ctx, "nonexistent-run-id-xyz")
	if err == nil {
		t.Fatal("Expected error for non-existent run")
	}
	if !mlflow.IsNotFound(err) {
		t.Errorf("Expected IsNotFound for run, got: %v", err)
	}
}

// TestDeleteTag tests deleting a tag from a run.
func TestDeleteRunTag(t *testing.T) {
	client, err := mlflow.NewClient(mlflow.WithInsecure())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Setup
	expName := fmt.Sprintf("e2e-tracking-deltag-%d", time.Now().UnixNano())
	expID, err := client.Tracking().CreateExperiment(ctx, expName)
	if err != nil {
		t.Fatalf("CreateExperiment() error = %v", err)
	}
	t.Cleanup(func() { _ = client.Tracking().DeleteExperiment(ctx, expID) })

	run, err := client.Tracking().CreateRun(ctx, expID)
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}
	runID := run.Info.RunID
	t.Cleanup(func() { _ = client.Tracking().DeleteRun(ctx, runID) })

	// Set then delete a tag
	err = client.Tracking().SetTag(ctx, runID, "temp_tag", "to_delete")
	if err != nil {
		t.Fatalf("SetTag() error = %v", err)
	}

	err = client.Tracking().DeleteTag(ctx, runID, "temp_tag")
	if err != nil {
		t.Fatalf("DeleteTag() error = %v", err)
	}

	// Verify tag is gone
	loaded, err := client.Tracking().GetRun(ctx, runID)
	if err != nil {
		t.Fatalf("GetRun() error = %v", err)
	}

	if _, ok := loaded.Data.Tags["temp_tag"]; ok {
		t.Error("Tag 'temp_tag' should have been deleted")
	}

	t.Log("DeleteTag test passed")
}

// TestSearchExperimentsPagination tests paginating through experiment results.
func TestSearchExperimentsPagination(t *testing.T) {
	client, err := mlflow.NewClient(mlflow.WithInsecure())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create 3 experiments with a unique prefix for filtering
	prefix := fmt.Sprintf("e2e-page-%d", time.Now().UnixNano())
	for i := 1; i <= 3; i++ {
		name := fmt.Sprintf("%s-%d", prefix, i)
		id, err := client.Tracking().CreateExperiment(ctx, name)
		if err != nil {
			t.Fatalf("CreateExperiment(%d) error = %v", i, err)
		}
		t.Cleanup(func() { _ = client.Tracking().DeleteExperiment(ctx, id) })
	}

	// Paginate with max_results=1, collecting all results
	filter := fmt.Sprintf("name LIKE '%s%%'", prefix)
	var allNames []string
	pageToken := ""

	for {
		opts := []tracking.SearchExperimentsOption{
			tracking.WithExperimentsFilter(filter),
			tracking.WithExperimentsMaxResults(1),
		}
		if pageToken != "" {
			opts = append(opts, tracking.WithExperimentsPageToken(pageToken))
		}

		result, err := client.Tracking().SearchExperiments(ctx, opts...)
		if err != nil {
			t.Fatalf("SearchExperiments() error = %v", err)
		}

		for _, e := range result.Experiments {
			allNames = append(allNames, e.Name)
		}

		if result.NextPageToken == "" {
			break
		}
		pageToken = result.NextPageToken
	}

	if len(allNames) != 3 {
		t.Errorf("Expected 3 experiments across pages, got %d: %v", len(allNames), allNames)
	}
	t.Logf("Paginated through %d experiments: %v", len(allNames), allNames)

	t.Log("SearchExperimentsPagination test passed")
}

// TestSearchRunsPagination tests paginating through run results.
func TestSearchRunsPagination(t *testing.T) {
	client, err := mlflow.NewClient(mlflow.WithInsecure())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create experiment with 3 runs
	expName := fmt.Sprintf("e2e-run-page-%d", time.Now().UnixNano())
	expID, err := client.Tracking().CreateExperiment(ctx, expName)
	if err != nil {
		t.Fatalf("CreateExperiment() error = %v", err)
	}
	t.Cleanup(func() { _ = client.Tracking().DeleteExperiment(ctx, expID) })

	for i := 1; i <= 3; i++ {
		run, err := client.Tracking().CreateRun(ctx, expID,
			tracking.WithRunName(fmt.Sprintf("run-%d", i)),
		)
		if err != nil {
			t.Fatalf("CreateRun(%d) error = %v", i, err)
		}
		t.Cleanup(func() { _ = client.Tracking().DeleteRun(ctx, run.Info.RunID) })
	}

	// Paginate with max_results=1
	var allRunIDs []string
	pageToken := ""

	for {
		opts := []tracking.SearchRunsOption{
			tracking.WithRunsMaxResults(1),
		}
		if pageToken != "" {
			opts = append(opts, tracking.WithRunsPageToken(pageToken))
		}

		result, err := client.Tracking().SearchRuns(ctx, []string{expID}, opts...)
		if err != nil {
			t.Fatalf("SearchRuns() error = %v", err)
		}

		for _, r := range result.Runs {
			allRunIDs = append(allRunIDs, r.Info.RunID)
		}

		if result.NextPageToken == "" {
			break
		}
		pageToken = result.NextPageToken
	}

	if len(allRunIDs) != 3 {
		t.Errorf("Expected 3 runs across pages, got %d", len(allRunIDs))
	}
	t.Logf("Paginated through %d runs", len(allRunIDs))

	t.Log("SearchRunsPagination test passed")
}
