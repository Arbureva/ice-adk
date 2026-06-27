<div align="center">

# ❄️ IceADK

**一个标准、易用的 Go 语言 Agent 开发套件（ADK）。**

[![Go Reference](https://pkg.go.dev/badge/github.com/Arbureva/ice-adk.svg)](https://pkg.go.dev/github.com/Arbureva/ice-adk) [![Go Version](https://img.shields.io/github/go-mod/go-version/Arbureva/ice-adk)](go.mod) [![Go Report Card](https://goreportcard.com/badge/github.com/Arbureva/ice-adk)](https://goreportcard.com/report/github.com/Arbureva/ice-adk) [![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE) ![Dependencies](https://img.shields.io/badge/dependencies-0-success) ![Status](https://img.shields.io/badge/status-active%20development-orange)

![OpenAI](https://img.shields.io/badge/OpenAI-supported-412991) ![Anthropic](https://img.shields.io/badge/Anthropic-supported-D4A27F) ![DeepSeek](https://img.shields.io/badge/DeepSeek-supported-4D6BFE)

[English](README.md) · **简体中文**

</div>

---

IceADK 为 Go 应用提供一种统一、干净的方式来对接大语言模型。它把每家厂商**原生、忠于协议的客户端**，与一个对请求、响应、流式、工具调用做归一化的**统一 chat 层**配合在一起——业务代码只写一次，即可在 OpenAI、Anthropic、DeepSeek 之间无改动切换。

它采用 `database/sql` 的驱动模型：核心不导入任何厂商包，应用通过空白导入（blank import）按需启用驱动。整个套件**仅依赖标准库——零第三方依赖。**

```go
cli := chat.New()
_ = cli.Use(adapter.OpenAI, openai.Config{APIKey: key})

msg, _ := cli.Chat(ctx, adapter.Request{
    Provider: adapter.OpenAI,
    Data:     &openai.Request{Model: "gpt-4o", Messages: []openai.Message{openai.UserMessage("hi")}},
})
out, _ := chat.Result(msg)
fmt.Println(out.Text)
```

## ✨ 为什么选择 IceADK

- **原生包，不做有损抽象。** `pkg/openai`、`pkg/anthropic`、`pkg/deepseek` 各自直接讲自己的线缆协议（content block、SSE 事件、`reasoning_content` 等）。既可单独使用，也可通过统一层使用——由你决定，而非框架强加。
- **一个入口，三家厂商。** `pkg/chat` 按 `Provider` 路由请求，返回归一化结果。换模型是改配置，不是重写代码。
- **驱动注册式接入。** 厂商驱动在 `init()` 中自注册，你用空白导入启用。无 build tag，无中心化的 switch，新增后端无需改动核心。
- **工具一次声明，处处可用。** 用极简的 `func(ctx, json.RawMessage) (*tool.Result, error)` 接口声明一个工具，各驱动会把它渲染成对应厂商的原生工具格式——同一套工具集驱动三家的 function calling。
- **流式归一化。** 无论哪个后端，都是一条带类型的 chunk 通道（`text`、`thinking`、`tool_call`、`stop`、`usage`、`error`）。
- **配置文件友好。** 厂商配置是带 `snake_case` JSON tag 的普通结构体；`Use` 也接受从应用配置直接解码出的原始 JSON。

## 📦 安装

```bash
go get github.com/Arbureva/ice-adk
```

需要 Go 1.25+。

## 🗂 包结构

| 包 | 职责 |
| --- | --- |
| `pkg/chat` | 统一入口——`Client`、`Chat`、`ChatStream`、`Result` 及 chunk 辅助函数。不导入任何厂商包。 |
| `pkg/chat/drivers/{openai,anthropic,deepseek}` | 厂商驱动桥接。空白导入即启用，各自在 `init()` 中自注册。 |
| `pkg/adapter` | 中立信封——`Request`、`MessageAdapter`、`ChunkMessageAdapter` 以及 `Provider` 常量。 |
| `pkg/openai` · `pkg/anthropic` · `pkg/deepseek` | 原生协议客户端，可独立使用。 |
| `pkg/tool` | 与厂商无关的工具抽象——`Tool`、`Func`、`Reflect`、`Set`、`Result`。 |
| `pkg/ecode` | 共享的哨兵错误。 |

## 🚀 快速上手

### 配置厂商

用空白导入启用后端，再 `Use` 它：

```go
import (
    "github.com/Arbureva/ice-adk/pkg/adapter"
    "github.com/Arbureva/ice-adk/pkg/chat"
    "github.com/Arbureva/ice-adk/pkg/openai"

    _ "github.com/Arbureva/ice-adk/pkg/chat/drivers/openai"   // 注册 openai 驱动
    _ "github.com/Arbureva/ice-adk/pkg/chat/drivers/anthropic"
    _ "github.com/Arbureva/ice-adk/pkg/chat/drivers/deepseek"
)

cli := chat.New()
_ = cli.Use(adapter.OpenAI, openai.Config{APIKey: key, BaseURL: "https://api.openai.com/v1"})
```

### 非流式

```go
msg, _ := cli.Chat(ctx, adapter.Request{
    Provider: adapter.OpenAI,
    Data:     &openai.Request{Model: "gpt-4o", Messages: []openai.Message{openai.UserMessage("介绍一下你自己。")}},
})
out, _ := chat.Result(msg) // *chat.Completion：Text / Reasoning / ToolCalls / StopReason / Usage / Raw
fmt.Println(out.Text)
```

### 流式

```go
ch, _ := cli.ChatStream(ctx, adapter.Request{Provider: adapter.OpenAI, Data: req})
for c := range ch {
    switch c.Kind {
    case chat.ChunkText:
        fmt.Print(chat.MustText(&c))
    case chat.ChunkThinking:
        fmt.Print(chat.MustThinking(&c))
    case chat.ChunkUsage:
        fmt.Printf("\nUsage: %d\n", chat.MustUsage(&c).TotalTokens)
    case chat.ChunkError:
        return chat.MustError(&c)
    }
}
```

### 工具调用

工具只声明一次；同一个 `Set` 既用于向模型暴露工具，也用于派发模型发起的调用：

```go
type weatherArgs struct {
    City string `json:"city" desc:"城市名，例如 Shanghai"`
}

tools := tool.NewSet(tool.Func("get_weather", "查询某城市的当前天气",
    tool.Reflect(weatherArgs{}),
    func(ctx context.Context, raw json.RawMessage) (*tool.Result, error) {
        var a weatherArgs
        if err := json.Unmarshal(raw, &a); err != nil {
            return tool.Errf("参数有误: %v", err), nil
        }
        return tool.Textf("%s 当前 24°C，晴。", a.City), nil
    }))

msg, _ := cli.Chat(ctx, adapter.Request{
    Provider: adapter.OpenAI,
    Data:     &openai.Request{Model: "gpt-4o", Messages: msgs},
    Tools:    tools.RequestTools(),
})
out, _ := chat.Result(msg)
for _, call := range out.ToolCalls {
    res, _ := tools.Invoke(ctx, call.Name, call.Args)
    // 把 res.Content 作为该厂商的 tool / tool_result 消息回填，再发起下一轮请求
}
```

`chat.Result` 会把所有厂商的工具调用都归一成 `[]chat.ToolCall`，因此你的派发循环在各后端之间完全一致。只有“回填消息的重建”是厂商相关的（OpenAI/DeepSeek 用 `tool` 角色消息，Anthropic 用 `tool_use` / `tool_result` 内容块）。

## 📂 示例

可运行示例位于 [`example/`](example/) 下：

- [`example/chat`](example/chat) —— 非流式对话，每个厂商一个文件。
- [`example/chat-stream`](example/chat-stream) —— 流式，每个厂商一个文件。
- [`example/chat-tool`](example/chat-tool) —— 两轮工具调用循环，每个厂商一个文件。

## 🗺 路线图

IceADK 致力于成为一个完整、标准的 Go 语言 ADK。基础能力已经就位；上层能力都会包裹现有的 `tool.Tool` 接口与 `chat` 入口，因此引入它们无需改动已基于 IceADK 写好的代码。

- [x] 原生厂商客户端 —— OpenAI · Anthropic · DeepSeek
- [x] 带驱动注册的统一 chat 入口
- [x] 归一化 chunk 的流式
- [x] 工具调用
- [ ] **Agent** —— 在工具层之上实现带状态管理的 ReAct 循环
- [ ] **MCP** —— 将 Model Context Protocol 工具作为一等的 `tool.Tool` 实现接入
- [ ] **Skills** —— 由工具与提示词组合而成、可复用的封装能力
- [ ] **Cli** —— 用于运行、观察 Agent 与工具的命令行驱动

## 📄 许可证

基于 [MIT 许可证](LICENSE) 发布。

---

<div align="center">

[English](README.md) · **简体中文**

</div>