package logging

import (
	"bytes"
	"encoding/json"
	"log"
	"strings"
	"testing"
)

func TestMCPLoggerServerConnection(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	mcpLogger := NewMCPLogger()
	mcpLogger.logger.logger = log.New(&buf, "", 0)
	
	// Log server connection
	mcpLogger.LogServerConnection(ServerConnectionEvent{
		ServerName: "test-server",
		Status:     "connected",
		Command:    "npx",
		Args:       []string{"-y", "@test/server"},
		ToolCount:  5,
		Tools:      []string{"tool1", "tool2", "tool3", "tool4", "tool5"},
		Duration:   "1.5s",
	})
	
	// Parse JSON output
	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}
	
	// Verify fields
	if entry.Level != "INFO" {
		t.Errorf("Expected level INFO for connected status, got %s", entry.Level)
	}
	if entry.Fields["server_name"] != "test-server" {
		t.Errorf("Expected server_name='test-server', got %v", entry.Fields["server_name"])
	}
	if entry.Fields["status"] != "connected" {
		t.Errorf("Expected status='connected', got %v", entry.Fields["status"])
	}
	if entry.Fields["tool_count"] != float64(5) {
		t.Errorf("Expected tool_count=5, got %v", entry.Fields["tool_count"])
	}
}

func TestMCPLoggerToolExecution(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	mcpLogger := NewMCPLogger()
	mcpLogger.logger.logger = log.New(&buf, "", 0)
	
	// Log tool execution
	mcpLogger.LogToolExecution(ToolExecutionEvent{
		ServerName: "test-server",
		ToolName:   "test-tool",
		ToolUseID:  "tool-123",
		Arguments: map[string]interface{}{
			"param1": "value1",
			"param2": 42,
		},
		Status:     "completed",
		Duration:   "500ms",
		ResultSize: 1024,
	})
	
	// Parse JSON output
	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}
	
	// Verify fields
	if entry.Level != "INFO" {
		t.Errorf("Expected level INFO for completed status, got %s", entry.Level)
	}
	if entry.Fields["tool_name"] != "test-tool" {
		t.Errorf("Expected tool_name='test-tool', got %v", entry.Fields["tool_name"])
	}
	if entry.Fields["duration"] != "500ms" {
		t.Errorf("Expected duration='500ms', got %v", entry.Fields["duration"])
	}
}

func TestMCPLoggerToolExecutionError(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	mcpLogger := NewMCPLogger()
	mcpLogger.logger.logger = log.New(&buf, "", 0)
	
	// Log tool execution failure
	mcpLogger.LogToolExecution(ToolExecutionEvent{
		ServerName:   "test-server",
		ToolName:     "test-tool",
		Status:       "failed",
		ErrorMessage: "connection timeout",
	})
	
	// Parse JSON output
	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}
	
	// Verify error logging
	if entry.Level != "ERROR" {
		t.Errorf("Expected level ERROR for failed status, got %s", entry.Level)
	}
	if entry.Fields["error_message"] != "connection timeout" {
		t.Errorf("Expected error_message='connection timeout', got %v", entry.Fields["error_message"])
	}
}

func TestMCPLoggerToolResult(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	mcpLogger := NewMCPLogger()
	mcpLogger.logger.logger = log.New(&buf, "", 0)
	
	// Log tool result delivery
	mcpLogger.LogToolResult(ToolResultEvent{
		ToolUseID:      "tool-123",
		Status:         "delivered",
		ResultSize:     2048,
		ConversationID: "conv-456",
		Duration:       "200ms",
		HTTPStatus:     200,
	})
	
	// Parse JSON output
	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}
	
	// Verify fields
	if entry.Level != "INFO" {
		t.Errorf("Expected level INFO for delivered status, got %s", entry.Level)
	}
	if entry.Fields["tool_use_id"] != "tool-123" {
		t.Errorf("Expected tool_use_id='tool-123', got %v", entry.Fields["tool_use_id"])
	}
	if entry.Fields["http_status"] != float64(200) {
		t.Errorf("Expected http_status=200, got %v", entry.Fields["http_status"])
	}
}

