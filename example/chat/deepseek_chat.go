package main

import (
	"context"
	"os"

	"github.com/Arbureva/ice-adk/pkg/adapter"
	"github.com/Arbureva/ice-adk/pkg/chat"
	"github.com/Arbureva/ice-adk/pkg/deepseek"
)

func DeepseekChat(cli *chat.Client) (*adapter.MessageAdapter, error) {
	err := cli.Use(adapter.Deepseek, deepseek.Config{
		APIKey:  os.Getenv("DEEPSEEK_API_KEY"),
		BaseURL: "https://api.deepseek.com",
	})
	if err != nil {
		return nil, err
	}

	res, err := cli.Chat(context.Background(), adapter.Request{
		Provider: adapter.Deepseek,
		Data: &deepseek.Request{
			Model: "deepseek-v4-flash",
			Messages: []deepseek.Message{
				deepseek.UserMessage("介绍一下你自己？"),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}
