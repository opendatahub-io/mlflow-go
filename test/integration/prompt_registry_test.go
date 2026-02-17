//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/opendatahub-io/mlflow-go/mlflow"
	"github.com/opendatahub-io/mlflow-go/mlflow/promptregistry"
)

// TestPromptLifecycle tests the full prompt lifecycle:
// 1. Register a new prompt (creates v1)
// 2. Load the prompt by name (gets latest)
// 3. Modify locally and register new version (creates v2)
// 4. Load specific version (v1)
// 5. Verify versions are correct
func TestPromptLifecycle(t *testing.T) {
	client, err := mlflow.NewClient(mlflow.WithInsecure())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use unique name to avoid conflicts between test runs
	promptName := fmt.Sprintf("e2e-test-prompt-%d", time.Now().UnixNano())

	// Step 1: Register a new prompt
	t.Log("Step 1: Registering new prompt")
	v1, err := client.PromptRegistry().RegisterPrompt(ctx, promptName,
		"Hello {{name}}!",
		promptregistry.WithCommitMessage("Initial version"),
		promptregistry.WithTags(map[string]string{"test": "e2e"}),
	)
	if err != nil {
		t.Fatalf("RegisterPrompt() error = %v", err)
	}

	if v1.Version != 1 {
		t.Errorf("Expected version 1, got %d", v1.Version)
	}
	if v1.Template != "Hello {{name}}!" {
		t.Errorf("Template = %q, want %q", v1.Template, "Hello {{name}}!")
	}
	if v1.Name != promptName {
		t.Errorf("Name = %q, want %q", v1.Name, promptName)
	}
	t.Logf("Created %s v%d", v1.Name, v1.Version)

	// Step 2: Load the prompt by name (should get latest = v1)
	t.Log("Step 2: Loading prompt by name")
	loaded, err := client.PromptRegistry().LoadPrompt(ctx, promptName)
	if err != nil {
		t.Fatalf("LoadPrompt() error = %v", err)
	}

	if loaded.Version != 1 {
		t.Errorf("Loaded version = %d, want 1", loaded.Version)
	}
	if loaded.Template != v1.Template {
		t.Errorf("Loaded template differs from registered")
	}

	// Step 3: Modify locally and register new version
	t.Log("Step 3: Modifying and registering new version")
	modified := loaded.
		WithTemplate("Hello {{name}}, welcome to {{company}}!").
		WithCommitMessage("Added company variable")

	// Verify original is unchanged
	if loaded.Template != "Hello {{name}}!" {
		t.Error("Original prompt was modified")
	}

	v2, err := client.PromptRegistry().RegisterPrompt(ctx, promptName,
		modified.Template,
		promptregistry.WithCommitMessage(modified.CommitMessage),
	)
	if err != nil {
		t.Fatalf("RegisterPrompt() v2 error = %v", err)
	}

	if v2.Version != 2 {
		t.Errorf("Expected version 2, got %d", v2.Version)
	}
	if v2.Template != "Hello {{name}}, welcome to {{company}}!" {
		t.Errorf("v2 Template = %q", v2.Template)
	}
	t.Logf("Created %s v%d", v2.Name, v2.Version)

	// Step 4: Load specific version (v1)
	t.Log("Step 4: Loading specific version (v1)")
	v1Loaded, err := client.PromptRegistry().LoadPrompt(ctx, promptName, promptregistry.WithVersion(1))
	if err != nil {
		t.Fatalf("LoadPrompt(v1) error = %v", err)
	}

	if v1Loaded.Version != 1 {
		t.Errorf("v1Loaded.Version = %d, want 1", v1Loaded.Version)
	}
	if v1Loaded.Template != "Hello {{name}}!" {
		t.Errorf("v1 template was changed: %q", v1Loaded.Template)
	}

	// Step 5: Load latest (should be v2 now)
	t.Log("Step 5: Verifying latest is v2")
	latest, err := client.PromptRegistry().LoadPrompt(ctx, promptName)
	if err != nil {
		t.Fatalf("LoadPrompt() latest error = %v", err)
	}

	if latest.Version != 2 {
		t.Errorf("Latest version = %d, want 2", latest.Version)
	}

	t.Log("E2E test passed: full prompt lifecycle verified")
}

