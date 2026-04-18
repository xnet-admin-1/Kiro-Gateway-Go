package converters

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/yourusername/kiro-gateway-go/internal/models"
)

// DEPRECATED: ConvertOpenAIToKiro is deprecated and should not be used.
// Use ConvertOpenAIToConversationState instead for AWS Q Developer API compatibility.
// This function produces KiroRequest format which is not compatible with Q Developer endpoints.
// 
// ⚠️ WARNING: This function is DEPRECATED as of 2026-01-25
// - Produces incompatible KiroRequest format (not ConversationStateRequest)
// - Not compatible with Q Developer API endpoints
// - Does not support multimodal content correctly
// - Will be REMOVED in next major version
//
// Migration: Replace all calls with ConvertOpenAIToConversationState()
//
// ConvertOpenAIToKiro converts OpenAI request to Kiro request (LEGACY FORMAT - DO NOT USE)
func ConvertOpenAIToKiro(req *models.ChatCompletionRequest, conversationID, profileArn string, injectThinking bool) (*models.KiroRequest, error) {
	// Log deprecation warning
	// Note: Commented out to avoid log spam, but this function should not be used
	// log.Println("⚠️ WARNING: ConvertOpenAIToKiro() is DEPRECATED. Use ConvertOpenAIToConversationState() instead.")
	// Extract system prompt and convert messages
	systemPrompt, kiroMessages := convertMessages(req.Messages)
	
	// Convert tools
	var kiroTools []models.KiroTool
	if req.Tools != nil {
		kiroTools = convertTools(req.Tools)
	}
	
	// Inject thinking mode if enabled
	if injectThinking && systemPrompt != "" {
		systemPrompt = injectThinkingMode(systemPrompt)
	}
	
	// Build Kiro message with system prompt
	var content []models.KiroContentPart
	if systemPrompt != "" {
		content = append(content, models.KiroContentPart{
			Type: "text",
			Text: systemPrompt,
		})
	}
	
	// Add user messages
	content = append(content, kiroMessages...)
	
	kiroReq := &models.KiroRequest{
		ConversationID: conversationID,
		ProfileArn:     profileArn,
		Message: models.KiroMessage{
			Role:    "user",
			Content: content,
		},
		Tools: kiroTools,
	}
	
	return kiroReq, nil
}

// convertMessages converts OpenAI messages to Kiro format
func convertMessages(messages []models.Message) (string, []models.KiroContentPart) {
	var systemPrompt string
	var kiroContent []models.KiroContentPart
	var pendingToolResults []models.KiroContentPart
	
	for _, msg := range messages {
		switch msg.Role {
		case "system":
			// Extract system prompt
			systemPrompt += extractTextContent(msg.Content) + "\n"
			
		case "user":
			// Flush pending tool results first
			if len(pendingToolResults) > 0 {
				kiroContent = append(kiroContent, pendingToolResults...)
				pendingToolResults = nil
			}
			
			// Convert user message
			text := extractTextContent(msg.Content)
			if text != "" {
				kiroContent = append(kiroContent, models.KiroContentPart{
					Type: "text",
					Text: text,
				})
			}
			
			// Handle images
			images := extractImages(msg.Content)
			kiroContent = append(kiroContent, images...)
			
		case "assistant":
			// Convert assistant message with tool calls
			text := extractTextContent(msg.Content)
			if text != "" {
				kiroContent = append(kiroContent, models.KiroContentPart{
					Type: "text",
					Text: text,
				})
			}
			
			// Handle tool calls
			if msg.ToolCalls != nil {
				for _, tc := range msg.ToolCalls {
					var input map[string]interface{}
					if err := json.Unmarshal([]byte(tc.Function.Arguments), &input); err != nil {
						// If parsing fails, use empty object
						input = make(map[string]interface{})
					}
					
					kiroContent = append(kiroContent, models.KiroContentPart{
						Type:  "tool_use",
						ID:    tc.ID,
						Name:  tc.Function.Name,
						Input: input,
					})
				}
			}
			
		case "tool":
			// Collect tool results
			toolResult := models.KiroContentPart{
				Type:      "tool_result",
				ToolUseID: msg.ToolCallID,
				Content:   extractTextContent(msg.Content),
			}
			if toolResult.Content == "" {
				toolResult.Content = "(empty result)"
			}
			pendingToolResults = append(pendingToolResults, toolResult)
		}
	}
	
	// Flush remaining tool results
	if len(pendingToolResults) > 0 {
		kiroContent = append(kiroContent, pendingToolResults...)
	}
	
	return strings.TrimSpace(systemPrompt), kiroContent
}

