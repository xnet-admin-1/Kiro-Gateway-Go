package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/yourusername/kiro-gateway-go/internal/auth"
	"github.com/yourusername/kiro-gateway-go/internal/client"
	"github.com/yourusername/kiro-gateway-go/internal/converters"
	"github.com/yourusername/kiro-gateway-go/internal/models"
	"github.com/yourusername/kiro-gateway-go/internal/streaming"
	"github.com/yourusername/kiro-gateway-go/internal/logging"
	"github.com/yourusername/kiro-gateway-go/internal/validation"
)

// AnthropicAdapter handles Anthropic-compatible API requests
type AnthropicAdapter struct {
	authManager *auth.AuthManager
	client      *client.Client
	validator   *validation.RequestValidator
	config      *AdapterConfig
}

// AnthropicRequest represents an Anthropic Messages API request
type AnthropicRequest struct {
	Model       string                   `json:"model"`
	Messages    []AnthropicMessage       `json:"messages"`
	System      string                   `json:"system,omitempty"`
	MaxTokens   int                      `json:"max_tokens"`
	Temperature *float64                 `json:"temperature,omitempty"`
	TopP        *float64                 `json:"top_p,omitempty"`
	TopK        *int                     `json:"top_k,omitempty"`
	Stream      bool                     `json:"stream,omitempty"`
	Tools       []AnthropicTool          `json:"tools,omitempty"`
	Metadata    map[string]interface{}   `json:"metadata,omitempty"`
}

// AnthropicMessage represents a message in Anthropic format
type AnthropicMessage struct {
	Role    string                 `json:"role"` // "user" or "assistant"
	Content interface{}            `json:"content"` // string or []AnthropicContentBlock
}

// AnthropicContentBlock represents a content block
type AnthropicContentBlock struct {
	Type      string                 `json:"type"` // "text", "image", "tool_use", "tool_result"
	Text      string                 `json:"text,omitempty"`
	Source    *AnthropicImageSource  `json:"source,omitempty"`
	ID        string                 `json:"id,omitempty"`
	Name      string                 `json:"name,omitempty"`
	Input     map[string]interface{} `json:"input,omitempty"`
	ToolUseID string                 `json:"tool_use_id,omitempty"`
	Content   interface{}            `json:"content,omitempty"`
}

// AnthropicImageSource represents an image source
type AnthropicImageSource struct {
	Type      string `json:"type"` // "base64" or "url"
	MediaType string `json:"media_type"`
	Data      string `json:"data,omitempty"`
	URL       string `json:"url,omitempty"`
}

// AnthropicTool represents a tool definition
type AnthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// AnthropicResponse represents an Anthropic Messages API response
type AnthropicResponse struct {
	ID           string                  `json:"id"`
	Type         string                  `json:"type"` // "message"
	Role         string                  `json:"role"` // "assistant"
	Content      []AnthropicContentBlock `json:"content"`
	Model        string                  `json:"model"`
	StopReason   string                  `json:"stop_reason,omitempty"`
	StopSequence string                  `json:"stop_sequence,omitempty"`
	Usage        *AnthropicUsage         `json:"usage,omitempty"`
}

// AnthropicUsage represents token usage
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// AnthropicStreamEvent represents a streaming event
type AnthropicStreamEvent struct {
	Type         string                  `json:"type"`
	Message      *AnthropicResponse      `json:"message,omitempty"`
	Index        int                     `json:"index,omitempty"`
	ContentBlock *AnthropicContentBlock  `json:"content_block,omitempty"`
	Delta        *AnthropicDelta         `json:"delta,omitempty"`
	Usage        *AnthropicUsage         `json:"usage,omitempty"`
}

// AnthropicDelta represents a delta in streaming
type AnthropicDelta struct {
	Type         string                 `json:"type,omitempty"`
	Text         string                 `json:"text,omitempty"`
	StopReason   string                 `json:"stop_reason,omitempty"`
	StopSequence string                 `json:"stop_sequence,omitempty"`
}

// NewAnthropicAdapter creates a new Anthropic adapter
func NewAnthropicAdapter(authManager *auth.AuthManager, client *client.Client, validator *validation.RequestValidator, config *AdapterConfig) *AnthropicAdapter {
	return &AnthropicAdapter{
		authManager: authManager,
		client:      client,
		validator:   validator,
		config:      config,
	}
}

