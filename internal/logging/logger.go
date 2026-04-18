package logging

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured logging capabilities
type Logger struct {
	component string
	level     LogLevel
	logger    *log.Logger
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Component string                 `json:"component"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Stack     string                 `json:"stack,omitempty"`
}

// NewLogger creates a new logger for the specified component
func NewLogger(component string) *Logger {
	level := LevelInfo
	
	// Check environment variable for log level
	if envLevel := os.Getenv("LOG_LEVEL"); envLevel != "" {
		switch strings.ToUpper(envLevel) {
		case "DEBUG":
			level = LevelDebug
		case "INFO":
			level = LevelInfo
		case "WARN":
			level = LevelWarn
		case "ERROR":
			level = LevelError
		}
	}
	
	return &Logger{
		component: component,
		level:     level,
		logger:    log.New(os.Stdout, "", 0),
	}
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// GetLevel returns the current log level
func (l *Logger) GetLevel() LogLevel {
	return l.level
}

// IsDebugEnabled returns true if debug logging is enabled
func (l *Logger) IsDebugEnabled() bool {
	return l.level <= LevelDebug
}

// log writes a structured log entry
func (l *Logger) log(level LogLevel, message string, fields map[string]interface{}, err error) {
	if level < l.level {
		return
	}
	
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     level.String(),
		Component: l.component,
		Message:   message,
		Fields:    fields,
	}
	
	if err != nil {
		entry.Error = err.Error()
		
		// Add stack trace for errors
		if level == LevelError {
			entry.Stack = getStackTrace()
		}
	}
	
	// Marshal to JSON
	jsonBytes, marshalErr := json.Marshal(entry)
	if marshalErr != nil {
		// Fallback to plain text if JSON marshaling fails
		l.logger.Printf("[%s] %s: %s (JSON marshal error: %v)", level.String(), l.component, message, marshalErr)
		return
	}
	
	l.logger.Println(string(jsonBytes))
}

// Debug logs a debug message
func (l *Logger) Debug(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LevelDebug, message, f, nil)
}

// Info logs an info message
func (l *Logger) Info(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LevelInfo, message, f, nil)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LevelWarn, message, f, nil)
}

// Error logs an error message
func (l *Logger) Error(message string, err error, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LevelError, message, f, err)
}

// WithFields creates a new logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *FieldLogger {
	return &FieldLogger{
		logger: l,
		fields: fields,
	}
}

// FieldLogger is a logger with pre-set fields
type FieldLogger struct {
	logger *Logger
	fields map[string]interface{}
}

// Debug logs a debug message with pre-set fields
func (fl *FieldLogger) Debug(message string, additionalFields ...map[string]interface{}) {
	fields := fl.mergeFields(additionalFields...)
	fl.logger.log(LevelDebug, message, fields, nil)
}

// Info logs an info message with pre-set fields
func (fl *FieldLogger) Info(message string, additionalFields ...map[string]interface{}) {
	fields := fl.mergeFields(additionalFields...)
	fl.logger.log(LevelInfo, message, fields, nil)
}

// Warn logs a warning message with pre-set fields
func (fl *FieldLogger) Warn(message string, additionalFields ...map[string]interface{}) {
	fields := fl.mergeFields(additionalFields...)
	fl.logger.log(LevelWarn, message, fields, nil)
}

// Error logs an error message with pre-set fields
func (fl *FieldLogger) Error(message string, err error, additionalFields ...map[string]interface{}) {
	fields := fl.mergeFields(additionalFields...)
	fl.logger.log(LevelError, message, fields, err)
}

// mergeFields merges pre-set fields with additional fields
func (fl *FieldLogger) mergeFields(additionalFields ...map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	
	// Copy pre-set fields
	for k, v := range fl.fields {
		merged[k] = v
	}
	
	// Add additional fields (overwrite if duplicate)
	if len(additionalFields) > 0 {
		for k, v := range additionalFields[0] {
			merged[k] = v
		}
	}
	
	return merged
}

// getStackTrace returns a formatted stack trace
func getStackTrace() string {
	const maxDepth = 32
	var pcs [maxDepth]uintptr
	n := runtime.Callers(3, pcs[:]) // Skip runtime.Callers, getStackTrace, and log
	
	frames := runtime.CallersFrames(pcs[:n])
	
	var builder strings.Builder
	for {
		frame, more := frames.Next()
		
		// Skip runtime and logging package frames
		if !strings.Contains(frame.File, "runtime/") && !strings.Contains(frame.File, "logging/") {
			builder.WriteString(fmt.Sprintf("%s:%d %s\n", frame.File, frame.Line, frame.Function))
		}
		
		if !more {
			break
		}
	}
	
	return builder.String()
}
