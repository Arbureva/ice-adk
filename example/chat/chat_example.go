package main

import (
	"fmt"

	"github.com/Arbureva/ice-adk/pkg/chat"
	_ "github.com/Arbureva/ice-adk/pkg/chat/drivers/anthropic"
	_ "github.com/Arbureva/ice-adk/pkg/chat/drivers/deepseek"
	_ "github.com/Arbureva/ice-adk/pkg/chat/drivers/openai"
)

func main() {
	cli := chat.New()

	// res, err := OpenAiChat(cli)
	//res, err := DeepseekChat(cli)
	res, err := AnthropicChat(cli)
	if err != nil {
		panic(err)
	}

	msg, ok := chat.Result(res)
	if !ok {
		panic("not ok")
	}

	fmt.Printf("AI: %s\n| Input: %d | Output: %d | Reasoning: %s |\n", msg.Text, msg.Usage.TotalTokens, msg.Usage.OutputTokens, msg.Reasoning)
}
