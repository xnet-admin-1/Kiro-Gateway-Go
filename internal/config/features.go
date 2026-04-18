package config

import (
	"os"
	"strconv"
)

// BetaFeatures holds configuration for beta features
type BetaFeatures struct {
	EnableExtendedContext bool // Enable 1M token context window (Sonnet 4.5 only)
	EnableExtendedThinking bool // Enable multi-hour extended thinking
	WarnOnBetaFeatures    bool // Warn users when using beta features
}

// LoadBetaFeatures loads beta feature configuration from environment
func LoadBetaFeatures() BetaFeatures {
	return BetaFeatures{
		EnableExtendedContext:  parseBoolEnv("ENABLE_EXTENDED_CONTEXT", false),
		EnableExtendedThinking: parseBoolEnv("ENABLE_EXTENDED_THINKING", true),
		WarnOnBetaFeatures:     parseBoolEnv("WARN_ON_BETA_FEATURES", true),
	}
}

// parseBoolEnv gets a boolean environment variable with a default value
func parseBoolEnv(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	
	return boolValue
}
