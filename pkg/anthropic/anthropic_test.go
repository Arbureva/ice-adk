package anthropic

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient(t *testing.T, h http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	c := New(Config{APIKey: "k", BaseURL: srv.URL, MaxRetries: -1})
	return c, srv
}

func TestChat(t *testing.T) {
	var gotBody []byte
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		if r.Header.Get("X-Api-Key") != "k" || r.Header.Get("Anthropic-Version") != DefaultVersion {
			t.Errorf("headers: key=%q ver=%q", r.Header.Get("X-Api-Key"), r.Header.Get("Anthropic-Version"))
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"msg_1","type":"message","role":"assistant","model":"claude",
			"content":[{"type":"text","text":"Hello!"}],"stop_reason":"end_turn",
			"usage":{"input_tokens":5,"output_tokens":3}}`)
	})
	defer srv.Close()

	msg, err := c.Chat(context.Background(), &Request{
		Model:     "claude-sonnet-4-5",
		MaxTokens: 100,
		System:    "be brief",
		Messages:  []Message{UserText("hi")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if msg.Text() != "Hello!" || msg.StopReason != StopEndTurn {
		t.Errorf("msg = %+v", msg)
	}
	// system projected to top-level string
	if !strings.Contains(string(gotBody), `"system":"be brief"`) {
		t.Errorf("system not marshalled: %s", gotBody)
	}
}

func TestStreamToolUse(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl, _ := w.(http.Flusher)
		events := []struct{ ev, data string }{
			{"message_start", `{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude","content":[],"usage":{"input_tokens":10,"output_tokens":0}}}`},
			{"content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`},
			{"content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Let me check"}}`},
			{"content_block_stop", `{"type":"content_block_stop","index":0}`},
			{"content_block_start", `{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_1","name":"get_weather","input":{}}}`},
			{"content_block_delta", `{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"city\":"}}`},
			{"content_block_delta", `{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"\"SF\"}"}}`},
			{"content_block_stop", `{"type":"content_block_stop","index":1}`},
			{"message_delta", `{"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":15}}`},
			{"message_stop", `{"type":"message_stop"}`},
		}
		for _, e := range events {
			io.WriteString(w, "event: "+e.ev+"\ndata: "+e.data+"\n\n")
			if fl != nil {
				fl.Flush()
			}
		}
	})
	defer srv.Close()

	var text strings.Builder
	final, err := c.StreamFunc(context.Background(), &Request{
		Model: "claude-sonnet-4-5", MaxTokens: 100, Messages: []Message{UserText("weather?")},
	}, func(ev StreamEvent) error {
		if t, ok := ev.TextDelta(); ok {
			text.WriteString(t)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if text.String() != "Let me check" {
		t.Errorf("streamed text = %q", text.String())
	}
	if final.Text() != "Let me check" {
		t.Errorf("accumulated text = %q", final.Text())
	}
	if final.StopReason != StopToolUse {
		t.Errorf("stop = %q", final.StopReason)
	}
	tus := final.ToolUses()
	if len(tus) != 1 {
		t.Fatalf("tool uses = %d", len(tus))
	}
	if tus[0].ID != "toolu_1" || tus[0].Name != "get_weather" {
		t.Errorf("tool use = %+v", tus[0])
	}
	// the tool input JSON must be reassembled across the two deltas
	var input map[string]string
	if err := json.Unmarshal(tus[0].Input, &input); err != nil {
		t.Fatalf("input not valid JSON %q: %v", tus[0].Input, err)
	}
	if input["city"] != "SF" {
		t.Errorf("reassembled input = %v", input)
	}
	if final.Usage == nil || final.Usage.InputTokens != 10 || final.Usage.OutputTokens != 15 {
		t.Errorf("usage = %+v", final.Usage)
	}
}

func TestAPIError(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		io.WriteString(w, `{"type":"error","error":{"type":"overloaded_error","message":"overloaded"}}`)
	})
	defer srv.Close()

	_, err := c.Chat(context.Background(), &Request{Model: "claude", MaxTokens: 10, Messages: []Message{UserText("x")}})
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("err type = %T (%v)", err, err)
	}
	if apiErr.Type != ErrTypeOverloaded || !apiErr.Retryable() {
		t.Errorf("apiErr = %+v", apiErr)
	}
}
