package main

import (
	"IceADK/pkg/adapter"
	"IceADK/pkg/chat"
	"IceADK/pkg/openai"
	"context"
	"fmt"

	_ "IceADK/pkg/chat/drivers/openai"
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

	res, err := cli.Chat(context.Background(), adapter.Request{
		Provider: adapter.OpenAI,
		Data: &openai.Request{
			Model: "gpt-oss:20b",
			Messages: []openai.Message{
				openai.UserMessage("Introduce yourself in one sentence."),
			},
		},
	})
	if err != nil {
		panic(err)
	}

	msg, ok := chat.Result(res)
	if !ok {
		panic("not ok")
	}

	fmt.Printf("AI: %s\n| Input: %d | Output: %d | Reasoning: %s |\n", msg.Text, msg.Usage.TotalTokens, msg.Usage.OutputTokens, msg.Reasoning)

	//  AI: I am ChatGPT, a large language model trained by OpenAI, here to help you with information, ideas, and solutions.
	//	| Input: 155 | Output: 82 | Reasoning:  |
}
