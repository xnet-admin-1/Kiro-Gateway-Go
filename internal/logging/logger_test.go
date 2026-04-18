package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"os"
	"strings"
	"testing"
)

func TestNewLogger(t *testing.T) {
	logger := NewLogger("test-component")
	
	if logger.component != "test-component" {
		t.Errorf("Expected component 'test-component', got '%s'", logger.component)
	}
	
	if logger.level != LevelInfo {
		t.Errorf("Expected default level INFO, got %v", logger.level)
	}
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
	}
	
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.level.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.level.String())
			}
		})
	}
}

func TestLogOutput(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	logger := NewLogger("test")
	logger.logger = log.New(&buf, "", 0)
	
	// Log a message
	logger.Info("test message", map[string]interface{}{
		"key": "value",
	})
	
	// Parse JSON output
	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}
	
	// Verify fields
	if entry.Level != "INFO" {
		t.Errorf("Expected level INFO, got %s", entry.Level)
	}
	if entry.Component != "test" {
		t.Errorf("Expected component 'test', got '%s'", entry.Component)
	}
	if entry.Message != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", entry.Message)
	}
	if entry.Fields["key"] != "value" {
		t.Errorf("Expected field key='value', got %v", entry.Fields["key"])
	}
}

func TestLogWithError(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	logger := NewLogger("test")
	logger.logger = log.New(&buf, "", 0)
	
	// Log an error
	testErr := errors.New("test error")
	logger.Error("error occurred", testErr, map[string]interface{}{
		"context": "testing",
	})
	
	// Parse JSON output
	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}
	
	// Verify error fields
	if entry.Level != "ERROR" {
		t.Errorf("Expected level ERROR, got %s", entry.Level)
	}
	if entry.Error != "test error" {
		t.Errorf("Expected error 'test error', got '%s'", entry.Error)
	}
	if entry.Stack == "" {
		t.Error("Expected stack trace for error, got empty string")
	}
}

func TestLogLevelFiltering(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	logger := NewLogger("test")
	logger.logger = log.New(&buf, "", 0)
	logger.SetLevel(LevelWarn)
	
	// Log at different levels
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	
	output := buf.String()
	
	// Debug and Info should be filtered out
	if strings.Contains(output, "debug message") {
		t.Error("Debug message should be filtered out")
	}
	if strings.Contains(output, "info message") {
		t.Error("Info message should be filtered out")
	}
	
	// Warn should be logged
	if !strings.Contains(output, "warn message") {
		t.Error("Warn message should be logged")
	}
}

func TestFieldLogger(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	logger := NewLogger("test")
	logger.logger = log.New(&buf, "", 0)
	
	// Create field logger with pre-set fields
	fieldLogger := logger.WithFields(map[string]interface{}{
		"request_id": "req-123",
		"user_id":    "user-456",
	})
	
	// Log with additional fields
	fieldLogger.Info("test message", map[string]interface{}{
		"action": "test",
	})
	
	// Parse JSON output
	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}
	
	// Verify all fields are present
	if entry.Fields["request_id"] != "req-123" {
		t.Errorf("Expected request_id='req-123', got %v", entry.Fields["request_id"])
	}
	if entry.Fields["user_id"] != "user-456" {
		t.Errorf("Expected user_id='user-456', got %v", entry.Fields["user_id"])
	}
	if entry.Fields["action"] != "test" {
		t.Errorf("Expected action='test', got %v", entry.Fields["action"])
	}
}

func TestLogLevelFromEnv(t *testing.T) {
	// Save original env
	originalLevel := os.Getenv("LOG_LEVEL")
	defer os.Setenv("LOG_LEVEL", originalLevel)
	
	tests := []struct {
		envValue string
		expected LogLevel
	}{
		{"DEBUG", LevelDebug},
		{"INFO", LevelInfo},
		{"WARN", LevelWarn},
		{"ERROR", LevelError},
		{"invalid", LevelInfo}, // Default to INFO for invalid values
	}
	
	for _, tt := range tests {
		t.Run(tt.envValue, func(t *testing.T) {
			os.Setenv("LOG_LEVEL", tt.envValue)
			logger := NewLogger("test")
			
			if logger.level != tt.expected {
				t.Errorf("Expected level %v, got %v", tt.expected, logger.level)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration string
		contains string
	}{
		{"microseconds", "500µs", "µs"},
		{"milliseconds", "50ms", "ms"},
		{"seconds", "2.5s", "s"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(tt.duration, tt.contains) {
				t.Errorf("Expected duration to contain '%s', got '%s'", tt.contains, tt.duration)
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int
		expected string
	}{
		{100, "100B"},
		{1024, "1.0KB"},
		{1024 * 1024, "1.0MB"},
		{1024 * 1024 * 1024, "1.0GB"},
	}
	
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatSize(tt.bytes)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		contains string
	}{
		{"simple object", map[string]string{"key": "value"}, `"key":"value"`},
		{"array", []int{1, 2, 3}, "[1,2,3]"},
		{"string", "test", `"test"`},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MarshalJSON(tt.input)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("Expected JSON to contain '%s', got '%s'", tt.contains, result)
			}
		})
	}
}
