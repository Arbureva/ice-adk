package anthropic

import "encoding/json"

// Request is the body of POST /v1/messages. Pointer fields are omitted when nil
// so the server applies its own defaults.
//
// See https://docs.anthropic.com/en/api/messages.
type Request struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`

	// System prompt: either a plain string (System) or an array of text blocks
	// (SystemBlocks, e.g. for prompt caching). At most one should be set.
	System       string         `json:"-"`
	SystemBlocks []ContentBlock `json:"-"`

	Temperature   *float64 `json:"temperature,omitempty"`
	TopP          *float64 `json:"top_p,omitempty"`
	TopK          *int     `json:"top_k,omitempty"`
	StopSequences []string `json:"stop_sequences,omitempty"`

	Stream bool `json:"stream,omitempty"`

	Tools      []Tool      `json:"tools,omitempty"`
	ToolChoice *ToolChoice `json:"tool_choice,omitempty"`

	Thinking *Thinking `json:"thinking,omitempty"`

	Metadata map[string]string `json:"metadata,omitempty"`
}

type requestWire struct {
	Model         string            `json:"model"`
	Messages      []Message         `json:"messages"`
	MaxTokens     int               `json:"max_tokens"`
	System        json.RawMessage   `json:"system,omitempty"`
	Temperature   *float64          `json:"temperature,omitempty"`
	TopP          *float64          `json:"top_p,omitempty"`
	TopK          *int              `json:"top_k,omitempty"`
	StopSequences []string          `json:"stop_sequences,omitempty"`
	Stream        bool              `json:"stream,omitempty"`
	Tools         []Tool            `json:"tools,omitempty"`
	ToolChoice    *ToolChoice       `json:"tool_choice,omitempty"`
	Thinking      *Thinking         `json:"thinking,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// MarshalJSON projects System/SystemBlocks onto the single "system" key.
func (r Request) MarshalJSON() ([]byte, error) {
	w := requestWire{
		Model:         r.Model,
		Messages:      r.Messages,
		MaxTokens:     r.MaxTokens,
		Temperature:   r.Temperature,
		TopP:          r.TopP,
		TopK:          r.TopK,
		StopSequences: r.StopSequences,
		Stream:        r.Stream,
		Tools:         r.Tools,
		ToolChoice:    r.ToolChoice,
		Thinking:      r.Thinking,
		Metadata:      r.Metadata,
	}
	switch {
	case len(r.SystemBlocks) > 0:
		b, err := json.Marshal(r.SystemBlocks)
		if err != nil {
			return nil, err
		}
		w.System = b
	case r.System != "":
		b, err := json.Marshal(r.System)
		if err != nil {
			return nil, err
		}
		w.System = b
	}
	return json.Marshal(w)
}

// Tool is a tool definition exposed to the model.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
	// Type is set for server/builtin tools; empty for custom tools.
	Type string `json:"type,omitempty"`
}

// ToolChoice constrains tool usage. Type is "auto", "any", "tool", or "none".
// Name is required only when Type == "tool".
type ToolChoice struct {
	Type                   string `json:"type"`
	Name                   string `json:"name,omitempty"`
	DisableParallelToolUse bool   `json:"disable_parallel_tool_use,omitempty"`
}

func ToolChoiceAuto() *ToolChoice { return &ToolChoice{Type: "auto"} }
func ToolChoiceAny() *ToolChoice  { return &ToolChoice{Type: "any"} }
func ToolChoiceNone() *ToolChoice { return &ToolChoice{Type: "none"} }
func ToolChoiceTool(n string) *ToolChoice {
	return &ToolChoice{Type: "tool", Name: n}
}

// Thinking enables extended thinking. Type is "enabled" or "disabled".
type Thinking struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens,omitempty"`
}

// EnableThinking returns a Thinking config with the given token budget.
func EnableThinking(budget int) *Thinking {
	return &Thinking{Type: "enabled", BudgetTokens: budget}
}
