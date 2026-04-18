package converters

import (
	"testing"

	"github.com/yourusername/kiro-gateway-go/internal/models"
)

func TestNormalizeModelID(t *testing.T) {
	tests := []struct {
		name         string
		modelID      string
		hiddenModels []string
		want         string
	}{
		{
			name:         "replace underscores",
			modelID:      "claude_3_5_sonnet",
			hiddenModels: nil,
			want:         "claude-3-5-sonnet",
		},
		{
			name:         "already normalized",
			modelID:      "claude-3-5-sonnet",
			hiddenModels: nil,
			want:         "claude-3-5-sonnet",
		},
		{
			name:         "hidden model",
			modelID:      "claude-3-5-sonnet-v2",
			hiddenModels: []string{"claude-3-5-sonnet-v2:0"},
			want:         "claude-3-5-sonnet-v2:0",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeModelID(tt.modelID, tt.hiddenModels)
			if got != tt.want {
				t.Errorf("NormalizeModelID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertOpenAIToKiro(t *testing.T) {
	req := &models.ChatCompletionRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []models.Message{
			{
				Role:    "system",
				Content: "You are a helpful assistant.",
			},
			{
				Role:    "user",
				Content: "Hello!",
			},
		},
	}
	
	kiroReq, err := ConvertOpenAIToKiro(req, "conv_123", "", true)
	if err != nil {
		t.Fatalf("ConvertOpenAIToKiro() error = %v", err)
	}
	
	if kiroReq.ConversationID != "conv_123" {
		t.Errorf("ConversationID = %v, want conv_123", kiroReq.ConversationID)
	}
	
	if kiroReq.Message.Role != "user" {
		t.Errorf("Message.Role = %v, want user", kiroReq.Message.Role)
	}
	
	if len(kiroReq.Message.Content) == 0 {
		t.Error("Message.Content is empty")
	}
}

func TestExtractTextContent(t *testing.T) {
	tests := []struct {
		name    string
		content interface{}
		want    string
	}{
		{
			name:    "string content",
			content: "Hello, world!",
			want:    "Hello, world!",
		},
		{
			name: "array content",
			content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Hello",
				},
				map[string]interface{}{
					"type": "text",
					"text": " world!",
				},
			},
			want: "Hello world!",
		},
		{
			name:    "nil content",
			content: nil,
			want:    "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTextContent(tt.content)
			if got != tt.want {
				t.Errorf("extractTextContent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertTools(t *testing.T) {
	tools := []models.Tool{
		{
			Type: "function",
			Function: models.FunctionDef{
				Name:        "get_weather",
				Description: "Get weather",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"location": map[string]interface{}{
							"type": "string",
						},
					},
				},
			},
		},
	}
	
	kiroTools := convertTools(tools)
	
	if len(kiroTools) != 1 {
		t.Fatalf("len(kiroTools) = %d, want 1", len(kiroTools))
	}
	
	if kiroTools[0].Name != "get_weather" {
		t.Errorf("kiroTools[0].Name = %v, want get_weather", kiroTools[0].Name)
	}
}
