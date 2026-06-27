package anthropic

// SSE event types emitted on the Messages streaming endpoint.
const (
	EventMessageStart      = "message_start"
	EventContentBlockStart = "content_block_start"
	EventContentBlockDelta = "content_block_delta"
	EventContentBlockStop  = "content_block_stop"
	EventMessageDelta      = "message_delta"
	EventMessageStop       = "message_stop"
	EventPing              = "ping"
	EventError             = "error"
)

// Delta sub-types carried inside content_block_delta.
const (
	DeltaText      = "text_delta"
	DeltaInputJSON = "input_json_delta"
	DeltaThinking  = "thinking_delta"
	DeltaSignature = "signature_delta"
)

// StreamEvent is one decoded SSE event. A single struct covers every event
// kind; the populated fields depend on Type.
type StreamEvent struct {
	Type string `json:"type"`

	// content_block_* index
	Index int `json:"index,omitempty"`

	// message_start
	Message *Message `json:"message,omitempty"`

	// content_block_start
	ContentBlock *ContentBlock `json:"content_block,omitempty"`

	// content_block_delta / message_delta
	Delta *Delta `json:"delta,omitempty"`

	// message_delta
	Usage *Usage `json:"usage,omitempty"`

	// error
	Error *StreamError `json:"error,omitempty"`
}

// Delta is the polymorphic delta payload for content_block_delta (text /
// input_json / thinking / signature) and message_delta (stop_reason /
// stop_sequence). Fields are mutually exclusive per event.
type Delta struct {
	Type        string `json:"type,omitempty"`
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
	Thinking    string `json:"thinking,omitempty"`
	Signature   string `json:"signature,omitempty"`

	StopReason   string `json:"stop_reason,omitempty"`
	StopSequence string `json:"stop_sequence,omitempty"`
}

// StreamError is the payload of an "error" SSE event.
type StreamError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (e *StreamError) Error() string {
	return "anthropic: stream error: " + e.Type + ": " + e.Message
}

// TextDelta reports an incremental text fragment, if this event carries one.
func (e StreamEvent) TextDelta() (string, bool) {
	if e.Type == EventContentBlockDelta && e.Delta != nil && e.Delta.Type == DeltaText {
		return e.Delta.Text, true
	}
	return "", false
}

// InputJSONDelta reports a tool-input JSON fragment, if present.
func (e StreamEvent) InputJSONDelta() (string, bool) {
	if e.Type == EventContentBlockDelta && e.Delta != nil && e.Delta.Type == DeltaInputJSON {
		return e.Delta.PartialJSON, true
	}
	return "", false
}

// ThinkingDelta reports an incremental thinking fragment, if present.
func (e StreamEvent) ThinkingDelta() (string, bool) {
	if e.Type == EventContentBlockDelta && e.Delta != nil && e.Delta.Type == DeltaThinking {
		return e.Delta.Thinking, true
	}
	return "", false
}
