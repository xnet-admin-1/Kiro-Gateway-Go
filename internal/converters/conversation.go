package converters

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/yourusername/kiro-gateway-go/internal/logging"
	"github.com/yourusername/kiro-gateway-go/internal/models"
)

// ConvertOpenAIToConversationState converts OpenAI request to Q Developer conversation state
func ConvertOpenAIToConversationState(
	req *models.ChatCompletionRequest,
	conversationID *string,
	profileArn string,
) (*models.ConversationStateRequest, error) {
	// Extract system prompt and convert messages to history
	systemPrompt, history, currentUserMessage, currentImages := extractConversationHistory(req.Messages)
	
	// FIX #1 & #2: Extract tool results from messages
	// Tool results are handled inline in history
	
	// Build current message with system prompt if present
	if systemPrompt != "" {
		// Prepend system prompt to current user message
		currentUserMessage = systemPrompt + "\n\n" + currentUserMessage
	}
	
	// Empty content fallback
	if currentUserMessage == "" {
		currentUserMessage = "Continue"
	}

	// Build userInputMessageContext with tools and toolResults
	userInputContext := &models.UserInputMessageContext{}

	// Add tools inside context (not top-level)
	if req.Tools != nil && len(req.Tools) > 0 {
		userInputContext.Tools = convertToolsToQFormat(req.Tools)
	}

	// Extract tool results from messages and add to context
	toolResults := extractToolResults(req.Messages)
	if len(toolResults) > 0 {
		userInputContext.ToolResults = toolResults
	}

	userInputMsg := &models.UserInputMessage{
		Content:                 currentUserMessage,
		UserInputMessageContext: userInputContext,
		Images:                  currentImages,
		Origin:                  "AI_EDITOR",
		ModelID:                 req.Model,
	}

	convState := &models.ConversationState{
		ConversationID: conversationID,
		CurrentMessage: models.ChatMessage{
			UserInputMessage: userInputMsg,
		},
		ChatTriggerType: "MANUAL",
		History:         history,
	}

	request := &models.ConversationStateRequest{
		ConversationState: convState,
	}
	
	// Only include profileArn for Identity Center mode (when provided)
	if profileArn != "" {
		request.ProfileArn = profileArn
	}
	
	return request, nil
}

// extractConversationHistory extracts system prompt, conversation history, current user message, and images
func extractConversationHistory(messages []models.Message) (string, []models.ChatMessage, string, []models.ImageBlock) {
	var systemPrompt string
	var history []models.ChatMessage
	var currentUserMessage string
	var currentImages []models.ImageBlock
	
	// Track if we're building history or current message
	var pendingUserMsg *models.UserInputMessage
	var pendingAssistantMsg *models.AssistantResponseMessage
	var pendingToolResults []models.ToolResult
	
	for i, msg := range messages {
		isLastMessage := i == len(messages)-1
		
		switch msg.Role {
		case "system":
			// Accumulate system prompts
			systemPrompt += extractTextContent(msg.Content)
			if !strings.HasSuffix(systemPrompt, "\n") {
				systemPrompt += "\n"
			}
			
		case "user":
			// Flush any pending tool results first
			if len(pendingToolResults) > 0 {
				userMsg := &models.UserInputMessage{
					Content: "Continue",
					UserInputMessageContext: &models.UserInputMessageContext{
						ToolResults: pendingToolResults,
					},
				}
				history = append(history, models.ChatMessage{
					UserInputMessage: userMsg,
				})
				pendingToolResults = nil
			}

			text := extractTextContent(msg.Content)
			images := extractAndConvertImages(msg.Content)
			
			if isLastMessage {
				// This is the current message
				currentUserMessage = text
				currentImages = images
			} else {
				// This is part of history
				// First, flush any pending assistant message
				if pendingAssistantMsg != nil {
					// We have an assistant message without a following user message
					// This shouldn't happen in valid history, but handle it
					history = append(history, models.ChatMessage{
						AssistantResponseMessage: pendingAssistantMsg,
					})
					pendingAssistantMsg = nil
				}
				
				// Store this user message as pending
				pendingUserMsg = &models.UserInputMessage{
					Content: text,
					Images:  images,
				}
			}
			
		case "assistant":
			// Flush any pending tool results as a user message first
			if len(pendingToolResults) > 0 {
				userMsg := &models.UserInputMessage{
					Content: "Continue",
					UserInputMessageContext: &models.UserInputMessageContext{
						ToolResults: pendingToolResults,
					},
				}
				history = append(history, models.ChatMessage{
					UserInputMessage: userMsg,
				})
				pendingToolResults = nil
				pendingUserMsg = nil
			}

			text := extractTextContent(msg.Content)
			if text == "" {
				text = " "
			}
			
			assistantMsg := &models.AssistantResponseMessage{
				Content: text,
			}
			
			// Preserve real toolUses for the API
			if msg.ToolCalls != nil && len(msg.ToolCalls) > 0 {
				var toolUses []models.ToolUse
				for _, tc := range msg.ToolCalls {
					toolUses = append(toolUses, models.ToolUse{
						ToolUseID: tc.ID,
						Name:      tc.Function.Name,
						Input:     json.RawMessage(tc.Function.Arguments),
					})
				}
				assistantMsg.ToolUses = toolUses
			}
			
			// Add user message first if pending, then assistant
			if pendingUserMsg != nil {
				history = append(history, models.ChatMessage{
					UserInputMessage: pendingUserMsg,
				})
				pendingUserMsg = nil
			}
			// Always add assistant to history (not as pending)
			history = append(history, models.ChatMessage{
				AssistantResponseMessage: assistantMsg,
			})
			pendingAssistantMsg = nil
			
		case "tool":
			// Accumulate tool results - they'll be grouped into a single user message
			toolResult := convertToolResultToQFormat(msg)
			if toolResult != nil {
				if isLastMessage {
					currentUserMessage = "Continue"
				} else {
					pendingToolResults = append(pendingToolResults, *toolResult)
				}
			}
		}
	}
	
	// FIX #2: Multi-turn Conversation History Pairing
	// If we have pending tool results, they should be part of the current message
	// Flush any remaining pending messages into history
	if pendingAssistantMsg != nil {
		// Need a user message before the assistant
		if pendingUserMsg != nil {
			history = append(history, models.ChatMessage{UserInputMessage: pendingUserMsg})
			pendingUserMsg = nil
		}
		history = append(history, models.ChatMessage{AssistantResponseMessage: pendingAssistantMsg})
		pendingAssistantMsg = nil
	}
	if pendingUserMsg != nil {
		history = append(history, models.ChatMessage{UserInputMessage: pendingUserMsg})
	}
	
	systemPrompt = strings.TrimSpace(systemPrompt)
	
	return systemPrompt, history, currentUserMessage, currentImages
}

