package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yourusername/kiro-gateway-go/internal/logging"
	"github.com/yourusername/kiro-gateway-go/internal/models"
)

// ToolResultBuilder builds follow-up requests with tool results for Q Developer
type ToolResultBuilder struct {
	conversationID string
	profileArn     string
	toolResults    []models.ToolResult
	originalReq    *models.ConversationStateRequest
}

// NewToolResultBuilder creates a new tool result builder
func NewToolResultBuilder(conversationID, profileArn string, originalReq *models.ConversationStateRequest) *ToolResultBuilder {
	return &ToolResultBuilder{
		conversationID: conversationID,
		profileArn:     profileArn,
		toolResults:    make([]models.ToolResult, 0),
		originalReq:    originalReq,
	}
}

// AddToolResult adds a tool result to the builder
func (b *ToolResultBuilder) AddToolResult(toolUseID string, result *sdk.CallToolResult) {
	// Convert MCP content to Q Developer format
	content := convertMCPContentToQ(result.Content)
	
	// Determine status based on IsError flag
	status := "success"
	if result.IsError {
		status = "error"
	}
	
	toolResult := models.ToolResult{
		ToolUseID: toolUseID,
		Content:   content,
		Status:    status,
	}
	
	b.toolResults = append(b.toolResults, toolResult)
}

// AddToolError adds a tool error to the builder
func (b *ToolResultBuilder) AddToolError(toolUseID string, err error) {
	toolResult := models.ToolResult{
		ToolUseID: toolUseID,
		Content: []models.ToolResultContent{
			{
				Text: fmt.Sprintf("Tool execution failed: %v", err),
			},
		},
		Status: "error",
	}
	
	b.toolResults = append(b.toolResults, toolResult)
}

// Build creates a ConversationStateRequest with tool results
// This preserves the conversation context and includes all accumulated tool results
func (b *ToolResultBuilder) Build() (*models.ConversationStateRequest, error) {
	if len(b.toolResults) == 0 {
		return nil, fmt.Errorf("no tool results to send")
	}
	
	// Create user input message with tool results
	// According to Q Developer API spec, tool results go in the userInputMessage
	userInputMsg := &models.UserInputMessage{
		Content:     "Here are the tool execution results.", // Required content field
		ToolResults: b.toolResults,
		Origin:      "CLI", // Match original request origin
	}
	
	// Preserve model ID if it was in the original request
	if b.originalReq != nil && 
	   b.originalReq.ConversationState != nil && 
	   b.originalReq.ConversationState.CurrentMessage.UserInputMessage != nil {
		userInputMsg.ModelID = b.originalReq.ConversationState.CurrentMessage.UserInputMessage.ModelID
	}
	
	// Create conversation state with tool results
	// CRITICAL: Use the conversation ID from the previous response
	convID := &b.conversationID
	convState := &models.ConversationState{
		ConversationID: convID, // Use pointer to conversation ID
		CurrentMessage: models.ChatMessage{
			UserInputMessage: userInputMsg,
		},
		ChatTriggerType: "MANUAL",
		// Don't include history - Q Developer maintains it server-side
	}
	
	// Build the request
	request := &models.ConversationStateRequest{
		ConversationState: convState,
		Source:            "CLI", // Match original request
	}
	
	// Include profile ARN if present (for Identity Center mode)
	if b.profileArn != "" {
		request.ProfileArn = b.profileArn
	}
	
	// Preserve tools if they were in the original request
	if b.originalReq != nil && len(b.originalReq.Tools) > 0 {
		request.Tools = b.originalReq.Tools
	}
	
	return request, nil
}

