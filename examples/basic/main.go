// ABOUTME: Basic example demonstrating the MLflow Prompt Registry SDK.
// ABOUTME: Shows loading, modifying, and registering prompts.

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ederign/mlflow-go/mlflow"
)

func main() {
	// Create client (reads from environment variables)
	// Required: MLFLOW_TRACKING_URI
	// Optional: MLFLOW_TRACKING_TOKEN, MLFLOW_INSECURE_SKIP_TLS_VERIFY
	client, err := mlflow.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Example 1: Load a prompt
	fmt.Println("=== Loading Prompt ===")
	prompt, err := client.LoadPrompt(ctx, "greeting-prompt")
	if err != nil {
		if mlflow.IsNotFound(err) {
			fmt.Println("Prompt not found, creating one...")
			prompt, err = createExamplePrompt(ctx, client)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatal(err)
		}
	}

	fmt.Printf("Loaded: %s v%d\n", prompt.Name, prompt.Version)
	fmt.Printf("Template: %s\n", prompt.Template)
	fmt.Printf("Description: %s\n", prompt.Description)
	fmt.Println()

	// Example 2: Load a specific version
	fmt.Println("=== Loading Specific Version ===")
	promptV1, err := client.LoadPrompt(ctx, prompt.Name, mlflow.WithVersion(1))
	if err != nil {
		log.Printf("Could not load version 1: %v", err)
	} else {
		fmt.Printf("Version 1 template: %s\n", promptV1.Template)
	}
	fmt.Println()

	// Example 3: Modify locally and register new version
	fmt.Println("=== Modifying and Registering ===")
	modified := prompt.
		WithTemplate("Hello {{name}}! Bella and Dora welcome you to {{company}}!").
		WithDescription("Added Bella and Dora")

	fmt.Printf("Local modification (original unchanged): %s\n", modified.Template)
	fmt.Printf("Original still: %s\n", prompt.Template)

	// Register the modification as a new version
	newVersion, err := client.RegisterPrompt(ctx, modified.Name, modified.Template,
		mlflow.WithDescription(modified.Description),
	)
	if err != nil {
		log.Printf("Could not register new version: %v", err)
	} else {
		fmt.Printf("Registered new version: v%d\n", newVersion.Version)
	}
}

func createExamplePrompt(ctx context.Context, client *mlflow.Client) (*mlflow.Prompt, error) {
	return client.RegisterPrompt(ctx, "greeting-prompt",
		"Hello {{name}}, welcome to {{company}}!",
		mlflow.WithDescription("Initial greeting template"),
		mlflow.WithTags(map[string]string{
			"author":   "example",
			"category": "onboarding",
		}),
	)
}

func init() {
	// Check required environment variable
	if os.Getenv("MLFLOW_TRACKING_URI") == "" {
		fmt.Println("Usage: MLFLOW_TRACKING_URI=http://localhost:5000 go run main.go")
		fmt.Println()
		fmt.Println("Start MLflow server first: make dev/up")
		fmt.Println()
		os.Exit(1)
	}
}
