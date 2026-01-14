// ABOUTME: Defines the Prompt type representing a prompt from the registry.
// ABOUTME: Provides immutable modification methods for local editing.

package mlflow

import (
	"time"
)

// Prompt represents a prompt version from the MLflow Prompt Registry.
// Prompt values are snapshots of server state at load time.
// Modifications to a Prompt do not affect the registry until RegisterPrompt is called.
type Prompt struct {
	// Name is the prompt identifier in the registry.
	Name string

	// Version is the version number (1, 2, 3, ...).
	// Zero if this is a new prompt not yet registered.
	Version int

	// Template is the prompt template content.
	// May contain {{variable}} placeholders.
	Template string

	// Description is the version description or commit message.
	Description string

	// Tags are key-value metadata pairs.
	Tags map[string]string

	// CreatedAt is when this version was created.
	// Zero if not yet registered.
	CreatedAt time.Time

	// UpdatedAt is when this version was last updated.
	// Zero if not yet registered.
	UpdatedAt time.Time
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
		for k, v := range p.Tags {
			clone.Tags[k] = v
		}
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
