package main

import (
	"IceADK/pkg/adapter"
	"IceADK/pkg/chat"
	_ "IceADK/pkg/chat/drivers/openai"
	"IceADK/pkg/openai"
	"context"
	"fmt"
)

func main() {
	cli := chat.New()

	err := cli.Use(adapter.OpenAI, openai.Config{
		APIKey:  "test",
		BaseURL: "http://studio:11434/v1",
	})
	if err != nil {
		panic(err)
	}

	ch, err := cli.ChatStream(context.Background(), adapter.Request{
		Provider: adapter.OpenAI,
		Data: &openai.Request{
			Model: "gpt-oss:20b",
			Messages: []openai.Message{
				openai.UserMessage("写一篇1000字的文章"),
			},
			Stream: true,
		},
	})
	if err != nil {
		panic(err)
	}

	for c := range ch {
		switch c.Kind {
		case chat.ChunkText:
			fmt.Printf(chat.MustText(&c))

		case chat.ChunkToolCall:
			f := chat.MustToolCall(&c)
			fmt.Printf("Use Tool Call: %s\n", f.Name)

		case chat.ChunkStop:
			fmt.Printf("\nChunkStop: %s\n", chat.MustStop(&c))

		case chat.ChunkUsage:
			usg := chat.MustUsage(&c)
			fmt.Printf("\nUsage: %d\n", usg.TotalTokens)

		case chat.ChunkError:
			err := chat.MustError(&c)
			fmt.Printf("\nError: %s\n", err)
		}
	}
}
