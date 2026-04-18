package tokenizer

import (
	"encoding/json"
	"fmt"

	"github.com/pkoukk/tiktoken-go"
)

// Tokenizer handles token counting for various models
type Tokenizer struct {
	encoding *tiktoken.Tiktoken
}

// NewTokenizer creates a new tokenizer for the given model
func NewTokenizer(model string) (*Tokenizer, error) {
	// Map model names to tiktoken encodings
	encodingName := getEncodingForModel(model)
	
	encoding, err := tiktoken.GetEncoding(encodingName)
	if err != nil {
		return nil, fmt.Errorf("failed to get encoding: %w", err)
	}
	
	return &Tokenizer{
		encoding: encoding,
	}, nil
}

// CountTokens counts tokens in a text string
func (t *Tokenizer) CountTokens(text string) int {
	if text == "" {
		return 0
	}
	
	tokens := t.encoding.Encode(text, nil, nil)
	return len(tokens)
}

// CountMessageTokens counts tokens in a message array
func (t *Tokenizer) CountMessageTokens(messages []map[string]interface{}) int {
	totalTokens := 0
	
	for _, msg := range messages {
		// Add tokens for message structure
		totalTokens += 4 // Every message has overhead
		
		// Count role tokens
		if role, ok := msg["role"].(string); ok {
			totalTokens += t.CountTokens(role)
		}
		
		// Count content tokens
		if content := msg["content"]; content != nil {
			totalTokens += t.countContentTokens(content)
		}
		
		// Count tool calls
		if toolCalls, ok := msg["tool_calls"].([]interface{}); ok {
			for _, tc := range toolCalls {
				if tcMap, ok := tc.(map[string]interface{}); ok {
					totalTokens += t.countToolCallTokens(tcMap)
				}
			}
		}
	}
	
	// Add tokens for message array structure
	totalTokens += 3
	
	return totalTokens
}

// CountToolsTokens counts tokens in tools array
func (t *Tokenizer) CountToolsTokens(tools []map[string]interface{}) int {
	if len(tools) == 0 {
		return 0
	}
	
	totalTokens := 0
	
	for _, tool := range tools {
		// Count function definition
		if function, ok := tool["function"].(map[string]interface{}); ok {
			// Name
			if name, ok := function["name"].(string); ok {
				totalTokens += t.CountTokens(name)
			}
			
			// Description
			if desc, ok := function["description"].(string); ok {
				totalTokens += t.CountTokens(desc)
			}
			
			// Parameters (JSON schema)
			if params, ok := function["parameters"].(map[string]interface{}); ok {
				paramsJSON, _ := json.Marshal(params)
				totalTokens += t.CountTokens(string(paramsJSON))
			}
		}
		
		// Overhead per tool
		totalTokens += 10
	}
	
	return totalTokens
}

// countContentTokens counts tokens in message content
func (t *Tokenizer) countContentTokens(content interface{}) int {
	switch v := content.(type) {
	case string:
		return t.CountTokens(v)
		
	case []interface{}:
		totalTokens := 0
		for _, part := range v {
			if partMap, ok := part.(map[string]interface{}); ok {
				if partType, ok := partMap["type"].(string); ok {
					switch partType {
					case "text":
						if text, ok := partMap["text"].(string); ok {
							totalTokens += t.CountTokens(text)
						}
					case "image_url":
						// Images have fixed token cost
						totalTokens += 85 // Base cost for image
					}
				}
			}
		}
		return totalTokens
		
	default:
		return 0
	}
}

// countToolCallTokens counts tokens in a tool call
func (t *Tokenizer) countToolCallTokens(toolCall map[string]interface{}) int {
	totalTokens := 0
	
	if function, ok := toolCall["function"].(map[string]interface{}); ok {
		// Function name
		if name, ok := function["name"].(string); ok {
			totalTokens += t.CountTokens(name)
		}
		
		// Function arguments
		if args, ok := function["arguments"].(string); ok {
			totalTokens += t.CountTokens(args)
		}
	}
	
	// Overhead for tool call structure
	totalTokens += 5
	
	return totalTokens
}

// getEncodingForModel returns the tiktoken encoding name for a model
func getEncodingForModel(model string) string {
	// Claude models use cl100k_base encoding (same as GPT-4)
	// This is an approximation since Claude uses a different tokenizer
	return "cl100k_base"
}

// CalculateTokensFromContextUsage calculates tokens from context usage percentage
func CalculateTokensFromContextUsage(contextUsagePercent float64, completionTokens int, maxContextTokens int) (promptTokens, totalTokens int) {
	if contextUsagePercent <= 0 || maxContextTokens <= 0 {
		return 0, completionTokens
	}
	
	// Context usage includes both input and output
	totalTokens = int(float64(maxContextTokens) * contextUsagePercent / 100.0)
	
	// Subtract completion tokens to get prompt tokens
	promptTokens = totalTokens - completionTokens
	if promptTokens < 0 {
		promptTokens = 0
	}
	
	return promptTokens, totalTokens
}

// GetMaxContextTokens returns max context tokens for a model
func GetMaxContextTokens(model string) int {
	// Model context limits
	switch model {
	case "claude-3-5-sonnet-20241022", "claude-3-5-sonnet-20240620":
		return 200000
	case "claude-3-opus-20240229":
		return 200000
	case "claude-3-sonnet-20240229":
		return 200000
	case "claude-3-haiku-20240307":
		return 200000
	default:
		return 200000 // Default to 200k
	}
}

// ApplyClaudeCorrection applies correction coefficient for Claude models
// Claude's tokenizer differs from tiktoken, so we apply a correction factor
func ApplyClaudeCorrection(tokens int) int {
	// Claude typically uses ~15% more tokens than tiktoken estimates
	return int(float64(tokens) * 1.15)
}

// Close releases tokenizer resources
func (t *Tokenizer) Close() {
	// tiktoken-go doesn't require explicit cleanup
	// This method is here for future compatibility
}
