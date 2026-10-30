package workspace_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/opendatahub-io/mlflow-go/mlflow"
)

func TestWorkspaceLookupAndCreationErrors(t *testing.T) {
	const notFoundMessage = "Workspace 'new-project' not found in the Kubernetes cluster. Each MLflow workspace maps 1:1 to a namespace."
	const creationMessage = "Namespace creation is not supported by this provider"
	var callsMu sync.Mutex
	var calls []string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callsMu.Lock()
		calls = append(calls, r.Method+" "+r.URL.Path)
		callsMu.Unlock()
		if got := r.Header.Get("X-MLFLOW-WORKSPACE"); got != "" {
			t.Errorf("workspace management request carried workspace header %q", got)
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/mlflow/api/3.0/mlflow/workspaces/new-project":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error_code":"RESOURCE_DOES_NOT_EXIST","message":%q}`, notFoundMessage)
		case r.Method == http.MethodPost && r.URL.Path == "/mlflow/api/3.0/mlflow/workspaces":
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintf(w, `{"detail":%q}`, creationMessage)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()
	client, err := mlflow.NewClient(
		mlflow.WithTrackingURI(server.URL+"/mlflow"),
		mlflow.WithHTTPClient(server.Client()),
		mlflow.WithToken(""),
		mlflow.WithWorkspace("new-project"),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, getErr := client.Workspaces().GetWorkspace(context.Background(), "new-project")
	var getAPIError *mlflow.APIError
	if !errors.As(getErr, &getAPIError) || !mlflow.IsNotFound(getErr) {
		t.Fatalf("expected typed not-found error, got %v", getErr)
	}
	if getAPIError.Code != "RESOURCE_DOES_NOT_EXIST" || getAPIError.Message != notFoundMessage {
		t.Fatalf("GET lost server details: %#v", getAPIError)
	}
	if !errors.Is(getErr, getAPIError) {
		t.Fatal("GET error lost APIError identity")
	}
	// Simulate the caller deciding to create after a 404. The SDK itself must
	// not make that decision or infer provider capabilities from the 403.
	_, createErr := client.Workspaces().CreateWorkspace(context.Background(), "new-project")
	assertForbidden := func(err error) {
		t.Helper()
		var apiErr *mlflow.APIError
		if !errors.As(err, &apiErr) || !mlflow.IsPermissionDenied(err) {
			t.Fatalf("expected typed forbidden error, got %v", err)
		}
		if apiErr.StatusCode != 403 || apiErr.Code != "" || !strings.Contains(apiErr.Message, creationMessage) {
			t.Fatalf("POST lost server details: %#v", apiErr)
		}
		if !errors.Is(err, apiErr) || mlflow.IsAlreadyExists(err) || mlflow.IsNotFound(err) {
			t.Fatalf("POST error identity/classification changed: %v", err)
		}
	}
	assertForbidden(createErr)
	_, ensureErr := client.Workspaces().EnsureWorkspace(context.Background(), "new-project")
	assertForbidden(ensureErr)
	callsMu.Lock()
	defer callsMu.Unlock()
	if want := []string{
		"GET /mlflow/api/3.0/mlflow/workspaces/new-project",
		"POST /mlflow/api/3.0/mlflow/workspaces",
		"POST /mlflow/api/3.0/mlflow/workspaces",
	}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("requests = %v, want %v (no retries or fallback after 403)", calls, want)
	}
}
