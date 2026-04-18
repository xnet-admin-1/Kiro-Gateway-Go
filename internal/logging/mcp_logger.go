package logging

import (
	"encoding/json"
	"fmt"
	"time"
)

// MCPLogger provides specialized logging for MCP operations
type MCPLogger struct {
	logger *Logger
}

// NewMCPLogger creates a new MCP logger
func NewMCPLogger() *MCPLogger {
	return &MCPLogger{
		logger: NewLogger("mcp"),
	}
}

// ServerConnectionEvent logs MCP server connection events
type ServerConnectionEvent struct {
	ServerName    string   `json:"server_name"`
	Status        string   `json:"status"` // "connecting", "connected", "failed", "disconnected"
	Command       string   `json:"command,omitempty"`
	Args          []string `json:"args,omitempty"`
	ToolCount     int      `json:"tool_count,omitempty"`
	Tools         []string `json:"tools,omitempty"`
	FailureCount  int      `json:"failure_count,omitempty"`
	NextRetryTime string   `json:"next_retry_time,omitempty"`
	Duration      string   `json:"duration,omitempty"`
}

// LogServerConnection logs a server connection event
func (ml *MCPLogger) LogServerConnection(event ServerConnectionEvent) {
	fields := map[string]interface{}{
		"server_name": event.ServerName,
		"status":      event.Status,
	}
	
	if event.Command != "" {
		fields["command"] = event.Command
	}
	if len(event.Args) > 0 {
		fields["args"] = event.Args
	}
	if event.ToolCount > 0 {
		fields["tool_count"] = event.ToolCount
	}
	if len(event.Tools) > 0 {
		fields["tools"] = event.Tools
	}
	if event.FailureCount > 0 {
		fields["failure_count"] = event.FailureCount
	}
	if event.NextRetryTime != "" {
		fields["next_retry_time"] = event.NextRetryTime
	}
	if event.Duration != "" {
		fields["duration"] = event.Duration
	}
	
	message := fmt.Sprintf("MCP server %s: %s", event.ServerName, event.Status)
	
	switch event.Status {
	case "connected":
		ml.logger.Info(message, fields)
	case "failed", "disconnected":
		ml.logger.Warn(message, fields)
	default:
		ml.logger.Debug(message, fields)
	}
}

