package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/yourusername/kiro-gateway-go/internal/auth"
	"github.com/yourusername/kiro-gateway-go/internal/client"
	"github.com/yourusername/kiro-gateway-go/internal/converters"
	"github.com/yourusername/kiro-gateway-go/internal/models"
	"github.com/yourusername/kiro-gateway-go/internal/streaming"
	"github.com/yourusername/kiro-gateway-go/internal/validation"
)

// OpenAIAdapter handles OpenAI-compatible API requests
type OpenAIAdapter struct {
	authManager *auth.AuthManager
	client      *client.Client
	validator   *validation.RequestValidator
	config      *AdapterConfig
}

// AdapterConfig holds configuration for adapters
type AdapterConfig struct {
	ProfileArn           string
	HiddenModels         []string
	InjectThinking       bool
	EnableExtendedContext bool
	EnableExtendedThinking bool
}

// NewOpenAIAdapter creates a new OpenAI adapter
func NewOpenAIAdapter(authManager *auth.AuthManager, client *client.Client, validator *validation.RequestValidator, config *AdapterConfig) *OpenAIAdapter {
	return &OpenAIAdapter{
		authManager: authManager,
		client:      client,
		validator:   validator,
		config:      config,
	}
}

// HandleChatCompletions handles POST /v1/chat/completions
func (a *OpenAIAdapter) HandleChatCompletions(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	ctx := context.WithValue(r.Context(), "requestID", requestID)
	
	log.Printf("[%s] OpenAI API: /v1/chat/completions", requestID)
	
	// Parse request
	var req models.ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	
	// Normalize model ID
	originalModel := req.Model
	req.Model = converters.NormalizeModelID(req.Model, a.config.HiddenModels)
	if originalModel != req.Model {
		log.Printf("[%s] Normalized model: %s -> %s", requestID, originalModel, req.Model)
	}
	
	// Validate request
	if err := a.validator.ValidateRequest(&req); err != nil {
		log.Printf("[%s] Validation error: %v", requestID, err)
		writeError(w, http.StatusBadRequest, "Validation failed", err)
		return
	}
	
	log.Printf("[%s] Model: %s, Stream: %v, Messages: %d", requestID, req.Model, req.Stream, len(req.Messages))
	
	// Generate conversation ID (nil for first message)
	var convID *string
	// For first message, use nil; for subsequent messages, use the ID
	// TODO: Implement conversation history tracking to reuse IDs
	// For now, always use nil to start new conversations
	convID = nil
	
	// Convert to Q Developer conversation state format using shared converter
	convStateReq, err := converters.ConvertOpenAIToConversationState(&req, convID, a.config.ProfileArn)
	if err != nil {
		log.Printf("[%s] Conversion error: %v", requestID, err)
		writeError(w, http.StatusInternalServerError, "Failed to convert request", err)
		return
	}
	
	// Log the request for debugging (redacted in production)
	if os.Getenv("DEBUG") == "true" || os.Getenv("DEBUG") == "1" {
		if reqBytes, err := json.MarshalIndent(convStateReq, "", "  "); err == nil {
			log.Printf("[%s] Conversation state request:\n%s", requestID, string(reqBytes))
		}
	} else {
		log.Printf("[%s] Conversation state request: [REDACTED] (set DEBUG=true to view)", requestID)
	}
	
	// Handle streaming vs non-streaming
	if req.Stream {
		a.handleStreamingRequest(ctx, w, r, &req, convStateReq, requestID)
	} else {
		a.handleNonStreamingRequest(ctx, w, r, &req, convStateReq, requestID)
	}
}

