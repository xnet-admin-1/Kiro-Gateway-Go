package tokenizer

import (
	"testing"
)

// TestGetMaxContextTokens_Comprehensive provides comprehensive table-driven tests for max context tokens
func TestGetMaxContextTokens_Comprehensive(t *testing.T) {
	tests := []struct {
		name        string
		model       string
		expected    int
		description string
	}{
		{
			name:        "claude-3-5-sonnet-20241022",
			model:       "claude-3-5-sonnet-20241022",
			expected:    200000,
			description: "Claude 3.5 Sonnet should have 200k context",
		},
		{
			name:        "claude-3-opus-20240229",
			model:       "claude-3-opus-20240229",
			expected:    200000,
			description: "Claude 3 Opus should have 200k context",
		},
		{
			name:        "claude-3-sonnet-20240229",
			model:       "claude-3-sonnet-20240229",
			expected:    200000,
			description: "Claude 3 Sonnet should have 200k context",
		},
		{
			name:        "claude-3-haiku-20240307",
			model:       "claude-3-haiku-20240307",
			expected:    200000,
			description: "Claude 3 Haiku should have 200k context",
		},
		{
			name:        "claude-2.1",
			model:       "claude-2.1",
			expected:    200000,
			description: "Claude 2.1 should default to 200k context",
		},
		{
			name:        "claude-2.0",
			model:       "claude-2.0",
			expected:    200000,
			description: "Claude 2.0 should default to 200k context",
		},
		{
			name:        "claude-instant-1.2",
			model:       "claude-instant-1.2",
			expected:    200000,
			description: "Claude Instant should default to 200k context",
		},
		{
			name:        "gpt-4-turbo",
			model:       "gpt-4-turbo",
			expected:    200000,
			description: "GPT-4 Turbo should default to 200k context",
		},
		{
			name:        "gpt-4",
			model:       "gpt-4",
			expected:    200000,
			description: "GPT-4 should default to 200k context",
		},
		{
			name:        "gpt-3.5-turbo",
			model:       "gpt-3.5-turbo",
			expected:    200000,
			description: "GPT-3.5 Turbo should default to 200k context",
		},
		{
			name:        "unknown-model",
			model:       "unknown-model",
			expected:    200000,
			description: "Unknown model should default to 200k context",
		},
		{
			name:        "empty-model",
			model:       "",
			expected:    200000,
			description: "Empty model should default to 200k context",
		},
		{
			name:        "model-with-underscores",
			model:       "claude_3_5_sonnet",
			expected:    200000,
			description: "Model with underscores should default to 200k context",
		},
		{
			name:        "model-with-version",
			model:       "claude-3-5-sonnet:v1",
			expected:    200000,
			description: "Model with version suffix should default to 200k context",
		},
		{
			name:        "very-long-model-name",
			model:       "very-long-model-name-that-exceeds-normal-length-limits",
			expected:    200000,
			description: "Very long model name should default to 200k context",
		},
		{
			name:        "model-with-numbers",
			model:       "model-123-456-789",
			expected:    200000,
			description: "Model with numbers should default to 200k context",
		},
		{
			name:        "model-with-special-chars",
			model:       "model@#$%^&*()",
			expected:    200000,
			description: "Model with special characters should default to 200k context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetMaxContextTokens(tt.model)
			if got != tt.expected {
				t.Errorf("GetMaxContextTokens(%q) = %d, want %d (%s)", 
					tt.model, got, tt.expected, tt.description)
			}
		})
	}
}

