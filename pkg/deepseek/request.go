package deepseek

import "encoding/json"

// Request is the body of POST /chat/completions. Pointer fields are omitted
// when nil so the server applies its own defaults.
//
// See https://api-docs.deepseek.com/api/create-chat-completion.
type Request struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`

	MaxTokens int `json:"max_tokens,omitempty"`

	// Sampling parameters. Note: in thinking mode these are accepted for
	// compatibility but have no effect.
	Temperature      *float64 `json:"temperature,omitempty"`
	TopP             *float64 `json:"top_p,omitempty"`
	PresencePenalty  *float64 `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`
	Stop             []string `json:"stop,omitempty"`

	Stream        bool           `json:"stream,omitempty"`
	StreamOptions *StreamOptions `json:"stream_options,omitempty"`

	Tools      []Tool      `json:"tools,omitempty"`
	ToolChoice *ToolChoice `json:"tool_choice,omitempty"`

	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`

	// ReasoningEffort controls thinking depth ("low"/"medium"/"high"/"max");
	// low/medium are mapped to high on current models.
	ReasoningEffort string `json:"reasoning_effort,omitempty"`

	// Thinking toggles thinking mode. Type is "enabled" or "disabled".
	// Thinking is enabled by default on V4 models.
	Thinking *Thinking `json:"thinking,omitempty"`
}

// StreamOptions tunes streaming behaviour.
type StreamOptions struct {
	// IncludeUsage requests a final chunk carrying usage totals.
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// Thinking toggles thinking (chain-of-thought) mode.
type Thinking struct {
	Type string `json:"type"` // "enabled" | "disabled"
}

// EnableThinking / DisableThinking are convenience constructors.
func EnableThinking() *Thinking  { return &Thinking{Type: "enabled"} }
func DisableThinking() *Thinking { return &Thinking{Type: "disabled"} }

// Tool is a tool/function definition. Type is "function".
type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// Function describes a callable function exposed to the model.
type Function struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// NewTool is a convenience constructor for a function tool.
func NewTool(name, description string, parameters json.RawMessage) Tool {
	return Tool{
		Type:     "function",
		Function: Function{Name: name, Description: description, Parameters: parameters},
	}
}

// ToolChoice constrains tool usage. It marshals either as a bare string
// ("none"/"auto"/"required") or as an object selecting a named function.
type ToolChoice struct {
	Mode string // "none" | "auto" | "required"
	Name string // forces a specific function when set
}

func (t ToolChoice) MarshalJSON() ([]byte, error) {
	if t.Name != "" {
		return json.Marshal(map[string]any{
			"type":     "function",
			"function": map[string]string{"name": t.Name},
		})
	}
	mode := t.Mode
	if mode == "" {
		mode = "auto"
	}
	return json.Marshal(mode)
}

func ToolChoiceAuto() *ToolChoice     { return &ToolChoice{Mode: "auto"} }
func ToolChoiceRequired() *ToolChoice { return &ToolChoice{Mode: "required"} }
func ToolChoiceNone() *ToolChoice     { return &ToolChoice{Mode: "none"} }
func ToolChoiceFunc(name string) *ToolChoice {
	return &ToolChoice{Name: name}
}

// ResponseFormat controls structured output. Type is "text" or "json_object".
type ResponseFormat struct {
	Type string `json:"type"`
}

// JSONObjectFormat is a convenience constructor.
func JSONObjectFormat() *ResponseFormat { return &ResponseFormat{Type: "json_object"} }
