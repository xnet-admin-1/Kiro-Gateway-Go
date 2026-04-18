package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yourusername/kiro-gateway-go/internal/adapters"
	"github.com/yourusername/kiro-gateway-go/internal/auth"
	"github.com/yourusername/kiro-gateway-go/internal/cache"
	"github.com/yourusername/kiro-gateway-go/internal/client"
	"github.com/yourusername/kiro-gateway-go/internal/concurrency"
	"github.com/yourusername/kiro-gateway-go/internal/converters"
	"github.com/yourusername/kiro-gateway-go/internal/errors"
	"github.com/yourusername/kiro-gateway-go/internal/logging"
	"github.com/yourusername/kiro-gateway-go/internal/models"
	"github.com/yourusername/kiro-gateway-go/internal/streaming"
	"github.com/yourusername/kiro-gateway-go/internal/validation"
)

// RequestContext holds request-specific information
type RequestContext struct {
	RequestID      string
	StartTime      time.Time
	ConversationID string // Q Developer conversation ID for follow-up requests
	ProfileArn     string // Profile ARN for Identity Center mode
	OriginalReq    *models.ConversationStateRequest // Original request for context preservation
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

// handleChatCompletions handles POST /v1/chat/completions
func (h *Handler) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Generate request ID and add to context
	requestID := generateRequestID()
	baseCtx := context.WithValue(r.Context(), "requestID", requestID)
	r = r.WithContext(baseCtx)
	
	reqCtx := &RequestContext{
		RequestID:   requestID,
		StartTime:   time.Now(),
		ProfileArn:  "", // Will be set after auth manager is accessed
		OriginalReq: nil, // Will be set after conversion
	}
	
	// Ensure context cleanup after response completion
	// This prevents memory leaks from abandoned conversations
	defer func() {
		if h.contextManager != nil {
			h.contextManager.Remove(requestID)
			log.Printf("[%s] Conversation context cleaned up", requestID)
		}
	}()
	
	log.Printf("[%s] Request to /v1/chat/completions", requestID)
	
	// Parse request
	var req models.ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorWithRequestID(w, http.StatusBadRequest, "Invalid request body", err, requestID)
		return
	}
	
	// Normalize model ID BEFORE validation (strip Bedrock prefix/suffix)
	originalModel := req.Model
	req.Model = converters.NormalizeModelID(req.Model, h.config.HiddenModels)
	if originalModel != req.Model {
		log.Printf("[%s] Normalized model: %s -> %s", requestID, originalModel, req.Model)
	}
	
	// Validate request against AWS Q Developer limits
	if err := h.validator.ValidateRequest(&req); err != nil {
		log.Printf("[%s] Validation error: %v", requestID, err)
		if valErr, ok := err.(*validation.ValidationError); ok {
			h.writeValidationError(w, valErr, http.StatusBadRequest)
		} else {
			h.writeErrorWithRequestID(w, http.StatusBadRequest, "Validation failed", err, requestID)
		}
		return
	}
	
	// Log beta feature warnings if enabled
	if h.config.BetaFeatures.WarnOnBetaFeatures {
		if h.config.BetaFeatures.EnableExtendedContext {
			if modelLimit, err := validation.GetModelLimit(req.Model); err == nil && modelLimit.SupportsExtendedContext {
				log.Printf("[%s] BETA FEATURE: Extended context window (1M tokens) enabled for %s", requestID, req.Model)
			}
		}
		if h.config.BetaFeatures.EnableExtendedThinking {
			if modelLimit, err := validation.GetModelLimit(req.Model); err == nil && modelLimit.SupportsExtendedThinking {
				log.Printf("[%s] BETA FEATURE: Extended thinking enabled for %s", requestID, req.Model)
			}
		}
	}
	
	
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		userID = "anonymous"
	}
	
	// Determine priority based on user
	priority := concurrency.GetPriorityFromUser(userID, nil)
	
	// Check load shedding
	if h.loadShedder != nil {
		if shouldShed, reason := h.loadShedder.ShouldShed(priority); shouldShed {
			retryAfter := h.loadShedder.GetRetryAfter()
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
			h.writeErrorWithRequestID(w, http.StatusServiceUnavailable, reason, nil, requestID)
			log.Printf("[%s] Request shed due to high load (priority: %s, retry after: %v)", 
				requestID, concurrency.GetPriorityName(priority), retryAfter)
			return
		}
	}
	
	log.Printf("[%s] Model: %s, Stream: %v, Messages: %d, Priority: %s", 
		requestID, req.Model, req.Stream, len(req.Messages), concurrency.GetPriorityName(priority))
	
	// Use normalized model ID
	modelID := req.Model
	
	// Generate conversation ID - use nil for first message
	// According to Q Developer API spec: null for first message, UUID for subsequent
	var convID *string = nil
	
	// Get profile ARN (for Identity Center mode with bearer token or SigV4)
	profileArn := h.authManager.GetProfileArn()
	
	// Store profile ARN in request context
	reqCtx.ProfileArn = profileArn
	
	
	// Convert to Q Developer conversation state format
	convStateReq, err := converters.ConvertOpenAIToConversationState(&req, convID, profileArn)
	if err != nil {
		h.writeErrorWithRequestID(w, http.StatusBadRequest, "Failed to convert request", err, requestID)
		return
	}
	
	// Store original request in context for tool result follow-ups
	reqCtx.OriginalReq = convStateReq
	
	// Log the request for debugging (redacted in production)
	if os.Getenv("DEBUG") == "true" || os.Getenv("DEBUG") == "1" {
		if reqBytes, err := json.MarshalIndent(convStateReq, "", "  "); err == nil {
			log.Printf("[%s] Conversation state request:\n%s", requestID, string(reqBytes))
		}
	} else {
	}
	
	// Detect if multimodal content (images) is present
	hasImages := false
	if convStateReq.ConversationState != nil && 
	   convStateReq.ConversationState.CurrentMessage.UserInputMessage != nil &&
	   len(convStateReq.ConversationState.CurrentMessage.UserInputMessage.Images) > 0 {
		hasImages = true
		log.Printf("[%s] Multimodal content detected: %d image(s)", requestID, 
			len(convStateReq.ConversationState.CurrentMessage.UserInputMessage.Images))
	}
	
	// Determine timeout based on content type
	timeout := h.config.FirstTokenTimeout
	if hasImages {
		timeout = h.config.MultimodalFirstTokenTimeout
	}
	
	// Create context with appropriate timeout
	// For streaming: use long timeout since response can take minutes
	// For non-streaming: use first-token timeout
	var ctx context.Context
	var cancel context.CancelFunc
	if req.Stream {
		ctx, cancel = context.WithTimeout(baseCtx, 10*time.Minute)
	} else {
		ctx, cancel = context.WithTimeout(baseCtx, timeout)
	}
	defer cancel()
	
	// Create HTTP client
	httpClient := client.NewClient(h.authManager, client.Config{
		MaxConnections:       h.config.MaxConnections,
		KeepAliveConnections: h.config.KeepAliveConnections,
		ConnectionTimeout:    h.config.ConnectionTimeout,
		MaxRetries:           3,
		RequestID:            requestID,
		UseQDeveloper:        true,
		APIEndpoint:          h.config.APIEndpoint,
	})
	defer httpClient.Close()
	
	apiEndpoint := adapters.DetermineAPIEndpoint(h.authManager.GetAuthMode(), true)
	var cachedResponse *models.ChatCompletionResponse
	hasTools := req.Tools != nil && len(req.Tools) > 0
	shouldUseCache := !req.Stream && !hasImages && !hasTools && h.responseCache != nil && h.responseCache.IsEnabled()
	
	if shouldUseCache {
		cacheKey := cache.CacheKey{
			ModelID:      req.Model,
			Prompt:       extractPromptFromMessages(req.Messages),
			SystemPrompt: extractSystemPrompt(req.Messages),
			Temperature:  req.Temperature,
			MaxTokens:    req.MaxTokens,
		}
		
		if cacheKey.IsCacheable() {
			if cached, found := h.responseCache.Get(cacheKey); found {
				log.Printf("[%s] CACHE HIT", requestID)
				var response models.ChatCompletionResponse
				if err := json.Unmarshal([]byte(cached.Response), &response); err == nil {
					cachedResponse = &response
				}
			}
		}
	}
	
	// If we have a cached response, use it
	if cachedResponse != nil {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-ID", requestID)
		w.Header().Set("X-Cache-Hit", "true")
		if err := json.NewEncoder(w).Encode(cachedResponse); err != nil {
			log.Printf("[%s] Failed to encode cached response: %v", requestID, err)
			h.writeErrorWithRequestID(w, http.StatusInternalServerError, "Failed to encode response", err, requestID)
		}
		duration := time.Since(reqCtx.StartTime)
		log.Printf("[%s] Request completed from cache in %v", requestID, duration)
		return
	}
	
	// Make request to Q Developer API with timeout context
	resp, err := httpClient.PostStream(ctx, apiEndpoint, convStateReq)
	if err != nil {
		log.Printf("[%s] HTTP request error: %v", requestID, err)
		// Classify network/connection errors
		apiErr := &errors.APIError{
			Kind:      errors.ErrorKindInternalServerError,
			Message:   "Failed to connect to Kiro API",
			RequestID: requestID,
			Cause:     err,
		}
		h.writeAPIErrorWithRequestID(w, apiErr, requestID)
		return
	}
	
	// Log response status for debugging
	log.Printf("[%s] API response status: %d %s", requestID, resp.StatusCode, resp.Status)
	log.Printf("[%s] API response headers: Content-Type=%s", requestID, resp.Header.Get("Content-Type"))
	
	// Check response status and classify errors
	if resp.StatusCode != http.StatusOK {
		body, _ := client.ReadBody(resp)
		log.Printf("[%s] AWS API error: %d - %s", requestID, resp.StatusCode, string(body))
		log.Printf("[%s] Request was sent to: q.%s.amazonaws.com%s", requestID, 
			h.authManager.GetRegion(), apiEndpoint)
		
		// Log full response headers for debugging
		log.Printf("[%s] Response headers: %v", requestID, resp.Header)
		
		// Log full request details for debugging
		log.Printf("[%s] Request method: POST", requestID)
		log.Printf("[%s] Request URL: https://q.%s.amazonaws.com%s", requestID,
			h.authManager.GetRegion(), apiEndpoint)
		
		// Try to parse AWS error response
		var awsError map[string]interface{}
		if err := json.Unmarshal(body, &awsError); err == nil {
			log.Printf("[%s] AWS error response: %+v", requestID, awsError)
		}
		
		// Classify the error with request ID
		apiErr := errors.ClassifyError(resp.StatusCode, body, nil)
		if apiErr.RequestID == "" {
			apiErr.RequestID = requestID
		}
		
		h.writeAPIErrorWithRequestID(w, apiErr, requestID)
		return
	}
	
	// Handle streaming vs non-streaming
	if req.Stream {
		h.handleStreamingWithRequestID(w, ctx, resp, modelID, &req, reqCtx)
	} else {
		h.handleNonStreamingWithRequestID(w, ctx, resp, modelID, &req, reqCtx, convStateReq)
	}
	
	duration := time.Since(reqCtx.StartTime)
	log.Printf("[%s] Request completed in %v", requestID, duration)
	
	// Increment quota usage
	if userID, ok := r.Context().Value("userID").(string); ok {
		h.quotaTracker.IncrementQuota(userID, 1, 1) // 1 agentic request, 1 inference call
	}
}

