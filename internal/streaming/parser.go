package streaming

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/yourusername/kiro-gateway-go/internal/models"
)

// StreamEvent represents a parsed event from Kiro stream
type StreamEvent struct {
	Type string
	
	// Content events
	Content         string
	ThinkingContent string
	
	// Tool events
	ToolCall      *models.ToolCall
	ToolUses      []models.ToolUse // Q Developer tool uses (different from ToolCall)
	ToolUseEvent  *ToolUseEventData
	
	// Metadata
	ConversationID string
	UtteranceID    string
	
	// Usage events
	Usage                  *models.TokenUsage
	ContextUsagePercentage float64
	
	// Error
	Error error
}

// ToolUseEventData represents a streaming tool use event from Q Developer
type ToolUseEventData struct {
	Name      string
	ToolUseID string
	Input     string
	Stop      bool
}

// ParseKiroStream parses Kiro SSE stream into events
func ParseKiroStream(ctx context.Context, resp *http.Response, firstTokenTimeout time.Duration) (<-chan StreamEvent, error) {
	contentType := resp.Header.Get("Content-Type")
	
	// AWS Q Developer always returns binary event-stream format,
	// even when Content-Type says application/json
	// Try binary event-stream first for any response
	if strings.Contains(contentType, "application/vnd.amazon.eventstream") ||
		strings.Contains(contentType, "application/json") {
		return parseEventStreamBinary(ctx, resp)
	}
	
	// Handle SSE streaming response (text/event-stream)
	eventChan := make(chan StreamEvent, 64)
	
	go func() {
		defer close(eventChan)
		defer resp.Body.Close()
		
		scanner := bufio.NewScanner(resp.Body)
		firstToken := true
		timeoutTimer := time.NewTimer(firstTokenTimeout)
		defer timeoutTimer.Stop()
		
		// Current tool call being accumulated
		var currentToolCall *models.ToolCall
		
		for {
			// Check for timeout on first token
			if firstToken {
				select {
				case <-ctx.Done():
					eventChan <- StreamEvent{Type: "error", Error: ctx.Err()}
					return
				case <-timeoutTimer.C:
					eventChan <- StreamEvent{Type: "error", Error: fmt.Errorf("first token timeout")}
					return
				default:
				}
			}
			
			// Read next line
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					eventChan <- StreamEvent{Type: "error", Error: err}
				}
				return
			}
			
			line := scanner.Text()
			
			// Skip empty lines
			if line == "" {
				continue
			}
			
			// Parse SSE line
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			
			data := strings.TrimSpace(line[5:])
			if data == "" || data == "[DONE]" {
				continue
			}
			
			// First token received
			if firstToken {
				firstToken = false
				timeoutTimer.Stop()
			}
			
			// Try parsing as Q Developer ChatResponseStream format first
			var qEvent models.ChatResponseStream
			if err := json.Unmarshal([]byte(data), &qEvent); err == nil {
				// Handle Q Developer response format
				if qEvent.AssistantResponseEvent != nil {
					eventChan <- StreamEvent{
						Type:    "content",
						Content: qEvent.AssistantResponseEvent.Content,
					}
					continue
				}
				
				if qEvent.MessageMetadataEvent != nil {
					// Extract conversation ID for follow-up requests
					if qEvent.MessageMetadataEvent.ConversationID != "" {
						eventChan <- StreamEvent{
							Type:           "metadata",
							ConversationID: qEvent.MessageMetadataEvent.ConversationID,
							UtteranceID:    qEvent.MessageMetadataEvent.UtteranceID,
						}
					}
					
					// Final message indicator
					if qEvent.MessageMetadataEvent.FinalMessage {
						eventChan <- StreamEvent{
							Type: "done",
						}
					}
					continue
				}
				
				if qEvent.Error != nil {
					eventChan <- StreamEvent{
						Type:  "error",
						Error: fmt.Errorf("API error: %s (code: %s)", qEvent.Error.Message, qEvent.Error.Code),
					}
					return
				}
				
				// Check for tool uses in the response
				// Note: Q Developer sends tool uses as part of the conversation state
				// They appear in the full message, not in streaming chunks
				// We'll need to handle this differently - tool uses come at the end
				
				// Skip other event types for now (code references, follow-ups, etc.)
				continue
			}
			
			// Fallback to legacy Kiro format
			var event models.KiroEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				// Skip malformed events
				continue
			}
			
			// Process event based on type
			switch {
			case event.ContentBlockDelta != nil && event.ContentBlockDelta.Delta != nil:
				delta := event.ContentBlockDelta.Delta
				
				if delta.Type == "text_delta" && delta.Text != "" {
					eventChan <- StreamEvent{
						Type:    "content",
						Content: delta.Text,
					}
				} else if delta.Type == "thinking_delta" && delta.Thinking != "" {
					eventChan <- StreamEvent{
						Type:            "thinking",
						ThinkingContent: delta.Thinking,
					}
				} else if delta.Type == "input_json_delta" && delta.PartialJSON != "" {
					// Accumulate tool input
					if currentToolCall != nil {
						currentToolCall.Function.Arguments += delta.PartialJSON
					}
				}
				
			case event.ContentBlock != nil && event.ContentBlock.Type == "tool_use":
				// Start new tool call
				currentToolCall = &models.ToolCall{
					ID:   event.ContentBlock.ID,
					Type: "function",
					Function: models.FunctionCall{
						Name:      event.ContentBlock.Name,
						Arguments: "",
					},
				}
				
			case event.Type == "contentBlockStop":
				// Finish current tool call
				if currentToolCall != nil {
					eventChan <- StreamEvent{
						Type:     "tool_call",
						ToolCall: currentToolCall,
					}
					currentToolCall = nil
				}
				
			case event.Metadata != nil:
				// Extract usage information
				if event.Metadata.Usage != nil {
					eventChan <- StreamEvent{
						Type: "usage",
						Usage: &models.TokenUsage{
							PromptTokens:     event.Metadata.Usage.InputTokens,
							CompletionTokens: event.Metadata.Usage.OutputTokens,
							TotalTokens:      event.Metadata.Usage.InputTokens + event.Metadata.Usage.OutputTokens,
						},
					}
				}
				
				if event.Metadata.ContextUsage != nil {
					eventChan <- StreamEvent{
						Type:                   "context_usage",
						ContextUsagePercentage: event.Metadata.ContextUsage.Percentage,
					}
				}
			}
		}
	}()
	
	return eventChan, nil
}