// TestNotFoundError tests that IsNotFound works correctly.
func TestNotFoundError(t *testing.T) {
	client, err := mlflow.NewClient(mlflow.WithInsecure())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()

	_, err = client.PromptRegistry().LoadPrompt(ctx, "nonexistent-prompt-xyz-123456")
	if err == nil {
		t.Fatal("Expected error for non-existent prompt")
	}

	if !mlflow.IsNotFound(err) {
		t.Errorf("Expected IsNotFound, got: %v", err)
	}
}

// TestLoadWithTags tests that tags are preserved correctly.
func TestLoadWithTags(t *testing.T) {
	client, err := mlflow.NewClient(mlflow.WithInsecure())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	promptName := fmt.Sprintf("e2e-tags-test-%d", time.Now().UnixNano())

	// Register with tags
	_, err = client.PromptRegistry().RegisterPrompt(ctx, promptName,
		"Template with tags",
		promptregistry.WithTags(map[string]string{
			"team":     "ml-platform",
			"category": "greeting",
			"env":      "test",
		}),
	)
	if err != nil {
		t.Fatalf("RegisterPrompt() error = %v", err)
	}

	// Load and verify tags
	loaded, err := client.PromptRegistry().LoadPrompt(ctx, promptName)
	if err != nil {
		t.Fatalf("LoadPrompt() error = %v", err)
	}

	if loaded.Tags["team"] != "ml-platform" {
		t.Errorf("Tags[team] = %q, want %q", loaded.Tags["team"], "ml-platform")
	}
	if loaded.Tags["category"] != "greeting" {
		t.Errorf("Tags[category] = %q, want %q", loaded.Tags["category"], "greeting")
	}
	if loaded.Tags["env"] != "test" {
		t.Errorf("Tags[env] = %q, want %q", loaded.Tags["env"], "test")
	}

	// Internal tags should not be exposed
	if _, ok := loaded.Tags["mlflow.prompt.is_prompt"]; ok {
		t.Error("Internal tag should not be in user tags")
	}
	if _, ok := loaded.Tags["mlflow.prompt.text"]; ok {
		t.Error("Template tag should not be in user tags")
	}
}

// TestListPrompts tests listing all prompts.
func TestListPrompts(t *testing.T) {
	client, err := mlflow.NewClient(mlflow.WithInsecure())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()

	// Create a unique prompt to ensure we have at least one
	promptName := fmt.Sprintf("e2e-list-test-%d", time.Now().UnixNano())
	_, err = client.PromptRegistry().RegisterPrompt(ctx, promptName, "List test template")
	if err != nil {
		t.Fatalf("RegisterPrompt() error = %v", err)
	}

	// List all prompts
	list, err := client.PromptRegistry().ListPrompts(ctx)
	if err != nil {
		t.Fatalf("ListPrompts() error = %v", err)
	}

	if len(list.Prompts) == 0 {
		t.Error("Expected at least one prompt")
	}

	// Find our prompt in the list
	found := false
	for _, p := range list.Prompts {
		if p.Name == promptName {
			found = true
			if p.LatestVersion < 1 {
				t.Errorf("LatestVersion = %d, want >= 1", p.LatestVersion)
			}
			break
		}
	}

	if !found {
		t.Errorf("Created prompt %s not found in list", promptName)
	}

	t.Logf("Listed %d prompts", len(list.Prompts))
}

