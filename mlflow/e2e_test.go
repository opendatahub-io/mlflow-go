// ABOUTME: End-to-end tests for the MLflow Prompt Registry SDK.
// ABOUTME: Tests full workflow: create, load, version prompts against real MLflow server.

//go:build integration

package mlflow

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestE2E_PromptLifecycle tests the full prompt lifecycle:
// 1. Register a new prompt (creates v1)
// 2. Load the prompt by name (gets latest)
// 3. Modify locally and register new version (creates v2)
// 4. Load specific version (v1)
// 5. Verify versions are correct
func TestE2E_PromptLifecycle(t *testing.T) {
	client, err := NewClient(WithInsecure())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use unique name to avoid conflicts between test runs
	promptName := fmt.Sprintf("e2e-test-prompt-%d", time.Now().UnixNano())

	// Step 1: Register a new prompt
	t.Log("Step 1: Registering new prompt")
	v1, err := client.RegisterPrompt(ctx, promptName,
		"Hello {{name}}!",
		WithDescription("Initial version"),
		WithTags(map[string]string{"test": "e2e"}),
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
	loaded, err := client.LoadPrompt(ctx, promptName)
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
		WithDescription("Added company variable")

	// Verify original is unchanged
	if loaded.Template != "Hello {{name}}!" {
		t.Error("Original prompt was modified")
	}

	v2, err := client.RegisterPrompt(ctx, promptName,
		modified.Template,
		WithDescription(modified.Description),
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
	v1Loaded, err := client.LoadPrompt(ctx, promptName, WithVersion(1))
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
	latest, err := client.LoadPrompt(ctx, promptName)
	if err != nil {
		t.Fatalf("LoadPrompt() latest error = %v", err)
	}

	if latest.Version != 2 {
		t.Errorf("Latest version = %d, want 2", latest.Version)
	}

	t.Log("E2E test passed: full prompt lifecycle verified")
}

// TestE2E_NotFoundError tests that IsNotFound works correctly.
func TestE2E_NotFoundError(t *testing.T) {
	client, err := NewClient(WithInsecure())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()

	_, err = client.LoadPrompt(ctx, "nonexistent-prompt-xyz-123456")
	if err == nil {
		t.Fatal("Expected error for non-existent prompt")
	}

	if !IsNotFound(err) {
		t.Errorf("Expected IsNotFound, got: %v", err)
	}
}

// TestE2E_LoadWithTags tests that tags are preserved correctly.
func TestE2E_LoadWithTags(t *testing.T) {
	client, err := NewClient(WithInsecure())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	promptName := fmt.Sprintf("e2e-tags-test-%d", time.Now().UnixNano())

	// Register with tags
	_, err = client.RegisterPrompt(ctx, promptName,
		"Template with tags",
		WithTags(map[string]string{
			"team":     "ml-platform",
			"category": "greeting",
			"env":      "test",
		}),
	)
	if err != nil {
		t.Fatalf("RegisterPrompt() error = %v", err)
	}

	// Load and verify tags
	loaded, err := client.LoadPrompt(ctx, promptName)
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