// parseEventStreamBinary handles AWS event-stream binary format responses
// Uses incremental decoding pattern from AWS SDK Go v2 for proper context cancellation
func parseEventStreamBinary(ctx context.Context, resp *http.Response) (<-chan StreamEvent, error) {
	eventChan := make(chan StreamEvent, 64)
	
	go func() {
		defer close(eventChan)
		defer resp.Body.Close()
		defer func() { if r := recover(); r != nil { log.Printf("[ERROR] Binary parser panic: %v", r) } }()
		
		// Wrap response body with context-aware reader
		ctxReader := newContextReader(ctx, resp.Body)
		
		// Reusable payload buffer (10KB like AWS SDK)
		payloadBuf := make([]byte, 10*1024)
		
		for {
			// Reset buffer to zero length, keep capacity
			payloadBuf = payloadBuf[0:0]
			
			// Decode ONE message at a time (AWS SDK pattern)
			msg, err := parseMessage(ctxReader)
			if err != nil {
				if err == io.EOF {
					// Stream ended normally
					eventChan <- StreamEvent{Type: "done"}
					return
				}
				// Check if error is due to context cancellation
				if ctx.Err() != nil {
					eventChan <- StreamEvent{Type: "error", Error: ctx.Err()}
					return
				}
				eventChan <- StreamEvent{Type: "error", Error: fmt.Errorf("failed to decode message: %w", err)}
				return
			}
			
			// Process message immediately
			event := processEventStreamMessage(msg)
			if event != nil {
				// Send event with cancellation check
				select {
				case eventChan <- *event:
					// Event sent successfully
				case <-ctx.Done():
					eventChan <- StreamEvent{Type: "error", Error: ctx.Err()}
					return
				}
			}
		}
	}()
	
	return eventChan, nil
}

