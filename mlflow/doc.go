// ABOUTME: Package mlflow provides a Go SDK for the MLflow platform.
// ABOUTME: This is the main package containing the Client and Prompt types.

// Package mlflow provides a Go SDK for the MLflow platform.
//
// The SDK currently supports the MLflow Prompt Registry, with additional
// MLflow functionality planned for future releases.
//
// # Quick Start
//
// Create a client and load a prompt:
//
//	client, err := mlflow.NewClient()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	prompt, err := client.LoadPrompt(ctx, "my-prompt")
//	if err != nil {
//	    if mlflow.IsNotFound(err) {
//	        log.Fatal("prompt not found")
//	    }
//	    log.Fatal(err)
//	}
//
//	fmt.Println(prompt.Template)
//
// # Configuration
//
// The client reads configuration from environment variables by default:
//
//   - MLFLOW_TRACKING_URI: MLflow server URL (required)
//   - MLFLOW_TRACKING_TOKEN: Authentication token (optional)
//   - MLFLOW_INSECURE_SKIP_TLS_VERIFY: Allow HTTP connections (optional)
//
// Configuration can also be provided explicitly:
//
//	client, err := mlflow.NewClient(
//	    mlflow.WithTrackingURI("https://mlflow.example.com"),
//	    mlflow.WithToken("my-token"),
//	)
//
// # Error Handling
//
// All API errors are returned as typed errors that can be inspected:
//
//	if mlflow.IsNotFound(err) {
//	    // Handle 404
//	}
//	if mlflow.IsUnauthorized(err) {
//	    // Handle 401 - invalid token
//	}
//	if mlflow.IsPermissionDenied(err) {
//	    // Handle 403 - lacks permission
//	}
//
// # Thread Safety
//
// The Client is safe for concurrent use after construction.
// Prompt values are immutable; modification methods return copies.
package mlflow
