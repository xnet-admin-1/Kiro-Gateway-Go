package tokenizer

import (
	"testing"
)

func TestCountTokens(t *testing.T) {
	tok, err := NewTokenizer("claude-3-5-sonnet-20241022")
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}
	defer tok.Close()
	
	tests := []struct {
		name     string
		text     string
		minCount int
		maxCount int
	}{
		{
			name:     "empty string",
			text:     "",
			minCount: 0,
			maxCount: 0,
		},
		{
			name:     "simple text",
			text:     "Hello, world!",
			minCount: 3,
			maxCount: 5,
		},
		{
			name:     "longer text",
			text:     "The quick brown fox jumps over the lazy dog.",
			minCount: 9,
			maxCount: 12,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := tok.CountTokens(tt.text)
			if count < tt.minCount || count > tt.maxCount {
				t.Errorf("CountTokens(%q) = %d, want between %d and %d", tt.text, count, tt.minCount, tt.maxCount)
			}
		})
	}
}

func TestCountMessageTokens(t *testing.T) {
	tok, err := NewTokenizer("claude-3-5-sonnet-20241022")
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}
	defer tok.Close()
	
	messages := []map[string]interface{}{
		{
			"role":    "user",
			"content": "Hello!",
		},
		{
			"role":    "assistant",
			"content": "Hi there! How can I help you?",
		},
	}
	
	count := tok.CountMessageTokens(messages)
	if count < 10 || count > 30 {
		t.Errorf("CountMessageTokens() = %d, want between 10 and 30", count)
	}
}

func TestGetMaxContextTokens(t *testing.T) {
	tests := []struct {
		model    string
		expected int
	}{
		{"claude-3-5-sonnet-20241022", 200000},
		{"claude-3-opus-20240229", 200000},
		{"unknown-model", 200000},
	}
	
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			max := GetMaxContextTokens(tt.model)
			if max != tt.expected {
				t.Errorf("GetMaxContextTokens(%q) = %d, want %d", tt.model, max, tt.expected)
			}
		})
	}
}

func TestApplyClaudeCorrection(t *testing.T) {
	tests := []struct {
		input    int
		minExpected int
		maxExpected int
	}{
		{100, 114, 116},
		{1000, 1140, 1160},
		{0, 0, 0},
	}
	
	for _, tt := range tests {
		result := ApplyClaudeCorrection(tt.input)
		if result < tt.minExpected || result > tt.maxExpected {
			t.Errorf("ApplyClaudeCorrection(%d) = %d, want between %d and %d", tt.input, result, tt.minExpected, tt.maxExpected)
		}
	}
}

func TestCalculateTokensFromContextUsage(t *testing.T) {
	tests := []struct {
		name              string
		contextPercent    float64
		completionTokens  int
		maxContextTokens  int
		wantPromptTokens  int
		wantTotalTokens   int
	}{
		{
			name:              "50% usage",
			contextPercent:    50.0,
			completionTokens:  100,
			maxContextTokens:  1000,
			wantPromptTokens:  400,
			wantTotalTokens:   500,
		},
		{
			name:              "zero percent",
			contextPercent:    0,
			completionTokens:  100,
			maxContextTokens:  1000,
			wantPromptTokens:  0,
			wantTotalTokens:   100,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			promptTokens, totalTokens := CalculateTokensFromContextUsage(
				tt.contextPercent,
				tt.completionTokens,
				tt.maxContextTokens,
			)
			
			if promptTokens != tt.wantPromptTokens {
				t.Errorf("promptTokens = %d, want %d", promptTokens, tt.wantPromptTokens)
			}
			if totalTokens != tt.wantTotalTokens {
				t.Errorf("totalTokens = %d, want %d", totalTokens, tt.wantTotalTokens)
			}
		})
	}
}
func TestCountToolsTokens(t *testing.T) {
	tok, err := NewTokenizer("claude-3-5-sonnet-20241022")
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}
	defer tok.Close()

	// Test with empty tools
	count := tok.CountToolsTokens(nil)
	if count != 0 {
		t.Errorf("CountToolsTokens(nil) = %d, want 0", count)
	}

	// Test with empty slice
	count = tok.CountToolsTokens([]map[string]interface{}{})
	if count != 0 {
		t.Errorf("CountToolsTokens([]) = %d, want 0", count)
	}

	// Test with mock tools
	tools := []map[string]interface{}{
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "get_weather",
				"description": "Get the current weather",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"location": map[string]interface{}{
							"type":        "string",
							"description": "The city and state",
						},
					},
				},
			},
		},
	}

	count = tok.CountToolsTokens(tools)
	if count <= 0 {
		t.Errorf("CountToolsTokens(tools) = %d, want > 0", count)
	}
}

