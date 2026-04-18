package converters

import (
	"testing"

	"github.com/yourusername/kiro-gateway-go/internal/models"
)

// TestNormalizeModelID_Comprehensive provides comprehensive table-driven tests for model ID normalization
func TestNormalizeModelID_Comprehensive(t *testing.T) {
	tests := []struct {
		name         string
		modelID      string
		hiddenModels []string
		want         string
		description  string
	}{
		{
			name:         "replace single underscore",
			modelID:      "claude_sonnet",
			hiddenModels: nil,
			want:         "claude-sonnet",
			description:  "Single underscore should be replaced with hyphen",
		},
		{
			name:         "replace multiple underscores",
			modelID:      "claude_3_5_sonnet_20241022",
			hiddenModels: nil,
			want:         "claude-3-5-sonnet-20241022",
			description:  "All underscores should be replaced with hyphens",
		},
		{
			name:         "already normalized with hyphens",
			modelID:      "claude-3-5-sonnet-20241022",
			hiddenModels: nil,
			want:         "claude-3-5-sonnet-20241022",
			description:  "Already normalized model IDs should remain unchanged",
		},
		{
			name:         "mixed underscores and hyphens",
			modelID:      "claude_3-5_sonnet",
			hiddenModels: nil,
			want:         "claude-3-5-sonnet",
			description:  "Mixed separators should be normalized to hyphens",
		},
		{
			name:         "empty model ID",
			modelID:      "",
			hiddenModels: nil,
			want:         "",
			description:  "Empty model ID should return empty string",
		},
		{
			name:         "model ID with numbers",
			modelID:      "gpt_4_turbo_preview",
			hiddenModels: nil,
			want:         "gpt-4-turbo-preview",
			description:  "Model IDs with numbers should be normalized",
		},
		{
			name:         "hidden model exact match",
			modelID:      "claude-3-5-sonnet",
			hiddenModels: []string{"claude-3-5-sonnet", "gpt-4"},
			want:         "claude-3-5-sonnet",
			description:  "Exact match in hidden models should return as-is",
		},
		{
			name:         "hidden model with version suffix",
			modelID:      "claude-haiku",
			hiddenModels: []string{"claude-haiku:v1", "claude-haiku:v2"},
			want:         "claude-haiku:v1",
			description:  "Should return first matching hidden model with version",
		},
		{
			name:         "no hidden models provided",
			modelID:      "claude_opus",
			hiddenModels: []string{},
			want:         "claude-opus",
			description:  "Empty hidden models list should just normalize",
		},
		{
			name:         "nil hidden models",
			modelID:      "claude_opus",
			hiddenModels: nil,
			want:         "claude-opus",
			description:  "Nil hidden models should just normalize",
		},
		{
			name:         "no matching hidden models",
			modelID:      "claude-opus",
			hiddenModels: []string{"claude-sonnet:v1", "gpt-4:v1"},
			want:         "claude-opus",
			description:  "No matching hidden models should return normalized ID",
		},
		{
			name:         "case sensitive matching",
			modelID:      "Claude_3_5_Sonnet",
			hiddenModels: []string{"claude-3-5-sonnet"},
			want:         "Claude-3-5-Sonnet",
			description:  "Case should be preserved in normalization",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeModelID(tt.modelID, tt.hiddenModels)
			if got != tt.want {
				t.Errorf("NormalizeModelID(%q, %v) = %q, want %q (%s)", 
					tt.modelID, tt.hiddenModels, got, tt.want, tt.description)
			}
		})
	}
}