// TestListPromptsWithFilter tests listing prompts with name filter.
func TestListPromptsWithFilter(t *testing.T) {
	client, err := mlflow.NewClient(mlflow.WithInsecure())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()

	// Create prompts with a unique prefix
	prefix := fmt.Sprintf("e2e-filter-%d", time.Now().UnixNano())
	_, err = client.PromptRegistry().RegisterPrompt(ctx, prefix+"-alpha", "Alpha template")
	if err != nil {
		t.Fatalf("RegisterPrompt() error = %v", err)
	}
	_, err = client.PromptRegistry().RegisterPrompt(ctx, prefix+"-beta", "Beta template")
	if err != nil {
		t.Fatalf("RegisterPrompt() error = %v", err)
	}

	// List with name filter
	list, err := client.PromptRegistry().ListPrompts(ctx, promptregistry.WithNameFilter(prefix+"%"))
	if err != nil {
		t.Fatalf("ListPrompts() with filter error = %v", err)
	}

	if len(list.Prompts) < 2 {
		t.Errorf("Expected at least 2 prompts matching filter, got %d", len(list.Prompts))
	}

	// Verify all results match the filter
	for _, p := range list.Prompts {
		if len(p.Name) < len(prefix) || p.Name[:len(prefix)] != prefix {
			t.Errorf("Prompt %s doesn't match filter %s%%", p.Name, prefix)
		}
	}

	t.Logf("Filtered list returned %d prompts", len(list.Prompts))
}

// TestListPromptVersions tests listing versions of a prompt.
func TestListPromptVersions(t *testing.T) {
	client, err := mlflow.NewClient(mlflow.WithInsecure())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()

	// Create a prompt with multiple versions
	promptName := fmt.Sprintf("e2e-versions-test-%d", time.Now().UnixNano())

	_, err = client.PromptRegistry().RegisterPrompt(ctx, promptName, "Version 1 template",
		promptregistry.WithCommitMessage("First version"))
	if err != nil {
		t.Fatalf("RegisterPrompt() v1 error = %v", err)
	}

	_, err = client.PromptRegistry().RegisterPrompt(ctx, promptName, "Version 2 template",
		promptregistry.WithCommitMessage("Second version"))
	if err != nil {
		t.Fatalf("RegisterPrompt() v2 error = %v", err)
	}

	_, err = client.PromptRegistry().RegisterPrompt(ctx, promptName, "Version 3 template",
		promptregistry.WithCommitMessage("Third version"))
	if err != nil {
		t.Fatalf("RegisterPrompt() v3 error = %v", err)
	}

	// List versions
	versions, err := client.PromptRegistry().ListPromptVersions(ctx, promptName)
	if err != nil {
		t.Fatalf("ListPromptVersions() error = %v", err)
	}

	if len(versions.Versions) != 3 {
		t.Errorf("Expected 3 versions, got %d", len(versions.Versions))
	}

	// Verify order (newest first)
	if len(versions.Versions) >= 3 {
		if versions.Versions[0].Version != 3 {
			t.Errorf("First version = %d, want 3 (newest first)", versions.Versions[0].Version)
		}
		if versions.Versions[2].Version != 1 {
			t.Errorf("Last version = %d, want 1", versions.Versions[2].Version)
		}
	}

	// Verify commit messages are present
	for _, v := range versions.Versions {
		if v.CommitMessage == "" {
			t.Errorf("Version %d has empty commit message", v.Version)
		}
	}

	// Template should be empty in listing (use LoadPrompt to get full content)
	for _, v := range versions.Versions {
		if v.Template != "" {
			t.Errorf("Version %d should have empty Template in listing, got %q", v.Version, v.Template)
		}
	}

	t.Logf("Listed %d versions of %s", len(versions.Versions), promptName)
}

