// ABOUTME: Tests for the Prompt type.
// ABOUTME: Verifies immutable modification methods work correctly.

package mlflow

import (
	"testing"
	"time"
)

func TestPrompt_Clone_Nil(t *testing.T) {
	var p *Prompt
	clone := p.Clone()
	if clone != nil {
		t.Error("expected nil for nil prompt")
	}
}

func TestPrompt_Clone_Basic(t *testing.T) {
	original := &Prompt{
		Name:        "test-prompt",
		Version:     3,
		Template:    "Hello, {{name}}!",
		Description: "A greeting",
		Tags:        map[string]string{"team": "ml"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	clone := original.Clone()

	// Verify all fields are copied
	if clone.Name != original.Name {
		t.Errorf("Name = %q, want %q", clone.Name, original.Name)
	}
	if clone.Version != original.Version {
		t.Errorf("Version = %d, want %d", clone.Version, original.Version)
	}
	if clone.Template != original.Template {
		t.Errorf("Template = %q, want %q", clone.Template, original.Template)
	}
	if clone.Description != original.Description {
		t.Errorf("Description = %q, want %q", clone.Description, original.Description)
	}
	if clone.Tags["team"] != "ml" {
		t.Errorf("Tags[team] = %q, want %q", clone.Tags["team"], "ml")
	}

	// Verify tags are a deep copy
	clone.Tags["team"] = "modified"
	if original.Tags["team"] == "modified" {
		t.Error("modifying clone's tags should not affect original")
	}
}

func TestPrompt_Clone_NilTags(t *testing.T) {
	original := &Prompt{
		Name: "test",
		Tags: nil,
	}

	clone := original.Clone()

	if clone.Tags != nil {
		t.Error("nil tags should remain nil after clone")
	}
}

func TestPrompt_WithTemplate(t *testing.T) {
	original := &Prompt{
		Name:     "test",
		Template: "old template",
	}

	modified := original.WithTemplate("new template")

	if modified.Template != "new template" {
		t.Errorf("modified.Template = %q, want %q", modified.Template, "new template")
	}
	if original.Template != "old template" {
		t.Error("original should not be modified")
	}
}

func TestPrompt_WithDescription(t *testing.T) {
	original := &Prompt{
		Name:        "test",
		Description: "old desc",
	}

	modified := original.WithDescription("new desc")

	if modified.Description != "new desc" {
		t.Errorf("modified.Description = %q, want %q", modified.Description, "new desc")
	}
	if original.Description != "old desc" {
		t.Error("original should not be modified")
	}
}

func TestPrompt_WithTag(t *testing.T) {
	original := &Prompt{
		Name: "test",
		Tags: map[string]string{"existing": "value"},
	}

	modified := original.WithTag("new", "tag")

	if modified.Tags["new"] != "tag" {
		t.Errorf("modified.Tags[new] = %q, want %q", modified.Tags["new"], "tag")
	}
	if modified.Tags["existing"] != "value" {
		t.Error("existing tag should be preserved")
	}
	if _, ok := original.Tags["new"]; ok {
		t.Error("original should not be modified")
	}
}

func TestPrompt_WithTag_NilTags(t *testing.T) {
	original := &Prompt{
		Name: "test",
		Tags: nil,
	}

	modified := original.WithTag("key", "value")

	if modified.Tags["key"] != "value" {
		t.Errorf("modified.Tags[key] = %q, want %q", modified.Tags["key"], "value")
	}
}

func TestPrompt_WithTag_UpdateExisting(t *testing.T) {
	original := &Prompt{
		Name: "test",
		Tags: map[string]string{"key": "old"},
	}

	modified := original.WithTag("key", "new")

	if modified.Tags["key"] != "new" {
		t.Errorf("modified.Tags[key] = %q, want %q", modified.Tags["key"], "new")
	}
	if original.Tags["key"] != "old" {
		t.Error("original should not be modified")
	}
}

func TestPrompt_Chaining(t *testing.T) {
	original := &Prompt{
		Name:     "test",
		Template: "original",
	}

	modified := original.
		WithTemplate("new template").
		WithDescription("new desc").
		WithTag("env", "prod")

	if modified.Template != "new template" {
		t.Errorf("Template = %q", modified.Template)
	}
	if modified.Description != "new desc" {
		t.Errorf("Description = %q", modified.Description)
	}
	if modified.Tags["env"] != "prod" {
		t.Errorf("Tags[env] = %q", modified.Tags["env"])
	}
	if original.Template != "original" {
		t.Error("original should not be modified")
	}
}
