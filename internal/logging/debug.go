package logging

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	debugEnabled bool
	debugOnce    sync.Once
	debugLogger  *log.Logger
	debugFile    *os.File
)

// InitDebugLogging initializes the debug logging system
func InitDebugLogging() {
	debugOnce.Do(func() {
		// Check if DEBUG mode is enabled
		debugEnv := os.Getenv("DEBUG")
		debugEnabled = debugEnv == "true" || debugEnv == "1"
		
		if debugEnabled {
			// Create logs directory if it doesn't exist
			logsDir := "logs"
			if err := os.MkdirAll(logsDir, 0755); err != nil {
				log.Printf("[WARNING] Failed to create logs directory: %v", err)
				return
			}
			
			// Create debug log file with timestamp
			timestamp := time.Now().Format("20060102-150405")
			debugLogPath := filepath.Join(logsDir, fmt.Sprintf("debug-%s.log", timestamp))
			
			var err error
			debugFile, err = os.OpenFile(debugLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				log.Printf("[WARNING] Failed to create debug log file: %v", err)
				return
			}
			
			// Create logger that writes to both file and stdout
			debugLogger = log.New(debugFile, "[DEBUG] ", log.LstdFlags|log.Lmicroseconds)
			
			log.Printf("[SUCCESS] Debug logging enabled - writing to %s", debugLogPath)
		}
	})
}

// DebugLog logs a debug message if DEBUG mode is enabled
func DebugLog(format string, args ...interface{}) {
	if !debugEnabled {
		return
	}
	
	message := fmt.Sprintf(format, args...)
	
	// Write to debug log file
	if debugLogger != nil {
		debugLogger.Println(message)
	}
	
	// Also write to stdout with [DEBUG] prefix
	log.Printf("[DEBUG] %s", message)
}

// IsDebugEnabled returns whether debug mode is enabled
func IsDebugEnabled() bool {
	return debugEnabled
}

// CloseDebugLog closes the debug log file
func CloseDebugLog() {
	if debugFile != nil {
		debugFile.Close()
	}
}
