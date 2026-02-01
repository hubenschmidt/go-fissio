package core

type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "tool"
)

type Message struct {
	Role       MessageRole `json:"role"`
	Content    string      `json:"content"`
	Name       string      `json:"name,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

func NewSystemMessage(content string) Message {
	return Message{Role: RoleSystem, Content: content}
}

func NewUserMessage(content string) Message {
	return Message{Role: RoleUser, Content: content}
}

func NewAssistantMessage(content string) Message {
	return Message{Role: RoleAssistant, Content: content}
}

func NewToolMessage(toolCallID, content string) Message {
	return Message{Role: RoleTool, Content: content, ToolCallID: toolCallID}
}
