package transport

import (
	"fmt"
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

// A caller-supplied Authorization header (e.g. one the tracking server told us to
// forward to a presigned object-store URL) must survive on a direct foreign-origin
// request. The round-tripper must not strip it. Regression test for issue #33.
func TestTokenRoundTripper_PreservesCallerAuthOnForeignOrigin(t *testing.T) {
	var gotAuth string
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer foreign.Close()

	trackingOrigin := mustParseURL(t, "https://tracking.example.com")
	rt := NewTokenRoundTripper(http.DefaultTransport, "secret", trackingOrigin)

	req, _ := http.NewRequest(http.MethodGet, foreign.URL+"/callback", nil)
	req.Header.Set("Authorization", "SharedKey acct:sig")

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip error: %v", err)
	}
	resp.Body.Close()

	if gotAuth != "SharedKey acct:sig" {
		t.Errorf("foreign host received Authorization = %q, want %q", gotAuth, "SharedKey acct:sig")
	}
}

func TestTokenFileRoundTripper_PreservesCallerAuthOnForeignOrigin(t *testing.T) {
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
	req.Header.Set("Authorization", "SharedKey acct:sig")

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip error: %v", err)
	}
	resp.Body.Close()

	if gotAuth != "SharedKey acct:sig" {
		t.Errorf("foreign host received Authorization = %q, want %q", gotAuth, "SharedKey acct:sig")
	}
}

// stripAuthOnCrossOriginRedirect must remove a caller-supplied Authorization
// header whenever a redirect crosses an exact-origin boundary. net/http alone
// does not cover the subdomain, different-port, or different-scheme cases
// because it compares hostnames only and ignores scheme and port.
func TestStripAuthOnCrossOriginRedirect(t *testing.T) {
	newReq := func(rawURL string) *http.Request {
		req, err := http.NewRequest(http.MethodGet, rawURL, nil)
		if err != nil {
			t.Fatalf("http.NewRequest(%q): %v", rawURL, err)
		}
		req.Header.Set("Authorization", "SharedKey acct:sig")
		return req
	}

	tests := []struct {
		name     string
		from     string
		to       string
		wantAuth string
	}{
		{"same exact origin keeps auth", "https://tracking.example.com/a", "https://tracking.example.com/b", "SharedKey acct:sig"},
		{"subdomain strips auth", "https://tracking.example.com/a", "https://sub.tracking.example.com/b", ""},
		{"different port strips auth", "https://tracking.example.com:443/a", "https://tracking.example.com:8443/b", ""},
		{"different scheme strips auth", "https://tracking.example.com/a", "http://tracking.example.com/b", ""},
		{"foreign host strips auth", "https://tracking.example.com/a", "https://evil.example/b", ""},
	}

	check := stripAuthOnCrossOriginRedirect(nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newReq(tt.to)
			if err := check(req, []*http.Request{newReq(tt.from)}); err != nil {
				t.Fatalf("CheckRedirect error: %v", err)
			}
			if got := req.Header.Get("Authorization"); got != tt.wantAuth {
				t.Errorf("Authorization = %q, want %q", got, tt.wantAuth)
			}
		})
	}
}

func TestStripAuthOnCrossOriginRedirect_DelegatesToPrev(t *testing.T) {
	sentinel := fmt.Errorf("prev decided")
	called := false
	check := stripAuthOnCrossOriginRedirect(func(_ *http.Request, _ []*http.Request) error {
		called = true
		return sentinel
	})

	req, _ := http.NewRequest(http.MethodGet, "https://tracking.example.com/a", nil)
	via, _ := http.NewRequest(http.MethodGet, "https://tracking.example.com/b", nil)

	if err := check(req, []*http.Request{via}); err != sentinel {
		t.Errorf("error = %v, want sentinel", err)
	}
	if !called {
		t.Error("prev CheckRedirect was not invoked")
	}
}

func TestStripAuthOnCrossOriginRedirect_DefaultCap(t *testing.T) {
	check := stripAuthOnCrossOriginRedirect(nil)
	req, _ := http.NewRequest(http.MethodGet, "https://tracking.example.com/a", nil)

	via := make([]*http.Request, 10)
	for i := range via {
		via[i] = req
	}
	if err := check(req, via); err == nil {
		t.Error("expected error after 10 redirects, got nil")
	}
	if err := check(req, via[:9]); err != nil {
		t.Errorf("unexpected error at 9 redirects: %v", err)
	}
}

// End-to-end: the tracking server redirects to the same host on a different port
// (a different exact origin that net/http would NOT strip on its own). A
// caller-supplied Authorization must not reach the redirect target.
func TestWrapClientWithAuth_StripsCallerAuthOnCrossOriginRedirect_Token(t *testing.T) {
	assertCallerAuthStrippedOnCrossOriginRedirect(t, func(trackingURL *url.URL) *http.Client {
		return wrapClientWithAuth(&http.Client{}, "tracking-token", "", trackingURL)
	})
}

func TestWrapClientWithAuth_StripsCallerAuthOnCrossOriginRedirect_TokenFile(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenFile, []byte("tracking-token"), 0600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
	assertCallerAuthStrippedOnCrossOriginRedirect(t, func(trackingURL *url.URL) *http.Client {
		return wrapClientWithAuth(&http.Client{}, "", tokenFile, trackingURL)
	})
}

func assertCallerAuthStrippedOnCrossOriginRedirect(t *testing.T, newClient func(trackingURL *url.URL) *http.Client) {
	t.Helper()

	var dstAuth string
	dst := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dstAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer dst.Close()

	// Same host (127.0.0.1), different port: net/http would carry the header
	// across; our CheckRedirect must strip it.
	tracking := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, dst.URL+"/callback", http.StatusTemporaryRedirect)
	}))
	defer tracking.Close()

	client := newClient(mustParseURL(t, tracking.URL))
	req, err := http.NewRequest(http.MethodGet, tracking.URL+"/start", nil)
	if err != nil {
		t.Fatalf("http.NewRequest error: %v", err)
	}
	req.Header.Set("Authorization", "SharedKey acct:sig")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	resp.Body.Close()

	if dstAuth != "" {
		t.Errorf("redirect target received Authorization = %q, want empty", dstAuth)
	}
}