// handleStreamingRequest handles streaming chat completions
func (a *OpenAIAdapter) handleStreamingRequest(ctx context.Context, w http.ResponseWriter, r *http.Request, req *models.ChatCompletionRequest, convStateReq *models.ConversationStateRequest, requestID string) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "Streaming not supported", nil)
		return
	}
	
	// Determine API endpoint using shared logic
	apiEndpoint := DetermineAPIEndpoint(a.authManager.GetAuthMode(), true)
	
	// Send request to AWS Q Developer API using PostStream
	resp, err := a.client.PostStream(ctx, apiEndpoint, convStateReq)
	if err != nil {
		log.Printf("[%s] Failed to send message: %v", requestID, err)
		writeStreamError(w, flusher, "Failed to send message", err)
		return
	}
	defer resp.Body.Close()
	
	// Parse event stream using ParseKiroStream
	eventChan, err := streaming.ParseKiroStream(ctx, resp, 30*time.Second)
	if err != nil {
		log.Printf("[%s] Failed to parse stream: %v", requestID, err)
		writeStreamError(w, flusher, "Failed to parse stream", err)
		return
	}
	
	var fullContent strings.Builder
	var toolCalls []models.ToolCall
	messageID := fmt.Sprintf("chatcmpl-%s", requestID)
	created := time.Now().Unix()
	
	for event := range eventChan {
		if event.Error != nil {
			log.Printf("[%s] Stream error: %v", requestID, event.Error)
			break
		}
		
		// Convert Kiro event to OpenAI format
		chunk := convertStreamEventToOpenAI(&event, messageID, req.Model, created, &fullContent, &toolCalls)
		if chunk != nil {
			// Send SSE chunk
			data, _ := json.Marshal(chunk)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
	
	// Send final [DONE] message
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
	
	log.Printf("[%s] Streaming completed", requestID)
}

// handleNonStreamingRequest handles non-streaming chat completions
func (a *OpenAIAdapter) handleNonStreamingRequest(ctx context.Context, w http.ResponseWriter, r *http.Request, req *models.ChatCompletionRequest, convStateReq *models.ConversationStateRequest, requestID string) {
	// Determine API endpoint using shared logic
	apiEndpoint := DetermineAPIEndpoint(a.authManager.GetAuthMode(), true)
	
	// Send request to AWS Q Developer API using PostStream
	resp, err := a.client.PostStream(ctx, apiEndpoint, convStateReq)
	if err != nil {
		log.Printf("[%s] Failed to send message: %v", requestID, err)
		writeError(w, http.StatusInternalServerError, "Failed to send message", err)
		return
	}
	defer resp.Body.Close()
	
	// Parse event stream and accumulate response
	eventChan, err := streaming.ParseKiroStream(ctx, resp, 30*time.Second)
	if err != nil {
		log.Printf("[%s] Failed to parse stream: %v", requestID, err)
		writeError(w, http.StatusInternalServerError, "Failed to parse stream", err)
		return
	}
	
	var fullContent strings.Builder
	var toolCalls []models.ToolCall
	var usage *models.TokenUsage
	
	for event := range eventChan {
		if event.Error != nil {
			log.Printf("[%s] Stream error: %v", requestID, event.Error)
			writeError(w, http.StatusInternalServerError, "Stream error", event.Error)
			return
		}
		
		// Accumulate content
		accumulateStreamEvent(&event, &fullContent, &toolCalls, &usage)
	}
	
	// Build final response
	messageID := fmt.Sprintf("chatcmpl-%s", requestID)
	created := time.Now().Unix()
	
	response := models.ChatCompletionResponse{
		ID:      messageID,
		Object:  "chat.completion",
		Created: created,
		Model:   req.Model,
		Choices: []models.ChatCompletionChoice{
			{
				Index: 0,
				Message: &models.ResponseMessage{
					Role:      "assistant",
					Content:   fullContent.String(),
					ToolCalls: toolCalls,
				},
				FinishReason: determineFinishReason(toolCalls),
			},
		},
		Usage: usage,
	}
	
	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	
	log.Printf("[%s] Non-streaming completed", requestID)
}

// HandleModels handles GET /v1/models
func (a *OpenAIAdapter) HandleModels(w http.ResponseWriter, r *http.Request) {
	// Return list of available models
	modelIDs := []string{
		"claude-sonnet-4-5",
		"claude-3-5-sonnet-20241022",
		"claude-3-5-sonnet-20240620",
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
	}
	
	// Add hidden models if configured
	modelIDs = append(modelIDs, a.config.HiddenModels...)
	
	modelList := models.NewModelList(modelIDs)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(modelList)
}

// Helper functions

func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

func writeError(w http.ResponseWriter, status int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	errorResp := map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
			"type":    "invalid_request_error",
			"code":    status,
		},
	}
	
	if err != nil {
		errorResp["error"].(map[string]interface{})["details"] = err.Error()
	}
	
	json.NewEncoder(w).Encode(errorResp)
}

func writeStreamError(w http.ResponseWriter, flusher http.Flusher, message string, err error) {
	errorData := map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
			"type":    "api_error",
		},
	}
	
	if err != nil {
		errorData["error"].(map[string]interface{})["details"] = err.Error()
	}
	
	data, _ := json.Marshal(errorData)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

func convertStreamEventToOpenAI(event *streaming.StreamEvent, messageID, model string, created int64, content *strings.Builder, toolCalls *[]models.ToolCall) *models.ChatCompletionResponse {
	// Convert StreamEvent to OpenAI streaming chunk
	
	if event.Content != "" {
		content.WriteString(event.Content)
		
		return &models.ChatCompletionResponse{
			ID:      messageID,
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
	}
	
	if event.ThinkingContent != "" {
		return &models.ChatCompletionResponse{
			ID:      messageID,
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
	}
	
	if event.ToolCall != nil {
		*toolCalls = append(*toolCalls, *event.ToolCall)
		
		return &models.ChatCompletionResponse{
			ID:      messageID,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   model,
			Choices: []models.ChatCompletionChoice{
				{
					Index: 0,
					Delta: &models.ResponseMessage{
						ToolCalls: []models.ToolCall{*event.ToolCall},
					},
				},
			},
		}
	}
	
	return nil
}

func accumulateStreamEvent(event *streaming.StreamEvent, content *strings.Builder, toolCalls *[]models.ToolCall, usage **models.TokenUsage) {
	if event.Content != "" {
		content.WriteString(event.Content)
	}
	
	if event.ThinkingContent != "" {
		content.WriteString(event.ThinkingContent)
	}
	
	if event.ToolCall != nil {
		*toolCalls = append(*toolCalls, *event.ToolCall)
	}
	
	if event.Usage != nil {
		*usage = event.Usage
	}
}

func determineFinishReason(toolCalls []models.ToolCall) string {
	if len(toolCalls) > 0 {
		return "tool_calls"
	}
	return "stop"
}
