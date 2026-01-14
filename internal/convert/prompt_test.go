// ABOUTME: Tests for prompt type conversion functions.
// ABOUTME: Verifies correct mapping between proto types and public SDK types.

package convert

import (
	"testing"
	"time"

	"github.com/ederign/mlflow-go/internal/gen/mlflowpb"
)

func ptr[T any](v T) *T {
	return &v
}

func TestModelVersionToPrompt_Nil(t *testing.T) {
	result := ModelVersionToPrompt(nil)
	if result != nil {
		t.Error("expected nil for nil input")
	}
}

func TestModelVersionToPrompt_Basic(t *testing.T) {
	mv := &mlflowpb.ModelVersion{
		Name:                 ptr("greeting-prompt"),
		Version:              ptr("3"),
		Description:          ptr("A friendly greeting"),
		CreationTimestamp:    ptr(int64(1700000000000)),
		LastUpdatedTimestamp: ptr(int64(1700000100000)),
		Tags: []*mlflowpb.ModelVersionTag{
			{Key: ptr(TagIsPrompt), Value: ptr("true")},
			{Key: ptr(TagPromptText), Value: ptr("Hello, {{name}}!")},
			{Key: ptr("team"), Value: ptr("ml")},
		},
	}

	p := ModelVersionToPrompt(mv)

	if p.Name != "greeting-prompt" {
		t.Errorf("Name = %q, want %q", p.Name, "greeting-prompt")
	}
	if p.Version != 3 {
		t.Errorf("Version = %d, want %d", p.Version, 3)
	}
	if p.Template != "Hello, {{name}}!" {
		t.Errorf("Template = %q, want %q", p.Template, "Hello, {{name}}!")
	}
	if p.Description != "A friendly greeting" {
		t.Errorf("Description = %q, want %q", p.Description, "A friendly greeting")
	}
	if p.Tags["team"] != "ml" {
		t.Errorf("Tags[team] = %q, want %q", p.Tags["team"], "ml")
	}
	// Internal tags should not be exposed
	if _, ok := p.Tags[TagIsPrompt]; ok {
		t.Error("internal tag TagIsPrompt should not be in user tags")
	}
	if _, ok := p.Tags[TagPromptText]; ok {
		t.Error("internal tag TagPromptText should not be in user tags")
	}
}

func TestModelVersionToPrompt_TimestampConversion(t *testing.T) {
	ts := int64(1700000000000) // 2023-11-14 22:13:20 UTC
	mv := &mlflowpb.ModelVersion{
		Name:              ptr("test"),
		Version:           ptr("1"),
		CreationTimestamp: &ts,
	}

	p := ModelVersionToPrompt(mv)

	expected := time.UnixMilli(1700000000000)
	if !p.CreatedAt.Equal(expected) {
		t.Errorf("CreatedAt = %v, want %v", p.CreatedAt, expected)
	}
}

func TestModelVersionToPrompt_DescriptionFromTag(t *testing.T) {
	mv := &mlflowpb.ModelVersion{
		Name:        ptr("test"),
		Version:     ptr("1"),
		Description: ptr("model version desc"),
		Tags: []*mlflowpb.ModelVersionTag{
			{Key: ptr(TagDescription), Value: ptr("prompt tag desc")},
		},
	}

	p := ModelVersionToPrompt(mv)

	// Tag description should take precedence
	if p.Description != "prompt tag desc" {
		t.Errorf("Description = %q, want %q", p.Description, "prompt tag desc")
	}
}

func TestModelVersionToPrompt_EmptyTags(t *testing.T) {
	mv := &mlflowpb.ModelVersion{
		Name:    ptr("test"),
		Version: ptr("1"),
	}

	p := ModelVersionToPrompt(mv)

	if p.Tags == nil {
		t.Error("Tags should be initialized even when empty")
	}
	if len(p.Tags) != 0 {
		t.Errorf("Tags length = %d, want 0", len(p.Tags))
	}
}

func TestPromptToModelVersionTags_Nil(t *testing.T) {
	result := PromptToModelVersionTags(nil)
	if result != nil {
		t.Error("expected nil for nil input")
	}
}

func TestPromptToModelVersionTags_Basic(t *testing.T) {
	p := &Prompt{
		Template:    "Hello, {{name}}!",
		Description: "A greeting prompt",
		Tags: map[string]string{
			"team": "ml",
			"env":  "prod",
		},
	}

	tags := PromptToModelVersionTags(p)

	// Should have: is_prompt, prompt_text, description, team, env = 5 tags
	if len(tags) != 5 {
		t.Errorf("tags length = %d, want 5", len(tags))
	}

	tagMap := make(map[string]string)
	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			tagMap[*tag.Key] = *tag.Value
		}
	}

	if tagMap[TagIsPrompt] != "true" {
		t.Errorf("TagIsPrompt = %q, want %q", tagMap[TagIsPrompt], "true")
	}
	if tagMap[TagPromptText] != "Hello, {{name}}!" {
		t.Errorf("TagPromptText = %q, want %q", tagMap[TagPromptText], "Hello, {{name}}!")
	}
	if tagMap[TagDescription] != "A greeting prompt" {
		t.Errorf("TagDescription = %q, want %q", tagMap[TagDescription], "A greeting prompt")
	}
	if tagMap["team"] != "ml" {
		t.Errorf("team tag = %q, want %q", tagMap["team"], "ml")
	}
}

func TestPromptToModelVersionTags_NoDescription(t *testing.T) {
	p := &Prompt{
		Template: "Hello!",
	}

	tags := PromptToModelVersionTags(p)

	// Should have: is_prompt, prompt_text = 2 tags (no description)
	if len(tags) != 2 {
		t.Errorf("tags length = %d, want 2", len(tags))
	}
}

func TestTimestampToTime_Nil(t *testing.T) {
	result := timestampToTime(nil)
	if !result.IsZero() {
		t.Error("expected zero time for nil input")
	}
}

func TestTimestampToTime_Zero(t *testing.T) {
	zero := int64(0)
	result := timestampToTime(&zero)
	if !result.IsZero() {
		t.Error("expected zero time for zero input")
	}
}

func TestGetString_Nil(t *testing.T) {
	result := getString(nil)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestGetString_Value(t *testing.T) {
	s := "hello"
	result := getString(&s)
	if result != "hello" {
		t.Errorf("expected %q, got %q", "hello", result)
	}
}
