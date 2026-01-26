package promptregistry

import (
	"testing"
	"time"
)

func TestPromptVersion_Clone_Nil(t *testing.T) {
	var pv *PromptVersion
	clone := pv.Clone()
	if clone != nil {
		t.Error("expected nil for nil PromptVersion")
	}
}

func TestPromptVersion_Clone_Basic(t *testing.T) {
	original := &PromptVersion{
		Name:          "test-prompt",
		Version:       3,
		Template:      "Hello, {{name}}!",
		CommitMessage: "A greeting",
		Tags:          map[string]string{"team": "ml"},
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
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
	if clone.CommitMessage != original.CommitMessage {
		t.Errorf("CommitMessage = %q, want %q", clone.CommitMessage, original.CommitMessage)
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

func TestPromptVersion_Clone_NilTags(t *testing.T) {
	original := &PromptVersion{
		Name: "test",
		Tags: nil,
	}

	clone := original.Clone()

	if clone.Tags != nil {
		t.Error("nil tags should remain nil after clone")
	}
}

func TestPromptVersion_WithTemplate(t *testing.T) {
	original := &PromptVersion{
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

func TestPromptVersion_WithCommitMessage(t *testing.T) {
	original := &PromptVersion{
		Name:          "test",
		CommitMessage: "old msg",
	}

	modified := original.WithCommitMessage("new msg")

	if modified.CommitMessage != "new msg" {
		t.Errorf("modified.CommitMessage = %q, want %q", modified.CommitMessage, "new msg")
	}
	if original.CommitMessage != "old msg" {
		t.Error("original should not be modified")
	}
}

func TestPromptVersion_WithTag(t *testing.T) {
	original := &PromptVersion{
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

func TestPromptVersion_WithTag_NilTags(t *testing.T) {
	original := &PromptVersion{
		Name: "test",
		Tags: nil,
	}

	modified := original.WithTag("key", "value")

	if modified.Tags["key"] != "value" {
		t.Errorf("modified.Tags[key] = %q, want %q", modified.Tags["key"], "value")
	}
}

func TestPromptVersion_WithTag_UpdateExisting(t *testing.T) {
	original := &PromptVersion{
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

func TestPromptVersion_Chaining(t *testing.T) {
	original := &PromptVersion{
		Name:     "test",
		Template: "original",
	}

	modified := original.
		WithTemplate("new template").
		WithCommitMessage("new msg").
		WithTag("env", "prod")

	if modified.Template != "new template" {
		t.Errorf("Template = %q", modified.Template)
	}
	if modified.CommitMessage != "new msg" {
		t.Errorf("CommitMessage = %q", modified.CommitMessage)
	}
	if modified.Tags["env"] != "prod" {
		t.Errorf("Tags[env] = %q", modified.Tags["env"])
	}
	if original.Template != "original" {
		t.Error("original should not be modified")
	}
}

func TestPromptVersion_Clone_ModelConfig(t *testing.T) {
	temp := 0.7
	original := &PromptVersion{
		Name: "test",
		ModelConfig: &PromptModelConfig{
			Provider:      "openai",
			Temperature:   &temp,
			StopSequences: []string{"stop1", "stop2"},
			ExtraParams:   map[string]any{"foo": "bar"},
		},
	}

	clone := original.Clone()

	// Verify fields are copied
	if clone.ModelConfig.Provider != "openai" {
		t.Errorf("Provider = %q, want %q", clone.ModelConfig.Provider, "openai")
	}
	if *clone.ModelConfig.Temperature != 0.7 {
		t.Errorf("Temperature = %v, want %v", *clone.ModelConfig.Temperature, 0.7)
	}

	// Verify StopSequences is a deep copy
	clone.ModelConfig.StopSequences[0] = "modified"
	if original.ModelConfig.StopSequences[0] == "modified" {
		t.Error("modifying clone's StopSequences should not affect original")
	}

	// Verify ExtraParams is a deep copy
	clone.ModelConfig.ExtraParams["foo"] = "modified"
	if original.ModelConfig.ExtraParams["foo"] == "modified" {
		t.Error("modifying clone's ExtraParams should not affect original")
	}

	// Verify ModelConfig struct itself is a deep copy
	clone.ModelConfig.Provider = "anthropic"
	if original.ModelConfig.Provider == "anthropic" {
		t.Error("modifying clone's ModelConfig should not affect original")
	}
}

func TestPromptVersion_Clone_NilModelConfig(t *testing.T) {
	original := &PromptVersion{
		Name:        "test",
		ModelConfig: nil,
	}

	clone := original.Clone()

	if clone.ModelConfig != nil {
		t.Error("nil ModelConfig should remain nil after clone")
	}
}