func TestMCPLoggerToolDiscovery(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	mcpLogger := NewMCPLogger()
	mcpLogger.logger.logger = log.New(&buf, "", 0)
	
	// Log tool discovery
	mcpLogger.LogToolDiscovery(ToolDiscoveryEvent{
		ServerName: "test-server",
		Status:     "completed",
		ToolCount:  10,
		Tools:      []string{"tool1", "tool2", "tool3"},
		PageCount:  2,
		Duration:   "1s",
	})
	
	// Parse JSON output
	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}
	
	// Verify fields
	if entry.Level != "INFO" {
		t.Errorf("Expected level INFO for completed status, got %s", entry.Level)
	}
	if entry.Fields["tool_count"] != float64(10) {
		t.Errorf("Expected tool_count=10, got %v", entry.Fields["tool_count"])
	}
	if entry.Fields["page_count"] != float64(2) {
		t.Errorf("Expected page_count=2, got %v", entry.Fields["page_count"])
	}
}

func TestMCPLoggerReconnection(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	mcpLogger := NewMCPLogger()
	mcpLogger.logger.logger = log.New(&buf, "", 0)
	
	// Log reconnection attempt
	mcpLogger.LogReconnection(ReconnectionEvent{
		ServerName:    "test-server",
		Status:        "succeeded",
		Attempt:       3,
		BackoffTime:   "4s",
		NextRetryTime: "2024-01-01T12:00:00Z",
	})
	
	// Parse JSON output
	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}
	
	// Verify fields
	if entry.Level != "INFO" {
		t.Errorf("Expected level INFO for succeeded status, got %s", entry.Level)
	}
	if entry.Fields["attempt"] != float64(3) {
		t.Errorf("Expected attempt=3, got %v", entry.Fields["attempt"])
	}
}

func TestMCPLoggerCircuitBreaker(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	mcpLogger := NewMCPLogger()
	mcpLogger.logger.logger = log.New(&buf, "", 0)
	
	// Log circuit breaker opened
	mcpLogger.LogCircuitBreaker(CircuitBreakerEvent{
		ServerName:    "test-server",
		Status:        "opened",
		FailureCount:  5,
		NextRetryTime: "2024-01-01T12:00:00Z",
	})
	
	// Parse JSON output
	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}
	
	// Verify fields
	if entry.Level != "WARN" {
		t.Errorf("Expected level WARN for opened status, got %s", entry.Level)
	}
	if entry.Fields["failure_count"] != float64(5) {
		t.Errorf("Expected failure_count=5, got %v", entry.Fields["failure_count"])
	}
}

func TestSanitizeArguments(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "sanitize password",
			input: map[string]interface{}{
				"username": "user",
				"password": "secret123",
			},
			expected: map[string]interface{}{
				"username": "user",
				"password": "[REDACTED]",
			},
		},
		{
			name: "sanitize token",
			input: map[string]interface{}{
				"api_key": "abc123",
				"data":    "public",
			},
			expected: map[string]interface{}{
				"api_key": "[REDACTED]",
				"data":    "public",
			},
		},
		{
			name: "nested sanitization",
			input: map[string]interface{}{
				"config": map[string]interface{}{
					"secret": "hidden",
					"name":   "visible",
				},
			},
			expected: map[string]interface{}{
				"config": map[string]interface{}{
					"secret": "[REDACTED]",
					"name":   "visible",
				},
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeArguments(tt.input)
			
			// Check each expected field
			for key, expectedValue := range tt.expected {
				if resultValue, ok := result[key]; !ok {
					t.Errorf("Expected key '%s' not found in result", key)
				} else {
					// Handle nested maps
					if expectedMap, ok := expectedValue.(map[string]interface{}); ok {
						resultMap, ok := resultValue.(map[string]interface{})
						if !ok {
							t.Errorf("Expected nested map for key '%s'", key)
							continue
						}
						for nestedKey, nestedExpected := range expectedMap {
							if resultMap[nestedKey] != nestedExpected {
								t.Errorf("For key '%s.%s': expected %v, got %v", 
									key, nestedKey, nestedExpected, resultMap[nestedKey])
							}
						}
					} else if resultValue != expectedValue {
						t.Errorf("For key '%s': expected %v, got %v", key, expectedValue, resultValue)
					}
				}
			}
		})
	}
}

