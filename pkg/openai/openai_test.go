package openai

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
	c := New(Config{APIKey: "sk-test", BaseURL: srv.URL, MaxRetries: -1})
	return c, srv
}

func TestChat(t *testing.T) {
	var gotBody []byte
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Errorf("auth header = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"chatcmpl-1","object":"chat.completion","model":"gpt-4o",
			"choices":[{"index":0,"message":{"role":"assistant","content":"Hi there"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}}`)
	})
	defer srv.Close()

	resp, err := c.Chat(context.Background(), &Request{
		Model:    "gpt-4o",
		Messages: []Message{SystemMessage("be nice"), UserMessage("hello")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text() != "Hi there" {
		t.Errorf("Text() = %q", resp.Text())
	}
	if resp.FinishReason() != FinishStop {
		t.Errorf("finish = %q", resp.FinishReason())
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 5 {
		t.Errorf("usage = %+v", resp.Usage)
	}
	// request must carry stream:false implicitly (omitted) and our messages
	if !strings.Contains(string(gotBody), `"content":"hello"`) {
		t.Errorf("request body = %s", gotBody)
	}
}

func TestContentMarshal(t *testing.T) {
	// string content
	b, _ := json.Marshal(UserMessage("hi"))
	if string(b) != `{"role":"user","content":"hi"}` {
		t.Errorf("string content = %s", b)
	}
	// multimodal content
	b, _ = json.Marshal(UserParts(TextPart("look"), ImageURLPart("http://x/y.png", "low")))
	if !strings.Contains(string(b), `"type":"image_url"`) || !strings.Contains(string(b), `"content":[`) {
		t.Errorf("multi content = %s", b)
	}
	// assistant tool-call message: content omitted, tool_calls present
	idx := 0
	b, _ = json.Marshal(Message{Role: RoleAssistant, ToolCalls: []ToolCall{{
		Index: &idx, ID: "call_1", Type: "function",
		Function: FunctionCall{Name: "f", Arguments: `{"a":1}`},
	}}})
	if strings.Contains(string(b), `"content"`) {
		t.Errorf("tool-call msg should omit content: %s", b)
	}
	// round-trip unmarshal of array content
	var m Message
	if err := json.Unmarshal([]byte(`{"role":"user","content":[{"type":"text","text":"z"}]}`), &m); err != nil {
		t.Fatal(err)
	}
	if len(m.MultiContent) != 1 || m.MultiContent[0].Text != "z" {
		t.Errorf("unmarshal multi = %+v", m)
	}
}

func TestToolChoiceMarshal(t *testing.T) {
	b, _ := json.Marshal(ToolChoiceRequired())
	if string(b) != `"required"` {
		t.Errorf("required = %s", b)
	}
	b, _ = json.Marshal(ToolChoiceFunc("get_weather"))
	if !strings.Contains(string(b), `"name":"get_weather"`) {
		t.Errorf("func = %s", b)
	}
}

func TestStream(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl, _ := w.(http.Flusher)
		chunks := []string{
			`{"id":"c1","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant","content":"Hel"},"finish_reason":null}]}`,
			`{"id":"c1","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{"content":"lo"},"finish_reason":null}]}`,
			// tool call split across two chunks
			`{"id":"c1","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_9","type":"function","function":{"name":"get_weather","arguments":"{\"loc"}}]},"finish_reason":null}]}`,
			`{"id":"c1","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\":\"SF\"}"}}]},"finish_reason":"tool_calls"}]}`,
			`{"id":"c1","object":"chat.completion.chunk","model":"gpt-4o","choices":[],"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`,
		}
		for _, ch := range chunks {
			io.WriteString(w, "data: "+ch+"\n\n")
			if fl != nil {
				fl.Flush()
			}
		}
		io.WriteString(w, "data: [DONE]\n\n")
		if fl != nil {
			fl.Flush()
		}
	})
	defer srv.Close()

	var streamed strings.Builder
	final, err := c.StreamFunc(context.Background(), &Request{
		Model:    "gpt-4o",
		Messages: []Message{UserMessage("hi")},
	}, func(chunk *ChatCompletionChunk) error {
		if len(chunk.Choices) > 0 {
			streamed.WriteString(chunk.Choices[0].Delta.Content)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if streamed.String() != "Hello" {
		t.Errorf("streamed text = %q", streamed.String())
	}
	if final.Text() != "Hello" {
		t.Errorf("accumulated text = %q", final.Text())
	}
	tcs := final.ToolCalls()
	if len(tcs) != 1 {
		t.Fatalf("tool calls = %d", len(tcs))
	}
	if tcs[0].ID != "call_9" || tcs[0].Function.Name != "get_weather" {
		t.Errorf("tool call = %+v", tcs[0])
	}
	if tcs[0].Function.Arguments != `{"loc":"SF"}` {
		t.Errorf("reassembled args = %q", tcs[0].Function.Arguments)
	}
	if final.FinishReason() != FinishToolCalls {
		t.Errorf("finish = %q", final.FinishReason())
	}
	if final.Usage == nil || final.Usage.TotalTokens != 3 {
		t.Errorf("usage = %+v", final.Usage)
	}
}

func TestAPIError(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "req_42")
		w.WriteHeader(http.StatusTooManyRequests)
		io.WriteString(w, `{"error":{"message":"slow down","type":"rate_limit_error","code":"rate_limited"}}`)
	})
	defer srv.Close()

	_, err := c.Chat(context.Background(), &Request{Model: "gpt-4o", Messages: []Message{UserMessage("x")}})
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("err type = %T", err)
	}
	if apiErr.StatusCode != 429 || apiErr.Type != "rate_limit_error" || !apiErr.Retryable() || apiErr.RequestID != "req_42" {
		t.Errorf("apiErr = %+v", apiErr)
	}
}
