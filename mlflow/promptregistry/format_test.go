package promptregistry

import (
	"testing"
)

func TestPromptVersion_Format_TextPrompt(t *testing.T) {
	pv := &PromptVersion{
		Name:     "test",
		Template: "Hello, {{name}}! Welcome to {{company}}.",
	}

	result, err := pv.Format(map[string]string{"name": "Alice", "company": "Acme"})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	if result.Template != "Hello, Alice! Welcome to Acme." {
		t.Errorf("Template = %q, want %q", result.Template, "Hello, Alice! Welcome to Acme.")
	}

	// Original should be unchanged
	if pv.Template != "Hello, {{name}}! Welcome to {{company}}." {
		t.Error("original prompt should not be modified")
	}
}

func TestPromptVersion_Format_ChatPrompt(t *testing.T) {
	pv := &PromptVersion{
		Name: "test",
		Messages: []ChatMessage{
			{Role: "system", Content: "You are a helpful assistant for {{company}}."},
			{Role: "user", Content: "Hello, my name is {{name}}."},
		},
	}

	result, err := pv.Format(map[string]string{"name": "Bob", "company": "Acme"})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	if result.Messages[0].Content != "You are a helpful assistant for Acme." {
		t.Errorf("Messages[0].Content = %q", result.Messages[0].Content)
	}
	if result.Messages[1].Content != "Hello, my name is Bob." {
		t.Errorf("Messages[1].Content = %q", result.Messages[1].Content)
	}

	// Original should be unchanged
	if pv.Messages[0].Content != "You are a helpful assistant for {{company}}." {
		t.Error("original prompt should not be modified")
	}
}

func TestPromptVersion_Format_MissingVariable(t *testing.T) {
	pv := &PromptVersion{
		Name:     "test",
		Template: "Hello, {{name}}! Welcome to {{company}}.",
	}

	_, err := pv.Format(map[string]string{"name": "Alice"})
	if err == nil {
		t.Error("expected error for missing variable")
	}
}

func TestPromptVersion_Format_NoVariables(t *testing.T) {
	pv := &PromptVersion{
		Name:     "test",
		Template: "Hello, world!",
	}

	result, err := pv.Format(map[string]string{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	if result.Template != "Hello, world!" {
		t.Errorf("Template = %q, want %q", result.Template, "Hello, world!")
	}
}

func TestPromptVersion_Format_Nil(t *testing.T) {
	var pv *PromptVersion

	_, err := pv.Format(map[string]string{})
	if err == nil {
		t.Error("expected error for nil PromptVersion")
	}
}

func TestPromptVersion_FormatAsText_Success(t *testing.T) {
	pv := &PromptVersion{
		Name:     "test",
		Template: "Hello, {{name}}!",
	}

	result, err := pv.FormatAsText(map[string]string{"name": "Alice"})
	if err != nil {
		t.Fatalf("FormatAsText() error = %v", err)
	}

	if result != "Hello, Alice!" {
		t.Errorf("result = %q, want %q", result, "Hello, Alice!")
	}
}

func TestPromptVersion_FormatAsText_ChatPromptError(t *testing.T) {
	pv := &PromptVersion{
		Name: "test",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	_, err := pv.FormatAsText(map[string]string{})
	if err == nil {
		t.Error("expected error for chat prompt")
	}
}

func TestPromptVersion_FormatAsMessages_Success(t *testing.T) {
	pv := &PromptVersion{
		Name: "test",
		Messages: []ChatMessage{
			{Role: "system", Content: "You are {{role}}."},
			{Role: "user", Content: "Hi!"},
		},
	}

	result, err := pv.FormatAsMessages(map[string]string{"role": "a helpful assistant"})
	if err != nil {
		t.Fatalf("FormatAsMessages() error = %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("got %d messages, want 2", len(result))
	}
	if result[0].Content != "You are a helpful assistant." {
		t.Errorf("result[0].Content = %q", result[0].Content)
	}
	if result[0].Role != "system" {
		t.Errorf("result[0].Role = %q, want %q", result[0].Role, "system")
	}
}

func TestPromptVersion_FormatAsMessages_TextPromptError(t *testing.T) {
	pv := &PromptVersion{
		Name:     "test",
		Template: "Hello",
	}

	_, err := pv.FormatAsMessages(map[string]string{})
	if err == nil {
		t.Error("expected error for text prompt")
	}
}

func TestPromptVersion_IsChat(t *testing.T) {
	textPrompt := &PromptVersion{
		Name:     "text",
		Template: "Hello",
	}

	chatPrompt := &PromptVersion{
		Name: "chat",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	if textPrompt.IsChat() {
		t.Error("text prompt should not be chat")
	}
	if !chatPrompt.IsChat() {
		t.Error("chat prompt should be chat")
	}
}

func TestSubstituteVars_MultipleOccurrences(t *testing.T) {
	template := "{{name}} and {{name}} are the same"
	result, err := substituteVars(template, map[string]string{"name": "Alice"})
	if err != nil {
		t.Fatalf("substituteVars() error = %v", err)
	}

	expected := "Alice and Alice are the same"
	if result != expected {
		t.Errorf("result = %q, want %q", result, expected)
	}
}

func TestSubstituteVars_PartialMatch(t *testing.T) {
	template := "Hello {{name}}, your id is {{id}}"
	_, err := substituteVars(template, map[string]string{"name": "Alice"})
	if err == nil {
		t.Error("expected error for missing id variable")
	}
}
