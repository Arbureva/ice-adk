package deepseek

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

func TestChatReasoning(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"1","object":"chat.completion","model":"deepseek-reasoner",
			"choices":[{"index":0,"message":{"role":"assistant","reasoning_content":"9.8 = 9.80 > 9.11","content":"9.8 is greater"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":10,"completion_tokens":15,"total_tokens":25,"prompt_cache_hit_tokens":2,"prompt_cache_miss_tokens":8,"completion_tokens_details":{"reasoning_tokens":7}}}`)
	})
	defer srv.Close()

	resp, err := c.Chat(context.Background(), &Request{
		Model:    "deepseek-reasoner",
		Messages: []Message{UserMessage("9.11 or 9.8?")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text() != "9.8 is greater" {
		t.Errorf("Text() = %q", resp.Text())
	}
	if resp.Reasoning() != "9.8 = 9.80 > 9.11" {
		t.Errorf("Reasoning() = %q", resp.Reasoning())
	}
	if resp.Usage == nil || resp.Usage.PromptCacheHitTokens != 2 ||
		resp.Usage.CompletionTokensDetails == nil || resp.Usage.CompletionTokensDetails.ReasoningTokens != 7 {
		t.Errorf("usage = %+v", resp.Usage)
	}
}

// reasoning_content must NEVER be re-sent: the API 400s if it is echoed back.
func TestReasoningNotMarshalled(t *testing.T) {
	m := Message{Role: RoleAssistant, Content: "answer", ReasoningContent: "secret cot"}
	b, _ := json.Marshal(m)
	if strings.Contains(string(b), "reasoning_content") || strings.Contains(string(b), "secret cot") {
		t.Errorf("reasoning_content leaked into marshalled message: %s", b)
	}
	// but it must still be readable on unmarshal
	var got Message
	json.Unmarshal([]byte(`{"role":"assistant","content":"a","reasoning_content":"cot"}`), &got)
	if got.ReasoningContent != "cot" {
		t.Errorf("reasoning_content not parsed: %+v", got)
	}
}

func TestStreamReasoning(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl, _ := w.(http.Flusher)
		for _, ch := range []string{
			`{"id":"1","object":"chat.completion.chunk","model":"deepseek-reasoner","choices":[{"index":0,"delta":{"role":"assistant","reasoning_content":"think"},"finish_reason":null}]}`,
			`{"id":"1","object":"chat.completion.chunk","model":"deepseek-reasoner","choices":[{"index":0,"delta":{"reasoning_content":"ing..."},"finish_reason":null}]}`,
			`{"id":"1","object":"chat.completion.chunk","model":"deepseek-reasoner","choices":[{"index":0,"delta":{"content":"Final"},"finish_reason":null}]}`,
			`{"id":"1","object":"chat.completion.chunk","model":"deepseek-reasoner","choices":[{"index":0,"delta":{"content":" answer"},"finish_reason":"stop"}]}`,
		} {
			io.WriteString(w, "data: "+ch+"\n\n")
			if fl != nil {
				fl.Flush()
			}
		}
		io.WriteString(w, "data: [DONE]\n\n")
	})
	defer srv.Close()

	final, err := c.StreamFunc(context.Background(), &Request{
		Model: "deepseek-reasoner", Messages: []Message{UserMessage("hi")},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if final.Reasoning() != "thinking..." {
		t.Errorf("accumulated reasoning = %q", final.Reasoning())
	}
	if final.Text() != "Final answer" {
		t.Errorf("accumulated text = %q", final.Text())
	}
	if final.FinishReason() != FinishStop {
		t.Errorf("finish = %q", final.FinishReason())
	}
}
