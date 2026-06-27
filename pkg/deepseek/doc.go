// Package deepseek implements the DeepSeek chat completions API natively.
//
// The wire shape mirrors OpenAI's chat completions but is modelled
// independently here (no cross-provider abstraction). DeepSeek-specific fields
// are first-class: ReasoningContent (chain-of-thought) on messages and deltas,
// the prompt cache hit/miss token counters in Usage, the Thinking toggle, and
// the insufficient_system_resource finish reason. A compatibility/adapter layer
// is expected to live elsewhere.
//
//	c := deepseek.New(deepseek.Config{APIKey: cfg.DeepSeek.APIKey})
//
//	resp, err := c.Chat(ctx, &deepseek.Request{
//		Model:    "deepseek-reasoner",
//		Messages: []deepseek.Message{deepseek.UserMessage("9.11 or 9.8 — which is greater?")},
//	})
//	fmt.Println(resp.Reasoning()) // chain-of-thought
//	fmt.Println(resp.Text())      // final answer
package deepseek
