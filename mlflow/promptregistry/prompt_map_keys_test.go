package promptregistry

import (
	"bytes"
	"encoding/json"
	"math/big"
	"testing"
)

func TestPromptVersion_MapKeyIsolation(t *testing.T) {
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
	marshalConfig := func(t *testing.T, cfg *PromptModelConfig) []byte {
		t.Helper()
		data, err := json.Marshal(cfg)
		if err != nil {
			t.Fatalf("json.Marshal(ModelConfig) error = %v", err)
		}
		return data
	}
	for _, transform := range transforms {
		t.Run(transform.name, func(t *testing.T) {
			for _, mutateOriginal := range []bool{false, true} {
				name := "mutate-clone"
				if mutateOriginal {
					name = "mutate-original"
				}
				t.Run(name, func(t *testing.T) {
					labelKey, sharedKey := big.NewInt(84), big.NewInt(42)
					original := &PromptVersion{
						Template: "Hello, {{name}}!",
						ModelConfig: &PromptModelConfig{ExtraParams: map[string]any{
							"labels": map[*big.Int]string{labelKey: "original"},
							"shared": map[*big.Int]*big.Int{sharedKey: sharedKey},
							"alias":  sharedKey,
						}},
					}
					before := marshalConfig(t, original.ModelConfig)
					clone, err := transform.apply(original)
					if err != nil {
						t.Fatalf("%s() error = %v", transform.name, err)
					}
					if !bytes.Equal(marshalConfig(t, original.ModelConfig), before) ||
						!bytes.Equal(marshalConfig(t, clone.ModelConfig), before) {
						t.Fatal("cloning changed the serialized map configuration")
					}
					labels := clone.ModelConfig.ExtraParams["labels"].(map[*big.Int]string)
					shared := clone.ModelConfig.ExtraParams["shared"].(map[*big.Int]*big.Int)
					if len(labels) != 1 || len(shared) != 1 {
						t.Fatal("cloning changed the number of map entries")
					}
					for key := range labels {
						if key == labelKey {
							t.Error("cloned label key still references the original key")
						}
					}
					for key, value := range shared {
						if key == sharedKey {
							t.Error("cloned shared key still references the original key")
						}
						if key != value || key != clone.ModelConfig.ExtraParams["alias"] {
							t.Error("cloning lost the alias between map keys and values")
						}
					}
					source, target := clone.ModelConfig, original.ModelConfig
					if mutateOriginal {
						source, target = target, source
					}
					for key := range source.ExtraParams["labels"].(map[*big.Int]string) {
						key.SetInt64(9)
					}
					for key := range source.ExtraParams["shared"].(map[*big.Int]*big.Int) {
						key.SetInt64(7)
					}
					if bytes.Equal(marshalConfig(t, source), before) {
						t.Fatal("mutating map keys did not change the source configuration")
					}
					if !bytes.Equal(marshalConfig(t, target), before) {
						t.Error("mutating map keys changed the other prompt's serialized configuration")
					}
				})
			}
		})
	}
}