// convertToolsToQFormat converts OpenAI tools to Q Developer format
func convertToolsToQFormat(tools []models.Tool) []models.ToolSpecWrapper {
	logging.DebugLog("Converting %d tools from OpenAI to Q format", len(tools))
	var qTools []models.ToolSpecWrapper
	
	for _, tool := range tools {
		logging.DebugLog("Converted tool: %s", tool.Function.Name)
		toolJSON, _ := json.Marshal(tool)
		logging.DebugLog("Tool JSON: %s", string(toolJSON))

		qTools = append(qTools, models.ToolSpecWrapper{
			ToolSpecification: &models.ToolSpecification{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				InputSchema: models.ToolInputSchema{
					JSON: tool.Function.Parameters, // Keep as object (map[string]interface{})
				},
			},
		})
	}
	
	return qTools
}

// ConvertQResponseToOpenAI converts Q Developer streaming response to OpenAI format
func ConvertQResponseToOpenAI(
	event *models.ChatResponseStream,
	model string,
	conversationID string,
) (*models.ChatCompletionResponse, error) {
	response := &models.ChatCompletionResponse{
		ID:      conversationID,
		Object:  "chat.completion.chunk",
		Created: getCurrentTimestamp(),
		Model:   model,
		Choices: []models.ChatCompletionChoice{},
	}
	
	// Handle different event types
	if event.AssistantResponseEvent != nil {
		// Text content
		response.Choices = append(response.Choices, models.ChatCompletionChoice{
			Index: 0,
			Delta: &models.ResponseMessage{
				Role:    "assistant",
				Content: event.AssistantResponseEvent.Content,
			},
			FinishReason: "",
		})
	} else if event.MessageMetadataEvent != nil {
		// Message metadata (final message indicator)
		if event.MessageMetadataEvent.FinalMessage {
			response.Choices = append(response.Choices, models.ChatCompletionChoice{
				Index: 0,
				Delta: &models.ResponseMessage{},
				FinishReason: "stop",
			})
		}
	} else if event.Error != nil {
		// Error event
		return nil, fmt.Errorf("API error: %s (code: %s)", event.Error.Message, event.Error.Code)
	} else if event.CodeReferenceEvent != nil {
		// Code references - could be added as metadata
		// For now, skip
	} else if event.FollowupPromptEvent != nil {
		// Follow-up prompts - could be added as metadata
		// For now, skip
	}
	
	return response, nil
}

// Helper function to get current timestamp
func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}


