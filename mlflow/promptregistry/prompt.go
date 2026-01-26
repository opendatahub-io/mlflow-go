// Package promptregistry provides types and operations for MLflow Prompt Registry.
//
// # Storage Model: OSS MLflow vs Databricks Unity Catalog
//
// This SDK targets OSS MLflow, which stores prompts using the Model Registry:
//   - A prompt is a RegisteredModel with tag "mlflow.prompt.is_prompt=true"
//   - Each prompt version is a ModelVersion
//   - The template text is stored in ModelVersion.Tags["mlflow.prompt.text"]
//
// Databricks Unity Catalog has native prompt support with a first-class
// PromptVersion.template field. This SDK does not target Unity Catalog.
//
// Because OSS stores templates in tags (not a dedicated field), loading a prompt
// requires fetching the ModelVersion with its tags. The SDK handles this
// transparently, exposing a clean Prompt type to users.
//
// Reference: https://github.com/mlflow/mlflow/blob/master/mlflow/prompts/_prompt_registry.py
package promptregistry

import (
	"maps"
	"time"
)

// ChatMessage represents a single message in a chat prompt.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// PromptVersion represents a prompt version from the MLflow Prompt Registry.
// PromptVersion values are snapshots of server state at load time.
// Modifications do not affect the registry until RegisterPrompt is called.
type PromptVersion struct {
	// Name is the prompt identifier in the registry.
	Name string `json:"name"`

	// Version is the version number (1, 2, 3, ...).
	// Zero if this is a new prompt not yet registered.
	Version int `json:"version"`

	// Template is the prompt template content for text prompts.
	// May contain {{variable}} placeholders.
	// Empty for chat prompts (use Messages instead).
	Template string `json:"template,omitempty"`

	// Messages contains the chat messages for chat prompts.
	// Each message may contain {{variable}} placeholders in Content.
	// Nil for text prompts (use Template instead).
	Messages []ChatMessage `json:"messages,omitempty"`

	// CommitMessage is the version commit message.
	CommitMessage string `json:"commit_message"`

	// Aliases are the aliases pointing to this version (e.g., "production", "staging").
	Aliases []string `json:"aliases,omitempty"`

	// ModelConfig contains optional model configuration.
	ModelConfig *PromptModelConfig `json:"model_config,omitempty"`

	// Tags are key-value metadata pairs.
	Tags map[string]string `json:"tags"`

	// CreatedAt is when this version was created.
	// Zero if not yet registered.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when this version was last updated.
	// Zero if not yet registered.
	UpdatedAt time.Time `json:"updated_at"`
}

// IsChat returns true if this is a chat prompt (has Messages), false for text prompts.
func (v *PromptVersion) IsChat() bool {
	return v.Messages != nil
}

// Prompt represents prompt metadata from a listing operation.
// Use LoadPrompt to get full PromptVersion with template content.
type Prompt struct {
	// Name is the prompt identifier in the registry.
	Name string `json:"name"`

	// Description is the prompt description.
	Description string `json:"description"`

	// LatestVersion is the highest version number, 0 if no versions exist.
	LatestVersion int `json:"latest_version"`

	// Tags are key-value metadata pairs.
	Tags map[string]string `json:"tags"`

	// CreationTimestamp is when the prompt was created.
	CreationTimestamp time.Time `json:"creation_timestamp"`
}

// PromptList contains prompts and a pagination token for the next page.
type PromptList struct {
	// Prompts is the list of prompt metadata in this page.
	Prompts []Prompt `json:"prompts"`

	// NextPageToken is the token to fetch the next page.
	// Empty if there are no more pages.
	NextPageToken string `json:"next_page_token"`
}

// PromptVersionList contains prompt versions and a pagination token.
type PromptVersionList struct {
	// Versions is the list of prompt versions in this page.
	// Template field will be empty; use LoadPrompt with WithVersion to get full content.
	Versions []PromptVersion `json:"versions"`

	// NextPageToken is the token to fetch the next page.
	// Empty if there are no more pages.
	NextPageToken string `json:"next_page_token"`
}

// Clone returns a deep copy of the PromptVersion.
// Use this to create a modified version for registration.
func (v *PromptVersion) Clone() *PromptVersion {
	if v == nil {
		return nil
	}

	clone := &PromptVersion{
		Name:          v.Name,
		Version:       v.Version,
		Template:      v.Template,
		CommitMessage: v.CommitMessage,
		CreatedAt:     v.CreatedAt,
		UpdatedAt:     v.UpdatedAt,
	}

	if v.ModelConfig != nil {
		cfg := *v.ModelConfig
		if v.ModelConfig.StopSequences != nil {
			cfg.StopSequences = make([]string, len(v.ModelConfig.StopSequences))
			copy(cfg.StopSequences, v.ModelConfig.StopSequences)
		}
		if v.ModelConfig.ExtraParams != nil {
			cfg.ExtraParams = make(map[string]any, len(v.ModelConfig.ExtraParams))
			maps.Copy(cfg.ExtraParams, v.ModelConfig.ExtraParams)
		}
		clone.ModelConfig = &cfg
	}

	if v.Messages != nil {
		clone.Messages = make([]ChatMessage, len(v.Messages))
		copy(clone.Messages, v.Messages)
	}

	if v.Aliases != nil {
		clone.Aliases = make([]string, len(v.Aliases))
		copy(clone.Aliases, v.Aliases)
	}

	if v.Tags != nil {
		clone.Tags = make(map[string]string, len(v.Tags))
		maps.Copy(clone.Tags, v.Tags)
	}

	return clone
}

// WithTemplate returns a copy with the template replaced.
func (v *PromptVersion) WithTemplate(template string) *PromptVersion {
	clone := v.Clone()
	clone.Template = template
	return clone
}

// WithCommitMessage returns a copy with the commit message replaced.
func (v *PromptVersion) WithCommitMessage(msg string) *PromptVersion {
	clone := v.Clone()
	clone.CommitMessage = msg
	return clone
}

// WithTag returns a copy with the tag added or updated.
func (v *PromptVersion) WithTag(key, value string) *PromptVersion {
	clone := v.Clone()
	if clone.Tags == nil {
		clone.Tags = make(map[string]string)
	}
	clone.Tags[key] = value
	return clone
}