// extractTextContent extracts text from message content
func extractTextContent(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		var text strings.Builder
		for _, item := range v {
			if part, ok := item.(map[string]interface{}); ok {
				if part["type"] == "text" {
					if t, ok := part["text"].(string); ok {
						text.WriteString(t)
					}
				}
			}
		}
		return text.String()
	default:
		return ""
	}
}

// extractImages extracts images from message content
func extractImages(content interface{}) []models.KiroContentPart {
	var images []models.KiroContentPart
	
	if parts, ok := content.([]interface{}); ok {
		for _, item := range parts {
			if part, ok := item.(map[string]interface{}); ok {
				if part["type"] == "image_url" {
					if imageURL, ok := part["image_url"].(map[string]interface{}); ok {
						if url, ok := imageURL["url"].(string); ok {
							image := convertImageURL(url)
							if image != nil {
								images = append(images, *image)
							}
						}
					}
				}
			}
		}
	}
	
	return images
}

// convertImageURL converts OpenAI image URL to Kiro format
func convertImageURL(url string) *models.KiroContentPart {
	// Handle data URLs: data:image/png;base64,iVBORw0KG...
	if strings.HasPrefix(url, "data:") {
		parts := strings.SplitN(url, ",", 2)
		if len(parts) != 2 {
			return nil
		}
		
		// Extract media type
		mediaType := "image/png"
		if strings.Contains(parts[0], "image/jpeg") {
			mediaType = "image/jpeg"
		} else if strings.Contains(parts[0], "image/webp") {
			mediaType = "image/webp"
		} else if strings.Contains(parts[0], "image/gif") {
			mediaType = "image/gif"
		}
		
		return &models.KiroContentPart{
			Type: "image",
			Source: &models.KiroImageSource{
				Type:      "base64",
				MediaType: mediaType,
				Data:      parts[1],
			},
		}
	}
	
	// Handle HTTP URLs - download and convert to base64
	// TODO: Implement HTTP image download
	return nil
}

// convertTools converts OpenAI tools to Kiro format
func convertTools(tools []models.Tool) []models.KiroTool {
	kiroTools := make([]models.KiroTool, 0, len(tools))
	
	for _, tool := range tools {
		if tool.Type != "function" {
			continue
		}
		
		kiroTools = append(kiroTools, models.KiroTool{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			InputSchema: tool.Function.Parameters,
		})
	}
	
	return kiroTools
}

// injectThinkingMode injects thinking mode tag into system prompt
func injectThinkingMode(systemPrompt string) string {
	thinkingTag := `<thinking_mode>
When faced with complex problems or questions, you should think through your reasoning step by step before providing your final answer. Use <thinking> tags to show your thought process.
</thinking_mode>`
	
	return systemPrompt + "\n\n" + thinkingTag
}

// NormalizeModelID normalizes model ID for Kiro API
func NormalizeModelID(modelID string, hiddenModels []string) string {
	// Strip Bedrock-style prefix and suffix if present
	// Format: anthropic.claude-3-5-sonnet-20241022-v2:0 -> claude-3-5-sonnet-20241022-v2
	normalized := modelID
	
	// Remove "anthropic." prefix
	if strings.HasPrefix(normalized, "anthropic.") {
		normalized = strings.TrimPrefix(normalized, "anthropic.")
	}
	
	// Remove version suffix (e.g., ":0", "-v1:0")
	if idx := strings.LastIndex(normalized, ":"); idx != -1 {
		normalized = normalized[:idx]
	}
	
	// Replace underscores with hyphens
	normalized = strings.ReplaceAll(normalized, "_", "-")
	
	// Check if it's a hidden model (exact match or prefix match)
	for _, hidden := range hiddenModels {
		if normalized == hidden || modelID == hidden {
			return hidden
		}
		// Check if the normalized model is a prefix of a hidden model
		if strings.HasPrefix(hidden, normalized) {
			return hidden
		}
	}
	
	return normalized
}

// GenerateConversationID generates a unique conversation ID using UUID
func GenerateConversationID() string {
	// Generate a UUID v4 for conversation ID
	// Format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to timestamp-based ID if random fails
		return fmt.Sprintf("conv_%d", time.Now().UnixNano())
	}
	
	// Set version (4) and variant bits
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// Helper to encode image to base64
func encodeImageToBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