// convertAnthropicToOpenAI converts Anthropic Messages API format to OpenAI Chat Completions format
func (a *AnthropicAdapter) convertAnthropicToOpenAI(req *AnthropicRequest) *models.ChatCompletionRequest {
	openAIReq := &models.ChatCompletionRequest{
		Model:     req.Model,
		Messages:  make([]models.Message, 0),
		MaxTokens: req.MaxTokens,
		Stream:    req.Stream,
	}
	
	// Handle optional temperature
	if req.Temperature != nil {
		openAIReq.Temperature = *req.Temperature
	}
	
	// Add system message if present
	if req.System != "" {
		openAIReq.Messages = append(openAIReq.Messages, models.Message{
			Role:    "system",
			Content: req.System,
		})
	}
	
	// Convert Anthropic messages to OpenAI format
	for _, msg := range req.Messages {
		openAIMsg := models.Message{
			Role: msg.Role,
		}
		
		// Handle content - can be string or array of content blocks
		switch content := msg.Content.(type) {
		case string:
			// Simple text content
			openAIMsg.Content = content
			
		case []interface{}:
			// Array of content blocks - need to convert to OpenAI format
			var contentParts []interface{}
			
			for _, block := range content {
				if blockMap, ok := block.(map[string]interface{}); ok {
					blockType, _ := blockMap["type"].(string)
					
					switch blockType {
					case "text":
						// Text block
						if text, ok := blockMap["text"].(string); ok {
							contentParts = append(contentParts, map[string]interface{}{
								"type": "text",
								"text": text,
							})
						}
						
					case "image":
						// Image block - convert to OpenAI format
						if source, ok := blockMap["source"].(map[string]interface{}); ok {
							if sourceType, _ := source["type"].(string); sourceType == "base64" {
								if data, ok := source["data"].(string); ok {
									if mediaType, ok := source["media_type"].(string); ok {
										// OpenAI format: data:image/png;base64,<data>
										dataURL := fmt.Sprintf("data:%s;base64,%s", mediaType, data)
										contentParts = append(contentParts, map[string]interface{}{
											"type": "image_url",
											"image_url": map[string]interface{}{
												"url": dataURL,
											},
										})
									}
								}
							}
						}
					}
				}
			}
			
			// If we have multiple parts, use array format; otherwise use simple string
			if len(contentParts) == 1 {
				if textPart, ok := contentParts[0].(map[string]interface{}); ok {
					if textPart["type"] == "text" {
						openAIMsg.Content = textPart["text"]
					} else {
						openAIMsg.Content = contentParts
					}
				}
			} else if len(contentParts) > 0 {
				openAIMsg.Content = contentParts
			}
		}
		
		openAIReq.Messages = append(openAIReq.Messages, openAIMsg)
	}
	
	// Convert tools if present
	if len(req.Tools) > 0 {
		openAIReq.Tools = make([]models.Tool, 0, len(req.Tools))
		for _, tool := range req.Tools {
			openAIReq.Tools = append(openAIReq.Tools, models.Tool{
				Type: "function",
				Function: models.FunctionDef{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  tool.InputSchema,
				},
			})
		}
	}
	
	return openAIReq
}

// HandleMessages handles POST /v1/messages
func (a *AnthropicAdapter) HandleMessages(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	ctx := context.WithValue(r.Context(), "requestID", requestID)
	
	log.Printf("[%s] Anthropic API: /v1/messages", requestID)
	
	// Parse request
	var req AnthropicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", "Invalid request body", err)
		return
	}
	
	// Validate required fields
	if req.Model == "" {
		writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", "Model is required", nil)
		return
	}
	if req.MaxTokens == 0 {
		writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", "max_tokens is required", nil)
		return
	}
	if len(req.Messages) == 0 {
		writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", "At least one message is required", nil)
		return
	}
	
	// Normalize model ID
	originalModel := req.Model
	req.Model = converters.NormalizeModelID(req.Model, a.config.HiddenModels)
	if originalModel != req.Model {
		log.Printf("[%s] Normalized model: %s -> %s", requestID, originalModel, req.Model)
	}
	
	log.Printf("[%s] Model: %s, Stream: %v, Messages: %d", requestID, req.Model, req.Stream, len(req.Messages))
	
	// Convert Anthropic format to OpenAI format first
	openAIReq := a.convertAnthropicToOpenAI(&req)
	
	// Then use the shared converter to convert to AWS Q Developer format
	var convID *string // nil for first message
	convStateReq, err := converters.ConvertOpenAIToConversationState(openAIReq, convID, a.config.ProfileArn)
	if err != nil {
		log.Printf("[%s] Conversion error: %v", requestID, err)
		writeAnthropicError(w, http.StatusInternalServerError, "api_error", "Failed to convert request", err)
		return
	}
	
	// Log the request for debugging (redacted in production)
	if logging.IsDebugEnabled() {
		if reqBytes, err := json.MarshalIndent(convStateReq, "", "  "); err == nil {
			log.Printf("[%s] Conversation state request:\n%s", requestID, string(reqBytes))
		}
	} else {
		log.Printf("[%s] Conversation state request: [REDACTED] (set DEBUG=true to view)", requestID)
	}
	
	// Handle streaming vs non-streaming
	if req.Stream {
		a.handleStreamingMessages(ctx, w, r, &req, convStateReq, requestID)
	} else {
		a.handleNonStreamingMessages(ctx, w, r, &req, convStateReq, requestID)
	}
}

