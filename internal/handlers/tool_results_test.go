package handlers

import (
	"context"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yourusername/kiro-gateway-go/internal/models"
)

func TestToolResultBuilder_Build(t *testing.T) {
	// Create original request
	originalReq := &models.ConversationStateRequest{
		ConversationState: &models.ConversationState{
			ConversationID: stringPtr("conv-123"),
			CurrentMessage: models.ChatMessage{
				UserInputMessage: &models.UserInputMessage{
					Content: "Test message",
					Origin:  "CLI",
					ModelID: "claude-sonnet-4-5",
				},
			},
			ChatTriggerType: "MANUAL",
		},
		Source:     "CLI",
		ProfileArn: "arn:aws:codewhisperer:us-east-1:123456789012:profile/test",
		Tools: []models.ToolSpecWrapper{
			{
				ToolSpecification: &models.ToolSpecification{
					Name:        "test-tool",
					Description: "A test tool",
					InputSchema: models.ToolInputSchema{
						JSON: `{"type":"object"}`,
					},
				},
			},
		},
	}
	
	// Create builder
	builder := NewToolResultBuilder("conv-456", "arn:aws:codewhisperer:us-east-1:123456789012:profile/test", originalReq)
	
	// Add a successful tool result
	result := &sdk.CallToolResult{
		Content: []sdk.Content{
			&sdk.TextContent{Text: "Tool executed successfully"},
		},
		IsError: false,
	}
	builder.AddToolResult("tool-use-123", result)
	
	// Add an error tool result
	builder.AddToolError("tool-use-456", nil)
	
	// Build the request
	req, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	
	// Verify conversation ID
	if req.ConversationState.ConversationID == nil {
		t.Error("ConversationID should not be nil")
	} else if *req.ConversationState.ConversationID != "conv-456" {
		t.Errorf("ConversationID = %v, want conv-456", *req.ConversationState.ConversationID)
	}
	
	// Verify profile ARN
	if req.ProfileArn != "arn:aws:codewhisperer:us-east-1:123456789012:profile/test" {
		t.Errorf("ProfileArn = %v, want arn:aws:codewhisperer:us-east-1:123456789012:profile/test", req.ProfileArn)
	}
	
	// Verify tool results
	if req.ConversationState.CurrentMessage.UserInputMessage == nil {
		t.Fatal("UserInputMessage should not be nil")
	}
	
	toolResults := req.ConversationState.CurrentMessage.UserInputMessage.ToolResults
	if len(toolResults) != 2 {
		t.Fatalf("Expected 2 tool results, got %d", len(toolResults))
	}
	
	// Verify first result (success)
	if toolResults[0].ToolUseID != "tool-use-123" {
		t.Errorf("ToolUseID = %v, want tool-use-123", toolResults[0].ToolUseID)
	}
	if toolResults[0].Status != "success" {
		t.Errorf("Status = %v, want success", toolResults[0].Status)
	}
	if len(toolResults[0].Content) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(toolResults[0].Content))
	}
	if toolResults[0].Content[0].Text != "Tool executed successfully" {
		t.Errorf("Content text = %v, want 'Tool executed successfully'", toolResults[0].Content[0].Text)
	}
	
	// Verify second result (error)
	if toolResults[1].ToolUseID != "tool-use-456" {
		t.Errorf("ToolUseID = %v, want tool-use-456", toolResults[1].ToolUseID)
	}
	if toolResults[1].Status != "error" {
		t.Errorf("Status = %v, want error", toolResults[1].Status)
	}
	
	// Verify origin is preserved
	if req.ConversationState.CurrentMessage.UserInputMessage.Origin != "CLI" {
		t.Errorf("Origin = %v, want CLI", req.ConversationState.CurrentMessage.UserInputMessage.Origin)
	}
	
	// Verify model ID is preserved
	if req.ConversationState.CurrentMessage.UserInputMessage.ModelID != "claude-sonnet-4-5" {
		t.Errorf("ModelID = %v, want claude-sonnet-4-5", req.ConversationState.CurrentMessage.UserInputMessage.ModelID)
	}
	
	// Verify tools are preserved
	if len(req.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(req.Tools))
	}
	
	// Verify source is preserved
	if req.Source != "CLI" {
		t.Errorf("Source = %v, want CLI", req.Source)
	}
}

