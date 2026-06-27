package anthropic

import (
	"encoding/json"
	"strings"
)

// Roles.
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// Stop reasons returned by the API.
const (
	StopEndTurn      = "end_turn"
	StopMaxTokens    = "max_tokens"
	StopStopSequence = "stop_sequence"
	StopToolUse      = "tool_use"
	StopPauseTurn    = "pause_turn"
	StopRefusal      = "refusal"
)

// Message is used both as an input message (Role + Content) and as the
// top-level response object, which adds id/type/model/stop_reason/usage. The
// response-only fields carry omitempty so the same struct marshals cleanly in
// either direction.
type Message struct {
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content"`

	// Response-only fields (populated by Chat / accumulated by Stream).
	ID           string `json:"id,omitempty"`
	Type         string `json:"type,omitempty"`
	Model        string `json:"model,omitempty"`
	StopReason   string `json:"stop_reason,omitempty"`
	StopSequence string `json:"stop_sequence,omitempty"`
	Usage        *Usage `json:"usage,omitempty"`
}

// Usage reports token accounting for a request/response.
type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// Text concatenates all text blocks in the message, ignoring other blocks.
func (msg *Message) Text() string {
	var b strings.Builder
	for i := range msg.Content {
		if msg.Content[i].Type == BlockText {
			b.WriteString(msg.Content[i].Text)
		}
	}
	return b.String()
}

// Thinking concatenates all thinking blocks in the message.
func (msg *Message) Thinking() string {
	var b strings.Builder
	for i := range msg.Content {
		if msg.Content[i].Type == BlockThinking {
			b.WriteString(msg.Content[i].Thinking)
		}
	}
	return b.String()
}

// ToolUses returns every tool_use block in the message.
func (msg *Message) ToolUses() []ContentBlock {
	var out []ContentBlock
	for i := range msg.Content {
		if msg.Content[i].Type == BlockToolUse {
			out = append(out, msg.Content[i])
		}
	}
	return out
}

// ----- Convenience constructors -------------------------------------------

func UserText(text string) Message {
	return Message{Role: RoleUser, Content: []ContentBlock{TextBlock(text)}}
}
func AssistantText(text string) Message {
	return Message{Role: RoleAssistant, Content: []ContentBlock{TextBlock(text)}}
}
func UserBlocks(blocks ...ContentBlock) Message {
	return Message{Role: RoleUser, Content: blocks}
}
func AssistantBlocks(blocks ...ContentBlock) Message {
	return Message{Role: RoleAssistant, Content: blocks}
}

// NewMessageFromJson parses an Anthropic-format message from raw JSON.
func NewMessageFromJson(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
