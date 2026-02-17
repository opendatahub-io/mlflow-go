package main

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"time"

	"github.com/opendatahub-io/mlflow-go/mlflow"
	"github.com/opendatahub-io/mlflow-go/mlflow/promptregistry"
)

func main() {
	ctx := context.Background()

	if os.Getenv("MLFLOW_DEMO_WORKSPACES") == "true" {
		// Workspace isolation demo (requires midstream server)
		// Run with: make run-sample-workspaces
		fmt.Println("=== Workspace Isolation Demo ===")
		runWorkspaceDemo(ctx)
		fmt.Println("\n=== Workspace demo completed successfully! ===")
	} else {
		// Prompt lifecycle demo
		// Run with: make run-sample
		client, err := mlflow.NewClient(mlflow.WithInsecure())
		if err != nil {
			log.Fatalf("Failed to create client: %v", err)
		}
		fmt.Printf("Connected to MLflow at %s\n\n", client.TrackingURI())

		runPromptDemo(ctx, client)
		fmt.Println("\n=== All operations completed successfully! ===")
	}
}

// runPromptDemo demonstrates the full prompt lifecycle: register, load, version,
// alias, format, chat prompts, and delete operations.
func runPromptDemo(ctx context.Context, client *mlflow.Client) {
	promptName := fmt.Sprintf("bella-dora-walks-%d", rand.IntN(10000))
	chatPromptName := fmt.Sprintf("bella-dora-assistant-chat-%d", rand.IntN(10000))

	// === 1. Register text prompts ===
	fmt.Println("=== 1. RegisterPrompt: Creating a new prompt ===")
	prompt1, err := client.PromptRegistry().RegisterPrompt(ctx, promptName,
		"Time to walk Bella and Dora! Meeting at {{location}} at {{time}}.",
		promptregistry.WithCommitMessage("Basic walk reminder for Bella and Dora"),
		promptregistry.WithTags(map[string]string{"author": "sample-app", "dogs": "bella,dora"}),
	)
	if err != nil {
		log.Fatalf("Failed to register prompt: %v", err)
	}
	printPromptVersion(prompt1)

	fmt.Println("\n=== 1b. RegisterPrompt: Creating version 2 ===")
	prompt2, err := client.PromptRegistry().RegisterPrompt(ctx, promptName,
		"Hey {{owner}}! Bella and Dora are ready for their walk at {{time}}. Don't forget the treats!",
		promptregistry.WithCommitMessage("Added owner name and treats reminder"),
	)
	if err != nil {
		log.Fatalf("Failed to register prompt v2: %v", err)
	}
	printPromptVersion(prompt2)

	// === 2. Load latest version ===
	fmt.Println("\n=== 2. LoadPrompt: Loading latest version (uses @latest alias) ===")
	latestPrompt, err := client.PromptRegistry().LoadPrompt(ctx, promptName)
	if err != nil {
		log.Fatalf("Failed to load latest prompt: %v", err)
	}
	fmt.Printf("  Loaded version %d (latest)\n", latestPrompt.Version)
	printPromptVersion(latestPrompt)

	// === 3. Load specific version ===
	fmt.Println("\n=== 3. LoadPrompt with WithVersion: Loading version 1 ===")
	v1Prompt, err := client.PromptRegistry().LoadPrompt(ctx, promptName, promptregistry.WithVersion(1))
	if err != nil {
		log.Fatalf("Failed to load prompt version 1: %v", err)
	}
	printPromptVersion(v1Prompt)

	// === 3b. Aliases ===
	fmt.Println("\n=== 3b. SetPromptAlias: Creating 'production' and 'staging' aliases ===")
	err = client.PromptRegistry().SetPromptAlias(ctx, promptName, "production", 1)
	if err != nil {
		log.Fatalf("Failed to set production alias: %v", err)
	}
	fmt.Printf("  Set 'production' alias -> v1\n")

	err = client.PromptRegistry().SetPromptAlias(ctx, promptName, "staging", 2)
	if err != nil {
		log.Fatalf("Failed to set staging alias: %v", err)
	}
	fmt.Printf("  Set 'staging' alias -> v2\n")

	// === 3c. Load by alias ===
	fmt.Println("\n=== 3c. LoadPrompt with WithAlias: Loading 'production' alias ===")
	prodPrompt, err := client.PromptRegistry().LoadPrompt(ctx, promptName, promptregistry.WithAlias("production"))
	if err != nil {
		log.Fatalf("Failed to load production alias: %v", err)
	}
	printPromptVersion(prodPrompt)
	fmt.Printf("  Aliases:     %v\n", prodPrompt.Aliases)

	// === 3d. Format text prompt ===
	fmt.Println("\n=== 3d. FormatAsText: Formatting text prompt with variables ===")
	formattedText, err := prodPrompt.FormatAsText(map[string]string{
		"location": "Central Park",
		"time":     "3pm",
	})
	if err != nil {
		log.Fatalf("Failed to format prompt: %v", err)
	}
	fmt.Printf("  Formatted:   %s\n", formattedText)

	// === 3e. Delete alias ===
	fmt.Println("\n=== 3e. DeletePromptAlias: Removing 'staging' alias ===")
	err = client.PromptRegistry().DeletePromptAlias(ctx, promptName, "staging")
	if err != nil {
		log.Fatalf("Failed to delete staging alias: %v", err)
	}
	fmt.Printf("  Deleted 'staging' alias\n")

	// === 4. List prompts ===
	fmt.Println("\n=== 4. ListPrompts: Listing all prompts ===")
	promptList, err := client.PromptRegistry().ListPrompts(ctx, promptregistry.WithMaxResults(5))
	if err != nil {
		log.Fatalf("Failed to list prompts: %v", err)
	}
	fmt.Printf("  Found %d prompts:\n", len(promptList.Prompts))
	for _, info := range promptList.Prompts {
		fmt.Printf("    - %s (latest: v%d)\n", info.Name, info.LatestVersion)
	}
	if promptList.NextPageToken != "" {
		fmt.Println("  (more prompts available)")
	}

	// === 5. List prompt versions ===
	fmt.Println("\n=== 5. ListPromptVersions: Listing versions from a listed prompt ===")
	if len(promptList.Prompts) > 0 {
		selectedPrompt := promptList.Prompts[0]
		fmt.Printf("  Selected prompt: %s\n", selectedPrompt.Name)

		versionList, err := client.PromptRegistry().ListPromptVersions(ctx, selectedPrompt.Name)
		if err != nil {
			log.Fatalf("Failed to list prompt versions: %v", err)
		}
		fmt.Printf("  Found %d versions:\n", len(versionList.Versions))
		for _, v := range versionList.Versions {
			fmt.Printf("    - v%d: %s\n", v.Version, v.CommitMessage)
		}
	} else {
		fmt.Println("  No prompts found to list versions for")
	}

	// === 6. Register chat prompt ===
	fmt.Println("\n=== 6. RegisterChatPrompt: Creating a chat prompt ===")
	messages := []promptregistry.ChatMessage{
		{Role: "system", Content: "You are a helpful dog walking assistant for {{owner}}. You help schedule walks for Bella and Dora."},
		{Role: "user", Content: "When should I walk {{dog_name}} today? The weather is {{weather}}."},
	}

	temp := 0.7
	modelConfig := &promptregistry.PromptModelConfig{
		Provider:    "openai",
		ModelName:   "gpt-4",
		Temperature: &temp,
	}

	chatPrompt, err := client.PromptRegistry().RegisterChatPrompt(ctx, chatPromptName,
		messages,
		promptregistry.WithCommitMessage("Chat assistant for Bella and Dora walks"),
		promptregistry.WithModelConfig(modelConfig),
		promptregistry.WithTags(map[string]string{"type": "assistant", "dogs": "bella,dora"}),
	)
	if err != nil {
		log.Fatalf("Failed to register chat prompt: %v", err)
	}
	printPromptVersion(chatPrompt)

	// === 7. Format chat prompt ===
	fmt.Println("\n=== 7. FormatAsMessages: Formatting the chat prompt ===")
	loadedChatPrompt, err := client.PromptRegistry().LoadPrompt(ctx, chatPromptName)
	if err != nil {
		log.Fatalf("Failed to load chat prompt: %v", err)
	}

	formattedMessages, err := loadedChatPrompt.FormatAsMessages(map[string]string{
		"owner":    "Alice",
		"dog_name": "Bella",
		"weather":  "sunny and 72°F",
	})
	if err != nil {
		log.Fatalf("Failed to format chat prompt: %v", err)
	}

	fmt.Println("  Formatted messages:")
	for i, msg := range formattedMessages {
		fmt.Printf("    [%d] %s: %s\n", i, msg.Role, msg.Content)
	}

	if loadedChatPrompt.ModelConfig != nil {
		fmt.Println("  Model config:")
		fmt.Printf("    Provider: %s\n", loadedChatPrompt.ModelConfig.Provider)
		fmt.Printf("    Model:    %s\n", loadedChatPrompt.ModelConfig.ModelName)
		if loadedChatPrompt.ModelConfig.Temperature != nil {
			fmt.Printf("    Temp:     %.1f\n", *loadedChatPrompt.ModelConfig.Temperature)
		}
	}

	// === 8. Delete operations ===
	fmt.Println("\n=== 8. Delete Operations: Cleaning up prompts ===")
	deleteDemo(ctx, client, promptName, chatPromptName)
}

