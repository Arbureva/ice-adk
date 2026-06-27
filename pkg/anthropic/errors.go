package anthropic

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Anthropic error type strings (the error.type field).
const (
	ErrTypeInvalidRequest  = "invalid_request_error"
	ErrTypeAuthentication  = "authentication_error"
	ErrTypePermission      = "permission_error"
	ErrTypeNotFound        = "not_found_error"
	ErrTypeRequestTooLarge = "request_too_large"
	ErrTypeRateLimit       = "rate_limit_error"
	ErrTypeAPI             = "api_error"
	ErrTypeOverloaded      = "overloaded_error"
)

// APIError is returned for any non-2xx response. It implements error.
type APIError struct {
	StatusCode int
	Type       string
	Message    string
	RequestID  string
	RetryAfter time.Duration
}

func (e *APIError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("anthropic: %d %s: %s (request-id: %s)",
			e.StatusCode, e.Type, e.Message, e.RequestID)
	}
	return fmt.Sprintf("anthropic: %d %s: %s", e.StatusCode, e.Type, e.Message)
}

// Retryable reports whether the request may be retried.
func (e *APIError) Retryable() bool {
	switch {
	case e.StatusCode == http.StatusTooManyRequests:
		return true
	case e.StatusCode == http.StatusRequestTimeout:
		return true
	case e.StatusCode >= 500:
		return true
	case e.Type == ErrTypeOverloaded:
		return true
	default:
		return false
	}
}

type errorEnvelope struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// parseAPIError builds an *APIError from a non-2xx response without closing it.
func parseAPIError(resp *http.Response) *APIError {
	e := &APIError{
		StatusCode: resp.StatusCode,
		Type:       ErrTypeAPI,
		Message:    http.StatusText(resp.StatusCode),
		RequestID:  resp.Header.Get("Request-Id"),
		RetryAfter: parseRetryAfter(resp.Header),
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if len(body) == 0 {
		return e
	}
	var env errorEnvelope
	if err := json.Unmarshal(body, &env); err == nil && env.Error.Type != "" {
		e.Type = env.Error.Type
		e.Message = env.Error.Message
		return e
	}
	e.Message = string(body)
	return e
}
