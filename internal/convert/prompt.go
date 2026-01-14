// ABOUTME: Converts between MLflow protobuf types and public SDK types.
// ABOUTME: Handles Model Registry types used for prompt storage in OSS MLflow.

package convert

import (
	"strconv"
	"time"

	"github.com/ederign/mlflow-go/internal/gen/mlflowpb"
)

// Prompt tag keys used by MLflow to store prompt metadata in Model Registry.
const (
	TagPromptText   = "mlflow.prompt.text"
	TagIsPrompt     = "mlflow.prompt.is_prompt"
	TagPromptType   = "_mlflow_prompt_type"
	TagDescription  = "mlflow.prompt.description"
)

// Prompt represents a prompt loaded from the MLflow Prompt Registry.
// This is the public type exposed by the SDK.
type Prompt struct {
	Name        string
	Version     int
	Template    string
	Description string
	Tags        map[string]string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ModelVersionToPrompt converts a ModelVersion proto to a public Prompt.
// Returns nil if the input is nil.
func ModelVersionToPrompt(mv *mlflowpb.ModelVersion) *Prompt {
	if mv == nil {
		return nil
	}

	p := &Prompt{
		Name:      getString(mv.Name),
		Template:  "",
		Tags:      make(map[string]string),
		CreatedAt: timestampToTime(mv.CreationTimestamp),
		UpdatedAt: timestampToTime(mv.LastUpdatedTimestamp),
	}

	// Parse version string to int
	if mv.Version != nil {
		if v, err := strconv.Atoi(*mv.Version); err == nil {
			p.Version = v
		}
	}

	// Use model version description
	p.Description = getString(mv.Description)

	// Process tags - extract prompt template and user tags
	for _, tag := range mv.Tags {
		if tag == nil || tag.Key == nil {
			continue
		}
		key := *tag.Key
		value := getString(tag.Value)

		switch key {
		case TagPromptText:
			p.Template = value
		case TagDescription:
			// Prefer tag-based description over model version description
			if value != "" {
				p.Description = value
			}
		case TagIsPrompt, TagPromptType:
			// Internal tags, don't expose to user
		default:
			// User-defined tags
			p.Tags[key] = value
		}
	}

	return p
}

// PromptToModelVersionTags converts a Prompt to ModelVersion tags for registration.
// This creates the tags needed to store a prompt as a Model Registry entity.
func PromptToModelVersionTags(p *Prompt) []*mlflowpb.ModelVersionTag {
	if p == nil {
		return nil
	}

	tags := make([]*mlflowpb.ModelVersionTag, 0, len(p.Tags)+3)

	// Add prompt metadata tags
	tags = append(tags, stringTag(TagIsPrompt, "true"))
	tags = append(tags, stringTag(TagPromptText, p.Template))

	if p.Description != "" {
		tags = append(tags, stringTag(TagDescription, p.Description))
	}

	// Add user-defined tags
	for k, v := range p.Tags {
		tags = append(tags, stringTag(k, v))
	}

	return tags
}

// timestampToTime converts an MLflow timestamp (milliseconds since epoch) to time.Time.
func timestampToTime(ts *int64) time.Time {
	if ts == nil || *ts == 0 {
		return time.Time{}
	}
	return time.UnixMilli(*ts)
}

// getString safely dereferences a string pointer, returning empty string if nil.
func getString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// stringTag creates a ModelVersionTag with the given key and value.
func stringTag(key, value string) *mlflowpb.ModelVersionTag {
	return &mlflowpb.ModelVersionTag{
		Key:   &key,
		Value: &value,
	}
}