// contextReader wraps an io.Reader to make it context-aware
type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func newContextReader(ctx context.Context, r io.Reader) *contextReader {
	return &contextReader{ctx: ctx, r: r}
}

func (cr *contextReader) Read(p []byte) (n int, err error) {
	// Check if context is cancelled before reading
	select {
	case <-cr.ctx.Done():
		return 0, cr.ctx.Err()
	default:
	}
	
	// Perform the read
	return cr.r.Read(p)
}

// processEventStreamMessage processes a single event stream message and returns a StreamEvent
func processEventStreamMessage(msg *EventStreamMessage) *StreamEvent {
	// Check message type header
	messageType, ok := msg.Headers[":message-type"].(string)
	if !ok {
		return nil
	}
	
	// Only process "event" messages
	if messageType != "event" {
		return nil
	}
	
	// Check event type
	eventType, ok := msg.Headers[":event-type"].(string)
	if !ok {
		return nil
	}
	
	// Parse payload as JSON
	var payload map[string]interface{}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		// Skip non-JSON payloads
		return nil
	}
	
	// Extract content based on event type
	switch eventType {
	case "assistantResponseEvent":
		if contentStr, ok := payload["content"].(string); ok && contentStr != "" {
			return &StreamEvent{
				Type:    "content",
				Content: contentStr,
			}
		}
	case "toolUseEvent":
		name, _ := payload["name"].(string)
		toolUseID, _ := payload["toolUseId"].(string)
		input, _ := payload["input"].(string)
		stop, _ := payload["stop"].(bool)
		return &StreamEvent{
			Type: "tool_use_event",
			ToolUseEvent: &ToolUseEventData{
				Name:      name,
				ToolUseID: toolUseID,
				Input:     input,
				Stop:      stop,
			},
		}
	case "contextUsageEvent":
		// Ignore for now
		return nil
	case "reasoningContentEvent":
		if contentStr, ok := payload["content"].(string); ok && contentStr != "" {
			return &StreamEvent{
				Type:            "thinking",
				ThinkingContent: contentStr,
			}
		}
		return nil
	case "messageMetadataEvent":
		// Extract conversation ID for follow-up requests
		if convID, ok := payload["conversationId"].(string); ok && convID != "" {
			event := &StreamEvent{
				Type:           "metadata",
				ConversationID: convID,
			}
			if uttID, ok := payload["utteranceId"].(string); ok {
				event.UtteranceID = uttID
			}
			return event
		}
		
		// Check for tool uses in the message metadata
		// Tool uses appear in the full assistant response message
		if assistantMsg, ok := payload["assistantResponseMessage"].(map[string]interface{}); ok {
			if toolUsesRaw, ok := assistantMsg["toolUses"].([]interface{}); ok && len(toolUsesRaw) > 0 {
				// Parse tool uses
				toolUses := make([]models.ToolUse, 0, len(toolUsesRaw))
				for _, tuRaw := range toolUsesRaw {
					if tuMap, ok := tuRaw.(map[string]interface{}); ok {
						toolUse := models.ToolUse{}
						if id, ok := tuMap["toolUseId"].(string); ok {
							toolUse.ToolUseID = id
						}
						if name, ok := tuMap["name"].(string); ok {
							toolUse.Name = name
						}
						if input, ok := tuMap["input"]; ok {
							if inputBytes, err := json.Marshal(input); err == nil { toolUse.Input = json.RawMessage(inputBytes) }
						}
						toolUses = append(toolUses, toolUse)
					}
				}
				
				if len(toolUses) > 0 {
					return &StreamEvent{
						Type:     "tool_uses",
						ToolUses: toolUses,
					}
				}
			}
		}
		// Final message indicator - could extract metadata here
		return nil
	default:
		log.Printf("[WARN] Unhandled event type: %s", eventType)
		return nil
	}
	
	return nil
}

