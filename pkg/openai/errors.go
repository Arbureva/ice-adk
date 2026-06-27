package openai

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// APIError is returned for any non-2xx response. It implements error.
type APIError struct {
	StatusCode int    `json:"-"`
	RequestID  string `json:"-"`
	RetryAfter time.Duration

	// Type, Code, Param and Message come from the error envelope.
	Type    string `json:"type"`
	Code    string `json:"code"`
	Param   string `json:"param"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	id := ""
	if e.RequestID != "" {
		id = " (request-id: " + e.RequestID + ")"
	}
	return fmt.Sprintf("openai: %d %s/%s: %s%s", e.StatusCode, e.Type, e.Code, e.Message, id)
}

// Retryable reports whether the request may be retried.
func (e *APIError) Retryable() bool {
	switch {
	case e.StatusCode == http.StatusTooManyRequests:
		return true
	case e.StatusCode == http.StatusRequestTimeout:
		return true
	case e.StatusCode == http.StatusConflict:
		return true
	case e.StatusCode >= 500:
		return true
	default:
		return false
	}
}

type errorEnvelope struct {
	Error APIError `json:"error"`
}

// parseAPIError builds an *APIError from a non-2xx response without closing it.
func parseAPIError(resp *http.Response) *APIError {
	e := &APIError{
		StatusCode: resp.StatusCode,
		RequestID:  resp.Header.Get("X-Request-Id"),
		RetryAfter: parseRetryAfter(resp.Header),
		Message:    http.StatusText(resp.StatusCode),
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if len(body) == 0 {
		return e
	}
	var env errorEnvelope
	if err := json.Unmarshal(body, &env); err == nil && (env.Error.Message != "" || env.Error.Type != "") {
		env.Error.StatusCode = e.StatusCode
		env.Error.RequestID = e.RequestID
		env.Error.RetryAfter = e.RetryAfter
		return &env.Error
	}
	e.Message = string(body)
	return e
}
