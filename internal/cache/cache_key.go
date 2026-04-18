package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// CacheKey represents a unique identifier for a cached response
type CacheKey struct {
	ModelID      string
	Prompt       string
	SystemPrompt string
	Temperature  float64
	MaxTokens    int
}

// String returns a string representation of the cache key
func (k CacheKey) String() string {
	return fmt.Sprintf("model=%s,prompt=%s,system=%s,temp=%.2f,max=%d",
		k.ModelID, truncate(k.Prompt, 50), truncate(k.SystemPrompt, 30), k.Temperature, k.MaxTokens)
}

// Hash returns a SHA256 hash of the cache key
func (k CacheKey) Hash() string {
	// Create a deterministic string representation
	parts := []string{
		fmt.Sprintf("model:%s", k.ModelID),
		fmt.Sprintf("prompt:%s", k.Prompt),
		fmt.Sprintf("system:%s", k.SystemPrompt),
		fmt.Sprintf("temp:%.2f", k.Temperature),
		fmt.Sprintf("max:%d", k.MaxTokens),
	}
	
	// Sort to ensure consistent ordering
	sort.Strings(parts)
	
	// Hash the combined string
	data := strings.Join(parts, "|")
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// IsCacheable returns true if this request should be cached
func (k CacheKey) IsCacheable() bool {
	// Don't cache empty prompts
	if k.Prompt == "" {
		return false
	}
	
	// Don't cache if model is empty
	if k.ModelID == "" {
		return false
	}
	
	return true
}

// truncate truncates a string to the specified length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
