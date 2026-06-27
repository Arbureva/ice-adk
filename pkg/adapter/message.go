package adapter

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type MessageAdapter struct {
	Provider Provider    `json:"provider"` // 是什么平台的消息类型？
	Role     Role        `json:"role"`     // 为了快速校验角色
	Data     interface{} `json:"data"`     // 消息载体
}