// deleteDemo demonstrates delete operations for tags, versions, aliases, and prompts.
func deleteDemo(ctx context.Context, client *mlflow.Client, promptName, chatPromptName string) {
	// Delete version tags from the chat prompt
	fmt.Println("\n=== 8a. DeletePromptVersionTag: Removing 'type' tag from version 1 ===")
	err := client.PromptRegistry().DeletePromptVersionTag(ctx, chatPromptName, 1, "type")
	if err != nil {
		log.Fatalf("Failed to delete prompt version tag: %v", err)
	}
	fmt.Printf("  Deleted 'type' tag from %s v1\n", chatPromptName)

	fmt.Println("\n=== 8b. DeletePromptVersionTag: Removing 'dogs' tag from version 1 ===")
	err = client.PromptRegistry().DeletePromptVersionTag(ctx, chatPromptName, 1, "dogs")
	if err != nil {
		log.Fatalf("Failed to delete version tag: %v", err)
	}
	fmt.Printf("  Deleted 'dogs' tag from %s v1\n", chatPromptName)

	// Delete the chat prompt version and prompt
	fmt.Println("\n=== 8c. DeletePromptVersion: Deleting chat prompt version ===")
	err = client.PromptRegistry().DeletePromptVersion(ctx, chatPromptName, 1)
	if err != nil {
		log.Fatalf("Failed to delete prompt version: %v", err)
	}
	fmt.Printf("  Deleted version 1 of %s\n", chatPromptName)

	_, err = client.PromptRegistry().LoadPrompt(ctx, chatPromptName, promptregistry.WithVersion(1))
	if !mlflow.IsNotFound(err) {
		log.Fatalf("Expected NotFound error for deleted version, got: %v", err)
	}
	fmt.Printf("  Verified: loading deleted version returns NotFound\n")

	fmt.Println("\n=== 8d. DeletePrompt: Deleting chat prompt ===")
	err = client.PromptRegistry().DeletePrompt(ctx, chatPromptName)
	if err != nil {
		log.Fatalf("Failed to delete prompt: %v", err)
	}
	fmt.Printf("  Deleted prompt %s\n", chatPromptName)

	_, err = client.PromptRegistry().LoadPrompt(ctx, chatPromptName)
	if !mlflow.IsNotFound(err) {
		log.Fatalf("Expected NotFound error for deleted prompt, got: %v", err)
	}
	fmt.Printf("  Verified: loading deleted prompt returns NotFound\n")

	// Delete version with alias conflict handling
	fmt.Println("\n=== 8e. DeletePromptVersion with alias: Handling alias conflict ===")
	err = client.PromptRegistry().DeletePromptVersion(ctx, promptName, 1)
	if mlflow.IsAliasConflict(err) {
		fmt.Printf("  Cannot delete %s v1: alias 'production' points to it\n", promptName)
		fmt.Println("  Removing alias first...")
		err = client.PromptRegistry().DeletePromptAlias(ctx, promptName, "production")
		if err != nil {
			log.Fatalf("Failed to delete alias: %v", err)
		}
		fmt.Println("  Deleted 'production' alias")

		err = client.PromptRegistry().DeletePromptVersion(ctx, promptName, 1)
		if err != nil {
			log.Fatalf("Failed to delete version after removing alias: %v", err)
		}
		fmt.Printf("  Deleted version 1 of %s\n", promptName)
	} else if err != nil {
		log.Fatalf("Failed to delete prompt version: %v", err)
	} else {
		fmt.Printf("  Deleted version 1 of %s (no alias conflict)\n", promptName)
	}

	// Final cleanup
	fmt.Println("\n=== 8f. Cleanup: Deleting remaining versions and prompt ===")
	err = client.PromptRegistry().DeletePromptVersion(ctx, promptName, 2)
	if err != nil {
		log.Fatalf("Failed to delete version 2: %v", err)
	}
	fmt.Printf("  Deleted version 2 of %s\n", promptName)

	err = client.PromptRegistry().DeletePrompt(ctx, promptName)
	if err != nil {
		log.Fatalf("Failed to delete prompt: %v", err)
	}
	fmt.Printf("  Deleted prompt %s\n", promptName)

	_, err = client.PromptRegistry().LoadPrompt(ctx, promptName)
	if !mlflow.IsNotFound(err) {
		log.Fatalf("Expected NotFound error for deleted prompt, got: %v", err)
	}
	fmt.Printf("  Verified: loading deleted prompt returns NotFound\n")
}

