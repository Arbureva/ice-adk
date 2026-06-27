package openai

import "encoding/json"

// Request is the body of POST /v1/chat/completions. Pointer fields are omitted
// when nil so the server applies its own defaults.
//
// See https://platform.openai.com/docs/api-reference/chat/create.
type Request struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`

	// MaxTokens is deprecated by the API in favor of MaxCompletionTokens and
	// is not accepted by o-series/reasoning models. Set only one.
	MaxTokens           int `json:"max_tokens,omitempty"`
	MaxCompletionTokens int `json:"max_completion_tokens,omitempty"`

	Temperature      *float64 `json:"temperature,omitempty"`
	TopP             *float64 `json:"top_p,omitempty"`
	N                int      `json:"n,omitempty"`
	Stop             []string `json:"stop,omitempty"`
	PresencePenalty  *float64 `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`
	Seed             *int     `json:"seed,omitempty"`

	Stream        bool           `json:"stream,omitempty"`
	StreamOptions *StreamOptions `json:"stream_options,omitempty"`

	Tools      []Tool      `json:"tools,omitempty"`
	ToolChoice *ToolChoice `json:"tool_choice,omitempty"`

	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`

	// ReasoningEffort applies to reasoning models ("minimal"/"low"/"medium"/
	// "high"), ignored otherwise.
	ReasoningEffort string `json:"reasoning_effort,omitempty"`

	// User is a stable end-user identifier for abuse detection / cache bucketing.
	User string `json:"user,omitempty"`
}

// StreamOptions tunes streaming behaviour.
type StreamOptions struct {
	// IncludeUsage requests a final chunk carrying usage totals.
	IncludeUsage bool `json:"include_usage,omitempty"`
}

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
	Strict      bool            `json:"strict,omitempty"`
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
	// Mode is "none", "auto", or "required". Mutually exclusive with Name.
	Mode string
	// Name forces a specific function when set.
	Name string
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

// ToolChoiceAuto / Required / None / Func are convenience constructors.
func ToolChoiceAuto() *ToolChoice     { return &ToolChoice{Mode: "auto"} }
func ToolChoiceRequired() *ToolChoice { return &ToolChoice{Mode: "required"} }
func ToolChoiceNone() *ToolChoice     { return &ToolChoice{Mode: "none"} }
func ToolChoiceFunc(name string) *ToolChoice {
	return &ToolChoice{Name: name}
}

// ResponseFormat controls structured output. Type is "text", "json_object",
// or "json_schema"; JSONSchema is required for the latter.
type ResponseFormat struct {
	Type       string          `json:"type"`
	JSONSchema json.RawMessage `json:"json_schema,omitempty"`
}

// JSONObjectFormat / JSONSchemaFormat are convenience constructors.
func JSONObjectFormat() *ResponseFormat {
	return &ResponseFormat{Type: "json_object"}
}
func JSONSchemaFormat(schema json.RawMessage) *ResponseFormat {
	return &ResponseFormat{Type: "json_schema", JSONSchema: schema}
}
