package transport

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestFormatAuthHeader_BearerToken(t *testing.T) {
	got := FormatAuthHeader("my-secret-token")
	want := "Bearer my-secret-token"
	if got != want {
		t.Errorf("FormatAuthHeader() = %q, want %q", got, want)
	}
}

func TestFormatAuthHeader_BasicAuth(t *testing.T) {
	got := FormatAuthHeader("user:pass")
	want := "Basic dXNlcjpwYXNz"
	if got != want {
		t.Errorf("FormatAuthHeader() = %q, want %q", got, want)
	}
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse(%q): %v", raw, err)
	}
	return u
}

func TestTokenRoundTripper(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{
		Transport: NewTokenRoundTripper(http.DefaultTransport, "my-token", mustParseURL(t, server.URL)),
	}

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	resp.Body.Close()

	if receivedAuth != "Bearer my-token" {
		t.Errorf("Authorization = %q, want %q", receivedAuth, "Bearer my-token")
	}
}

func TestTokenFileRoundTripper(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenFile, []byte("file-token\n"), 0600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{
		Transport: NewTokenFileRoundTripper(http.DefaultTransport, tokenFile, mustParseURL(t, server.URL)),
	}

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	resp.Body.Close()

	if receivedAuth != "Bearer file-token" {
		t.Errorf("Authorization = %q, want %q", receivedAuth, "Bearer file-token")
	}
}

func TestTokenFileRoundTripper_Rotation(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenFile, []byte("token-v1"), 0600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{
		Transport: NewTokenFileRoundTripper(http.DefaultTransport, tokenFile, mustParseURL(t, server.URL)),
	}

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("first request error: %v", err)
	}
	resp.Body.Close()
	if receivedAuth != "Bearer token-v1" {
		t.Errorf("first request Authorization = %q, want %q", receivedAuth, "Bearer token-v1")
	}

	err = os.WriteFile(tokenFile, []byte("token-v2"), 0600)
	if err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	resp, err = client.Get(server.URL)
	if err != nil {
		t.Fatalf("second request error: %v", err)
	}
	resp.Body.Close()
	if receivedAuth != "Bearer token-v2" {
		t.Errorf("second request Authorization = %q, want %q", receivedAuth, "Bearer token-v2")
	}
}

func TestTokenFileRoundTripper_MissingFile(t *testing.T) {
	target := mustParseURL(t, "http://localhost:1")
	client := &http.Client{
		Transport: NewTokenFileRoundTripper(http.DefaultTransport, "/nonexistent/token", target),
	}

	_, err := client.Get("http://localhost:1") //nolint:noctx // test helper
	if err == nil {
		t.Error("expected error for missing token file")
	}
}

func TestTokenFileRoundTripper_EmptyFile(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenFile, []byte("  \n"), 0600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{
		Transport: NewTokenFileRoundTripper(http.DefaultTransport, tokenFile, mustParseURL(t, server.URL)),
	}

	_, err := client.Get(server.URL)
	if err == nil {
		t.Error("expected error for empty token file")
	}
}

func TestTokenRoundTripper_NoAuthOnRedirectToForeignHost(t *testing.T) {
	var foreignAuth string
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		foreignAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer foreign.Close()

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, foreign.URL+"/callback", http.StatusTemporaryRedirect)
	}))
	defer origin.Close()

	client := &http.Client{
		Transport: NewTokenRoundTripper(http.DefaultTransport, "secret", mustParseURL(t, origin.URL)),
	}

	resp, err := client.Get(origin.URL + "/start")
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	resp.Body.Close()

	if foreignAuth != "" {
		t.Errorf("foreign host received Authorization = %q, want empty", foreignAuth)
	}
}

func TestTokenFileRoundTripper_NoAuthOnRedirectToForeignHost(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenFile, []byte("secret"), 0600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	var foreignAuth string
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		foreignAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer foreign.Close()

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, foreign.URL+"/callback", http.StatusTemporaryRedirect)
	}))
	defer origin.Close()

	client := &http.Client{
		Transport: NewTokenFileRoundTripper(http.DefaultTransport, tokenFile, mustParseURL(t, origin.URL)),
	}

	resp, err := client.Get(origin.URL + "/start")
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	resp.Body.Close()

	if foreignAuth != "" {
		t.Errorf("foreign host received Authorization = %q, want empty", foreignAuth)
	}
}

func TestTokenRoundTripper_NilHeaderRequest(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	serverURL := mustParseURL(t, server.URL)
	rt := NewTokenRoundTripper(http.DefaultTransport, "nil-hdr-token", serverURL)

	req := &http.Request{
		Method: http.MethodGet,
		URL:    mustParseURL(t, server.URL+"/test"),
	}

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip error: %v", err)
	}
	resp.Body.Close()

	if gotAuth != "Bearer nil-hdr-token" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer nil-hdr-token")
	}
}

func TestTokenFileRoundTripper_NilHeaderRequest(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenFile, []byte("nil-hdr-file-token"), 0600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	serverURL := mustParseURL(t, server.URL)
	rt := NewTokenFileRoundTripper(http.DefaultTransport, tokenFile, serverURL)

	req := &http.Request{
		Method: http.MethodGet,
		URL:    mustParseURL(t, server.URL+"/test"),
	}

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip error: %v", err)
	}
	resp.Body.Close()

	if gotAuth != "Bearer nil-hdr-file-token" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer nil-hdr-file-token")
	}
}

func TestTokenRoundTripper_StripsAuthOnForeignOrigin(t *testing.T) {
	var gotAuth string
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer foreign.Close()

	trackingOrigin := mustParseURL(t, "https://tracking.example.com")
	rt := NewTokenRoundTripper(http.DefaultTransport, "secret", trackingOrigin)

	req, _ := http.NewRequest(http.MethodGet, foreign.URL+"/callback", nil)
	req.Header.Set("Authorization", "Bearer stale-token")

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip error: %v", err)
	}
	resp.Body.Close()

	if gotAuth != "" {
		t.Errorf("foreign host received Authorization = %q, want empty", gotAuth)
	}
}

func TestTokenFileRoundTripper_StripsAuthOnForeignOrigin(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenFile, []byte("secret"), 0600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	var gotAuth string
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer foreign.Close()

	trackingOrigin := mustParseURL(t, "https://tracking.example.com")
	rt := NewTokenFileRoundTripper(http.DefaultTransport, tokenFile, trackingOrigin)

	req, _ := http.NewRequest(http.MethodGet, foreign.URL+"/callback", nil)
	req.Header.Set("Authorization", "Bearer stale-token")

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip error: %v", err)
	}
	resp.Body.Close()

	if gotAuth != "" {
		t.Errorf("foreign host received Authorization = %q, want empty", gotAuth)
	}
}