// handleStreamingMessages handles streaming Anthropic messages
func (a *AnthropicAdapter) handleStreamingMessages(ctx context.Context, w http.ResponseWriter, r *http.Request, req *AnthropicRequest, convStateReq *models.ConversationStateRequest, requestID string) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeAnthropicError(w, http.StatusInternalServerError, "api_error", "Streaming not supported", nil)
		return
	}
	
	// Determine API endpoint using shared logic
	apiEndpoint := DetermineAPIEndpoint(a.authManager.GetAuthMode(), true)
	
	// Send request to AWS Q Developer API using PostStream
	resp, err := a.client.PostStream(ctx, apiEndpoint, convStateReq)
	if err != nil {
		log.Printf("[%s] Failed to send message: %v", requestID, err)
		writeStreamAnthropicError(w, flusher, "api_error", "Failed to send message", err)
		return
	}
	defer resp.Body.Close()
	
	// Parse event stream using ParseKiroStream
	eventChan, err := streaming.ParseKiroStream(ctx, resp, 30*time.Second)
	if err != nil {
		log.Printf("[%s] Failed to parse stream: %v", requestID, err)
		writeStreamAnthropicError(w, flusher, "api_error", "Failed to parse stream", err)
		return
	}
	
	messageID := fmt.Sprintf("msg_%s", requestID)
	
	// Send message_start event
	startEvent := AnthropicStreamEvent{
		Type: "message_start",
		Message: &AnthropicResponse{
			ID:    messageID,
			Type:  "message",
			Role:  "assistant",
			Model: req.Model,
		},
	}
	sendAnthropicStreamEvent(w, flusher, &startEvent)
	
	contentIndex := 0
	
	for event := range eventChan {
		if event.Error != nil {
			log.Printf("[%s] Stream error: %v", requestID, event.Error)
			break
		}
		
		// Convert Kiro event to Anthropic format
		anthropicEvents := a.convertKiroEventToAnthropic(&event, contentIndex)
		for _, evt := range anthropicEvents {
			sendAnthropicStreamEvent(w, flusher, evt)
		}
	}
	
	// Send message_delta and message_stop events
	stopEvent := AnthropicStreamEvent{
		Type: "message_delta",
		Delta: &AnthropicDelta{
			StopReason: "end_turn",
		},
	}
	sendAnthropicStreamEvent(w, flusher, &stopEvent)
	
	finalEvent := AnthropicStreamEvent{
		Type: "message_stop",
	}
	sendAnthropicStreamEvent(w, flusher, &finalEvent)
	
	log.Printf("[%s] Streaming completed", requestID)
}

// handleNonStreamingMessages handles non-streaming Anthropic messages
func (a *AnthropicAdapter) handleNonStreamingMessages(ctx context.Context, w http.ResponseWriter, r *http.Request, req *AnthropicRequest, convStateReq *models.ConversationStateRequest, requestID string) {
	// Determine API endpoint using shared logic
	apiEndpoint := DetermineAPIEndpoint(a.authManager.GetAuthMode(), true)
	
	// Send request to AWS Q Developer API using PostStream
	resp, err := a.client.PostStream(ctx, apiEndpoint, convStateReq)
	if err != nil {
		log.Printf("[%s] Failed to send message: %v", requestID, err)
		writeAnthropicError(w, http.StatusInternalServerError, "api_error", "Failed to send message", err)
		return
	}
	defer resp.Body.Close()
	
	// Parse event stream and accumulate response
	eventChan, err := streaming.ParseKiroStream(ctx, resp, 30*time.Second)
	if err != nil {
		log.Printf("[%s] Failed to parse stream: %v", requestID, err)
		writeAnthropicError(w, http.StatusInternalServerError, "api_error", "Failed to parse stream", err)
		return
	}
	
	var content []AnthropicContentBlock
	var usage *AnthropicUsage
	
	for event := range eventChan {
		if event.Error != nil {
			log.Printf("[%s] Stream error: %v", requestID, event.Error)
			writeAnthropicError(w, http.StatusInternalServerError, "api_error", "Stream error", event.Error)
			return
		}
		
		// Accumulate content
		a.accumulateAnthropicEvent(&event, &content, &usage)
	}
	
	// Build final response
	messageID := fmt.Sprintf("msg_%s", requestID)
	
	response := AnthropicResponse{
		ID:         messageID,
		Type:       "message",
		Role:       "assistant",
		Content:    content,
		Model:      req.Model,
		StopReason: "end_turn",
		Usage:      usage,
	}
	
	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	
	log.Printf("[%s] Non-streaming completed", requestID)
}