// handleStreamingWithRequestID handles streaming response with request ID tracking
func (h *Handler) handleStreamingWithRequestID(w http.ResponseWriter, ctx context.Context, resp *http.Response, model string, req *models.ChatCompletionRequest, reqCtx *RequestContext) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Request-ID", reqCtx.RequestID)
	
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.writeErrorWithRequestID(w, http.StatusInternalServerError, "Streaming not supported", nil, reqCtx.RequestID)
		return
	}
	flusher.Flush() // Send headers immediately so client knows stream is open
	
	// Convert Kiro stream to OpenAI format with conversation ID callback
	eventChan, err := streaming.StreamKiroToOpenAIWithCallback(ctx, resp, model, h.config.FirstTokenTimeout, nil, nil, func(conversationID string) {
		// Capture conversation ID for follow-up requests
		log.Printf("[%s] Captured conversation ID: %s", reqCtx.RequestID, conversationID)
		reqCtx.ConversationID = conversationID
	})
	if err != nil {
		h.writeErrorWithRequestID(w, http.StatusInternalServerError, "Failed to start streaming", err, reqCtx.RequestID)
		return
	}
	
	log.Printf("[%s] Starting streaming response", reqCtx.RequestID)
	
	// Collect tool uses for MCP execution
	var toolUses []models.ToolUse
	
	// Stream events to client with pacing for real SSE experience
	chunkCount := 0
	for chunk := range eventChan {
		fmt.Fprint(w, chunk)
		flusher.Flush()
		chunkCount++
	}
	
	// Tool calls are now forwarded to clients in OpenAI format
	// Clients handle MCP tool execution locally
	if len(toolUses) > 0 {
		log.Printf("[%s] Forwarded %d tool call(s) to client for local execution", reqCtx.RequestID, len(toolUses))
	}
	
	log.Printf("[%s] Streaming completed (%d chunks sent)", reqCtx.RequestID, chunkCount)
}