// TestDeletePromptVersion tests deleting a specific version.
func TestDeletePromptVersion(t *testing.T) {
	client, err := mlflow.NewClient(mlflow.WithInsecure())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	promptName := fmt.Sprintf("e2e-delete-version-%d", time.Now().UnixNano())

	// Create prompt with 2 versions
	_, err = client.PromptRegistry().RegisterPrompt(ctx, promptName, "Version 1",
		promptregistry.WithCommitMessage("First"))
	if err != nil {
		t.Fatalf("RegisterPrompt() v1 error = %v", err)
	}

	_, err = client.PromptRegistry().RegisterPrompt(ctx, promptName, "Version 2",
		promptregistry.WithCommitMessage("Second"))
	if err != nil {
		t.Fatalf("RegisterPrompt() v2 error = %v", err)
	}

	// Delete version 1
	err = client.PromptRegistry().DeletePromptVersion(ctx, promptName, 1)
	if err != nil {
		t.Fatalf("DeletePromptVersion() error = %v", err)
	}

	// Verify v1 is gone
	_, err = client.PromptRegistry().LoadPrompt(ctx, promptName, promptregistry.WithVersion(1))
	if !mlflow.IsNotFound(err) {
		t.Errorf("Expected IsNotFound for deleted version, got: %v", err)
	}

	// Verify v2 still exists
	v2, err := client.PromptRegistry().LoadPrompt(ctx, promptName, promptregistry.WithVersion(2))
	if err != nil {
		t.Fatalf("LoadPrompt(v2) error = %v", err)
	}
	if v2.Version != 2 {
		t.Errorf("Expected version 2, got %d", v2.Version)
	}

	// Cleanup
	_ = client.PromptRegistry().DeletePromptVersion(ctx, promptName, 2)
	_ = client.PromptRegistry().DeletePrompt(ctx, promptName)

	t.Log("DeletePromptVersion test passed")
}

// TestDeletePrompt tests deleting an entire prompt.
func TestDeletePrompt(t *testing.T) {
	client, err := mlflow.NewClient(mlflow.WithInsecure())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	promptName := fmt.Sprintf("e2e-delete-prompt-%d", time.Now().UnixNano())

	// Create prompt
	_, err = client.PromptRegistry().RegisterPrompt(ctx, promptName, "Template")
	if err != nil {
		t.Fatalf("RegisterPrompt() error = %v", err)
	}

	// Delete version first (required)
	err = client.PromptRegistry().DeletePromptVersion(ctx, promptName, 1)
	if err != nil {
		t.Fatalf("DeletePromptVersion() error = %v", err)
	}

	// Delete prompt
	err = client.PromptRegistry().DeletePrompt(ctx, promptName)
	if err != nil {
		t.Fatalf("DeletePrompt() error = %v", err)
	}

	// Verify prompt is gone
	_, err = client.PromptRegistry().LoadPrompt(ctx, promptName)
	if !mlflow.IsNotFound(err) {
		t.Errorf("Expected IsNotFound for deleted prompt, got: %v", err)
	}

	t.Log("DeletePrompt test passed")
}

// TestDeletePromptVersionTag tests deleting a tag from a prompt version.
func TestDeletePromptVersionTag(t *testing.T) {
	client, err := mlflow.NewClient(mlflow.WithInsecure())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	promptName := fmt.Sprintf("e2e-delete-vtag-%d", time.Now().UnixNano())

	// Create prompt with tags
	_, err = client.PromptRegistry().RegisterPrompt(ctx, promptName, "Template",
		promptregistry.WithTags(map[string]string{
			"keep":   "this",
			"delete": "this",
		}),
	)
	if err != nil {
		t.Fatalf("RegisterPrompt() error = %v", err)
	}

	// Delete tag from version 1
	err = client.PromptRegistry().DeletePromptVersionTag(ctx, promptName, 1, "delete")
	if err != nil {
		t.Fatalf("DeletePromptVersionTag() error = %v", err)
	}

	// Load and verify tags on version 1
	loaded, err := client.PromptRegistry().LoadPrompt(ctx, promptName, promptregistry.WithVersion(1))
	if err != nil {
		t.Fatalf("LoadPrompt(v1) error = %v", err)
	}

	if loaded.Tags["keep"] != "this" {
		t.Errorf("Tags[keep] = %q, want %q", loaded.Tags["keep"], "this")
	}
	if _, ok := loaded.Tags["delete"]; ok {
		t.Error("Tag 'delete' should have been removed from version")
	}

	// Cleanup
	_ = client.PromptRegistry().DeletePromptVersion(ctx, promptName, 1)
	_ = client.PromptRegistry().DeletePrompt(ctx, promptName)

	t.Log("DeletePromptVersionTag test passed")
}

