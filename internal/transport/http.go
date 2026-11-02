package transport

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/opendatahub-io/mlflow-go/internal/errors"
)

const maxResponseBodySize = 100 << 20 // 100 MiB

// defaultStreamHeaderTimeout is the fallback deadline for the response-header
// phase of a streaming transfer when the request context carries no deadline. It
// is deliberately generous: MLflow's mlflow-artifacts proxy fetches the whole
// remote object from the backing store before it sends any response header, so a
// large artifact can take well over the 30s API timeout to reach first byte. It
// only guards against a server that accepts the connection and never responds.
const defaultStreamHeaderTimeout = 5 * time.Minute

// Client handles HTTP communication with the MLflow API.
type Client struct {
	baseURL    *url.URL
	headers    map[string]string
	httpClient *http.Client // API calls; overall http.Client.Timeout applies
	// streamClient handles artifact uploads/downloads. It shares httpClient's
	// Transport (connection pool, dial and TLS handshake timeouts, auth and
	// workspace round-trippers) but drops the overall http.Client.Timeout, which
	// covers the entire exchange including body transfer and would otherwise abort
	// an arbitrarily large streaming upload/download mid-transfer. Streaming
	// transfers are bounded by the request context instead; the response-header
	// phase is separately bounded (the shared Transport may set no
	// ResponseHeaderTimeout) — for downloads in doRequestBody, and for uploads in
	// PutReader once the request body has been fully sent.
	streamClient *http.Client
	// streamHeaderTimeout bounds the response-header phase of a streaming transfer
	// (download, or upload once the request body has been sent) only when the
	// request context has no deadline. Zero disables the fallback (rely solely on
	// the context). It is independent of the API Timeout so a slow proxied
	// transfer is not cut off before the object is exchanged.
	streamHeaderTimeout time.Duration
	logger              *slog.Logger
}

// Config holds configuration for creating a transport Client.
type Config struct {
	BaseURL           string
	Headers           map[string]string
	HTTPClient        *http.Client
	Logger            *slog.Logger
	Timeout           time.Duration
	Insecure          bool
	Token             string
	TokenPath         string
	Workspace         string
	WorkspacesSupport bool
	// StreamHeaderTimeout bounds the response-header phase of a streaming
	// artifact transfer (download or upload) when the request context has no
	// deadline; for uploads the timer starts only after the request body has been
	// sent. It is independent of Timeout so a slow proxied transfer is not aborted
	// before the object is exchanged. Zero uses defaultStreamHeaderTimeout; a
	// negative value disables the fallback entirely (rely solely on the context).
	// Callers handling very large artifacts should supply a context deadline or a
	// negative value.
	StreamHeaderTimeout time.Duration
}

// New creates a new transport Client.
func New(cfg Config) (*Client, error) {
	baseURL, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	hasAuth := cfg.Token != "" || cfg.TokenPath != ""
	if cfg.Insecure && hasAuth {
		return nil, fmt.Errorf("refusing to send credentials over an insecure connection " +
			"(plain HTTP or TLS verification disabled); remove WithInsecure or the token/token-path option")
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		timeout := cfg.Timeout
		if timeout == 0 {
			timeout = 30 * time.Second
		}
		httpClient = &http.Client{Timeout: timeout}
		if cfg.Insecure {
			if dt, ok := http.DefaultTransport.(*http.Transport); ok {
				tr := dt.Clone()
				tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12, NextProtos: []string{"h2", "http/1.1"}} //nolint:gosec // user-requested via WithInsecure
				httpClient.Transport = tr
			} else {
				httpClient.Transport = &http.Transport{
					ForceAttemptHTTP2: true,
					TLSClientConfig:   &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12, NextProtos: []string{"h2", "http/1.1"}}, //nolint:gosec // user-requested via WithInsecure
				}
			}
		}
	}

	wrapped := wrapClientWithWorkspace(wrapClientWithAuth(httpClient, cfg.Token, cfg.TokenPath, baseURL), cfg.Workspace, cfg.BaseURL, cfg.WorkspacesSupport, cfg.Headers)

	// Derive the streaming client from wrapped so it shares the same Transport
	// (and thus connection pool, dial/TLS/response-header timeouts, and the auth
	// and workspace round-trippers), but without the overall Timeout that would
	// cap the duration of a large artifact transfer.
	streamClient := *wrapped
	streamClient.Timeout = 0

	// Resolve the streaming response-header fallback: 0 means "use the generous
	// default", a negative value means "disable the fallback and rely solely on
	// the request context".
	streamHeaderTimeout := cfg.StreamHeaderTimeout
	switch {
	case streamHeaderTimeout == 0:
		streamHeaderTimeout = defaultStreamHeaderTimeout
	case streamHeaderTimeout < 0:
		streamHeaderTimeout = 0
	}

	return &Client{
		baseURL:             baseURL,
		headers:             cfg.Headers,
		httpClient:          wrapped,
		streamClient:        &streamClient,
		streamHeaderTimeout: streamHeaderTimeout,
		logger:              cfg.Logger,
	}, nil
}