// TestExtractTextContent_Comprehensive provides comprehensive table-driven tests for text content extraction
func TestExtractTextContent_Comprehensive(t *testing.T) {
	tests := []struct {
		name        string
		content     interface{}
		want        string
		description string
	}{
		{
			name:        "simple string content",
			content:     "Hello, world!",
			want:        "Hello, world!",
			description: "Simple string should be returned as-is",
		},
		{
			name:        "empty string content",
			content:     "",
			want:        "",
			description: "Empty string should return empty",
		},
		{
			name:        "nil content",
			content:     nil,
			want:        "",
			description: "Nil content should return empty string",
		},
		{
			name: "array with single text block",
			content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Single block",
				},
			},
			want:        "Single block",
			description: "Single text block should extract text",
		},
		{
			name: "array with multiple text blocks",
			content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Hello",
				},
				map[string]interface{}{
					"type": "text",
					"text": " world",
				},
				map[string]interface{}{
					"type": "text",
					"text": "!",
				},
			},
			want:        "Hello world!",
			description: "Multiple text blocks should be concatenated",
		},
		{
			name: "array with mixed content types",
			content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Start",
				},
				map[string]interface{}{
					"type": "image",
					"url":  "http://example.com/image.jpg",
				},
				map[string]interface{}{
					"type": "text",
					"text": " End",
				},
			},
			want:        "Start End",
			description: "Non-text blocks should be ignored",
		},
		{
			name: "array with only non-text blocks",
			content: []interface{}{
				map[string]interface{}{
					"type": "image",
					"url":  "http://example.com/image.jpg",
				},
				map[string]interface{}{
					"type": "video",
					"url":  "http://example.com/video.mp4",
				},
			},
			want:        "",
			description: "Only non-text blocks should return empty",
		},
		{
			name: "array with missing text field",
			content: []interface{}{
				map[string]interface{}{
					"type": "text",
				},
			},
			want:        "",
			description: "Missing text field should be ignored",
		},
		{
			name: "array with non-string text field",
			content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": 123,
				},
			},
			want:        "",
			description: "Non-string text field should be ignored",
		},
		{
			name: "array with empty text blocks",
			content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "",
				},
				map[string]interface{}{
					"type": "text",
					"text": "",
				},
			},
			want:        "",
			description: "Empty text blocks should result in empty string",
		},
		{
			name: "array with whitespace text",
			content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "  ",
				},
				map[string]interface{}{
					"type": "text",
					"text": "\n\t",
				},
			},
			want:        "  \n\t",
			description: "Whitespace should be preserved",
		},
		{
			name:        "empty array",
			content:     []interface{}{},
			want:        "",
			description: "Empty array should return empty string",
		},
		{
			name:        "non-string, non-array content",
			content:     42,
			want:        "",
			description: "Invalid content type should return empty string",
		},
		{
			name:        "boolean content",
			content:     true,
			want:        "",
			description: "Boolean content should return empty string",
		},
		{
			name: "array with invalid block structure",
			content: []interface{}{
				"not a map",
				123,
				map[string]interface{}{
					"type": "text",
					"text": "valid",
				},
			},
			want:        "valid",
			description: "Invalid blocks should be ignored, valid ones processed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTextContent(tt.content)
			if got != tt.want {
				t.Errorf("extractTextContent(%v) = %q, want %q (%s)", 
					tt.content, got, tt.want, tt.description)
			}
		})
	}
}

// TestConvertTools_Comprehensive provides comprehensive table-driven tests for tool conversion
func TestConvertTools_Comprehensive(t *testing.T) {
	tests := []struct {
		name        string
		tools       []models.Tool
		wantCount   int
		wantFirst   string
		description string
	}{
		{
			name:        "empty tools array",
			tools:       []models.Tool{},
			wantCount:   0,
			wantFirst:   "",
			description: "Empty tools should return empty array",
		},
		{
			name:        "nil tools array",
			tools:       nil,
			wantCount:   0,
			wantFirst:   "",
			description: "Nil tools should return empty array",
		},
		{
			name: "single function tool",
			tools: []models.Tool{
				{
					Type: "function",
					Function: models.FunctionDef{
						Name:        "get_weather",
						Description: "Get current weather",
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
			},
			wantCount:   1,
			wantFirst:   "get_weather",
			description: "Single function tool should be converted",
		},
		{
			name: "multiple function tools",
			tools: []models.Tool{
				{
					Type: "function",
					Function: models.FunctionDef{
						Name:        "get_weather",
						Description: "Get weather",
					},
				},
				{
					Type: "function",
					Function: models.FunctionDef{
						Name:        "get_time",
						Description: "Get current time",
					},
				},
			},
			wantCount:   2,
			wantFirst:   "get_weather",
			description: "Multiple tools should all be converted",
		},
		{
			name: "tool with complex parameters",
			tools: []models.Tool{
				{
					Type: "function",
					Function: models.FunctionDef{
						Name:        "search_database",
						Description: "Search in database",
						Parameters: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"query": map[string]interface{}{
									"type":        "string",
									"description": "Search query",
								},
								"limit": map[string]interface{}{
									"type":    "integer",
									"minimum": 1,
									"maximum": 100,
								},
								"filters": map[string]interface{}{
									"type": "array",
									"items": map[string]interface{}{
										"type": "string",
									},
								},
							},
							"required": []string{"query"},
						},
					},
				},
			},
			wantCount:   1,
			wantFirst:   "search_database",
			description: "Complex parameters should be preserved",
		},
		{
			name: "tool with empty name",
			tools: []models.Tool{
				{
					Type: "function",
					Function: models.FunctionDef{
						Name:        "",
						Description: "Tool with empty name",
					},
				},
			},
			wantCount:   1,
			wantFirst:   "",
			description: "Empty name should be preserved",
		},
		{
			name: "tool with empty description",
			tools: []models.Tool{
				{
					Type: "function",
					Function: models.FunctionDef{
						Name:        "test_tool",
						Description: "",
					},
				},
			},
			wantCount:   1,
			wantFirst:   "test_tool",
			description: "Empty description should be allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertTools(tt.tools)
			
			if len(got) != tt.wantCount {
				t.Errorf("convertTools() returned %d tools, want %d (%s)", 
					len(got), tt.wantCount, tt.description)
				return
			}
			
			if tt.wantCount > 0 && got[0].Name != tt.wantFirst {
				t.Errorf("convertTools() first tool name = %q, want %q (%s)", 
					got[0].Name, tt.wantFirst, tt.description)
			}
		})
	}
}
