package main

import (
	"context"
	"encoding/json"
	"fmt"

	"IceADK/pkg/adapter"
	"IceADK/pkg/chat"
	"IceADK/pkg/openai"
	"IceADK/pkg/tool"

	_ "IceADK/pkg/chat/drivers/openai"
)

// getWeatherArgs is the tool's parameter struct; tool.Reflect turns it into the
// JSON Schema advertised to the model.
type getWeatherArgs struct {
	City  string `json:"city"            desc:"City name, e.g. Shanghai"`
	Units string `json:"units,omitempty" enum:"c,f" desc:"temperature unit"`
}

func main() {
	// 1. Declare tools once. The handler is the lowest-common-denominator shape:
	//    func(ctx, json.RawMessage) (*tool.Result, error). mcp / skills / cli
	//    packages would produce tools implementing the same tool.Tool interface.
	tools := tool.NewSet(tool.Func("get_weather", "Get the current weather for a city",
		tool.Reflect(getWeatherArgs{}),
		func(ctx context.Context, raw json.RawMessage) (*tool.Result, error) {
			var a getWeatherArgs
			if err := json.Unmarshal(raw, &a); err != nil {
				return tool.Errf("bad arguments: %v", err), nil
			}
			// ... real lookup goes here ...
			return tool.Textf("It is 24°C and sunny in %s.", a.City), nil
		}),
	)

	cli := chat.New()
	if err := cli.Use(adapter.OpenAI, openai.Config{
		APIKey:  "test",
		BaseURL: "http://studio:11434/v1",
	}); err != nil {
		panic(err)
	}

	messages := []openai.Message{openai.UserMessage("What's the weather in Shanghai? And what's your name? ")}

	// 2. First call: advertise the tools via adapter.Request.Tools. The driver
	//    renders set.RequestTools() into OpenAI's native function-tool list.
	res, err := cli.Chat(context.Background(), adapter.Request{
		Provider: adapter.OpenAI,
		Data:     &openai.Request{Model: "gpt-oss:20b", Messages: messages},
		Tools:    tools.RequestTools(),
	})
	if err != nil {
		panic(err)
	}
	out, _ := chat.Result(res)

	// 3. If the model asked for tools, run each via the same Set and append the
	//    results as native tool messages, then call again for the final answer.
	if len(out.ToolCalls) > 0 {
		// echo the assistant's tool-call turn back into the history
		asst := openai.Message{Role: openai.RoleAssistant}
		for _, call := range out.ToolCalls {
			asst.ToolCalls = append(asst.ToolCalls, openai.ToolCall{
				ID:       call.ID,
				Type:     "function",
				Function: openai.FunctionCall{Name: call.Name, Arguments: string(call.Args)},
			})
		}
		messages = append(messages, asst)

		for _, call := range out.ToolCalls {
			result, err := tools.Invoke(context.Background(), call.Name, call.Args)
			if err != nil {
				// unknown tool / host failure — surface it to the model as a result
				result = tool.Errf("tool %q could not be run: %v", call.Name, err)
			}
			messages = append(messages, openai.ToolMessage(call.ID, result.Content))
		}

		res, err = cli.Chat(context.Background(), adapter.Request{
			Provider: adapter.OpenAI,
			Data:     &openai.Request{Model: "gpt-oss:20b", Messages: messages},
			Tools:    tools.RequestTools(),
		})
		if err != nil {
			panic(err)
		}
		out, _ = chat.Result(res)
	}

	fmt.Printf("AI: %s\n", out.Text)
}