func TestCountContentTokens(t *testing.T) {
	tok, err := NewTokenizer("claude-3-5-sonnet-20241022")
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}
	defer tok.Close()

	tests := []struct {
		name    string
		content interface{}
		want    int
	}{
		{
			name:    "string content",
			content: "Hello, world!",
			want:    3, // Approximate
		},
		{
			name:    "empty string",
			content: "",
			want:    0,
		},
		{
			name: "array content with text",
			content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Hello, world!",
				},
			},
			want: 3, // Approximate
		},
		{
			name: "array content with image",
			content: []interface{}{
				map[string]interface{}{
					"type": "image_url",
					"image_url": map[string]interface{}{
						"url": "data:image/jpeg;base64,base64data",
					},
				},
			},
			want: 85, // Fixed token count for images
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := tok.countContentTokens(tt.content)
			// Allow some variance for text token counting
			if tt.name == "string content" || tt.name == "array content with text" {
				if count < tt.want-1 || count > tt.want+2 {
					t.Errorf("countContentTokens() = %d, want approximately %d", count, tt.want)
				}
			} else {
				if count != tt.want {
					t.Errorf("countContentTokens() = %d, want %d", count, tt.want)
				}
			}
		})
	}
}

func TestCountToolCallTokens(t *testing.T) {
	tok, err := NewTokenizer("claude-3-5-sonnet-20241022")
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}
	defer tok.Close()

	// Test with nil - still has 5 token overhead
	count := tok.countToolCallTokens(nil)
	if count != 5 {
		t.Errorf("countToolCallTokens(nil) = %d, want 5 (overhead)", count)
	}

	// Test with empty map - still has 5 token overhead
	count = tok.countToolCallTokens(map[string]interface{}{})
	if count != 5 {
		t.Errorf("countToolCallTokens({}) = %d, want 5 (overhead)", count)
	}

	// Test with tool call
	toolCall := map[string]interface{}{
		"id":   "call_123",
		"type": "function",
		"function": map[string]interface{}{
			"name":      "get_weather",
			"arguments": `{"location": "San Francisco"}`,
		},
	}

	count = tok.countToolCallTokens(toolCall)
	if count <= 5 {
		t.Errorf("countToolCallTokens(toolCall) = %d, want > 5", count)
	}
}

func TestClose(t *testing.T) {
	tok, err := NewTokenizer("claude-3-5-sonnet-20241022")
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}

	// Test that Close doesn't panic
	tok.Close()

	// Test that Close can be called multiple times
	tok.Close()
}

func TestNewTokenizer_UnsupportedModel(t *testing.T) {
	// The tokenizer doesn't actually fail for unsupported models,
	// it uses cl100k_base encoding for all models
	tok, err := NewTokenizer("unsupported-model")
	if err != nil {
		t.Errorf("NewTokenizer should not return error for unsupported model: %v", err)
	}
	if tok != nil {
		tok.Close()
	}
}

func TestNewTokenizer_EmptyModel(t *testing.T) {
	tok, err := NewTokenizer("")
	if err != nil {
		t.Fatalf("NewTokenizer should handle empty model: %v", err)
	}
	defer tok.Close()

	// Should use default encoding
	count := tok.CountTokens("Hello")
	if count <= 0 {
		t.Errorf("CountTokens should work with default encoding, got %d", count)
	}
}