// TestDeletePromptVersionWithAlias tests alias conflict error.
// Note: MLflow OSS does not enforce alias conflict protection, so this test
// verifies the behavior and skips the conflict check if not supported.
func TestDeletePromptVersionWithAlias(t *testing.T) {
	client, err := mlflow.NewClient(mlflow.WithInsecure())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	promptName := fmt.Sprintf("e2e-alias-conflict-%d", time.Now().UnixNano())

	// Create prompt
	_, err = client.PromptRegistry().RegisterPrompt(ctx, promptName, "Template")
	if err != nil {
		t.Fatalf("RegisterPrompt() error = %v", err)
	}

	// Set alias on version 1
	err = client.PromptRegistry().SetPromptAlias(ctx, promptName, "production", 1)
	if err != nil {
		t.Fatalf("SetPromptAlias() error = %v", err)
	}

	// Try to delete version with alias
	err = client.PromptRegistry().DeletePromptVersion(ctx, promptName, 1)
	if mlflow.IsAliasConflict(err) {
		// Databricks MLflow enforces alias conflict - test the full workflow
		t.Log("Alias conflict enforced, testing removal workflow")

		err = client.PromptRegistry().DeletePromptAlias(ctx, promptName, "production")
		if err != nil {
			t.Fatalf("DeletePromptAlias() error = %v", err)
		}

		err = client.PromptRegistry().DeletePromptVersion(ctx, promptName, 1)
		if err != nil {
			t.Fatalf("DeletePromptVersion() after alias removal error = %v", err)
		}
	} else if err != nil {
		t.Fatalf("DeletePromptVersion() error = %v", err)
	} else {
		// MLflow OSS allows deletion even with aliases
		t.Log("MLflow OSS: alias conflict not enforced, version deleted directly")
	}

	// Cleanup
	_ = client.PromptRegistry().DeletePrompt(ctx, promptName)

	t.Log("DeletePromptVersionWithAlias test passed")
}

