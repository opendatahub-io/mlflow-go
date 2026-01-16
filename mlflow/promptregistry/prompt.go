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

// Prompt represents a prompt version from the MLflow Prompt Registry.
// Prompt values are snapshots of server state at load time.
// Modifications to a Prompt do not affect the registry until RegisterPrompt is called.
type Prompt struct {
	// Name is the prompt identifier in the registry.
	Name string `json:"name"`

	// Version is the version number (1, 2, 3, ...).
	// Zero if this is a new prompt not yet registered.
	Version int `json:"version"`

	// Template is the prompt template content.
	// May contain {{variable}} placeholders.
	Template string `json:"template"`

	// Description is the version description or commit message.
	Description string `json:"description"`

	// Tags are key-value metadata pairs.
	Tags map[string]string `json:"tags"`

	// CreatedAt is when this version was created.
	// Zero if not yet registered.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when this version was last updated.
	// Zero if not yet registered.
	UpdatedAt time.Time `json:"updated_at"`
}

// PromptInfo represents prompt metadata from a listing operation.
// Use LoadPrompt to get full Prompt with template content.
type PromptInfo struct {
	// Name is the prompt identifier in the registry.
	Name string `json:"name"`

	// Description is the prompt description.
	Description string `json:"description"`

	// LatestVersion is the highest version number, 0 if no versions exist.
	LatestVersion int `json:"latest_version"`

	// Tags are key-value metadata pairs.
	Tags map[string]string `json:"tags"`

	// CreatedAt is when the prompt was created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the prompt was last updated.
	UpdatedAt time.Time `json:"updated_at"`
}

// PromptList contains prompts and a pagination token for the next page.
type PromptList struct {
	// Prompts is the list of prompt metadata in this page.
	Prompts []PromptInfo `json:"prompts"`

	// NextPageToken is the token to fetch the next page.
	// Empty if there are no more pages.
	NextPageToken string `json:"next_page_token"`
}

// PromptVersionList contains prompt versions and a pagination token.
type PromptVersionList struct {
	// Versions is the list of prompt versions in this page.
	// Template field will be empty; use LoadPrompt with WithVersion to get full content.
	Versions []Prompt `json:"versions"`

	// NextPageToken is the token to fetch the next page.
	// Empty if there are no more pages.
	NextPageToken string `json:"next_page_token"`
}

// Clone returns a deep copy of the Prompt.
// Use this to create a modified version for registration.
func (p *Prompt) Clone() *Prompt {
	if p == nil {
		return nil
	}

	clone := &Prompt{
		Name:        p.Name,
		Version:     p.Version,
		Template:    p.Template,
		Description: p.Description,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}

	if p.Tags != nil {
		clone.Tags = make(map[string]string, len(p.Tags))
		maps.Copy(clone.Tags, p.Tags)
	}

	return clone
}

// WithTemplate returns a copy with the template replaced.
func (p *Prompt) WithTemplate(template string) *Prompt {
	clone := p.Clone()
	clone.Template = template
	return clone
}

// WithDescription returns a copy with the description replaced.
func (p *Prompt) WithDescription(description string) *Prompt {
	clone := p.Clone()
	clone.Description = description
	return clone
}

// WithTag returns a copy with the tag added or updated.
func (p *Prompt) WithTag(key, value string) *Prompt {
	clone := p.Clone()
	if clone.Tags == nil {
		clone.Tags = make(map[string]string)
	}
	clone.Tags[key] = value
	return clone
}
