// ABOUTME: Sample application demonstrating the MLflow Go SDK.
// ABOUTME: Exercises RegisterPrompt, LoadPrompt, ListPrompts, and ListPromptVersions.

package main

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"time"

	"github.com/ederign/mlflow-go/mlflow"
)

func main() {
	ctx := context.Background()

	// Create client connecting to local MLflow server
	client, err := mlflow.NewClient(
		mlflow.WithTrackingURI("http://localhost:5000"),
		mlflow.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	fmt.Printf("Connected to MLflow at %s\n\n", client.TrackingURI())

	// Generate a unique prompt name for this run
	promptName := fmt.Sprintf("bella-dora-walks-%d", rand.IntN(10000))

	// === Path 1: RegisterPrompt - Create a new prompt ===
	fmt.Println("=== 1. RegisterPrompt: Creating a new prompt ===")
	prompt1, err := client.RegisterPrompt(ctx, promptName,
		"Time to walk Bella and Dora! Meeting at {{location}} at {{time}}.",
		mlflow.WithDescription("Basic walk reminder for Bella and Dora"),
		mlflow.WithTags(map[string]string{"author": "sample-app", "dogs": "bella,dora"}),
	)
	if err != nil {
		log.Fatalf("Failed to register prompt: %v", err)
	}
	printPrompt(prompt1)

	// Create a second version to demonstrate versioning
	fmt.Println("\n=== 1b. RegisterPrompt: Creating version 2 ===")
	prompt2, err := client.RegisterPrompt(ctx, promptName,
		"Hey {{owner}}! Bella and Dora are ready for their walk at {{time}}. Don't forget the treats!",
		mlflow.WithDescription("Added owner name and treats reminder"),
	)
	if err != nil {
		log.Fatalf("Failed to register prompt v2: %v", err)
	}
	printPrompt(prompt2)

	// === Path 2: LoadPrompt - Load the latest version ===
	fmt.Println("\n=== 2. LoadPrompt: Loading latest version ===")
	latestPrompt, err := client.LoadPrompt(ctx, promptName)
	if err != nil {
		log.Fatalf("Failed to load latest prompt: %v", err)
	}
	printPrompt(latestPrompt)

	// === Path 3: LoadPrompt with WithVersion - Load specific version ===
	fmt.Println("\n=== 3. LoadPrompt with WithVersion: Loading version 1 ===")
	v1Prompt, err := client.LoadPrompt(ctx, promptName, mlflow.WithVersion(1))
	if err != nil {
		log.Fatalf("Failed to load prompt version 1: %v", err)
	}
	printPrompt(v1Prompt)

	// === Path 4: ListPrompts - List all prompts ===
	fmt.Println("\n=== 4. ListPrompts: Listing all prompts ===")
	promptList, err := client.ListPrompts(ctx, mlflow.WithMaxResults(5))
	if err != nil {
		log.Fatalf("Failed to list prompts: %v", err)
	}
	fmt.Printf("  Found %d prompts:\n", len(promptList.Prompts))
	for _, info := range promptList.Prompts {
		fmt.Printf("    - %s (latest: v%d)\n", info.Name, info.LatestVersion)
	}
	if promptList.NextPageToken != "" {
		fmt.Printf("  (more prompts available, next page token: %s...)\n", promptList.NextPageToken[:20])
	}

	// === Path 5: ListPromptVersions - List versions of our prompt ===
	fmt.Println("\n=== 5. ListPromptVersions: Listing versions of our prompt ===")
	versionList, err := client.ListPromptVersions(ctx, promptName)
	if err != nil {
		log.Fatalf("Failed to list prompt versions: %v", err)
	}
	fmt.Printf("  Found %d versions of %s:\n", len(versionList.Versions), promptName)
	for _, v := range versionList.Versions {
		fmt.Printf("    - v%d: %s\n", v.Version, v.Description)
	}

	fmt.Println("\n=== All operations completed successfully! ===")
}

func printPrompt(p *mlflow.Prompt) {
	fmt.Printf("  Name:        %s\n", p.Name)
	fmt.Printf("  Version:     %d\n", p.Version)
	fmt.Printf("  Template:    %s\n", p.Template)
	fmt.Printf("  Description: %s\n", p.Description)
	if len(p.Tags) > 0 {
		fmt.Printf("  Tags:        %v\n", p.Tags)
	}
	if !p.CreatedAt.IsZero() {
		fmt.Printf("  Created:     %s\n", p.CreatedAt.Format(time.RFC3339))
	}
}

