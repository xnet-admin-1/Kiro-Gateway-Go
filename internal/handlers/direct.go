package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/yourusername/kiro-gateway-go/internal/auth"
	"github.com/yourusername/kiro-gateway-go/internal/client"
	"github.com/yourusername/kiro-gateway-go/internal/concurrency"
	"github.com/yourusername/kiro-gateway-go/internal/errors"
	"github.com/yourusername/kiro-gateway-go/internal/logging"
)

// handleDirectChat handles POST /api/chat (Q Developer native format)
// PURE PASSTHROUGH: Only adds auth headers, forwards raw bytes
func (h *Handler) handleDirectChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Generate request ID
	requestID := generateRequestID()
	ctx := context.WithValue(r.Context(), "requestID", requestID)
	r = r.WithContext(ctx)
	
	reqCtx := &RequestContext{
		RequestID:  requestID,
		StartTime:  time.Now(),
		ProfileArn: h.authManager.GetProfileArn(),
	}
	
	logging.DebugLog("[%s] Request to /api/chat (Direct API - Pure Passthrough)", requestID)
	
	// Read raw request body (pure passthrough - don't unmarshal!)
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeErrorWithRequestID(w, http.StatusBadRequest, "Failed to read request body", err, requestID)
		return
	}
	r.Body.Close()
	
	logging.DebugLog("[%s] Received request body: %d bytes", requestID, len(bodyBytes))
	
	// Validate JSON is well-formed (but don't unmarshal into structs)
	var jsonCheck interface{}
	if err := json.Unmarshal(bodyBytes, &jsonCheck); err != nil {
		h.writeErrorWithRequestID(w, http.StatusBadRequest, "Invalid JSON", err, requestID)
		return
	}
	
	logging.DebugLog("[%s] JSON validation passed", requestID)
	
	// Save to debug file if debug mode is enabled
	if logging.IsDebugEnabled() {
		debugPath := "logs/debug-request.json"
		if err := os.WriteFile(debugPath, bodyBytes, 0644); err == nil {
			logging.DebugLog("[%s] Request body saved to %s", requestID, debugPath)
		}
	}
	
	// Create context with timeout
	timeout := 5 * time.Minute // Default timeout for streaming
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	// Get user ID and priority
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		userID = "anonymous"
	}
	priority := concurrency.GetPriorityFromUser(userID, nil)
	
	// Check load shedding
	if h.loadShedder != nil {
		if shouldShed, reason := h.loadShedder.ShouldShed(priority); shouldShed {
			retryAfter := h.loadShedder.GetRetryAfter()
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
			h.writeErrorWithRequestID(w, http.StatusServiceUnavailable, reason, nil, requestID)
			return
		}
	}
	
	// Create HTTP client
	httpClient := client.NewClient(h.authManager, client.Config{
		MaxConnections:       h.config.MaxConnections,
		KeepAliveConnections: h.config.KeepAliveConnections,
		ConnectionTimeout:    h.config.ConnectionTimeout,
		MaxRetries:           3,
		RequestID:            requestID,
		UseQDeveloper:        true,
		APIEndpoint:          h.config.APIEndpoint,
		Logger:               log.Default(),
	})
	defer httpClient.Close()
	
	// Determine API endpoint
	// Q Developer API uses root endpoint "/" with X-Amz-Target header
	apiEndpoint := "/"
	logging.DebugLog("[%s] Using API endpoint: %s (Auth: %s)", requestID, apiEndpoint,
		map[auth.AuthMode]string{auth.AuthModeBearerToken: "Bearer", auth.AuthModeSigV4: "SigV4"}[h.authManager.GetAuthMode()])
	
	// Make request to Q Developer API with RAW BYTES (no unmarshaling/re-marshaling)
	logging.DebugLog("[%s] Forwarding raw request to AWS (pure passthrough)", requestID)
	resp, err := httpClient.PostStreamRaw(ctx, apiEndpoint, bodyBytes)
	if err != nil {
		logging.DebugLog("[%s] HTTP request error: %v", requestID, err)
		apiErr := &errors.APIError{
			Kind:      errors.ErrorKindInternalServerError,
			Message:   "Failed to connect to Q Developer API",
			RequestID: requestID,
			Cause:     err,
		}
		h.writeAPIErrorWithRequestID(w, apiErr, requestID)
		return
	}
	
	// Log response status
	logging.DebugLog("[%s] API response status: %d %s", requestID, resp.StatusCode, resp.Status)
	
	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		logging.DebugLog("[%s] AWS API error: %d - %s", requestID, resp.StatusCode, string(bodyBytes))
		
		apiErr := errors.ClassifyError(resp.StatusCode, bodyBytes, nil)
		if apiErr.RequestID == "" {
			apiErr.RequestID = requestID
		}
		h.writeAPIErrorWithRequestID(w, apiErr, requestID)
		return
	}
	
	// Stream response (pure pass-through)
	h.handleDirectStreamingPassthrough(w, ctx, resp, reqCtx)
	
	duration := time.Since(reqCtx.StartTime)
	logging.DebugLog("[%s] Request completed in %v", requestID, duration)
	
	// Increment quota
	if userID, ok := r.Context().Value("userID").(string); ok && h.quotaTracker != nil {
		h.quotaTracker.IncrementQuota(userID, 1, 1)
	}
}

// handleDirectStreamingPassthrough handles streaming response (pure passthrough)
func (h *Handler) handleDirectStreamingPassthrough(w http.ResponseWriter, ctx context.Context, resp *http.Response, reqCtx *RequestContext) {
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
	
	logging.DebugLog("[%s] Starting direct streaming response (pure passthrough)", reqCtx.RequestID)
	
	// Pure passthrough streaming with debug capture
	buffer := make([]byte, 4096)
	totalBytes := 0
	var fullResponse []byte
	
	for {
		select {
		case <-ctx.Done():
			logging.DebugLog("[%s] Context cancelled during streaming", reqCtx.RequestID)
			return
		default:
			n, err := resp.Body.Read(buffer)
			if n > 0 {
				totalBytes += n
				chunk := buffer[:n]
				
				// Capture for debug logging
				if logging.IsDebugEnabled() {
					fullResponse = append(fullResponse, chunk...)
				}
				
				w.Write(chunk)
				flusher.Flush()
			}
			if err != nil {
				if err != io.EOF {
					logging.DebugLog("[%s] Streaming error: %v", reqCtx.RequestID, err)
				}
				logging.DebugLog("[%s] Streaming completed: %d bytes", reqCtx.RequestID, totalBytes)
				
				// Log response content if debug enabled
				if logging.IsDebugEnabled() && len(fullResponse) > 0 {
					logging.DebugLog("[%s] Response content: %s", reqCtx.RequestID, string(fullResponse))
					// Save to file
					debugPath := "logs/debug-response.txt"
					if err := os.WriteFile(debugPath, fullResponse, 0644); err == nil {
						logging.DebugLog("[%s] Response saved to %s", reqCtx.RequestID, debugPath)
					}
				}
				return
			}
		}
	}
}
