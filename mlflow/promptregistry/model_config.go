package promptregistry

// PromptModelConfig contains optional model configuration for a prompt.
type PromptModelConfig struct {
	Provider         string         `json:"provider,omitempty"`
	ModelName        string         `json:"model_name,omitempty"`
	Temperature      *float64       `json:"temperature,omitempty"`
	MaxTokens        *int           `json:"max_tokens,omitempty"`
	TopP             *float64       `json:"top_p,omitempty"`
	TopK             *int           `json:"top_k,omitempty"`
	FrequencyPenalty *float64       `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64       `json:"presence_penalty,omitempty"`
	StopSequences    []string       `json:"stop_sequences,omitempty"`
	ExtraParams      map[string]any `json:"extra_params,omitempty"`
}
