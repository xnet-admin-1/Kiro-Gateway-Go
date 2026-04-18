package streaming

import (
	"context"
	"encoding/json"
	"log"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/yourusername/kiro-gateway-go/internal/models"
	"github.com/yourusername/kiro-gateway-go/pkg/tokenizer"
)

// splitIntoTokens splits text into small chunks that simulate token-level streaming.
// Splits on word boundaries, keeping whitespace attached to the following word.
func splitIntoTokens(text string) []string {
	if len(text) <= 4 {
		return []string{text}
	}
	var tokens []string
	var cur strings.Builder
	for i, r := range text {
		cur.WriteRune(r)
		// Split after whitespace when next char is non-whitespace, or at newlines
		if r == '\n' || (r == ' ' && i+1 < len(text)) {
			tokens = append(tokens, cur.String())
			cur.Reset()
		}
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

var xmlParamRe = regexp.MustCompile(`<([a-z_]+)>([^<]*)</[a-z_]+>`)

func extractXMLToolCalls(text string, tools []map[string]interface{}) ([]models.ToolCall, string) {
	if !strings.Contains(text, "</") {
		return nil, text
	}
	toolNames := make(map[string]bool)
	for _, t := range tools {
		if fn, ok := t["function"].(map[string]interface{}); ok {
			if name, ok := fn["name"].(string); ok {
				toolNames[name] = true
			}
		}
	}
	var calls []models.ToolCall
	clean := text
	for name := range toolNames {
		re := regexp.MustCompile(`(?s)<` + regexp.QuoteMeta(name) + `>\s*(.*?)</` + regexp.QuoteMeta(name) + `>`)
		for _, m := range re.FindAllStringSubmatch(text, -1) {
			args := make(map[string]string)
			for _, p := range xmlParamRe.FindAllStringSubmatch(m[1], -1) {
				args[p[1]] = p[2]
			}
			argsJSON, _ := json.Marshal(args)
			calls = append(calls, models.ToolCall{
				ID:   fmt.Sprintf("call_%s", generateCompletionID()),
				Type: "function",
				Function: models.FunctionCall{
					Name:      name,
					Arguments: string(argsJSON),
				},
			})
			clean = strings.Replace(clean, m[0], "", 1)
		}
	}
	return calls, strings.TrimSpace(clean)
}

// StreamKiroToOpenAI converts Kiro stream to OpenAI SSE format
// Returns a channel of SSE chunks and optionally captures conversation ID via callback
func StreamKiroToOpenAI(ctx context.Context, resp *http.Response, model string, firstTokenTimeout time.Duration, requestMessages []map[string]interface{}, requestTools []map[string]interface{}) (<-chan string, error) {
	return StreamKiroToOpenAIWithCallback(ctx, resp, model, firstTokenTimeout, requestMessages, requestTools, nil)
}

// StreamKiroToOpenAIWithCallback converts Kiro stream to OpenAI SSE format with conversation ID callback
// The callback is called when a conversation ID is received from the MessageMetadataEvent
func StreamKiroToOpenAIWithCallback(ctx context.Context, resp *http.Response, model string, firstTokenTimeout time.Duration, requestMessages []map[string]interface{}, requestTools []map[string]interface{}, onConversationID func(string)) (<-chan string, error) {
	// Parse Kiro stream
	eventChan, err := ParseKiroStream(ctx, resp, firstTokenTimeout)
	if err != nil {
		return nil, err
	}
	
	outputChan := make(chan string, 64)
	
	go func() {
		defer func() { if r := recover(); r != nil { log.Printf("[ERROR] Streaming converter panic: %v", r) } }()
		defer close(outputChan)
		
		completionID := generateCompletionID()
		created := time.Now().Unix()
		firstChunk := true
		
		var fullContent string
		var fullThinking string
		var toolCalls []*models.ToolCall
		var toolUses []models.ToolUse
		var usage *models.TokenUsage
		var contextUsagePercent float64
		var currentToolUseID string
		var currentToolUseName string
		var currentToolUseInput string
		
		for event := range eventChan {
			switch event.Type {
			case "metadata":
				// Capture conversation ID from metadata event
				if event.ConversationID != "" && onConversationID != nil {
					onConversationID(event.ConversationID)
				}
				
			case "content":
				fullContent += event.Content
				
				// Stream content immediately as-is (no word splitting)
				chunk := models.ChatCompletionResponse{
					ID:      completionID,
					Object:  "chat.completion.chunk",
					Created: created,
					Model:   model,
					Choices: []models.ChatCompletionChoice{
						{
							Index: 0,
							Delta: &models.ResponseMessage{
								Content: event.Content,
							},
						},
					},
				}
				if firstChunk {
					chunk.Choices[0].Delta.Role = "assistant"
					firstChunk = false
				}
				chunkJSON, _ := json.Marshal(chunk)
				outputChan <- fmt.Sprintf("data: %s\n\n", chunkJSON)
				
			case "thinking":
				fullThinking += event.ThinkingContent
				
				// Send as reasoning_content
				chunk := models.ChatCompletionResponse{
					ID:      completionID,
					Object:  "chat.completion.chunk",
					Created: created,
					Model:   model,
					Choices: []models.ChatCompletionChoice{
						{
							Index: 0,
							Delta: &models.ResponseMessage{
								ReasoningContent: event.ThinkingContent,
							},
						},
					},
				}
				
				chunkJSON, _ := json.Marshal(chunk)
				outputChan <- fmt.Sprintf("data: %s\n\n", chunkJSON)
				
			case "tool_call":
				if event.ToolCall != nil {
					toolCalls = append(toolCalls, event.ToolCall)
				}
				
			case "tool_use_event":
				// Streaming tool use from Q Developer API
				if event.ToolUseEvent != nil {
					tue := event.ToolUseEvent
					if tue.Stop {
						tc := &models.ToolCall{
							Index: len(toolCalls),
							ID:    currentToolUseID,
							Type:  "function",
							Function: models.FunctionCall{
								Name:      currentToolUseName,
								Arguments: currentToolUseInput,
							},
						}
						toolCalls = append(toolCalls, tc)
						chunk := models.ChatCompletionResponse{
							ID:      completionID,
							Object:  "chat.completion.chunk",
							Created: created,
							Model:   model,
							Choices: []models.ChatCompletionChoice{
								{
									Index: 0,
									Delta: &models.ResponseMessage{
										ToolCalls: []models.ToolCall{*tc},
									},
								},
							},
						}
						chunkJSON, _ := json.Marshal(chunk)
						outputChan <- fmt.Sprintf("data: %s\n\n", chunkJSON)
						currentToolUseID = ""
						currentToolUseName = ""
						currentToolUseInput = ""
					} else {
						if tue.ToolUseID != "" {
							currentToolUseID = tue.ToolUseID
						}
						if tue.Name != "" {
							currentToolUseName = tue.Name
						}
						currentToolUseInput += tue.Input
					}
				}
				
			case "tool_uses":
				// Convert Q Developer tool uses to OpenAI tool_calls format and stream immediately
				if event.ToolUses != nil {
					toolUses = append(toolUses, event.ToolUses...)
					
					// Convert each tool use to OpenAI tool_call format
					for _, toolUse := range event.ToolUses {
						// Create OpenAI tool_call
						toolCall := &models.ToolCall{
							ID:   toolUse.ToolUseID,
							Type: "function",
							Function: models.FunctionCall{
								Name:      toolUse.Name,
								Arguments: string(toolUse.Input),
							},
						}
						toolCalls = append(toolCalls, toolCall)
						
						// Stream tool_call chunk immediately
						chunk := models.ChatCompletionResponse{
							ID:      completionID,
							Object:  "chat.completion.chunk",
							Created: created,
							Model:   model,
							Choices: []models.ChatCompletionChoice{
								{
									Index: 0,
									Delta: &models.ResponseMessage{
										ToolCalls: []models.ToolCall{*toolCall},
									},
								},
							},
						}
						
						chunkJSON, _ := json.Marshal(chunk)
						outputChan <- fmt.Sprintf("data: %s\n\n", chunkJSON)
					}
				}
				
			case "usage":
				usage = event.Usage
				
			case "context_usage":
				contextUsagePercent = event.ContextUsagePercentage
				
			case "error":
				// Send error and stop
				errorChunk := map[string]interface{}{
					"error": map[string]interface{}{
						"message": event.Error.Error(),
						"type":    "stream_error",
					},
				}
				errorJSON, _ := json.Marshal(errorChunk)
				outputChan <- fmt.Sprintf("data: %s\n\n", errorJSON)
				return
			}
		}
		
		// Calculate token usage if not provided by API
		if usage == nil {
			usage = &models.TokenUsage{}
		}
		
		// Count completion tokens using tiktoken
		tok, err := tokenizer.NewTokenizer(model)
		if err == nil {
			defer tok.Close()
			
			completionTokens := tok.CountTokens(fullContent + fullThinking)
			completionTokens = tokenizer.ApplyClaudeCorrection(completionTokens)
			usage.CompletionTokens = completionTokens
			
			// Calculate prompt tokens from context usage if available
			if contextUsagePercent > 0 {
				maxTokens := tokenizer.GetMaxContextTokens(model)
				promptTokens, totalTokens := tokenizer.CalculateTokensFromContextUsage(
					contextUsagePercent,
					completionTokens,
					maxTokens,
				)
				usage.PromptTokens = promptTokens
				usage.TotalTokens = totalTokens
			} else if requestMessages != nil {
				// Fallback: count from request messages
				usage.PromptTokens = tok.CountMessageTokens(requestMessages)
				if requestTools != nil {
					usage.PromptTokens += tok.CountToolsTokens(requestTools)
				}
				usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
			} else {
				usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
			}
		}
		
		// Tool calls already emitted during streaming via tool_use_event handler

		// Send final chunk with usage
		finishReason := "stop"
		if len(toolCalls) > 0 || len(toolUses) > 0 {
			finishReason = "tool_calls"
		}
		
		finalChunk := models.ChatCompletionResponse{
			ID:      completionID,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   model,
			Choices: []models.ChatCompletionChoice{
				{
					Index:        0,
					Delta:        &models.ResponseMessage{},
					FinishReason: finishReason,
				},
			},
			Usage: usage,
		}
		
		finalJSON, _ := json.Marshal(finalChunk)
		outputChan <- fmt.Sprintf("data: %s\n\n", finalJSON)
		outputChan <- "data: [DONE]\n\n"
	}()
	
	return outputChan, nil
}

// CollectNonStreamingResponse collects full response for non-streaming mode
func CollectNonStreamingResponse(ctx context.Context, resp *http.Response, model string, firstTokenTimeout time.Duration, requestMessages []map[string]interface{}, requestTools []map[string]interface{}) (*models.ChatCompletionResponse, error) {
	// Parse stream
	eventChan, err := ParseKiroStream(ctx, resp, firstTokenTimeout)
	if err != nil {
		return nil, err
	}
	
	// Collect all events
	content, thinking, toolCalls, usage, err := CollectStreamContent(eventChan)
	if err != nil {
		return nil, err
	}

	// Extract XML-style tool calls from content if no structured tool calls found
	if len(toolCalls) == 0 && requestTools != nil {
		xmlCalls, cleanContent := extractXMLToolCalls(content, requestTools)
		if len(xmlCalls) > 0 {
			content = cleanContent
			for i := range xmlCalls {
				toolCalls = append(toolCalls, &xmlCalls[i])
			}
		}
	}
	
	// Calculate token usage if not provided
	if usage == nil {
		usage = &models.TokenUsage{}
	}
	
	tok, tokErr := tokenizer.NewTokenizer(model)
	if tokErr == nil {
		defer tok.Close()
		
		completionTokens := tok.CountTokens(content + thinking)
		completionTokens = tokenizer.ApplyClaudeCorrection(completionTokens)
		usage.CompletionTokens = completionTokens
		
		if requestMessages != nil {
			usage.PromptTokens = tok.CountMessageTokens(requestMessages)
			if requestTools != nil {
				usage.PromptTokens += tok.CountToolsTokens(requestTools)
			}
		}
		
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	
	// Build response
	finishReason := "stop"
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
	}
	
	message := &models.ResponseMessage{
		Role:    "assistant",
		Content: content,
	}
	
	if thinking != "" {
		message.ReasoningContent = thinking
	}
	
	if len(toolCalls) > 0 {
		openaiToolCalls := make([]models.ToolCall, len(toolCalls))
		for i, tc := range toolCalls {
			openaiToolCalls[i] = *tc
		}
		message.ToolCalls = openaiToolCalls
	}
	
	response := &models.ChatCompletionResponse{
		ID:      generateCompletionID(),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []models.ChatCompletionChoice{
			{
				Index:        0,
				Message:      message,
				FinishReason: finishReason,
			},
		},
		Usage: usage,
	}
	
	return response, nil
}

// generateCompletionID generates a unique completion ID
func generateCompletionID() string {
	return fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())
}

// ParseToolUses extracts tool uses from accumulated stream events
// Tool uses appear in the AssistantResponseMessage at the end of the stream
func ParseToolUses(events []StreamEvent) []models.ToolUse {
	// Look for tool uses in the collected events
	for _, event := range events {
		if event.Type == "tool_uses" && event.ToolUses != nil {
			return event.ToolUses
		}
	}
	return nil
}