// handleNonStreamingWithRequestID handles non-streaming response with request ID tracking
func (h *Handler) handleNonStreamingWithRequestID(w http.ResponseWriter, ctx context.Context, resp *http.Response, model string, req *models.ChatCompletionRequest, reqCtx *RequestContext, convStateReq *models.ConversationStateRequest) {
	// Convert messages to map format for tokenizer
	requestMessages := convertMessagesToMap(req.Messages)
	var requestTools []map[string]interface{}
	if req.Tools != nil {
		requestTools = convertToolsToMap(req.Tools)
	}
	
	// Collect full response
	response, err := streaming.CollectNonStreamingResponse(ctx, resp, model, h.config.FirstTokenTimeout, requestMessages, requestTools)
	if err != nil {
		h.writeErrorWithRequestID(w, http.StatusInternalServerError, "Failed to collect response", err, reqCtx.RequestID)
		return
	}
	
	// Store in cache if enabled and not multimodal
	hasImages := convStateReq.ConversationState != nil && 
		convStateReq.ConversationState.CurrentMessage.UserInputMessage != nil &&
		len(convStateReq.ConversationState.CurrentMessage.UserInputMessage.Images) > 0
	
	if h.responseCache != nil && h.responseCache.IsEnabled() && !hasImages {
		// Create cache key
		cacheKey := cache.CacheKey{
			ModelID:      model,
			Prompt:       extractPromptFromMessages(req.Messages),
			SystemPrompt: extractSystemPrompt(req.Messages),
			Temperature:  req.Temperature,
			MaxTokens:    req.MaxTokens,
		}
		
		if cacheKey.IsCacheable() {
			// Serialize response
			if responseBytes, err := json.Marshal(response); err == nil {
				h.responseCache.Set(cacheKey, string(responseBytes))
				log.Printf("[%s] Response cached", reqCtx.RequestID)
			} else {
				log.Printf("[%s] Failed to serialize response for caching: %v", reqCtx.RequestID, err)
			}
		}
	}
	
	// Note: Request ID is included in response headers
	
	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", reqCtx.RequestID)
	json.NewEncoder(w).Encode(response)
	
	log.Printf("[%s] Non-streaming completed", reqCtx.RequestID)
}

