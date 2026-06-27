// Package openai implements the OpenAI Chat Completions API natively.
//
// It speaks the wire protocol directly (no cross-provider abstraction); a
// compatibility/adapter layer is expected to live elsewhere.
//
//	c := openai.New(openai.Config{APIKey: cfg.OpenAI.APIKey})
//
//	// Non-streaming
//	resp, err := c.Chat(ctx, &openai.Request{
//		Model:    "gpt-4o",
//		Messages: []openai.Message{openai.UserMessage("Hello")},
//	})
//	fmt.Println(resp.Text())
//
//	// Streaming
//	stream, err := c.Stream(ctx, &openai.Request{
//		Model:    "gpt-4o",
//		Messages: []openai.Message{openai.UserMessage("Hello")},
//		StreamOptions: &openai.StreamOptions{IncludeUsage: true},
//	})
//	if err != nil { return err }
//	defer stream.Close()
//	for {
//		chunk, err := stream.Recv()
//		if errors.Is(err, io.EOF) { break }
//		if err != nil { return err }
//		if len(chunk.Choices) > 0 { fmt.Print(chunk.Choices[0].Delta.Content) }
//	}
//	final := stream.Completion()
package openai