// runWorkspaceDemo demonstrates workspace-based tenant isolation.
// Requires a midstream MLflow server with workspaces enabled.
func runWorkspaceDemo(ctx context.Context) {
	wsPromptName := fmt.Sprintf("ws-demo-%d", rand.IntN(10000))

	bella, err := mlflow.NewClient(
		mlflow.WithInsecure(),
		mlflow.WithHeaders(map[string]string{"X-MLFLOW-WORKSPACE": "team-bella"}),
	)
	if err != nil {
		log.Fatalf("Failed to create team-bella client: %v", err)
	}

	dora, err := mlflow.NewClient(
		mlflow.WithInsecure(),
		mlflow.WithHeaders(map[string]string{"X-MLFLOW-WORKSPACE": "team-dora"}),
	)
	if err != nil {
		log.Fatalf("Failed to create team-dora client: %v", err)
	}

	// Register a prompt in team-bella
	fmt.Println("  Registering prompt in team-bella workspace...")
	wsPrompt, err := bella.PromptRegistry().RegisterPrompt(ctx, wsPromptName,
		"Walk Bella at {{time}} in {{location}}!",
		promptregistry.WithCommitMessage("workspace demo"),
	)
	if err != nil {
		log.Fatalf("Failed to register in team-bella: %v", err)
	}
	fmt.Printf("  Created %s v%d in team-bella\n", wsPrompt.Name, wsPrompt.Version)

	// Load from team-bella (should succeed)
	fmt.Println("  Loading from team-bella (should succeed)...")
	loaded, err := bella.PromptRegistry().LoadPrompt(ctx, wsPromptName)
	if err != nil {
		log.Fatalf("Failed to load from team-bella: %v", err)
	}
	fmt.Printf("  Found in team-bella: %s v%d — %q\n", loaded.Name, loaded.Version, loaded.Template)

	// Try to load from team-dora (should fail — isolation)
	fmt.Println("  Loading from team-dora (should fail — isolation)...")
	_, err = dora.PromptRegistry().LoadPrompt(ctx, wsPromptName)
	if mlflow.IsNotFound(err) {
		fmt.Printf("  Confirmed: %s is NOT visible from team-dora\n", wsPromptName)
	} else if err != nil {
		log.Fatalf("Unexpected error from team-dora: %v", err)
	} else {
		log.Fatal("ERROR: prompt should not be visible from team-dora!")
	}

	// Cleanup
	_ = bella.PromptRegistry().DeletePromptVersion(ctx, wsPromptName, 1)
	_ = bella.PromptRegistry().DeletePrompt(ctx, wsPromptName)
	fmt.Println("  Workspace isolation demo complete!")
}

func printPromptVersion(pv *promptregistry.PromptVersion) {
	fmt.Printf("  Name:        %s\n", pv.Name)
	fmt.Printf("  Version:     %d\n", pv.Version)
	if pv.IsChat() {
		fmt.Printf("  Type:        chat (%d messages)\n", len(pv.Messages))
		for i, msg := range pv.Messages {
			content := msg.Content
			if len(content) > 60 {
				content = content[:60] + "..."
			}
			fmt.Printf("    [%d] %s: %s\n", i, msg.Role, content)
		}
	} else {
		fmt.Printf("  Type:        text\n")
		fmt.Printf("  Template:    %s\n", pv.Template)
	}
	fmt.Printf("  Commit:      %s\n", pv.CommitMessage)
	if pv.ModelConfig != nil {
		fmt.Printf("  Model:       %s/%s\n", pv.ModelConfig.Provider, pv.ModelConfig.ModelName)
	}
	if len(pv.Tags) > 0 {
		fmt.Printf("  Tags:        %v\n", pv.Tags)
	}
	if !pv.CreatedAt.IsZero() {
		fmt.Printf("  Created:     %s\n", pv.CreatedAt.Format(time.RFC3339))
	}
}
