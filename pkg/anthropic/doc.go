// Package anthropic implements the Anthropic Messages API natively.
//
// It speaks the wire protocol directly (content blocks, SSE event stream); no
// cross-provider abstraction is imposed here. A compatibility/adapter layer is
// expected to live elsewhere.
//
//	c := anthropic.New(anthropic.Config{APIKey: cfg.Anthropic.APIKey})
//
//	// Non-streaming
//	msg, err := c.Chat(ctx, &anthropic.Request{
//		Model:     "claude-sonnet-4-5",
//		MaxTokens: 1024,
//		Messages:  []anthropic.Message{anthropic.UserText("Hello")},
//	})
//	fmt.Println(msg.Text())
//
//	// Streaming
//	stream, err := c.Stream(ctx, &anthropic.Request{
//		Model:     "claude-sonnet-4-5",
//		MaxTokens: 1024,
//		Messages:  []anthropic.Message{anthropic.UserText("Hello")},
//	})
//	if err != nil { return err }
//	defer stream.Close()
//	for {
//		ev, err := stream.Recv()
//		if errors.Is(err, io.EOF) { break }
//		if err != nil { return err }
//		if t, ok := ev.TextDelta(); ok { fmt.Print(t) }
//	}
//	final := stream.Message()
package anthropic
