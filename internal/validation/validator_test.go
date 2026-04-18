package validation

import (
	"testing"

	"github.com/yourusername/kiro-gateway-go/internal/models"
)

func TestValidateRequest(t *testing.T) {
	validator := NewRequestValidator(true, false, false)
	
	tests := []struct {
		name    string
		req     *models.ChatCompletionRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			req: &models.ChatCompletionRequest{
				Model: "claude-sonnet-4",
				Messages: []models.Message{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing model",
			req: &models.ChatCompletionRequest{
				Messages: []models.Message{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr: true,
			errMsg:  "model is required",
		},
		{
			name: "empty messages",
			req: &models.ChatCompletionRequest{
				Model:    "claude-sonnet-4",
				Messages: []models.Message{},
			},
			wantErr: true,
			errMsg:  "at least one message is required",
		},
		{
			name: "last message not from user",
			req: &models.ChatCompletionRequest{
				Model: "claude-sonnet-4",
				Messages: []models.Message{
					{Role: "user", Content: "Hello"},
					{Role: "assistant", Content: "Hi"},
				},
			},
			wantErr: true,
			errMsg:  "last message must be from user",
		},
		{
			name: "message too large",
			req: &models.ChatCompletionRequest{
				Model: "claude-sonnet-4",
				Messages: []models.Message{
					{Role: "user", Content: string(make([]byte, MaxUserMessageChars+1))},
				},
			},
			wantErr: true,
			errMsg:  "user message exceeds maximum size",
		},
		{
			name: "max_tokens too large",
			req: &models.ChatCompletionRequest{
				Model: "claude-sonnet-4",
				Messages: []models.Message{
					{Role: "user", Content: "Hello"},
				},
				MaxTokens: MaxOutputTokens + 1,
			},
			wantErr: true,
			errMsg:  "max_tokens exceeds maximum output tokens for this model (using standard context)",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if valErr, ok := err.(*ValidationError); ok {
					if valErr.Message != tt.errMsg {
						t.Errorf("ValidateRequest() error message = %v, want %v", valErr.Message, tt.errMsg)
					}
				}
			}
		})
	}
}

func TestEstimateTokenCount(t *testing.T) {
	tests := []struct {
		name  string
		chars int
		want  int
	}{
		{"empty", 0, 0},
		{"small", 100, 21},
		{"medium", 1000, 212},
		{"large", 10000, 2127},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokenCount(tt.chars)
			if got != tt.want {
				t.Errorf("EstimateTokenCount(%d) = %d, want %d", tt.chars, got, tt.want)
			}
		})
	}
}

func TestIsSupportedImageFormat(t *testing.T) {
	tests := []struct {
		format string
		want   bool
	}{
		{"png", true},
		{"jpeg", true},
		{"jpg", true},
		{"webp", true},
		{"gif", true},
		{"bmp", false},
		{"tiff", false},
		{"svg", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			got := IsSupportedImageFormat(tt.format)
			if got != tt.want {
				t.Errorf("IsSupportedImageFormat(%s) = %v, want %v", tt.format, got, tt.want)
			}
		})
	}
}