// ToolExecutionEvent logs tool execution events
type ToolExecutionEvent struct {
	ServerName   string                 `json:"server_name"`
	ToolName     string                 `json:"tool_name"`
	ToolUseID    string                 `json:"tool_use_id,omitempty"`
	Arguments    map[string]interface{} `json:"arguments,omitempty"`
	Status       string                 `json:"status"` // "started", "completed", "failed", "timeout"
	Duration     string                 `json:"duration,omitempty"`
	ResultSize   int                    `json:"result_size,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
}

// LogToolExecution logs a tool execution event
func (ml *MCPLogger) LogToolExecution(event ToolExecutionEvent) {
	fields := map[string]interface{}{
		"server_name": event.ServerName,
		"tool_name":   event.ToolName,
		"status":      event.Status,
	}
	
	if event.ToolUseID != "" {
		fields["tool_use_id"] = event.ToolUseID
	}
	if event.Arguments != nil {
		// Sanitize arguments - don't log sensitive data
		sanitized := sanitizeArguments(event.Arguments)
		fields["arguments"] = sanitized
	}
	if event.Duration != "" {
		fields["duration"] = event.Duration
	}
	if event.ResultSize > 0 {
		fields["result_size"] = event.ResultSize
	}
	if event.ErrorMessage != "" {
		fields["error_message"] = event.ErrorMessage
	}
	
	message := fmt.Sprintf("Tool execution %s/%s: %s", event.ServerName, event.ToolName, event.Status)
	
	switch event.Status {
	case "completed":
		ml.logger.Info(message, fields)
	case "failed", "timeout":
		ml.logger.Error(message, fmt.Errorf(event.ErrorMessage), fields)
	default:
		ml.logger.Debug(message, fields)
	}
}

// ToolResultEvent logs tool result delivery events
type ToolResultEvent struct {
	ToolUseID        string `json:"tool_use_id"`
	Status           string `json:"status"` // "sending", "delivered", "failed"
	ResultSize       int    `json:"result_size,omitempty"`
	ConversationID   string `json:"conversation_id,omitempty"`
	Duration         string `json:"duration,omitempty"`
	ErrorMessage     string `json:"error_message,omitempty"`
	HTTPStatus       int    `json:"http_status,omitempty"`
}

// LogToolResult logs a tool result delivery event
func (ml *MCPLogger) LogToolResult(event ToolResultEvent) {
	fields := map[string]interface{}{
		"tool_use_id": event.ToolUseID,
		"status":      event.Status,
	}
	
	if event.ResultSize > 0 {
		fields["result_size"] = event.ResultSize
	}
	if event.ConversationID != "" {
		fields["conversation_id"] = event.ConversationID
	}
	if event.Duration != "" {
		fields["duration"] = event.Duration
	}
	if event.ErrorMessage != "" {
		fields["error_message"] = event.ErrorMessage
	}
	if event.HTTPStatus > 0 {
		fields["http_status"] = event.HTTPStatus
	}
	
	message := fmt.Sprintf("Tool result %s: %s", event.ToolUseID, event.Status)
	
	switch event.Status {
	case "delivered":
		ml.logger.Info(message, fields)
	case "failed":
		ml.logger.Error(message, fmt.Errorf(event.ErrorMessage), fields)
	default:
		ml.logger.Debug(message, fields)
	}
}

// ToolDiscoveryEvent logs tool discovery events
type ToolDiscoveryEvent struct {
	ServerName   string   `json:"server_name"`
	Status       string   `json:"status"` // "started", "completed", "failed"
	ToolCount    int      `json:"tool_count,omitempty"`
	Tools        []string `json:"tools,omitempty"`
	PageCount    int      `json:"page_count,omitempty"`
	Duration     string   `json:"duration,omitempty"`
	ErrorMessage string   `json:"error_message,omitempty"`
}

// LogToolDiscovery logs a tool discovery event
func (ml *MCPLogger) LogToolDiscovery(event ToolDiscoveryEvent) {
	fields := map[string]interface{}{
		"server_name": event.ServerName,
		"status":      event.Status,
	}
	
	if event.ToolCount > 0 {
		fields["tool_count"] = event.ToolCount
	}
	if len(event.Tools) > 0 {
		fields["tools"] = event.Tools
	}
	if event.PageCount > 0 {
		fields["page_count"] = event.PageCount
	}
	if event.Duration != "" {
		fields["duration"] = event.Duration
	}
	if event.ErrorMessage != "" {
		fields["error_message"] = event.ErrorMessage
	}
	
	message := fmt.Sprintf("Tool discovery for %s: %s", event.ServerName, event.Status)
	
	switch event.Status {
	case "completed":
		ml.logger.Info(message, fields)
	case "failed":
		ml.logger.Error(message, fmt.Errorf(event.ErrorMessage), fields)
	default:
		ml.logger.Debug(message, fields)
	}
}

// ReconnectionEvent logs reconnection events
type ReconnectionEvent struct {
	ServerName    string `json:"server_name"`
	Status        string `json:"status"` // "attempting", "succeeded", "failed"
	Attempt       int    `json:"attempt"`
	BackoffTime   string `json:"backoff_time,omitempty"`
	NextRetryTime string `json:"next_retry_time,omitempty"`
	ErrorMessage  string `json:"error_message,omitempty"`
}

// LogReconnection logs a reconnection event
func (ml *MCPLogger) LogReconnection(event ReconnectionEvent) {
	fields := map[string]interface{}{
		"server_name": event.ServerName,
		"status":      event.Status,
		"attempt":     event.Attempt,
	}
	
	if event.BackoffTime != "" {
		fields["backoff_time"] = event.BackoffTime
	}
	if event.NextRetryTime != "" {
		fields["next_retry_time"] = event.NextRetryTime
	}
	if event.ErrorMessage != "" {
		fields["error_message"] = event.ErrorMessage
	}
	
	message := fmt.Sprintf("Reconnection attempt %d for %s: %s", event.Attempt, event.ServerName, event.Status)
	
	switch event.Status {
	case "succeeded":
		ml.logger.Info(message, fields)
	case "failed":
		ml.logger.Warn(message, fields)
	default:
		ml.logger.Debug(message, fields)
	}
}

// CircuitBreakerEvent logs circuit breaker events
type CircuitBreakerEvent struct {
	ServerName    string `json:"server_name"`
	Status        string `json:"status"` // "opened", "closed", "half_open"
	FailureCount  int    `json:"failure_count"`
	NextRetryTime string `json:"next_retry_time,omitempty"`
}

// LogCircuitBreaker logs a circuit breaker event
func (ml *MCPLogger) LogCircuitBreaker(event CircuitBreakerEvent) {
	fields := map[string]interface{}{
		"server_name":   event.ServerName,
		"status":        event.Status,
		"failure_count": event.FailureCount,
	}
	
	if event.NextRetryTime != "" {
		fields["next_retry_time"] = event.NextRetryTime
	}
	
	message := fmt.Sprintf("Circuit breaker for %s: %s (failures: %d)", 
		event.ServerName, event.Status, event.FailureCount)
	
	switch event.Status {
	case "opened":
		ml.logger.Warn(message, fields)
	case "closed":
		ml.logger.Info(message, fields)
	default:
		ml.logger.Debug(message, fields)
	}
}

// sanitizeArguments removes sensitive data from arguments
func sanitizeArguments(args map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})
	
	sensitiveKeys := map[string]bool{
		"password":     true,
		"token":        true,
		"secret":       true,
		"api_key":      true,
		"apikey":       true,
		"access_token": true,
		"private_key":  true,
		"credentials":  true,
	}
	
	for k, v := range args {
		// Check if key is sensitive
		lowerKey := fmt.Sprintf("%v", k)
		if sensitiveKeys[lowerKey] {
			sanitized[k] = "[REDACTED]"
			continue
		}
		
		// Recursively sanitize nested maps
		if nestedMap, ok := v.(map[string]interface{}); ok {
			sanitized[k] = sanitizeArguments(nestedMap)
			continue
		}
		
		// Keep non-sensitive values
		sanitized[k] = v
	}
	
	return sanitized
}

// FormatDuration formats a duration for logging
func FormatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%.2fµs", float64(d.Microseconds()))
	}
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Milliseconds()))
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// FormatSize formats a byte size for logging
func FormatSize(bytes int) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// MarshalJSON marshals an object to JSON string for logging
func MarshalJSON(v interface{}) string {
	bytes, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("[JSON marshal error: %v]", err)
	}
	return string(bytes)
}

// ProtocolMessageEvent logs MCP protocol messages in debug mode
type ProtocolMessageEvent struct {
	ServerName string                 `json:"server_name"`
	Direction  string                 `json:"direction"` // "request", "response"
	MessageType string                `json:"message_type"` // "initialize", "tools/list", "tools/call", etc.
	Payload    map[string]interface{} `json:"payload,omitempty"`
	RawJSON    string                 `json:"raw_json,omitempty"`
}

// LogProtocolMessage logs a full MCP protocol message (only in debug mode)
func (ml *MCPLogger) LogProtocolMessage(event ProtocolMessageEvent) {
	// Only log protocol messages in debug mode
	if !ml.IsDebugEnabled() {
		return
	}
	
	fields := map[string]interface{}{
		"server_name":  event.ServerName,
		"direction":    event.Direction,
		"message_type": event.MessageType,
	}
	
	if event.Payload != nil {
		fields["payload"] = event.Payload
	}
	
	if event.RawJSON != "" {
		fields["raw_json"] = event.RawJSON
	}
	
	message := fmt.Sprintf("MCP protocol %s: %s/%s", event.Direction, event.ServerName, event.MessageType)
	ml.logger.Debug(message, fields)
}

// IsDebugEnabled checks if debug logging is enabled
func (ml *MCPLogger) IsDebugEnabled() bool {
	return ml.logger.IsDebugEnabled()
}

// LogProtocolRequest logs an MCP request message in debug mode
func (ml *MCPLogger) LogProtocolRequest(serverName, messageType string, payload interface{}) {
	if !ml.IsDebugEnabled() {
		return
	}
	
	// Convert payload to map for structured logging
	var payloadMap map[string]interface{}
	if payload != nil {
		jsonBytes, err := json.Marshal(payload)
		if err == nil {
			json.Unmarshal(jsonBytes, &payloadMap)
		}
	}
	
	ml.LogProtocolMessage(ProtocolMessageEvent{
		ServerName:  serverName,
		Direction:   "request",
		MessageType: messageType,
		Payload:     payloadMap,
		RawJSON:     MarshalJSON(payload),
	})
}

// LogProtocolResponse logs an MCP response message in debug mode
func (ml *MCPLogger) LogProtocolResponse(serverName, messageType string, payload interface{}) {
	if !ml.IsDebugEnabled() {
		return
	}
	
	// Convert payload to map for structured logging
	var payloadMap map[string]interface{}
	if payload != nil {
		jsonBytes, err := json.Marshal(payload)
		if err == nil {
			json.Unmarshal(jsonBytes, &payloadMap)
		}
	}
	
	ml.LogProtocolMessage(ProtocolMessageEvent{
		ServerName:  serverName,
		Direction:   "response",
		MessageType: messageType,
		Payload:     payloadMap,
		RawJSON:     MarshalJSON(payload),
	})
}