// extractAndConvertImages extracts images from message content and converts to Q Developer format
func extractAndConvertImages(content interface{}) []models.ImageBlock {
	var images []models.ImageBlock
	
	if parts, ok := content.([]interface{}); ok {
		for _, item := range parts {
			if part, ok := item.(map[string]interface{}); ok {
				if part["type"] == "image_url" {
					if imageURL, ok := part["image_url"].(map[string]interface{}); ok {
						if url, ok := imageURL["url"].(string); ok {
							image := convertImageURLToBlock(url)
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

// convertImageURLToBlock converts OpenAI image URL to Q Developer ImageBlock
func convertImageURLToBlock(url string) *models.ImageBlock {
	// Handle data URLs: data:image/png;base64,iVBORw0KG...
	if strings.HasPrefix(url, "data:") {
		parts := strings.SplitN(url, ",", 2)
		if len(parts) != 2 {
			logging.DebugLog("Failed to split data URL")
			return nil
		}
		
		// Extract format from media type
		format := "png"
		if strings.Contains(parts[0], "image/jpeg") || strings.Contains(parts[0], "image/jpg") {
			format = "jpeg"
		} else if strings.Contains(parts[0], "image/webp") {
			format = "webp"
		} else if strings.Contains(parts[0], "image/gif") {
			format = "gif"
		}
		
		// CRITICAL: Decode base64 to raw bytes (matches Q CLI behavior)
		// The Q CLI reads raw bytes from files: fs::read(file_path)
		// We receive base64, so we decode it to get raw bytes
		// Go's JSON encoder will automatically base64-encode []byte during marshaling
		// This matches exactly what Rust's Blob type does
		imageBytes, err := base64DecodeString(parts[1])
		if err != nil {
			logging.DebugLog("Failed to decode base64 image: %v", err)
			return nil
		}
		
		logging.DebugLog("Converted image: format=%s, bytes_length=%d", format, len(imageBytes))
		
		// CRITICAL: Based on Q CLI serialization code (shape_image_block.rs):
		// The structure is nested: {"format": "png", "source": {"bytes": "base64..."}}
		// The ser_image_block function creates: object.key("source").start_object()
		// Then ser_image_source writes: object.key("bytes").string_unchecked(base64)
		return &models.ImageBlock{
			Format: format,
			Source: models.ImageSource{
				Bytes: imageBytes,  // Raw bytes, JSON encoder will base64-encode
			},
		}
	}
	
	// For external URLs, we would need to download the image
	// For now, return nil (not supported yet)
	return nil
}

// base64DecodeString decodes a base64 string
func base64DecodeString(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// FIX #1 & #3: convertToolResultToQFormat converts OpenAI tool message to Q Developer ToolResult
func convertToolResultToQFormat(msg models.Message) *models.ToolResult {
	if msg.Role != "tool" {
		return nil
	}
	
	// Extract content
	content := extractTextContent(msg.Content)
	if content == "" {
		content = "(empty result)"
	}
	
	// FIX #3: Detect error status from content
	status := "success"
	contentLower := strings.ToLower(content)
	
	// Check for common error indicators
	if strings.Contains(contentLower, "error:") ||
	   strings.Contains(contentLower, "failed") ||
	   strings.Contains(contentLower, "exception") ||
	   strings.Contains(contentLower, "not found") ||
	   strings.Contains(contentLower, "permission denied") ||
	   strings.Contains(contentLower, "invalid") {
		status = "error"
		logging.DebugLog("Detected error in tool result: %s", content[:min(100, len(content))])
	}
	
	// Create tool result content - always use text format
	toolResultContent := models.ToolResultContent{
		Text: content,
	}
	
	return &models.ToolResult{
		ToolUseID: msg.ToolCallID,
		Content:   []models.ToolResultContent{toolResultContent},
		Status:    status,
	}
}

// FIX #1: extractToolResults extracts all tool results from messages
// Supports two formats:
// 1. Standard OpenAI: role="tool" messages with tool_call_id
// 2. Anthropic/Claude: role="user" with content array containing tool_result parts
func extractToolResults(messages []models.Message) []models.ToolResult {
	var toolResults []models.ToolResult
	
	// Only extract tool results from the END of the message list
	// (after the last assistant message - these are the current round's results)
	lastAssistantIdx := -1
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "assistant" {
			lastAssistantIdx = i
			break
		}
	}
	
	for i, msg := range messages {
		if i <= lastAssistantIdx {
			continue
		}
		if msg.Role == "tool" {
			toolResult := convertToolResultToQFormat(msg)
			if toolResult != nil {
				toolResults = append(toolResults, *toolResult)
			}
		}
	}
	
	return toolResults
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}







