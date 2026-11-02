package workspace_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/opendatahub-io/mlflow-go/mlflow"
)

// This middleware forwards requests unchanged and injects no credentials.
type transparentTransport struct {
	http.RoundTripper
}

func TestEnsureWorkspaceCustomTransportRecovery(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusConflict} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()
			var lookups atomic.Int32
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodPost:
					w.WriteHeader(status)
					fmt.Fprint(w, `{"error_code":"RESOURCE_ALREADY_EXISTS","message":"workspace exists"}`)
				case http.MethodGet:
					lookups.Add(1)
					fmt.Fprint(w, `{"workspace":{"name":"existing-ws"}}`)
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
			}))
			defer server.Close()
			httpClient := server.Client()
			httpClient.Transport = transparentTransport{RoundTripper: httpClient.Transport}
			client, err := mlflow.NewClient(mlflow.WithTrackingURI(server.URL), mlflow.WithToken(""), mlflow.WithHTTPClient(httpClient))
			if err != nil {
				t.Fatal(err)
			}
			workspace, err := client.Workspaces().EnsureWorkspace(context.Background(), "existing-ws")
			if err != nil || workspace == nil || workspace.Name != "existing-ws" || lookups.Load() != 1 {
				t.Fatalf("expected already-exists recovery: workspace=%v err=%v lookups=%d", workspace, err, lookups.Load())
			}
		})
	}
}