// convertMessagesToMap converts messages to map format for tokenizer
func convertMessagesToMap(messages []models.Message) []map[string]interface{} {
	result := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		msgMap := map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
		if msg.ToolCalls != nil {
			toolCalls := make([]interface{}, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				toolCalls[j] = map[string]interface{}{
					"id":   tc.ID,
					"type": tc.Type,
					"function": map[string]interface{}{
						"name":      tc.Function.Name,
						"arguments": tc.Function.Arguments,
					},
				}
			}
			msgMap["tool_calls"] = toolCalls
		}
		result[i] = msgMap
	}
	return result
}

// extractPromptFromMessages extracts the user prompt from messages
func extractPromptFromMessages(messages []models.Message) string {
	// Find the last user message
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			// Handle both string and interface{} content
			if content, ok := messages[i].Content.(string); ok {
				return content
			}
			return fmt.Sprintf("%v", messages[i].Content)
		}
	}
	return ""
}

// extractSystemPrompt extracts the system prompt from messages
func extractSystemPrompt(messages []models.Message) string {
	// Find the first system message
	for _, msg := range messages {
		if msg.Role == "system" {
			// Handle both string and interface{} content
			if content, ok := msg.Content.(string); ok {
				return content
			}
			return fmt.Sprintf("%v", msg.Content)
		}
	}
	return ""
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// convertToolsToMap converts tools to map format for tokenizer
func convertToolsToMap(tools []models.Tool) []map[string]interface{} {
	result := make([]map[string]interface{}, len(tools))
	for i, tool := range tools {
		result[i] = map[string]interface{}{
			"type": tool.Type,
			"function": map[string]interface{}{
				"name":        tool.Function.Name,
				"description": tool.Function.Description,
				"parameters":  tool.Function.Parameters,
			},
		}
	}
	return result
}

// writeErrorWithRequestID writes an error response with request ID
func (h *Handler) writeErrorWithRequestID(w http.ResponseWriter, statusCode int, message string, err error, requestID string) {
	log.Printf("[%s] Error: %s - %v", requestID, message, err)
	
	errorMsg := message
	if err != nil {
		errorMsg = fmt.Sprintf("%s: %v", message, err)
	}
	
	response := map[string]interface{}{
		"error": map[string]interface{}{
			"message":    errorMsg,
			"type":       "api_error",
			"code":       statusCode,
			"request_id": requestID,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// writeAPIErrorWithRequestID writes a classified API error response with request ID
func (h *Handler) writeAPIErrorWithRequestID(w http.ResponseWriter, apiErr *errors.APIError, requestID string) {
	// Ensure request ID is set
	if apiErr.RequestID == "" {
		apiErr.RequestID = requestID
	}
	
	// Determine HTTP status code
	statusCode := apiErr.StatusCode
	if statusCode == 0 {
		// Default status codes based on error kind
		switch apiErr.Kind {
		case errors.ErrorKindThrottling, errors.ErrorKindModelOverloaded, errors.ErrorKindMonthlyLimitReached:
			statusCode = http.StatusTooManyRequests
		case errors.ErrorKindContextWindowOverflow, errors.ErrorKindInvalidRequest:
			statusCode = http.StatusBadRequest
		case errors.ErrorKindAuthenticationFailed:
			statusCode = http.StatusUnauthorized
		case errors.ErrorKindInternalServerError:
			statusCode = http.StatusInternalServerError
		default:
			statusCode = http.StatusInternalServerError
		}
	}
	
	// Log the error with classification and request ID
	log.Printf("[%s] API Error [%s]: %s (Status: %d)", 
		requestID, apiErr.Kind.String(), apiErr.Message, statusCode)
	
	// Format the error response using the error classifier's formatter
	response := errors.FormatErrorResponse(apiErr, statusCode)
	
	// Write response with request ID header
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// Legacy functions for backward compatibility
func writeError(w http.ResponseWriter, statusCode int, message string, err error) {
	log.Printf("Error: %s - %v", message, err)
	
	errorMsg := message
	if err != nil {
		errorMsg = fmt.Sprintf("%s: %v", message, err)
	}
	
	response := map[string]interface{}{
		"error": map[string]interface{}{
			"message": errorMsg,
			"type":    "api_error",
			"code":    statusCode,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// writeAPIError writes a classified API error response (legacy)
func writeAPIError(w http.ResponseWriter, apiErr *errors.APIError) {
	// Determine HTTP status code
	statusCode := apiErr.StatusCode
	if statusCode == 0 {
		// Default status codes based on error kind
		switch apiErr.Kind {
		case errors.ErrorKindThrottling, errors.ErrorKindModelOverloaded, errors.ErrorKindMonthlyLimitReached:
			statusCode = http.StatusTooManyRequests
		case errors.ErrorKindContextWindowOverflow, errors.ErrorKindInvalidRequest:
			statusCode = http.StatusBadRequest
		case errors.ErrorKindAuthenticationFailed:
			statusCode = http.StatusUnauthorized
		case errors.ErrorKindInternalServerError:
			statusCode = http.StatusInternalServerError
		default:
			statusCode = http.StatusInternalServerError
		}
	}
	
	// Log the error with classification
	log.Printf("API Error [%s]: %s (Status: %d, RequestID: %s)", 
		apiErr.Kind.String(), apiErr.Message, statusCode, apiErr.RequestID)
	
	// Format the error response using the error classifier's formatter
	response := errors.FormatErrorResponse(apiErr, statusCode)
	
	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// writeValidationError writes a validation error response
func (h *Handler) writeValidationError(w http.ResponseWriter, valErr *validation.ValidationError, statusCode int) {
	log.Printf("Validation error: %v", valErr)
	
	response := map[string]interface{}{
		"error": map[string]interface{}{
			"message": valErr.Message,
			"type":    "validation_error",
			"code":    statusCode,
			"field":   valErr.Field,
		},
	}
	
	// Add limit and actual values if available
	if valErr.Limit != nil {
		response["error"].(map[string]interface{})["limit"] = valErr.Limit
	}
	if valErr.Actual != nil {
		response["error"].(map[string]interface{})["actual"] = valErr.Actual
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// NOTE: Tool execution removed - clients should handle MCP tool calling
// This gateway is a pure AWS Q Developer proxy
// Tool uses are logged but not executed server-side

// convertMCPContentToQ converts MCP SDK content to Q Developer format
func convertMCPContentToQ(content []sdk.Content) []models.ToolResultContent {
	result := make([]models.ToolResultContent, 0, len(content))
	
	for _, c := range content {
		// Use type assertion to determine content type
		switch v := c.(type) {
		case *sdk.TextContent:
			result = append(result, models.ToolResultContent{
				Text: v.Text,
			})
		case *sdk.ImageContent:
			// Q Developer doesn't support image tool results yet
			// Convert to text description
			result = append(result, models.ToolResultContent{
				Text: "[Image content not supported in tool results]",
			})
		case *sdk.EmbeddedResource:
			// Convert resource to JSON
			jsonBytes, _ := json.Marshal(v)
			result = append(result, models.ToolResultContent{
				JSON: string(jsonBytes),
			})
		default:
			// Unknown content type - convert to JSON
			jsonBytes, _ := json.Marshal(c)
			result = append(result, models.ToolResultContent{
				JSON: string(jsonBytes),
			})
		}
	}
	
	return result
}

// sendFollowUpRequestWithStatus sends a follow-up request to Q Developer API with tool results
// and returns the HTTP status code along with any error
func (h *Handler) sendFollowUpRequestWithStatus(ctx context.Context, followUpReq *models.ConversationStateRequest, reqCtx *RequestContext) (int, error) {
	log.Printf("[%s] Sending follow-up request to Q Developer with tool results", reqCtx.RequestID)
	
	// Create HTTP client with Q Developer configuration
	// Use the same configuration as the original request
	httpClient := client.NewClient(h.authManager, client.Config{
		MaxConnections:       h.config.MaxConnections,
		KeepAliveConnections: h.config.KeepAliveConnections,
		ConnectionTimeout:    h.config.ConnectionTimeout,
		MaxRetries:           3,
		RequestID:            reqCtx.RequestID + "-followup",
		UseQDeveloper:        true, // Always use Q Developer endpoint
		APIEndpoint:          h.config.APIEndpoint,
		Logger:               log.Default(),
	})
	defer httpClient.Close()
	
	// Determine API endpoint using shared logic
	apiEndpoint := adapters.DetermineAPIEndpoint(h.authManager.GetAuthMode(), true)
	
	log.Printf("[%s] Follow-up request endpoint: %s (Auth: %s)", reqCtx.RequestID, apiEndpoint,
		map[auth.AuthMode]string{auth.AuthModeBearerToken: "Bearer", auth.AuthModeSigV4: "SigV4"}[h.authManager.GetAuthMode()])
	
	// Create context with timeout for follow-up request
	// Use standard timeout since tool results are typically small
	followUpCtx, cancel := context.WithTimeout(ctx, h.config.FirstTokenTimeout)
	defer cancel()
	
	// Make request to Q Developer API
	resp, err := httpClient.PostStream(followUpCtx, apiEndpoint, followUpReq)
	if err != nil {
		log.Printf("[%s] Follow-up HTTP request error: %v", reqCtx.RequestID, err)
		return 0, fmt.Errorf("failed to send follow-up request: %w", err)
	}
	
	// Log response status
	log.Printf("[%s] Follow-up API response status: %d %s", reqCtx.RequestID, resp.StatusCode, resp.Status)
	
	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := client.ReadBody(resp)
		log.Printf("[%s] Follow-up API error: %d - %s", reqCtx.RequestID, resp.StatusCode, string(body))
		return resp.StatusCode, fmt.Errorf("follow-up request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	// Stream the response back to the client
	// Note: In the current implementation, we don't have direct access to the response writer
	// from within this function. The response will be logged but not streamed to the client.
	// This is acceptable for now since tool execution happens after the initial response.
	// In a future iteration, we could implement a callback mechanism to stream the response.
	
	// For now, just consume and log the response
	log.Printf("[%s] Processing follow-up response from Q Developer", reqCtx.RequestID)
	
	// Read and log the response for debugging
	// In production, this would be streamed to the client
	body, err := client.ReadBody(resp)
	if err != nil {
		log.Printf("[%s] Failed to read follow-up response: %v", reqCtx.RequestID, err)
		return resp.StatusCode, fmt.Errorf("failed to read follow-up response: %w", err)
	}
	
	log.Printf("[%s] Follow-up response received (%s)", reqCtx.RequestID, logging.FormatSize(len(body)))
	
	// Log first 500 characters for debugging (avoid logging huge responses)
	if len(body) > 500 {
		log.Printf("[%s] Follow-up response preview: %s...", reqCtx.RequestID, string(body[:500]))
	} else {
		log.Printf("[%s] Follow-up response: %s", reqCtx.RequestID, string(body))
	}
	
	return resp.StatusCode, nil
}

// sendFollowUpRequest sends a follow-up request (legacy wrapper)
func (h *Handler) sendFollowUpRequest(ctx context.Context, followUpReq *models.ConversationStateRequest, reqCtx *RequestContext) error {
	_, err := h.sendFollowUpRequestWithStatus(ctx, followUpReq, reqCtx)
	return err
}
