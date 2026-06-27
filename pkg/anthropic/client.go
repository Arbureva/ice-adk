package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Defaults applied when a Config field is left zero.
const (
	DefaultBaseURL = "https://api.anthropic.com"
	DefaultVersion = "2023-06-01"
	MessagesPath   = "/v1/messages"

	defaultTimeout    = 600 * time.Second
	defaultMaxRetries = 2
)

// Config configures a Client. Intended to be populated from the kit's config
// file rather than read from the environment by this package.
type Config struct {
	// APIKey is sent as the x-api-key header. Required.
	APIKey string `json:"api_key"`

	// BaseURL overrides the API root. Defaults to DefaultBaseURL.
	BaseURL string `json:"base_url"`

	// Version sets the anthropic-version header. Defaults to DefaultVersion.
	Version string `json:"version"`

	// Beta sets one or more anthropic-beta header values (joined with ",").
	Beta []string `json:"beta"`

	// Timeout bounds a single HTTP attempt. Defaults to defaultTimeout.
	// Ignored when HTTPClient is supplied.
	Timeout time.Duration `json:"timeout"`

	// MaxRetries on 429/5xx/overloaded and transport errors. Defaults to
	// defaultMaxRetries. A negative value disables retries.
	MaxRetries int `json:"max_retries"`

	// HTTPClient injects a pre-configured client. When nil one is built from
	// Timeout.
	HTTPClient *http.Client `json:"-"`
}

// Client is a concurrency-safe Anthropic Messages API client.
type Client struct {
	apiKey     string
	baseURL    string
	version    string
	beta       string
	maxRetries int
	hc         *http.Client
}

// New builds a Client, filling defaults for any zero-valued Config field.
func New(cfg Config) *Client {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	version := cfg.Version
	if version == "" {
		version = DefaultVersion
	}
	maxRetries := cfg.MaxRetries
	if maxRetries == 0 {
		maxRetries = defaultMaxRetries
	}
	hc := cfg.HTTPClient
	if hc == nil {
		timeout := cfg.Timeout
		if timeout == 0 {
			timeout = defaultTimeout
		}
		hc = &http.Client{Timeout: timeout}
	}
	return &Client{
		apiKey:     cfg.APIKey,
		baseURL:    baseURL,
		version:    version,
		beta:       strings.Join(cfg.Beta, ","),
		maxRetries: maxRetries,
		hc:         hc,
	}
}

func (c *Client) newHTTPRequest(ctx context.Context, body *Request, stream bool) (*http.Request, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("anthropic: empty API key")
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("anthropic: marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+MessagesPath, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("anthropic: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Anthropic-Version", c.version)
	if c.beta != "" {
		req.Header.Set("Anthropic-Beta", c.beta)
	}
	if stream {
		req.Header.Set("Accept", "text/event-stream")
	} else {
		req.Header.Set("Accept", "application/json")
	}
	return req, nil
}

// do executes the request with retry/backoff. On success the response body is
// open and the caller must close it; on a non-2xx status the body is consumed
// and an *APIError is returned.
func (c *Client) do(ctx context.Context, body *Request, stream bool) (*http.Response, error) {
	var lastErr error
	for attempt := 0; ; attempt++ {
		req, err := c.newHTTPRequest(ctx, body, stream)
		if err != nil {
			return nil, err
		}

		resp, err := c.hc.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("anthropic: do request: %w", err)
			if attempt < c.maxRetries && ctx.Err() == nil {
				if !sleepCtx(ctx, backoff(attempt, 0)) {
					return nil, ctx.Err()
				}
				continue
			}
			return nil, lastErr
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}

		apiErr := parseAPIError(resp)
		_ = resp.Body.Close()
		lastErr = apiErr
		if attempt < c.maxRetries && apiErr.Retryable() && ctx.Err() == nil {
			if !sleepCtx(ctx, backoff(attempt, apiErr.RetryAfter)) {
				return nil, ctx.Err()
			}
			continue
		}
		return nil, apiErr
	}
}

func backoff(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		return retryAfter
	}
	const base = 500 * time.Millisecond
	const max = 16 * time.Second
	d := time.Duration(float64(base) * math.Pow(2, float64(attempt)))
	if d > max {
		d = max
	}
	return time.Duration(rand.Int63n(int64(d) + 1))
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func drainAndClose(r io.ReadCloser) {
	_, _ = io.Copy(io.Discard, r)
	_ = r.Close()
}

func parseRetryAfter(h http.Header) time.Duration {
	v := h.Get("Retry-After")
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
	return 0
}