func TestMCPLoggerMessageFormat(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	mcpLogger := NewMCPLogger()
	mcpLogger.logger.logger = log.New(&buf, "", 0)
	
	// Log various events and check message format
	tests := []struct {
		name     string
		logFunc  func()
		contains string
	}{
		{
			name: "server connection message",
			logFunc: func() {
				mcpLogger.LogServerConnection(ServerConnectionEvent{
					ServerName: "test",
					Status:     "connected",
				})
			},
			contains: "MCP server test: connected",
		},
		{
			name: "tool execution message",
			logFunc: func() {
				buf.Reset()
				mcpLogger.LogToolExecution(ToolExecutionEvent{
					ServerName: "test",
					ToolName:   "my-tool",
					Status:     "completed",
				})
			},
			contains: "Tool execution test/my-tool: completed",
		},
		{
			name: "tool result message",
			logFunc: func() {
				buf.Reset()
				mcpLogger.LogToolResult(ToolResultEvent{
					ToolUseID: "tool-123",
					Status:    "delivered",
				})
			},
			contains: "Tool result tool-123: delivered",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc()
			
			var entry LogEntry
			if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
				t.Fatalf("Failed to parse log output: %v", err)
			}
			
			if !strings.Contains(entry.Message, tt.contains) {
				t.Errorf("Expected message to contain '%s', got '%s'", tt.contains, entry.Message)
			}
		})
	}
}

func TestMCPLoggerDebugMode(t *testing.T) {
	tests := []struct {
		name          string
		logLevel      LogLevel
		shouldLog     bool
		description   string
	}{
		{
			name:        "debug mode enabled",
			logLevel:    LevelDebug,
			shouldLog:   true,
			description: "Protocol messages should be logged when LOG_LEVEL=DEBUG",
		},
		{
			name:        "info mode",
			logLevel:    LevelInfo,
			shouldLog:   false,
			description: "Protocol messages should NOT be logged when LOG_LEVEL=INFO",
		},
		{
			name:        "warn mode",
			logLevel:    LevelWarn,
			shouldLog:   false,
			description: "Protocol messages should NOT be logged when LOG_LEVEL=WARN",
		},
		{
			name:        "error mode",
			logLevel:    LevelError,
			shouldLog:   false,
			description: "Protocol messages should NOT be logged when LOG_LEVEL=ERROR",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture log output
			var buf bytes.Buffer
			mcpLogger := NewMCPLogger()
			mcpLogger.logger.SetLevel(tt.logLevel)
			mcpLogger.logger.logger = log.New(&buf, "", 0)
			
			// Log protocol message
			mcpLogger.LogProtocolMessage(ProtocolMessageEvent{
				ServerName:  "test-server",
				Direction:   "request",
				MessageType: "tools/list",
				Payload: map[string]interface{}{
					"cursor": "",
				},
			})
			
			// Check if message was logged
			output := buf.String()
			hasOutput := len(output) > 0
			
			if hasOutput != tt.shouldLog {
				if tt.shouldLog {
					t.Errorf("%s: Expected protocol message to be logged, but got no output", tt.description)
				} else {
					t.Errorf("%s: Expected no protocol message, but got output: %s", tt.description, output)
				}
			}
			
			// If logged, verify the content
			if hasOutput && tt.shouldLog {
				var entry LogEntry
				if err := json.Unmarshal([]byte(output), &entry); err != nil {
					t.Fatalf("Failed to parse log output: %v", err)
				}
				
				if entry.Level != "DEBUG" {
					t.Errorf("Expected level DEBUG for protocol messages, got %s", entry.Level)
				}
				
				if entry.Fields["server_name"] != "test-server" {
					t.Errorf("Expected server_name='test-server', got %v", entry.Fields["server_name"])
				}
				
				if entry.Fields["direction"] != "request" {
					t.Errorf("Expected direction='request', got %v", entry.Fields["direction"])
				}
				
				if entry.Fields["message_type"] != "tools/list" {
					t.Errorf("Expected message_type='tools/list', got %v", entry.Fields["message_type"])
				}
			}
		})
	}
}

