package deepseek

import "encoding/json"

// Roles.
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

// Finish reasons.
const (
	FinishStop                       = "stop"
	FinishLength                     = "length"
	FinishContentFilter              = "content_filter"
	FinishToolCalls                  = "tool_calls"
	FinishInsufficientSystemResource = "insufficient_system_resource"
)

// Message is a chat message, used both as request input and response output.
//
// ReasoningContent (chain-of-thought) is present only on responses from
// thinking/reasoner models. The API rejects it on input, so it is excluded from
// marshalling (json:"-") and only populated on decode.
type Message struct {
	Role             string     `json:"role"`
	Content          string     `json:"content,omitempty"`
	Name             string     `json:"name,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
	ReasoningContent string     `json:"-"`
}

func (m *Message) UnmarshalJSON(b []byte) error {
	type alias struct {
		Role             string     `json:"role"`
		Content          string     `json:"content"`
		Name             string     `json:"name"`
		ToolCalls        []ToolCall `json:"tool_calls"`
		ToolCallID       string     `json:"tool_call_id"`
		ReasoningContent string     `json:"reasoning_content"`
	}
	var a alias
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	m.Role = a.Role
	m.Content = a.Content
	m.Name = a.Name
	m.ToolCalls = a.ToolCalls
	m.ToolCallID = a.ToolCallID
	m.ReasoningContent = a.ReasoningContent
	return nil
}

// ToolCall is a tool invocation requested by the model. Index is only set on
// streaming deltas (to correlate fragments across chunks).
type ToolCall struct {
	Index    *int         `json:"index,omitempty"`
	ID       string       `json:"id,omitempty"`
	Type     string       `json:"type,omitempty"` // "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall is the function name + JSON-encoded arguments of a tool call.
type FunctionCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// Usage reports token accounting, including DeepSeek's context-cache breakdown.
type Usage struct {
	PromptTokens            int                      `json:"prompt_tokens"`
	CompletionTokens        int                      `json:"completion_tokens"`
	TotalTokens             int                      `json:"total_tokens"`
	PromptCacheHitTokens    int                      `json:"prompt_cache_hit_tokens,omitempty"`
	PromptCacheMissTokens   int                      `json:"prompt_cache_miss_tokens,omitempty"`
	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"`
}

// CompletionTokensDetails breaks down completion tokens (e.g. reasoning).
type CompletionTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
}

// ChatCompletion is the non-streaming response object.
type ChatCompletion struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	SystemFingerprint string   `json:"system_fingerprint,omitempty"`
	Choices           []Choice `json:"choices"`
	Usage             *Usage   `json:"usage,omitempty"`
}

// Choice is one completion choice.
type Choice struct {
	Index        int             `json:"index"`
	Message      Message         `json:"message"`
	FinishReason string          `json:"finish_reason"`
	Logprobs     json.RawMessage `json:"logprobs,omitempty"`
}

// Text returns the content of the first choice, or "" if there is none.
func (r *ChatCompletion) Text() string {
	if len(r.Choices) == 0 {
		return ""
	}
	return r.Choices[0].Message.Content
}

// Reasoning returns the chain-of-thought of the first choice (thinking mode).
func (r *ChatCompletion) Reasoning() string {
	if len(r.Choices) == 0 {
		return ""
	}
	return r.Choices[0].Message.ReasoningContent
}

// FinishReason returns the finish reason of the first choice.
func (r *ChatCompletion) FinishReason() string {
	if len(r.Choices) == 0 {
		return ""
	}
	return r.Choices[0].FinishReason
}

// ToolCalls returns the tool calls of the first choice.
func (r *ChatCompletion) ToolCalls() []ToolCall {
	if len(r.Choices) == 0 {
		return nil
	}
	return r.Choices[0].Message.ToolCalls
}

// ----- Convenience constructors -------------------------------------------

func SystemMessage(text string) Message    { return Message{Role: RoleSystem, Content: text} }
func UserMessage(text string) Message      { return Message{Role: RoleUser, Content: text} }
func AssistantMessage(text string) Message { return Message{Role: RoleAssistant, Content: text} }

// ToolMessage builds a tool-result message answering a prior tool call.
func ToolMessage(toolCallID, content string) Message {
	return Message{Role: RoleTool, ToolCallID: toolCallID, Content: content}
}

// NewMessageFromJson parses a single DeepSeek message from raw JSON.
func NewMessageFromJson(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