// BuildFollowUpRequest is a convenience function that builds a follow-up request
// with tool results and returns it ready to send to Q Developer API
func BuildFollowUpRequest(
	ctx context.Context,
	conversationID string,
	profileArn string,
	toolResults []models.ToolResult,
	originalReq *models.ConversationStateRequest,
) (*models.ConversationStateRequest, error) {
	if len(toolResults) == 0 {
		return nil, fmt.Errorf("no tool results to send")
	}
	
	// Create user input message with tool results
	userInputMsg := &models.UserInputMessage{
		Content:     "Here are the tool execution results.",
		ToolResults: toolResults,
		Origin:      "CLI",
	}
	
	// Preserve model ID if present in original request
	if originalReq != nil && 
	   originalReq.ConversationState != nil && 
	   originalReq.ConversationState.CurrentMessage.UserInputMessage != nil {
		userInputMsg.ModelID = originalReq.ConversationState.CurrentMessage.UserInputMessage.ModelID
	}
	
	// Create conversation state
	convID := &conversationID
	convState := &models.ConversationState{
		ConversationID: convID,
		CurrentMessage: models.ChatMessage{
			UserInputMessage: userInputMsg,
		},
		ChatTriggerType: "MANUAL",
	}
	
	// Build request
	request := &models.ConversationStateRequest{
		ConversationState: convState,
		Source:            "CLI",
	}
	
	// Include profile ARN if present
	if profileArn != "" {
		request.ProfileArn = profileArn
	}
	
	// Preserve tools from original request
	if originalReq != nil && len(originalReq.Tools) > 0 {
		request.Tools = originalReq.Tools
	}
	
	return request, nil
}

// ConvertToolResultsToQFormat converts a slice of MCP tool results to Q Developer format
// This is useful when you have multiple tool results from different tool executions
func ConvertToolResultsToQFormat(toolUseID string, result *sdk.CallToolResult) models.ToolResult {
	content := convertMCPContentToQ(result.Content)
	
	status := "success"
	if result.IsError {
		status = "error"
	}
	
	return models.ToolResult{
		ToolUseID: toolUseID,
		Content:   content,
		Status:    status,
	}
}

// LogToolResultRequest logs the tool result request for debugging
func LogToolResultRequest(requestID string, request *models.ConversationStateRequest) {
	mcpLogger := logging.NewMCPLogger()
	
	// Calculate result size
	resultSize := 0
	if request.ConversationState != nil && 
	   request.ConversationState.CurrentMessage.UserInputMessage != nil &&
	   len(request.ConversationState.CurrentMessage.UserInputMessage.ToolResults) > 0 {
		
		for _, result := range request.ConversationState.CurrentMessage.UserInputMessage.ToolResults {
			resultJSON := logging.MarshalJSON(result)
			resultSize += len(resultJSON)
			
			// Log each tool result being sent
			mcpLogger.LogToolResult(logging.ToolResultEvent{
				ToolUseID:      result.ToolUseID,
				Status:         "sending",
				ResultSize:     len(resultJSON),
				ConversationID: getConversationID(request),
			})
		}
	}
	
	// Log the full request for debugging
	if reqBytes, err := json.MarshalIndent(request, "", "  "); err == nil {
		log.Printf("[%s] Tool result follow-up request (%s):\n%s", 
			requestID, logging.FormatSize(len(reqBytes)), string(reqBytes))
	} else {
		log.Printf("[%s] Failed to marshal tool result request: %v", requestID, err)
	}
}

// LogToolResultDelivery logs the result of tool result delivery
func LogToolResultDelivery(requestID string, toolUseID string, conversationID string, 
	resultSize int, duration time.Duration, httpStatus int, err error) {
	
	mcpLogger := logging.NewMCPLogger()
	
	event := logging.ToolResultEvent{
		ToolUseID:      toolUseID,
		ResultSize:     resultSize,
		ConversationID: conversationID,
		Duration:       logging.FormatDuration(duration),
		HTTPStatus:     httpStatus,
	}
	
	if err != nil {
		event.Status = "failed"
		event.ErrorMessage = err.Error()
	} else {
		event.Status = "delivered"
	}
	
	mcpLogger.LogToolResult(event)
}

// getConversationID extracts conversation ID from request
func getConversationID(request *models.ConversationStateRequest) string {
	if request.ConversationState != nil && request.ConversationState.ConversationID != nil {
		return *request.ConversationState.ConversationID
	}
	return ""
}
