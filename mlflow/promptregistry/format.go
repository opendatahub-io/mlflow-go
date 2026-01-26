package promptregistry

import (
	"fmt"
	"regexp"
	"strings"
)

// varPattern matches {{variable}} placeholders.
var varPattern = regexp.MustCompile(`\{\{(\w+)\}\}`)

// Format returns a new PromptVersion with all {{variable}} placeholders replaced.
// Returns an error if any variable in the template is not found in vars.
func (v *PromptVersion) Format(vars map[string]string) (*PromptVersion, error) {
	if v == nil {
		return nil, fmt.Errorf("mlflow: cannot format nil PromptVersion")
	}

	clone := v.Clone()

	if v.IsChat() {
		for i := range clone.Messages {
			formatted, err := substituteVars(clone.Messages[i].Content, vars)
			if err != nil {
				return nil, fmt.Errorf("mlflow: message %d: %w", i, err)
			}
			clone.Messages[i].Content = formatted
		}
	} else {
		formatted, err := substituteVars(clone.Template, vars)
		if err != nil {
			return nil, err
		}
		clone.Template = formatted
	}

	return clone, nil
}

// FormatAsText formats the prompt and returns the template string.
// Returns an error if this is a chat prompt or if any variable is not found.
func (v *PromptVersion) FormatAsText(vars map[string]string) (string, error) {
	if v == nil {
		return "", fmt.Errorf("mlflow: cannot format nil PromptVersion")
	}
	if v.IsChat() {
		return "", fmt.Errorf("mlflow: cannot format chat prompt as text; use FormatAsMessages")
	}

	return substituteVars(v.Template, vars)
}

// FormatAsMessages formats the prompt and returns the messages.
// Returns an error if this is a text prompt or if any variable is not found.
func (v *PromptVersion) FormatAsMessages(vars map[string]string) ([]ChatMessage, error) {
	if v == nil {
		return nil, fmt.Errorf("mlflow: cannot format nil PromptVersion")
	}
	if !v.IsChat() {
		return nil, fmt.Errorf("mlflow: cannot format text prompt as messages; use FormatAsText")
	}

	result := make([]ChatMessage, len(v.Messages))
	for i, msg := range v.Messages {
		formatted, err := substituteVars(msg.Content, vars)
		if err != nil {
			return nil, fmt.Errorf("mlflow: message %d: %w", i, err)
		}
		result[i] = ChatMessage{
			Role:    msg.Role,
			Content: formatted,
		}
	}

	return result, nil
}

// substituteVars replaces all {{variable}} placeholders in template with values from vars.
// Returns an error if any variable is not found in vars.
func substituteVars(template string, vars map[string]string) (string, error) {
	var missingVars []string

	result := varPattern.ReplaceAllStringFunc(template, func(match string) string {
		// Extract variable name from {{name}}
		name := match[2 : len(match)-2]
		if value, ok := vars[name]; ok {
			return value
		}
		missingVars = append(missingVars, name)
		return match
	})

	if len(missingVars) > 0 {
		return "", fmt.Errorf("mlflow: missing variables: %s", strings.Join(missingVars, ", "))
	}

	return result, nil
}
