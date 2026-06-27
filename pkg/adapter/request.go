package adapter

type Provider string

const (
	OpenAI    Provider = "openai"
	Anthropic Provider = "anthropic"
	Deepseek  Provider = "deepseek"
)

type Request struct {
	Provider Provider      `json:"provider"`
	Data     interface{}   `json:"data"`
	Tools    []interface{} `json:"tools"`
}
