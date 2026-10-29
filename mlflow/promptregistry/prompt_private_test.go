package promptregistry

import (
	"encoding/json"
	"math/big"
	"reflect"
	"testing"
)

// Only MarshalJSON is implemented: cloning must preserve private state without
// assuming that a JSON-marshaling value can also be unmarshaled.
type testPrivateJSONParams struct {
	stops  []string
	labels map[string]string
	number *big.Int
	self   *testPrivateJSONParams
}

func (p *testPrivateJSONParams) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Stops  []string
		Labels map[string]string
		Number *big.Int
	}{p.stops, p.labels, p.number})
}

func (p *testPrivateJSONParams) SetStop(stop string) {
	p.self.stops[0] = stop
}

func (p *testPrivateJSONParams) SetLabel(label string) {
	p.labels["key"] = label
}

func newTestPrivateModelConfig() *PromptModelConfig {
	number := big.NewInt(42)
	private := &testPrivateJSONParams{
		stops:  []string{"END"},
		labels: map[string]string{"key": "original"},
		number: number,
	}
	private.self = private
	return &PromptModelConfig{ExtraParams: map[string]any{
		"int":        number,
		"intAlias":   number,
		"intValue":   *big.NewInt(42),
		"rat":        big.NewRat(42, 5),
		"float":      big.NewFloat(42).SetPrec(128).SetMode(big.ToNegativeInf),
		"private":    private,
		"nilInt":     (*big.Int)(nil),
		"zeroInt":    new(big.Int),
		"zeroRat":    new(big.Rat),
		"zeroFloat":  new(big.Float),
		"nestedInts": []any{map[string]*big.Int{"value": big.NewInt(42)}},
	}}
}

func mutateTestPrivateModelConfig(cfg *PromptModelConfig) {
	cfg.ExtraParams["int"].(*big.Int).SetInt64(7)
	// SetInt64 reuses the value's private backing slice even for a struct copy.
	integer := cfg.ExtraParams["intValue"].(big.Int)
	integer.SetInt64(7)
	cfg.ExtraParams["intValue"] = integer
	cfg.ExtraParams["rat"].(*big.Rat).SetInt64(7)
	cfg.ExtraParams["float"].(*big.Float).SetFloat64(7)
	private := cfg.ExtraParams["private"].(*testPrivateJSONParams)
	private.SetStop("modified")
	private.SetLabel("modified")
	cfg.ExtraParams["nestedInts"].([]any)[0].(map[string]*big.Int)["value"].SetInt64(7)
}

func TestPromptVersion_PrivateModelConfigIsolation(t *testing.T) {
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
					original := &PromptVersion{Template: "Hello, {{name}}!", ModelConfig: newTestPrivateModelConfig()}
					if _, err := json.Marshal(original.ModelConfig); err != nil {
						t.Fatalf("test configuration is not JSON-serializable: %v", err)
					}
					clone, err := transform.apply(original)
					if err != nil {
						t.Fatalf("%s() error = %v", transform.name, err)
					}
					if !reflect.DeepEqual(original.ModelConfig, newTestPrivateModelConfig()) {
						t.Fatal("cloning modified the original private configuration state")
					}
					if !reflect.DeepEqual(clone.ModelConfig, original.ModelConfig) {
						t.Fatal("private model configuration values or concrete types changed during copy")
					}
					private := clone.ModelConfig.ExtraParams["private"].(*testPrivateJSONParams)
					if private.self != private || private.number != clone.ModelConfig.ExtraParams["int"] ||
						private.number != clone.ModelConfig.ExtraParams["intAlias"] {
						t.Error("private cycles or repeated references were not preserved")
					}
					source, target := clone.ModelConfig, original.ModelConfig
					if mutateOriginal {
						source, target = target, source
					}
					mutateTestPrivateModelConfig(source)
					if reflect.DeepEqual(source, newTestPrivateModelConfig()) {
						t.Fatal("public mutation methods did not change the source")
					}
					if got := target.ExtraParams["int"].(*big.Int).Int64(); got != 42 {
						t.Errorf("SetInt64 changed the other prompt's integer to %d, want 42", got)
					}
					if !reflect.DeepEqual(target, newTestPrivateModelConfig()) {
						t.Error("public mutation methods changed the other prompt's private configuration state")
					}
				})
			}
		})
	}
}