// Helper functions

func writeAnthropicError(w http.ResponseWriter, status int, errorType, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	errorResp := map[string]interface{}{
		"type":  "error",
		"error": map[string]interface{}{
			"type":    errorType,
			"message": message,
		},
	}
	
	if err != nil {
		errorResp["error"].(map[string]interface{})["details"] = err.Error()
	}
	
	json.NewEncoder(w).Encode(errorResp)
}

func writeStreamAnthropicError(w http.ResponseWriter, flusher http.Flusher, errorType, message string, err error) {
	errorEvent := AnthropicStreamEvent{
		Type: "error",
	}
	
	data, _ := json.Marshal(errorEvent)
	fmt.Fprintf(w, "event: error\ndata: %s\n\n", data)
	flusher.Flush()
}

func sendAnthropicStreamEvent(w http.ResponseWriter, flusher http.Flusher, event *AnthropicStreamEvent) {
	data, _ := json.Marshal(event)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
	flusher.Flush()
}

func (a *AnthropicAdapter) convertKiroEventToAnthropic(event *streaming.StreamEvent, contentIndex int) []*AnthropicStreamEvent {
	var events []*AnthropicStreamEvent
	
	// Convert content events
	if event.Content != "" {
		events = append(events, &AnthropicStreamEvent{
			Type:  "content_block_delta",
			Index: contentIndex,
			Delta: &AnthropicDelta{
				Type: "text_delta",
				Text: event.Content,
			},
		})
	}
	
	// Convert thinking events
	if event.ThinkingContent != "" {
		events = append(events, &AnthropicStreamEvent{
			Type:  "content_block_delta",
			Index: contentIndex,
			Delta: &AnthropicDelta{
				Type: "text_delta",
				Text: event.ThinkingContent,
			},
		})
	}
	
	// Convert tool call events
	if event.ToolCall != nil {
		events = append(events, &AnthropicStreamEvent{
			Type:  "content_block_start",
			Index: contentIndex,
			ContentBlock: &AnthropicContentBlock{
				Type: "tool_use",
				ID:   event.ToolCall.ID,
				Name: event.ToolCall.Function.Name,
			},
		})
		
		// Parse tool arguments if available
		if event.ToolCall.Function.Arguments != "" {
			var input map[string]interface{}
			if err := json.Unmarshal([]byte(event.ToolCall.Function.Arguments), &input); err == nil {
				events = append(events, &AnthropicStreamEvent{
					Type:  "content_block_delta",
					Index: contentIndex,
					Delta: &AnthropicDelta{
						Type: "input_json_delta",
					},
				})
			}
		}
	}
	
	return events
}

func (a *AnthropicAdapter) accumulateAnthropicEvent(event *streaming.StreamEvent, content *[]AnthropicContentBlock, usage **AnthropicUsage) {
	// Accumulate content
	if event.Content != "" {
		// Append to last text block or create new one
		if len(*content) > 0 && (*content)[len(*content)-1].Type == "text" {
			(*content)[len(*content)-1].Text += event.Content
		} else {
			*content = append(*content, AnthropicContentBlock{
				Type: "text",
				Text: event.Content,
			})
		}
	}
	
	// Accumulate thinking content
	if event.ThinkingContent != "" {
		// Append to last text block or create new one
		if len(*content) > 0 && (*content)[len(*content)-1].Type == "text" {
			(*content)[len(*content)-1].Text += event.ThinkingContent
		} else {
			*content = append(*content, AnthropicContentBlock{
				Type: "text",
				Text: event.ThinkingContent,
			})
		}
	}
	
	// Accumulate tool calls
	if event.ToolCall != nil {
		var input map[string]interface{}
		if event.ToolCall.Function.Arguments != "" {
			json.Unmarshal([]byte(event.ToolCall.Function.Arguments), &input)
		}
		
		*content = append(*content, AnthropicContentBlock{
			Type:  "tool_use",
			ID:    event.ToolCall.ID,
			Name:  event.ToolCall.Function.Name,
			Input: input,
		})
	}
	
	// Accumulate usage
	if event.Usage != nil {
		*usage = &AnthropicUsage{
			InputTokens:  event.Usage.PromptTokens,
			OutputTokens: event.Usage.CompletionTokens,
		}
	}
}