// Get performs a GET request to the specified path with query parameters.
func (c *Client) Get(ctx context.Context, path string, query url.Values, result any) error {
	return c.do(ctx, http.MethodGet, c.buildURL(path, query), nil, result)
}

// GetEscaped performs a GET request where escapedPath is already
// percent-encoded (e.g. via url.PathEscape). The encoding is preserved so a
// single path segment containing reserved characters is neither re-encoded nor
// split into multiple segments.
func (c *Client) GetEscaped(ctx context.Context, escapedPath string, query url.Values, result any) error {
	reqURL, err := c.buildEscapedURL(escapedPath, query)
	if err != nil {
		return err
	}
	return c.do(ctx, http.MethodGet, reqURL, nil, result)
}

// Post performs a POST request to the specified path with a JSON body.
func (c *Client) Post(ctx context.Context, path string, body, result any) error {
	return c.do(ctx, http.MethodPost, c.buildURL(path, nil), body, result)
}

// Delete performs a DELETE request to the specified path with a JSON body.
func (c *Client) Delete(ctx context.Context, path string, body, result any) error {
	return c.do(ctx, http.MethodDelete, c.buildURL(path, nil), body, result)
}

// DeleteEscaped performs a DELETE request where escapedPath is already
// percent-encoded (e.g. via url.PathEscape). See GetEscaped for the encoding
// contract.
func (c *Client) DeleteEscaped(ctx context.Context, escapedPath string, body, result any) error {
	reqURL, err := c.buildEscapedURL(escapedPath, nil)
	if err != nil {
		return err
	}
	return c.do(ctx, http.MethodDelete, reqURL, body, result)
}

// Patch performs a PATCH request to the specified path with a JSON body.
func (c *Client) Patch(ctx context.Context, path string, body, result any) error {
	return c.do(ctx, http.MethodPatch, c.buildURL(path, nil), body, result)
}

// GetBytes performs a GET request and returns the raw response body.
func (c *Client) GetBytes(ctx context.Context, path string, query url.Values) ([]byte, string, error) {
	return c.doRaw(ctx, http.MethodGet, path, query, nil, "", false)
}

