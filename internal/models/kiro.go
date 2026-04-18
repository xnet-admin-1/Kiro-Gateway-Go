package models

// DEPRECATED: Legacy Kiro API Types
// These types are DEPRECATED as of 2026-01-25 and will be removed in the next major version.
// They produce KiroRequest format which is NOT compatible with AWS Q Developer API.
//
// Use ConversationStateRequest and related types instead (defined in conversation.go).
//
// Migration Guide:
// - KiroRequest → ConversationStateRequest
// - KiroMessage → UserInputMessage
// - KiroContentPart → Use flat content string + separate Images array
// - KiroTool → ToolSpecWrapper

// DEPRECATED: Use ConversationStateRequest instead
type KiroRequest struct {
	ConversationID string       `json:"conversationId"`
	ProfileArn     string       `json:"profileArn,omitempty"`
	Message        KiroMessage  `json:"message"`
	Tools          []KiroTool   `json:"tools,omitempty"`
}

// DEPRECATED: Use UserInputMessage instead
type KiroMessage struct {
	Role    string             `json:"role"`
	Content []KiroContentPart  `json:"content"`
}

// DEPRECATED: Use flat content string + Images array instead
type KiroContentPart struct {
	Type string `json:"type"` // "text", "image", "tool_use", "tool_result"
	
	// For text
	Text string `json:"text,omitempty"`
	
	// For image
	Source *KiroImageSource `json:"source,omitempty"`
	
	// For tool_use
	ID    string                 `json:"id,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`
	
	// For tool_result
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`
}

// DEPRECATED: Use ImageSource instead
type KiroImageSource struct {
	Type      string `json:"type"` // "base64"
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// DEPRECATED: Use ToolSpecWrapper instead
type KiroTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// Kiro SSE Event Types

type KiroEvent struct {
	Type string `json:"type"`
	
	// For contentBlockDelta
	ContentBlockDelta *ContentBlockDelta `json:"contentBlockDelta,omitempty"`
	
	// For messageStart
	Message *KiroMessageStart `json:"message,omitempty"`
	
	// For contentBlockStart
	ContentBlock *ContentBlock `json:"contentBlock,omitempty"`
	
	// For messageStop
	StopReason string `json:"stopReason,omitempty"`
	
	// For metadata
	Metadata *KiroMetadata `json:"metadata,omitempty"`
}

type ContentBlockDelta struct {
	Delta *Delta `json:"delta,omitempty"`
}

type Delta struct {
	Type string `json:"type"` // "text_delta", "thinking_delta", "input_json_delta"
	Text string `json:"text,omitempty"`
	
	// For thinking
	Thinking string `json:"thinking,omitempty"`
	
	// For tool input
	PartialJSON string `json:"partial_json,omitempty"`
}

type KiroMessageStart struct {
	Role  string `json:"role"`
	Model string `json:"model,omitempty"`
}

type ContentBlock struct {
	Type string `json:"type"` // "text", "tool_use"
	
	// For tool_use
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type KiroMetadata struct {
	Usage              *KiroUsage `json:"usage,omitempty"`
	ContextUsage       *ContextUsage `json:"contextUsage,omitempty"`
	MeteringData       map[string]interface{} `json:"meteringData,omitempty"`
}

type KiroUsage struct {
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
}

type ContextUsage struct {
	Percentage float64 `json:"percentage"`
}