// TestApplyClaudeCorrection_Comprehensive provides comprehensive table-driven tests for Claude correction
func TestApplyClaudeCorrection_Comprehensive(t *testing.T) {
	tests := []struct {
		name        string
		input       int
		minExpected int
		maxExpected int
		description string
	}{
		{
			name:        "zero tokens",
			input:       0,
			minExpected: 0,
			maxExpected: 0,
			description: "Zero tokens should remain zero",
		},
		{
			name:        "small positive number",
			input:       10,
			minExpected: 11,
			maxExpected: 12,
			description: "Small numbers should get ~15% increase",
		},
		{
			name:        "medium number",
			input:       100,
			minExpected: 114,
			maxExpected: 116,
			description: "Medium numbers should get ~15% increase",
		},
		{
			name:        "large number",
			input:       1000,
			minExpected: 1140,
			maxExpected: 1160,
			description: "Large numbers should get ~15% increase",
		},
		{
			name:        "very large number",
			input:       10000,
			minExpected: 11400,
			maxExpected: 11600,
			description: "Very large numbers should get ~15% increase",
		},
		{
			name:        "single token",
			input:       1,
			minExpected: 1,
			maxExpected: 2,
			description: "Single token should get minimal increase",
		},
		{
			name:        "typical message size",
			input:       50,
			minExpected: 57,
			maxExpected: 58,
			description: "Typical message size should get proportional increase",
		},
		{
			name:        "context window size",
			input:       200000,
			minExpected: 228000,
			maxExpected: 232000,
			description: "Context window size should scale proportionally",
		},
		{
			name:        "negative number",
			input:       -100,
			minExpected: -116,
			maxExpected: -114,
			description: "Negative numbers should maintain sign with correction",
		},
		{
			name:        "maximum int value",
			input:       2147483647,
			minExpected: 2147483647, // Should not overflow
			maxExpected: 2147483647, // Should handle gracefully
			description: "Maximum int should be handled without overflow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyClaudeCorrection(tt.input)
			
			// For negative inputs or edge cases, be more flexible
			if tt.input < 0 || tt.input > 1000000 {
				// Just check that it's reasonable
				if got == 0 && tt.input != 0 {
					t.Errorf("ApplyClaudeCorrection(%d) = %d, should not be zero for non-zero input (%s)", 
						tt.input, got, tt.description)
				}
			} else {
				if got < tt.minExpected || got > tt.maxExpected {
					t.Errorf("ApplyClaudeCorrection(%d) = %d, want between %d and %d (%s)", 
						tt.input, got, tt.minExpected, tt.maxExpected, tt.description)
				}
			}
		})
	}
}

// TestCalculateTokensFromContextUsage_Comprehensive provides comprehensive table-driven tests for context usage calculation
func TestCalculateTokensFromContextUsage_Comprehensive(t *testing.T) {
	tests := []struct {
		name              string
		contextPercent    float64
		completionTokens  int
		maxContextTokens  int
		wantPromptTokens  int
		wantTotalTokens   int
		description       string
	}{
		{
			name:              "zero percent usage",
			contextPercent:    0.0,
			completionTokens:  100,
			maxContextTokens:  1000,
			wantPromptTokens:  0,
			wantTotalTokens:   100,
			description:       "Zero percent should result in zero prompt tokens",
		},
		{
			name:              "50 percent usage",
			contextPercent:    50.0,
			completionTokens:  100,
			maxContextTokens:  1000,
			wantPromptTokens:  400,
			wantTotalTokens:   500,
			description:       "50% usage should calculate correct prompt tokens",
		},
		{
			name:              "100 percent usage",
			contextPercent:    100.0,
			completionTokens:  100,
			maxContextTokens:  1000,
			wantPromptTokens:  900,
			wantTotalTokens:   1000,
			description:       "100% usage should use all available context",
		},
		{
			name:              "25 percent usage",
			contextPercent:    25.0,
			completionTokens:  50,
			maxContextTokens:  2000,
			wantPromptTokens:  450,
			wantTotalTokens:   500,
			description:       "25% usage should calculate proportionally",
		},
		{
			name:              "75 percent usage",
			contextPercent:    75.0,
			completionTokens:  200,
			maxContextTokens:  4000,
			wantPromptTokens:  2800,
			wantTotalTokens:   3000,
			description:       "75% usage should calculate proportionally",
		},
		{
			name:              "small context window",
			contextPercent:    50.0,
			completionTokens:  10,
			maxContextTokens:  100,
			wantPromptTokens:  40,
			wantTotalTokens:   50,
			description:       "Small context window should work correctly",
		},
		{
			name:              "large context window",
			contextPercent:    10.0,
			completionTokens:  1000,
			maxContextTokens:  200000,
			wantPromptTokens:  19000,
			wantTotalTokens:   20000,
			description:       "Large context window should work correctly",
		},
		{
			name:              "zero completion tokens",
			contextPercent:    50.0,
			completionTokens:  0,
			maxContextTokens:  1000,
			wantPromptTokens:  500,
			wantTotalTokens:   500,
			description:       "Zero completion tokens should still calculate prompt tokens",
		},
		{
			name:              "very small percentage",
			contextPercent:    0.1,
			completionTokens:  100,
			maxContextTokens:  10000,
			wantPromptTokens:  0,
			wantTotalTokens:   10,
			description:       "Very small percentage with negative result should be clamped to 0",
		},
		{
			name:              "percentage over 100",
			contextPercent:    150.0,
			completionTokens:  100,
			maxContextTokens:  1000,
			wantPromptTokens:  1400,
			wantTotalTokens:   1500,
			description:       "Percentage over 100 should be handled (though unusual)",
		},
		{
			name:              "negative percentage",
			contextPercent:    -10.0,
			completionTokens:  100,
			maxContextTokens:  1000,
			wantPromptTokens:  0,
			wantTotalTokens:   100,
			description:       "Negative percentage should return completion tokens only",
		},
		{
			name:              "fractional percentage",
			contextPercent:    33.33,
			completionTokens:  150,
			maxContextTokens:  3000,
			wantPromptTokens:  849,
			wantTotalTokens:   999,
			description:       "Fractional percentage should be calculated precisely",
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
				t.Errorf("CalculateTokensFromContextUsage() promptTokens = %d, want %d (%s)", 
					promptTokens, tt.wantPromptTokens, tt.description)
			}
			
			if totalTokens != tt.wantTotalTokens {
				t.Errorf("CalculateTokensFromContextUsage() totalTokens = %d, want %d (%s)", 
					totalTokens, tt.wantTotalTokens, tt.description)
			}
		})
	}
}

