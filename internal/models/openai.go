package models

import "time"

// OpenAI API Types

type ChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Tools       []Tool    `json:"tools,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

type Message struct {
	Role       string      `json:"role"`
	Content    interface{} `json:"content"` // string or []ContentPart
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

type ContentPart struct {
	Type       string    `json:"type"` // "text", "image_url", or "tool_result"
	Text       string    `json:"text,omitempty"`
	ImageURL   *ImageURL `json:"image_url,omitempty"`
	ToolUseID  string    `json:"tool_use_id,omitempty"`  // For tool_result type
	Content    string    `json:"content,omitempty"`      // For tool_result type
	IsError    *bool     `json:"is_error,omitempty"`     // For tool_result type
}

type ImageURL struct {
	URL string `json:"url"`
}

type Tool struct {
	Type     string       `json:"type"` // "function"
	Function FunctionDef  `json:"function"`
}

type FunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type ToolCall struct {
	Index    int          `json:"index"`
	ID       string       `json:"id"`
	Type     string       `json:"type"` // "function"
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Response types

type ChatCompletionResponse struct {
	ID      string                   `json:"id"`
	Object  string                   `json:"object"`
	Created int64                    `json:"created"`
	Model   string                   `json:"model"`
	Choices []ChatCompletionChoice   `json:"choices"`
	Usage   *TokenUsage              `json:"usage,omitempty"`
}

type ChatCompletionChoice struct {
	Index        int              `json:"index"`
	Message      *ResponseMessage `json:"message,omitempty"`
	Delta        *ResponseMessage `json:"delta,omitempty"`
	FinishReason string           `json:"finish_reason,omitempty"`
}

type ResponseMessage struct {
	Role             string     `json:"role,omitempty"`
	Content          string     `json:"content,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
}

type TokenUsage struct {
	PromptTokens     int                    `json:"prompt_tokens"`
	CompletionTokens int                    `json:"completion_tokens"`
	TotalTokens      int                    `json:"total_tokens"`
	CreditsUsed      map[string]interface{} `json:"credits_used,omitempty"`
}

// Model list types

type ModelList struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

type Model struct {
	ID          string `json:"id"`
	Object      string `json:"object"`
	Created     int64  `json:"created"`
	OwnedBy     string `json:"owned_by"`
	Description string `json:"description,omitempty"`
}

// Helper to create model list
func NewModelList(modelIDs []string) ModelList {
	models := make([]Model, len(modelIDs))
	created := time.Now().Unix()
	
	for i, id := range modelIDs {
		models[i] = Model{
			ID:          id,
			Object:      "model",
			Created:     created,
			OwnedBy:     "anthropic",
			Description: "Claude model via Kiro API",
		}
	}
	
	return ModelList{
		Object: "list",
		Data:   models,
	}
}