// TestWorkspaceIsolation tests that prompts registered in one workspace
// are not visible from another workspace. Requires a midstream MLflow server
// with MLFLOW_ENABLE_WORKSPACES=true and pre-created workspaces (team-bella, team-dora).
//
// Setup:
//
//	make dev/up-midstream
//	make dev/seed-workspaces
//
// Run with: MLFLOW_TEST_WORKSPACES=true make test/integration
func TestWorkspaceIsolation(t *testing.T) {
	if os.Getenv("MLFLOW_TEST_WORKSPACES") != "true" {
		t.Skip("Skipping workspace test: set MLFLOW_TEST_WORKSPACES=true (requires midstream server with workspaces)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create clients for two different workspaces
	clientA, err := mlflow.NewClient(
		mlflow.WithInsecure(),
		mlflow.WithHeaders(map[string]string{"X-MLFLOW-WORKSPACE": "team-bella"}),
	)
	if err != nil {
		t.Fatalf("NewClient(team-bella) error = %v", err)
	}

	clientB, err := mlflow.NewClient(
		mlflow.WithInsecure(),
		mlflow.WithHeaders(map[string]string{"X-MLFLOW-WORKSPACE": "team-dora"}),
	)
	if err != nil {
		t.Fatalf("NewClient(team-dora) error = %v", err)
	}

	promptNameA := fmt.Sprintf("ws-team-bella-%d", time.Now().UnixNano())
	promptNameB := fmt.Sprintf("ws-team-dora-%d", time.Now().UnixNano())

	// Register prompt in team-bella
	t.Log("Registering prompt in team-bella workspace")
	v1, err := clientA.PromptRegistry().RegisterPrompt(ctx, promptNameA,
		"Hello from team-bella!",
		promptregistry.WithCommitMessage("workspace isolation test"),
	)
	if err != nil {
		t.Fatalf("RegisterPrompt(team-bella) error = %v", err)
	}
	if v1.Version != 1 {
		t.Errorf("Expected version 1, got %d", v1.Version)
	}

	// Wait for search indexing (MLflow OSS has a known indexing delay)
	time.Sleep(500 * time.Millisecond)

	// Register prompt in team-dora
	t.Log("Registering prompt in team-dora workspace")
	v1b, err := clientB.PromptRegistry().RegisterPrompt(ctx, promptNameB,
		"Hello from team-dora!",
		promptregistry.WithCommitMessage("workspace isolation test"),
	)
	if err != nil {
		t.Fatalf("RegisterPrompt(team-dora) error = %v", err)
	}
	if v1b.Version != 1 {
		t.Errorf("Expected version 1, got %d", v1b.Version)
	}

	time.Sleep(500 * time.Millisecond)

	// Verify team-bella prompt is visible from team-bella
	t.Log("Verifying team-bella prompt is visible from team-bella")
	loaded, err := clientA.PromptRegistry().LoadPrompt(ctx, promptNameA)
	if err != nil {
		t.Fatalf("LoadPrompt(team-bella, promptA) error = %v", err)
	}
	if loaded.Template != "Hello from team-bella!" {
		t.Errorf("Template = %q, want %q", loaded.Template, "Hello from team-bella!")
	}

	// Verify team-bella prompt is NOT visible from team-dora
	t.Log("Verifying team-bella prompt is NOT visible from team-dora")
	_, err = clientB.PromptRegistry().LoadPrompt(ctx, promptNameA)
	if err == nil {
		t.Fatal("Expected error: team-bella prompt should not be visible from team-dora")
	}
	if !mlflow.IsNotFound(err) {
		t.Errorf("Expected IsNotFound from team-dora, got: %v", err)
	}

	// Verify team-dora prompt is NOT visible from team-bella
	t.Log("Verifying team-dora prompt is NOT visible from team-bella")
	_, err = clientA.PromptRegistry().LoadPrompt(ctx, promptNameB)
	if err == nil {
		t.Fatal("Expected error: team-dora prompt should not be visible from team-bella")
	}
	if !mlflow.IsNotFound(err) {
		t.Errorf("Expected IsNotFound from team-bella, got: %v", err)
	}

	// Verify team-dora prompt is visible from team-dora
	t.Log("Verifying team-dora prompt is visible from team-dora")
	loadedB, err := clientB.PromptRegistry().LoadPrompt(ctx, promptNameB)
	if err != nil {
		t.Fatalf("LoadPrompt(team-dora, promptB) error = %v", err)
	}
	if loadedB.Template != "Hello from team-dora!" {
		t.Errorf("Template = %q, want %q", loadedB.Template, "Hello from team-dora!")
	}

	// Verify list isolation: team-bella list should contain promptA but not promptB
	t.Log("Verifying list isolation")
	listA, err := clientA.PromptRegistry().ListPrompts(ctx,
		promptregistry.WithNameFilter("ws-team-%"),
	)
	if err != nil {
		t.Fatalf("ListPrompts(team-bella) error = %v", err)
	}
	foundA, foundB := false, false
	for _, p := range listA.Prompts {
		if p.Name == promptNameA {
			foundA = true
		}
		if p.Name == promptNameB {
			foundB = true
		}
	}
	if !foundA {
		t.Error("team-bella list should contain its own prompt")
	}
	if foundB {
		t.Error("team-bella list should NOT contain team-dora's prompt")
	}

	// Cleanup
	_ = clientA.PromptRegistry().DeletePromptVersion(ctx, promptNameA, 1)
	_ = clientA.PromptRegistry().DeletePrompt(ctx, promptNameA)
	_ = clientB.PromptRegistry().DeletePromptVersion(ctx, promptNameB, 1)
	_ = clientB.PromptRegistry().DeletePrompt(ctx, promptNameB)

	t.Log("Workspace isolation test passed")
}