// CollectStreamContent collects all content from stream
func CollectStreamContent(eventChan <-chan StreamEvent) (string, string, []*models.ToolCall, *models.TokenUsage, error) {
	var content strings.Builder
	var thinking strings.Builder
	var toolCalls []*models.ToolCall
	var usage *models.TokenUsage
	var curToolID, curToolName, curToolInput string
	
	for event := range eventChan {
		switch event.Type {
		case "content":
			content.WriteString(event.Content)
		case "thinking":
			thinking.WriteString(event.ThinkingContent)
		case "tool_call":
			if event.ToolCall != nil {
				toolCalls = append(toolCalls, event.ToolCall)
			}
		case "tool_use_event":
			if event.ToolUseEvent != nil {
				tue := event.ToolUseEvent
				if tue.Stop {
										tcIdx := len(toolCalls)
					toolCalls = append(toolCalls, &models.ToolCall{
						Index: tcIdx,
						ID:   curToolID,
						Type: "function",
						Function: models.FunctionCall{
							Name:      curToolName,
							Arguments: curToolInput,
						},
					})
					curToolID, curToolName, curToolInput = "", "", ""
				} else {
					if tue.ToolUseID != "" { curToolID = tue.ToolUseID }
					if tue.Name != "" { curToolName = tue.Name }
					curToolInput += tue.Input
				}
			}
		case "tool_uses":
			// Q Developer tool uses are handled separately
		case "usage":
			usage = event.Usage
		case "error":
			return "", "", nil, nil, event.Error
		}
	}
	
	return content.String(), thinking.String(), toolCalls, usage, nil
}


// parseJSONResponse handles non-streaming JSON responses
// This is a fallback for responses that claim to be JSON but might actually be event-stream
func parseJSONResponse(ctx context.Context, resp *http.Response) (<-chan StreamEvent, error) {
	eventChan := make(chan StreamEvent, 64)
	
	go func() {
		defer close(eventChan)
		defer resp.Body.Close()
		
		// Read the entire response body for debugging
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			eventChan <- StreamEvent{Type: "error", Error: fmt.Errorf("failed to read response body: %w", err)}
			return
		}
		
		// Log first 200 bytes for debugging
		debugLen := 200
		if len(bodyBytes) < debugLen {
			debugLen = len(bodyBytes)
		}
		log.Printf("Response body (first %d bytes): %q", debugLen, string(bodyBytes[:debugLen]))
		log.Printf("Response body length: %d bytes", len(bodyBytes))
		
		// Try parsing as AWS event-stream binary format using AWS SDK-compliant incremental parser
		reader := bytes.NewReader(bodyBytes)
		ctxReader := newContextReader(ctx, reader)
		
		var content strings.Builder
		foundContent := false
		
		for {
			msg, err := parseMessage(ctxReader)
			if err != nil {
				if err == io.EOF {
					// Reached end of stream
					break
				}
				// Not event-stream format, try JSON
				log.Printf("Event-stream parsing error: %v", err)
				break
			}
			
			// Process message
			event := processEventStreamMessage(msg)
			if event != nil && event.Type == "content" && event.Content != "" {
				content.WriteString(event.Content)
				foundContent = true
			}
		}
		
		if foundContent {
			// Successfully parsed event-stream
			eventChan <- StreamEvent{
				Type:    "content",
				Content: content.String(),
			}
			return
		}
		
		// If event-stream parsing failed, try parsing as plain JSON
		var jsonResp map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &jsonResp); err == nil {
			// Check if it's an error response
			if errMsg, ok := jsonResp["error"].(map[string]interface{}); ok {
				eventChan <- StreamEvent{
					Type:  "error",
					Error: fmt.Errorf("API error: %v", errMsg),
				}
				return
			}
			
			// Try to extract content from various possible fields
			if content, ok := jsonResp["content"].(string); ok && content != "" {
				eventChan <- StreamEvent{
					Type:    "content",
					Content: content,
				}
				return
			}
			
			if message, ok := jsonResp["message"].(string); ok && message != "" {
				eventChan <- StreamEvent{
					Type:    "content",
					Content: message,
				}
				return
			}
		}
		
		// Return error - couldn't parse as event-stream or JSON
		eventChan <- StreamEvent{Type: "error", Error: fmt.Errorf("failed to parse response: not valid event-stream or JSON")}
	}()
	
	return eventChan, nil
}