// GetBodyStream performs a GET request and returns the response body for
// streaming without capping the response size. Use this for artifact downloads,
// where the body is an arbitrarily large object stream the caller consumes
// incrementally rather than a buffered API response. The transfer is not subject
// to the overall http.Client.Timeout; bound it via ctx. The caller must close the
// returned ReadCloser.
func (c *Client) GetBodyStream(ctx context.Context, path string, query url.Values) (io.ReadCloser, error) {
	reqURL := c.buildURL(path, query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	if c.logger != nil {
		c.logger.Debug("request",
			"method", http.MethodGet,
			"url", reqURL.String(),
		)
	}
	return c.doRequestBody(req)
}

// PutBytes performs a PUT request with a raw body and content type.
func (c *Client) PutBytes(ctx context.Context, path string, body []byte, contentType string) error {
	_, _, err := c.doRaw(ctx, http.MethodPut, path, nil, body, contentType, false)
	return err
}

// PutReader performs a PUT request that streams body as the request payload
// without buffering it in memory, so arbitrarily large artifacts can be
// uploaded. When body's length is not known ahead of time (e.g. an *os.File or a
// plain io.Reader) the request uses chunked transfer encoding. The upload is not
// subject to the overall http.Client.Timeout; bound it via ctx. The caller
// retains ownership of body and is responsible for closing it if needed.
func (c *Client) PutReader(ctx context.Context, path string, body io.Reader, contentType string) error {
	reqURL := c.buildURL(path, nil)

	// net/http's Transport closes the request body (even on error). Wrap an
	// io.Closer body so the caller keeps ownership, per this method's contract.
	// *bytes.Reader/*strings.Reader/*bytes.Buffer are not closers, so they stay
	// unwrapped and http.NewRequest can still detect them to set Content-Length.
	if _, ok := body.(io.Closer); ok {
		body = io.NopCloser(body)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, reqURL.String(), body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	start := time.Now()
	if c.logger != nil {
		c.logger.Debug("request",
			"method", http.MethodPut,
			"url", reqURL.String(),
		)
	}

	// Bound the wait for response headers, mirroring the streaming download path,
	// but only when the caller supplied no deadline of its own. streamClient has no
	// overall Timeout and the shared Transport may set no ResponseHeaderTimeout
	// (e.g. http.DefaultTransport), so a server that consumes the whole body and
	// then never responds would otherwise block forever. The upload itself must
	// stay unbounded (a large artifact can take arbitrarily long to send), so the
	// timer starts only once net/http reports the request has been fully written
	// (via the WroteRequest httptrace callback): it caps the wait for headers, not
	// the transfer. Body EOF is not a safe trigger because the transport can read
	// EOF from the body before the final bytes finish transmitting.
	var (
		headerMu    sync.Mutex
		headerTimer *time.Timer
		stopHeaders = func() bool { return false }
	)
	if _, hasDeadline := ctx.Deadline(); !hasDeadline && c.streamHeaderTimeout > 0 && req.Body != nil {
		streamCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		timeout := c.streamHeaderTimeout
		var stopped bool
		trace := &httptrace.ClientTrace{
			WroteRequest: func(info httptrace.WroteRequestInfo) {
				if info.Err != nil {
					return
				}
				headerMu.Lock()
				defer headerMu.Unlock()
				if stopped {
					return
				}
				headerTimer = time.AfterFunc(timeout, cancel)
			},
		}
		req = req.WithContext(httptrace.WithClientTrace(streamCtx, trace))
		// stopHeaders disarms the timer and reports whether it had already fired.
		// Timer.Stop returning false means the AfterFunc has run (or is running),
		// so cancel has been called and streamCtx is cancelled; the caller must
		// then treat the request as a header timeout even if Do returned success,
		// because the response body is tied to that cancelled context. Reading a
		// separate "fired" flag would be racy here: the flag is set just before
		// cancel, so a reader could observe it still false while cancel is pending.
		stopHeaders = func() bool {
			headerMu.Lock()
			defer headerMu.Unlock()
			stopped = true
			return headerTimer != nil && !headerTimer.Stop()
		}
	}

	// Stream the (potentially very large) body with the timeout-free client so a
	// slow upload is not aborted by the overall http.Client.Timeout; the transfer
	// is bounded by ctx via the request instead. Redirect following is disabled:
	// Go turns 301/302/303 into a bodyless GET (and cannot replay a streamed body
	// for 307/308), so a followed redirect could return 200 while the server
	// stored nothing. Returning the 3xx verbatim lets the non-2xx check below
	// surface it as a failure.
	uploadClient := *c.streamClient
	uploadClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	// Bound the workspace probe separately from the body transfer. The probe to
	// /server-info runs inside uploadClient.Do before the request is written, so
	// PutReader's WroteRequest-based header timer never covers it. Without a caller
	// deadline and with a Transport that sets no ResponseHeaderTimeout, a server
	// that never answers the probe would hang the upload indefinitely. Run the
	// probe under a bounded context up front so its result is cached before the
	// (deliberately unbounded) body transfer begins; the Do below then reuses the
	// cached result instead of probing again.
	if _, hasDeadline := ctx.Deadline(); !hasDeadline && c.streamHeaderTimeout > 0 {
		if wrt := extractWorkspaceRT(&uploadClient); wrt != nil {
			probeCtx, cancelProbe := context.WithTimeout(ctx, c.streamHeaderTimeout)
			// shouldAttach performs the same gating (exemption, same-origin,
			// probeEnabled) the round-tripper applies during Do, forwarding this
			// request's auth headers to the probe, but here under probeCtx.
			_, probeErr := wrt.shouldAttach(req.Clone(probeCtx))
			cancelProbe()
			if probeErr != nil {
				return fmt.Errorf("request failed: %w", probeErr)
			}
		}
	}

	resp, err := uploadClient.Do(req)
	fired := stopHeaders()
	if err != nil {
		// If the header timer fired it canceled streamCtx, so Do failed with
		// context.Canceled; surface it as a header timeout so the caller can tell
		// it apart from its own cancellation, matching the download path.
		if fired {
			return fmt.Errorf("request failed: response headers not received within %s", c.streamHeaderTimeout)
		}
		return fmt.Errorf("request failed: %w", err)
	}
	if fired {
		// The header deadline fired at the same moment Do returned success. cancel
		// has already canceled streamCtx, so the response body is tied to a dead
		// context and reading it would fail with context.Canceled, masking the
		// timeout. Report the header timeout instead, matching doRequestBody.
		resp.Body.Close()
		return fmt.Errorf("request failed: response headers not received within %s", c.streamHeaderTimeout)
	}
	defer resp.Body.Close()

	if c.logger != nil {
		c.logger.Debug("response",
			"status", resp.StatusCode,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	}

	// The response to a PUT is small (status/metadata); read it fully so the
	// connection can be reused and any error body can be surfaced.
	respBody, err := readResponseBody(resp.Body)
	if err != nil {
		return err
	}
	// Treat anything outside 2xx as a failure. A streaming body has no
	// req.GetBody, so Go's client cannot replay it across a redirect and returns
	// the 3xx response verbatim; accepting it would report a stored-nothing
	// upload as success.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.parseError(resp.StatusCode, respBody)
	}
	return nil
}

// PostBytes performs a POST request with a raw body and content type.
func (c *Client) PostBytes(ctx context.Context, path string, query url.Values, body []byte, contentType string) error {
	_, _, err := c.doRaw(ctx, http.MethodPost, path, query, body, contentType, false)
	return err
}

// DoAbsolute performs an HTTP request to an absolute URL outside the tracking server base URL.
// This is used for presigned artifact upload/download URLs.
func (c *Client) DoAbsolute(ctx context.Context, method, absoluteURL string, headers map[string]string, body []byte) ([]byte, string, error) {
	return c.doAbsolute(ctx, method, absoluteURL, headers, body)
}

// DoAbsoluteGetBody performs a GET to an absolute URL and returns the response body for streaming.
// The caller must close the returned ReadCloser.
func (c *Client) DoAbsoluteGetBody(ctx context.Context, absoluteURL string, headers map[string]string) (io.ReadCloser, error) {
	return c.doAbsoluteBody(ctx, http.MethodGet, absoluteURL, headers)
}

// buildURL constructs the full request URL by appending path to the base URL
// prefix. The path is treated as a literal (raw) value: reserved characters are
// percent-encoded exactly once during serialization. This preserves raw-path
// handling for callers such as the artifacts client that pass storage paths
// verbatim. Callers that need a single, already-escaped path segment (e.g. a
// name containing "/") must use buildEscapedURL instead.
func (c *Client) buildURL(path string, query url.Values) *url.URL {
	// Build the escaped request path from the base URL's escaped prefix plus the
	// escaped literal suffix, then keep it in RawPath. This:
	//   - preserves any encoding in the base prefix (e.g. https://host/api%2Fv1)
	//     rather than decoding it to raw separators;
	//   - encodes reserved characters in path exactly once (literal handling);
	//   - unlike ResolveReference, does not collapse "." or ".." segments, so
	//     storage object paths containing them are left intact.
	rawPath := strings.TrimRight(c.baseURL.EscapedPath(), "/") + (&url.URL{Path: path}).EscapedPath()
	decoded, err := url.PathUnescape(rawPath)
	if err != nil {
		// rawPath is assembled from validly-escaped parts, so this is
		// unreachable in practice; fall back defensively.
		decoded = rawPath
	}
	u := *c.baseURL
	u.RawPath = rawPath
	u.Path = decoded
	u.RawQuery = query.Encode()
	return &u
}

// buildEscapedURL constructs the full request URL from an already
// percent-encoded path (e.g. produced by url.PathEscape). RawPath preserves the
// caller's escaping so it is not double-encoded, while Path holds the decoded
// form required by net/url. Unlike buildURL, this does not treat reserved
// characters as literals: it trusts the caller to have escaped each path
// segment.
func (c *Client) buildEscapedURL(escapedPath string, query url.Values) (*url.URL, error) {
	// Use the base URL's escaped path so any encoding in the base prefix
	// (e.g. https://host/api%2Fv1) is preserved rather than decoded into raw
	// separators when assigned to RawPath below.
	rawPath := strings.TrimRight(c.baseURL.EscapedPath(), "/") + escapedPath
	decoded, err := url.PathUnescape(rawPath)
	if err != nil {
		return nil, fmt.Errorf("invalid escaped path %q: %w", escapedPath, err)
	}
	u := *c.baseURL
	u.RawPath = rawPath
	u.Path = decoded
	u.RawQuery = query.Encode()
	return &u, nil
}

func (c *Client) do(ctx context.Context, method string, reqURL *url.URL, body, result any) error {
	// Encode body if present
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to encode request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, reqURL.String(), bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	// Log request
	start := time.Now()
	if c.logger != nil {
		c.logger.Debug("request",
			"method", method,
			"url", reqURL.String(),
		)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Log response
	duration := time.Since(start)
	if c.logger != nil {
		c.logger.Debug("response",
			"status", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
		)
	}

	// Read response body (bounded by maxResponseBodySize)
	respBody, err := readResponseBody(resp.Body)
	if err != nil {
		return err
	}

	// Handle error responses
	if resp.StatusCode >= 400 {
		return c.parseError(resp.StatusCode, respBody)
	}

	// Decode successful response
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

func (c *Client) doRaw(ctx context.Context, method, path string, query url.Values, body []byte, contentType string, jsonAccept bool) ([]byte, string, error) {
	reqURL := c.buildURL(path, query)

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL.String(), bodyReader)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil && contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if jsonAccept {
		req.Header.Set("Accept", "application/json")
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	start := time.Now()
	if c.logger != nil {
		c.logger.Debug("request",
			"method", method,
			"url", reqURL.String(),
		)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if c.logger != nil {
		c.logger.Debug("response",
			"status", resp.StatusCode,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	}

	respBody, err := readResponseBody(resp.Body)
	if err != nil {
		return nil, "", err
	}

	if resp.StatusCode >= 400 {
		return nil, "", c.parseError(resp.StatusCode, respBody)
	}

	return respBody, resp.Header.Get("Content-Type"), nil
}

func (c *Client) doAbsoluteBody(ctx context.Context, method, absoluteURL string, headers map[string]string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, method, absoluteURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	if c.logger != nil {
		c.logger.Debug("request",
			"method", method,
			"url", redactAbsoluteURLForLog(absoluteURL),
		)
	}

	// Absolute GETs are presigned artifact downloads: arbitrarily large object
	// streams the caller consumes incrementally, so they are not size-capped.
	return c.doRequestBody(req)
}

// doRequestBody performs req and returns its body as an uncapped stream for the
// caller to consume incrementally. It uses the timeout-free stream client so a
// large artifact transfer is not aborted by the overall http.Client.Timeout.
func (c *Client) doRequestBody(req *http.Request) (io.ReadCloser, error) {
	start := time.Now()

	client := c.streamClient
	// headerCancel bounds the response-header phase of a streaming request. The
	// stream client has no overall Timeout and its shared Transport may set no
	// ResponseHeaderTimeout (e.g. http.DefaultTransport), so without this a server
	// that accepts the connection but never sends headers would block forever.
	//
	// The bound is only a fallback: when the caller's context already carries a
	// deadline, that deadline governs the whole request (headers included) and we
	// add no timer. Otherwise we fall back to streamHeaderTimeout, which is
	// independent of the API Timeout — MLflow's mlflow-artifacts proxy fetches the
	// full remote object before sending headers, so a large artifact can take far
	// longer than the 30s API timeout to reach first byte. The timer fires cancel
	// if headers do not arrive in time; once they do we stop the timer and hand
	// cancel to the body's Close so the transfer stays bound only by the context.
	var headerCancel context.CancelFunc
	var headerTimer *time.Timer
	var headerTimeout time.Duration
	if _, hasDeadline := req.Context().Deadline(); !hasDeadline && c.streamHeaderTimeout > 0 {
		headerTimeout = c.streamHeaderTimeout
		var ctx context.Context
		ctx, headerCancel = context.WithCancel(req.Context())
		req = req.WithContext(ctx)
		headerTimer = time.AfterFunc(headerTimeout, headerCancel)
	}

	resp, err := client.Do(req)
	if err != nil {
		// If the header timer fired it canceled the request context, so Do failed
		// with context.Canceled. Stop reporting false then means the timer fired;
		// surface it as a header timeout so the caller can tell it apart from its
		// own cancellation, matching the success path and PutReader.
		fired := headerTimer != nil && !headerTimer.Stop()
		if headerCancel != nil {
			headerCancel()
		}
		if fired {
			return nil, fmt.Errorf("request failed: response headers not received within %s", headerTimeout)
		}
		return nil, fmt.Errorf("request failed: %w", err)
	}
	if headerTimer != nil && !headerTimer.Stop() {
		// The header deadline fired (Stop reports it did not stop a pending timer)
		// after Do returned success, so headerCancel has already canceled the
		// request context. The response body is tied to that context and its first
		// read would fail, so treat this as a header timeout rather than handing
		// back a stream that is already broken.
		resp.Body.Close()
		headerCancel()
		return nil, fmt.Errorf("request failed: response headers not received within %s", headerTimeout)
	}

	if c.logger != nil {
		c.logger.Debug("response",
			"status", resp.StatusCode,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	}

	if resp.StatusCode >= 400 {
		respBody, readErr := readResponseBody(resp.Body)
		resp.Body.Close()
		if headerCancel != nil {
			headerCancel()
		}
		if readErr != nil {
			return nil, readErr
		}
		return nil, c.parseError(resp.StatusCode, respBody)
	}

	if headerCancel != nil {
		// Keep the header-phase context alive for the duration of the stream;
		// release it when the caller closes the body.
		return &cancelReadCloser{ReadCloser: resp.Body, cancel: headerCancel}, nil
	}
	return resp.Body, nil
}

// cancelReadCloser wraps a response body and runs cancel when the body is closed,
// releasing a context created to bound the request's response-header phase.
type cancelReadCloser struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (c *cancelReadCloser) Close() error {
	err := c.ReadCloser.Close()
	c.cancel()
	return err
}

func (c *Client) doAbsolute(ctx context.Context, method, absoluteURL string, headers map[string]string, body []byte) ([]byte, string, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, absoluteURL, bodyReader)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	start := time.Now()
	if c.logger != nil {
		c.logger.Debug("request",
			"method", method,
			"url", redactAbsoluteURLForLog(absoluteURL),
		)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if c.logger != nil {
		c.logger.Debug("response",
			"status", resp.StatusCode,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	}

	respBody, err := readResponseBody(resp.Body)
	if err != nil {
		return nil, "", err
	}

	if resp.StatusCode >= 400 {
		return nil, "", c.parseError(resp.StatusCode, respBody)
	}

	return respBody, resp.Header.Get("Content-Type"), nil
}

func readResponseBody(r io.Reader) ([]byte, error) {
	limited := io.LimitReader(r, maxResponseBodySize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if int64(len(data)) > maxResponseBodySize {
		return nil, fmt.Errorf("response body exceeds maximum size of %d bytes", maxResponseBodySize)
	}
	return data, nil
}

func redactAbsoluteURLForLog(absoluteURL string) string {
	parsed, err := url.Parse(absoluteURL)
	if err != nil {
		return absoluteURL
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func wrapClientWithAuth(c *http.Client, token, tokenPath string, trackingURL *url.URL) *http.Client {
	if token == "" && tokenPath == "" {
		return c
	}
	base := c.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	var rt http.RoundTripper
	if tokenPath != "" {
		rt = NewTokenFileRoundTripper(base, tokenPath, trackingURL)
	} else {
		rt = NewTokenRoundTripper(base, token, trackingURL)
	}
	clone := *c
	clone.Transport = rt
	clone.CheckRedirect = stripAuthOnCrossOriginRedirect(c.CheckRedirect)
	return &clone
}

// stripAuthOnCrossOriginRedirect returns a CheckRedirect that deletes the
// Authorization header whenever a redirect destination differs, by exact origin
// (scheme+host+port), from either the initial request or the immediately
// preceding hop, then delegates the follow/stop decision to prev.
//
// net/http only strips Authorization when the destination host is neither the
// same host nor a subdomain of the origin, and it ignores scheme and port
// entirely (see shouldCopyHeaderOnRedirect in net/http/client.go). That leaves
// a caller-supplied credential able to survive a redirect to a subdomain or to
// the same host on a different scheme/port.
//
// Comparing against the initial request (via[0]) is essential, not just the
// preceding hop: net/http re-copies redirected headers from the initial
// request's headers on every hop (makeHeadersCopier), so a Del on one hop does
// not prevent a re-copy on a later hop whose origin matches the preceding hop
// but still differs from the initial (credential-owning) origin. We tighten
// this so a credential never leaks to a different exact origin, while genuine
// same-origin redirects keep it. Direct foreign-origin requests (e.g. presigned
// object-store URLs) are unaffected because they are not redirects.
func stripAuthOnCrossOriginRedirect(prev func(*http.Request, []*http.Request) error) func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) > 0 {
			origin := requestOrigin(req)
			if origin != requestOrigin(via[0]) || origin != requestOrigin(via[len(via)-1]) {
				req.Header.Del("Authorization")
			}
		}
		if prev != nil {
			return prev(req, via)
		}
		// Replicate net/http's default cap, which only applies when
		// CheckRedirect is nil; setting our own function opts out of it.
		if len(via) >= 10 {
			return fmt.Errorf("stopped after 10 redirects")
		}
		return nil
	}
}

func (c *Client) parseError(statusCode int, body []byte) error {
	apiErr := &errors.APIError{StatusCode: statusCode}
	// Gate on a JSON object, and decode fields independently: a non-string
	// message must not discard a valid MLflow error code (or vice versa).
	var fields struct {
		Code    json.RawMessage `json:"error_code"`
		Message json.RawMessage `json:"message"`
	}
	if json.Unmarshal(body, &fields) == nil {
		if err := json.Unmarshal(fields.Code, &apiErr.Code); err != nil {
			apiErr.Code = ""
		}
		if err := json.Unmarshal(fields.Message, &apiErr.Message); err != nil {
			apiErr.Message = ""
		}
	}
	if strings.TrimSpace(apiErr.Message) == "" {
		apiErr.Message = http.StatusText(statusCode)
		if apiErr.Message == "" {
			apiErr.Message = "HTTP request failed"
		}
		// Proxies and providers can return arbitrary JSON, HTML, or plain text.
		// Include only a bounded body excerpt; do not append request or response
		// headers or client configuration. Server bodies may contain sensitive
		// data. Quote the excerpt so it cannot inject log lines.
		const maxErrorBodyExcerpt = 4096
		excerpt := bytes.TrimSpace(body)
		if len(excerpt) > 0 {
			truncated := len(excerpt) > maxErrorBodyExcerpt
			if truncated {
				excerpt = excerpt[:maxErrorBodyExcerpt]
			}
			apiErr.Message += fmt.Sprintf(": response body %q", strings.ToValidUTF8(string(excerpt), "\uFFFD"))
			if truncated {
				apiErr.Message += " (truncated)"
			}
		} else {
			apiErr.Message += ": empty response body"
		}
	}
	return apiErr
}
