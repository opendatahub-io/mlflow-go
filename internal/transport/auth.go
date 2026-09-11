package transport

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// FormatAuthHeader returns the Authorization header value for a token.
// Tokens containing a colon are treated as user:password and encoded as Basic auth;
// all others are sent as Bearer tokens.
func FormatAuthHeader(token string) string {
	if strings.Contains(token, ":") {
		return "Basic " + base64.StdEncoding.EncodeToString([]byte(token))
	}
	return "Bearer " + token
}

// tokenRoundTripper injects a static Authorization header on requests
// whose origin (scheme + host) matches the configured tracking server.
type tokenRoundTripper struct {
	base      http.RoundTripper
	origin    string // "scheme://host" of the tracking server
	authValue string
}

// NewTokenRoundTripper wraps base to inject an Authorization header with the
// given token only for requests to trackingURL's origin.
func NewTokenRoundTripper(base http.RoundTripper, token string, trackingURL *url.URL) http.RoundTripper {
	return &tokenRoundTripper{
		base:      base,
		origin:    originFromURL(trackingURL),
		authValue: FormatAuthHeader(token),
	}
}

func (t *tokenRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	if requestOrigin(r) == t.origin {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("Authorization", t.authValue)
	} else {
		r.Header.Del("Authorization")
	}
	return t.base.RoundTrip(r)
}

// tokenFileRoundTripper re-reads a token file on every request to support
// Kubernetes projected service-account tokens that rotate. Credentials are
// only sent to the configured tracking server origin.
type tokenFileRoundTripper struct {
	base      http.RoundTripper
	origin    string // "scheme://host" of the tracking server
	tokenPath string
}

// NewTokenFileRoundTripper wraps base to read the token from path on every
// request, injecting it only for requests to trackingURL's origin.
func NewTokenFileRoundTripper(base http.RoundTripper, path string, trackingURL *url.URL) http.RoundTripper {
	return &tokenFileRoundTripper{
		base:      base,
		origin:    originFromURL(trackingURL),
		tokenPath: path,
	}
}

func (t *tokenFileRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if requestOrigin(req) != t.origin {
		r := req.Clone(req.Context())
		r.Header.Del("Authorization")
		return t.base.RoundTrip(r)
	}

	data, err := os.ReadFile(t.tokenPath)
	if err != nil {
		return nil, fmt.Errorf("mlflow: failed to read token file %q: %w", t.tokenPath, err)
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		return nil, fmt.Errorf("mlflow: token file %q is empty", t.tokenPath)
	}

	r := req.Clone(req.Context())
	if r.Header == nil {
		r.Header = make(http.Header)
	}
	r.Header.Set("Authorization", FormatAuthHeader(token))
	return t.base.RoundTrip(r)
}

func originFromURL(u *url.URL) string {
	return strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host)
}

func requestOrigin(r *http.Request) string {
	return strings.ToLower(r.URL.Scheme) + "://" + strings.ToLower(r.URL.Host)
}