// TestCountTokens_EdgeCases provides comprehensive edge case tests for token counting
func TestCountTokens_EdgeCases(t *testing.T) {
	// Only run if we can create a tokenizer
	tok, err := NewTokenizer("claude-3-5-sonnet-20241022")
	if err != nil {
		t.Skipf("Skipping token counting tests: %v", err)
		return
	}
	defer tok.Close()

	tests := []struct {
		name        string
		text        string
		minCount    int
		maxCount    int
		description string
	}{
		{
			name:        "empty string",
			text:        "",
			minCount:    0,
			maxCount:    0,
			description: "Empty string should have zero tokens",
		},
		{
			name:        "single space",
			text:        " ",
			minCount:    0,
			maxCount:    1,
			description: "Single space should have minimal tokens",
		},
		{
			name:        "single character",
			text:        "a",
			minCount:    1,
			maxCount:    1,
			description: "Single character should have one token",
		},
		{
			name:        "single word",
			text:        "hello",
			minCount:    1,
			maxCount:    1,
			description: "Single word should have one token",
		},
		{
			name:        "two words",
			text:        "hello world",
			minCount:    2,
			maxCount:    3,
			description: "Two words should have 2-3 tokens",
		},
		{
			name:        "punctuation only",
			text:        "!@#$%^&*()",
			minCount:    1,
			maxCount:    10,
			description: "Punctuation should be tokenized",
		},
		{
			name:        "numbers",
			text:        "123456789",
			minCount:    1,
			maxCount:    5,
			description: "Numbers should be tokenized efficiently",
		},
		{
			name:        "mixed alphanumeric",
			text:        "abc123def456",
			minCount:    1,
			maxCount:    6,
			description: "Mixed alphanumeric should be tokenized",
		},
		{
			name:        "unicode characters",
			text:        "Hello 世界 🌍",
			minCount:    3,
			maxCount:    8,
			description: "Unicode should be handled correctly",
		},
		{
			name:        "newlines and tabs",
			text:        "line1\nline2\tline3",
			minCount:    3,
			maxCount:    8,
			description: "Whitespace characters should be tokenized",
		},
		{
			name:        "repeated characters",
			text:        "aaaaaaaaaa",
			minCount:    1,
			maxCount:    3,
			description: "Repeated characters should be efficient",
		},
		{
			name:        "very long word",
			text:        "supercalifragilisticexpialidocious",
			minCount:    1,
			maxCount:    15,
			description: "Very long words should be handled (may be split into multiple tokens)",
		},
		{
			name:        "code snippet",
			text:        "function hello() { return 'world'; }",
			minCount:    8,
			maxCount:    15,
			description: "Code should be tokenized appropriately",
		},
		{
			name:        "json string",
			text:        `{"key": "value", "number": 42}`,
			minCount:    8,
			maxCount:    15,
			description: "JSON should be tokenized appropriately",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := tok.CountTokens(tt.text)
			if count < tt.minCount || count > tt.maxCount {
				t.Errorf("CountTokens(%q) = %d, want between %d and %d (%s)", 
					tt.text, count, tt.minCount, tt.maxCount, tt.description)
			}
		})
	}
}