func TestBuildFollowUpRequest(t *testing.T) {
	ctx := context.Background()
	
	// Create tool results
	toolResults := []models.ToolResult{
		{
			ToolUseID: "tool-123",
			Content: []models.ToolResultContent{
				{Text: "Result 1"},
			},
			Status: "success",
		},
		{
			ToolUseID: "tool-456",
			Content: []models.ToolResultContent{
				{Text: "Result 2"},
			},
			Status: "success",
		},
	}
	
	// Create original request
	originalReq := &models.ConversationStateRequest{
		ConversationState: &models.ConversationState{
			CurrentMessage: models.ChatMessage{
				UserInputMessage: &models.UserInputMessage{
					ModelID: "claude-sonnet-3-5",
				},
			},
		},
		Tools: []models.ToolSpecWrapper{
			{
				ToolSpecification: &models.ToolSpecification{
					Name: "test-tool",
				},
			},
		},
	}
	
	// Build follow-up request
	req, err := BuildFollowUpRequest(ctx, "conv-789", "arn:test", toolResults, originalReq)
	if err != nil {
		t.Fatalf("BuildFollowUpRequest() error = %v", err)
	}
	
	// Verify conversation ID
	if req.ConversationState.ConversationID == nil {
		t.Error("ConversationID should not be nil")
	} else if *req.ConversationState.ConversationID != "conv-789" {
		t.Errorf("ConversationID = %v, want conv-789", *req.ConversationState.ConversationID)
	}
	
	// Verify tool results
	if len(req.ConversationState.CurrentMessage.UserInputMessage.ToolResults) != 2 {
		t.Errorf("Expected 2 tool results, got %d", len(req.ConversationState.CurrentMessage.UserInputMessage.ToolResults))
	}
	
	// Verify model ID is preserved
	if req.ConversationState.CurrentMessage.UserInputMessage.ModelID != "claude-sonnet-3-5" {
		t.Errorf("ModelID = %v, want claude-sonnet-3-5", req.ConversationState.CurrentMessage.UserInputMessage.ModelID)
	}
	
	// Verify tools are preserved
	if len(req.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(req.Tools))
	}
}

func TestBuildFollowUpRequest_NoToolResults(t *testing.T) {
	ctx := context.Background()
	
	// Try to build with no tool results
	_, err := BuildFollowUpRequest(ctx, "conv-123", "arn:test", []models.ToolResult{}, nil)
	if err == nil {
		t.Error("Expected error when building with no tool results")
	}
}

func TestConvertToolResultsToQFormat(t *testing.T) {
	// Test successful result
	result := &sdk.CallToolResult{
		Content: []sdk.Content{
			&sdk.TextContent{Text: "Success"},
		},
		IsError: false,
	}
	
	qResult := ConvertToolResultsToQFormat("tool-123", result)
	
	if qResult.ToolUseID != "tool-123" {
		t.Errorf("ToolUseID = %v, want tool-123", qResult.ToolUseID)
	}
	if qResult.Status != "success" {
		t.Errorf("Status = %v, want success", qResult.Status)
	}
	if len(qResult.Content) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(qResult.Content))
	}
	if qResult.Content[0].Text != "Success" {
		t.Errorf("Content text = %v, want Success", qResult.Content[0].Text)
	}
	
	// Test error result
	errorResult := &sdk.CallToolResult{
		Content: []sdk.Content{
			&sdk.TextContent{Text: "Error occurred"},
		},
		IsError: true,
	}
	
	qErrorResult := ConvertToolResultsToQFormat("tool-456", errorResult)
	
	if qErrorResult.Status != "error" {
		t.Errorf("Status = %v, want error", qErrorResult.Status)
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
