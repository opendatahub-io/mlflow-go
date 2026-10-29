package promptregistry

import (
	"encoding/json"
	"reflect"
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

func TestPromptVersion_ModelConfigIsolation(t *testing.T) {
	transforms := []struct {
		name  string
		apply func(*PromptVersion) (*PromptVersion, error)
	}{
		{"Clone", func(v *PromptVersion) (*PromptVersion, error) { return v.Clone(), nil }},
		{"WithTemplate", func(v *PromptVersion) (*PromptVersion, error) { return v.WithTemplate("new"), nil }},
		{"WithTag", func(v *PromptVersion) (*PromptVersion, error) { return v.WithTag("key", "value"), nil }},
		{"WithCommitMessage", func(v *PromptVersion) (*PromptVersion, error) { return v.WithCommitMessage("new"), nil }},
		{"FormatText", func(v *PromptVersion) (*PromptVersion, error) {
			return v.Format(map[string]string{"name": "Alice"})
		}},
		{"FormatChat", func(v *PromptVersion) (*PromptVersion, error) {
			v.Messages = []ChatMessage{{Role: "user", Content: "Hello, {{name}}!"}}
			return v.Format(map[string]string{"name": "Alice"})
		}},
	}

	for _, transform := range transforms {
		t.Run(transform.name, func(t *testing.T) {
			for _, mutateOriginal := range []bool{false, true} {
				name := "mutate-clone"
				if mutateOriginal {
					name = "mutate-original"
				}
				t.Run(name, func(t *testing.T) {
					original := &PromptVersion{
						Template:    "Hello, {{name}}!",
						ModelConfig: newTestModelConfig(),
					}
					clone, err := transform.apply(original)
					if err != nil {
						t.Fatalf("%s() error = %v", transform.name, err)
					}
					if !reflect.DeepEqual(clone.ModelConfig, original.ModelConfig) {
						t.Fatal("model configuration values or types changed during copy")
					}

					source, target := clone.ModelConfig, original.ModelConfig
					if mutateOriginal {
						source, target = target, source
					}
					mutateTestModelConfig(source)
					if !reflect.DeepEqual(target, newTestModelConfig()) {
						t.Errorf("mutating one prompt changed the other's model configuration: %+v", target)
					}
				})
			}
		})
	}
}

func newTestModelConfig() *PromptModelConfig {
	temperature, topP, frequencyPenalty, presencePenalty := 0.7, 0.9, 0.1, 0.2
	maxTokens, topK := 1024, 40
	return &PromptModelConfig{
		Provider:         "openai",
		ModelName:        "test-model",
		Temperature:      &temperature,
		MaxTokens:        &maxTokens,
		TopP:             &topP,
		TopK:             &topK,
		FrequencyPenalty: &frequencyPenalty,
		PresencePenalty:  &presencePenalty,
		StopSequences:    []string{"stop"},
		ExtraParams: map[string]any{
			"object": map[string]any{
				"value":  "original",
				"object": map[string]any{"value": "original"},
				"array":  []any{"original", map[string]any{"value": "original"}, []any{"original"}},
			},
			"array":      []any{map[string]any{"value": "original"}, []any{"original"}},
			"int":        int(42),
			"int64":      int64(42),
			"float32":    float32(0.5),
			"float64":    float64(0.5),
			"jsonNumber": json.Number("42"),
			"bool":       true,
			"nil":        nil,
			"nilMap":     map[string]any(nil),
			"nilArray":   []any(nil),
			"emptyMap":   map[string]any{},
			"emptyArray": []any{},
		},
	}
}

func mutateTestModelConfig(cfg *PromptModelConfig) {
	*cfg.Temperature = 0
	*cfg.MaxTokens = 0
	*cfg.TopP = 0
	*cfg.TopK = 0
	*cfg.FrequencyPenalty = 0
	*cfg.PresencePenalty = 0
	cfg.StopSequences[0] = "modified"
	cfg.ExtraParams["int"] = 0
	object := cfg.ExtraParams["object"].(map[string]any)
	object["value"] = "modified"
	object["object"].(map[string]any)["value"] = "modified"
	array := object["array"].([]any)
	array[0] = "modified"
	array[1].(map[string]any)["value"] = "modified"
	array[2].([]any)[0] = "modified"
	array = cfg.ExtraParams["array"].([]any)
	array[0].(map[string]any)["value"] = "modified"
	array[1].([]any)[0] = "modified"
}

func TestPromptVersion_Clone_NilAndEmptyModelConfigFields(t *testing.T) {
	for _, cfg := range []*PromptModelConfig{
		{},
		{StopSequences: []string{}, ExtraParams: map[string]any{}},
	} {
		original := &PromptVersion{ModelConfig: cfg}
		clone := original.Clone()
		if !reflect.DeepEqual(clone.ModelConfig, original.ModelConfig) {
			t.Errorf("nil or empty model configuration fields changed: got %+v, want %+v", clone.ModelConfig, cfg)
		}
	}
}

func TestPromptVersion_Clone_CollectionsIsolation(t *testing.T) {
	newPrompt := func() *PromptVersion {
		return &PromptVersion{
			Messages: []ChatMessage{{Role: "user", Content: "Hello"}},
			Aliases:  []string{"production"},
			Tags:     map[string]string{"team": "ml"},
		}
	}
	for _, mutateOriginal := range []bool{false, true} {
		original := newPrompt()
		clone := original.Clone()
		if !reflect.DeepEqual(clone, original) {
			t.Fatal("collection values changed during copy")
		}
		source, target := clone, original
		if mutateOriginal {
			source, target = target, source
		}
		source.Messages[0].Content = "modified"
		source.Aliases[0] = "modified"
		source.Tags["team"] = "modified"
		if !reflect.DeepEqual(target, newPrompt()) {
			t.Errorf("mutating one prompt changed the other's collections: %+v", target)
		}
	}
}