func TestMCPLoggerProtocolRequest(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	mcpLogger := NewMCPLogger()
	mcpLogger.logger.SetLevel(LevelDebug)
	mcpLogger.logger.logger = log.New(&buf, "", 0)
	
	// Log protocol request
	mcpLogger.LogProtocolRequest("test-server", "tools/call", map[string]interface{}{
		"name": "test-tool",
		"arguments": map[string]interface{}{
			"param1": "value1",
			"param2": 42,
		},
	})
	
	// Parse JSON output
	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}
	
	// Verify fields
	if entry.Level != "DEBUG" {
		t.Errorf("Expected level DEBUG, got %s", entry.Level)
	}
	
	if entry.Fields["direction"] != "request" {
		t.Errorf("Expected direction='request', got %v", entry.Fields["direction"])
	}
	
	if entry.Fields["message_type"] != "tools/call" {
		t.Errorf("Expected message_type='tools/call', got %v", entry.Fields["message_type"])
	}
	
	// Verify payload is present
	if entry.Fields["payload"] == nil {
		t.Error("Expected payload field to be present")
	}
	
	// Verify raw_json is present
	if entry.Fields["raw_json"] == nil {
		t.Error("Expected raw_json field to be present")
	}
}

func TestMCPLoggerProtocolResponse(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	mcpLogger := NewMCPLogger()
	mcpLogger.logger.SetLevel(LevelDebug)
	mcpLogger.logger.logger = log.New(&buf, "", 0)
	
	// Log protocol response
	mcpLogger.LogProtocolResponse("test-server", "tools/list", map[string]interface{}{
		"tools": []map[string]interface{}{
			{
				"name":        "tool1",
				"description": "Test tool 1",
			},
			{
				"name":        "tool2",
				"description": "Test tool 2",
			},
		},
		"nextCursor": "",
	})
	
	// Parse JSON output
	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}
	
	// Verify fields
	if entry.Level != "DEBUG" {
		t.Errorf("Expected level DEBUG, got %s", entry.Level)
	}
	
	if entry.Fields["direction"] != "response" {
		t.Errorf("Expected direction='response', got %v", entry.Fields["direction"])
	}
	
	if entry.Fields["message_type"] != "tools/list" {
		t.Errorf("Expected message_type='tools/list', got %v", entry.Fields["message_type"])
	}
	
	// Verify payload is present
	if entry.Fields["payload"] == nil {
		t.Error("Expected payload field to be present")
	}
}

func TestMCPLoggerIsDebugEnabled(t *testing.T) {
	tests := []struct {
		name     string
		logLevel LogLevel
		expected bool
	}{
		{
			name:     "debug level",
			logLevel: LevelDebug,
			expected: true,
		},
		{
			name:     "info level",
			logLevel: LevelInfo,
			expected: false,
		},
		{
			name:     "warn level",
			logLevel: LevelWarn,
			expected: false,
		},
		{
			name:     "error level",
			logLevel: LevelError,
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpLogger := NewMCPLogger()
			mcpLogger.logger.SetLevel(tt.logLevel)
			
			result := mcpLogger.IsDebugEnabled()
			if result != tt.expected {
				t.Errorf("Expected IsDebugEnabled()=%v for level %s, got %v", 
					tt.expected, tt.logLevel.String(), result)
			}
		})
	}
}
