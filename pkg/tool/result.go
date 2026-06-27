package tool

import (
	"encoding/json"
	"fmt"
)

// Result is what a tool returns. Content is the textual payload fed back to the
// model as the tool's output (it becomes the provider's tool / tool_result
// message). IsError tells the model the call failed without aborting the loop.
// Meta is optional out-of-band data for the host (logging, UI, tracing) and is
// never sent to the model.
type Result struct {
	Content string          `json:"content"`
	IsError bool            `json:"is_error,omitempty"`
	Meta    json.RawMessage `json:"meta,omitempty"`
}

// Text returns a successful Result carrying s verbatim.
func Text(s string) *Result { return &Result{Content: s} }

// Textf is Text with fmt formatting.
func Textf(format string, a ...any) *Result {
	return &Result{Content: fmt.Sprintf(format, a...)}
}

// JSON marshals v as the Result content. A marshalling failure comes back as an
// error (not an IsError result): it is a host bug, not a tool outcome.
func JSON(v any) (*Result, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("tool: marshal result: %w", err)
	}
	return &Result{Content: string(b)}, nil
}

// Err returns a Result flagged as an error, carrying msg so the model can read
// what went wrong.
func Err(msg string) *Result { return &Result{Content: msg, IsError: true} }

// Errf is Err with fmt formatting.
func Errf(format string, a ...any) *Result {
	return &Result{Content: fmt.Sprintf(format, a...), IsError: true}
}

// WithMeta attaches host-only metadata and returns r for chaining.
func (r *Result) WithMeta(meta json.RawMessage) *Result {
	r.Meta = meta
	return r
}
