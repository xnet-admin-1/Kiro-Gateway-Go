package apikeys

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

// APIKey represents an API key with metadata
type APIKey struct {
	Key         string
	Name        string
	UserID      string
	CreatedAt   time.Time
	ExpiresAt   *time.Time
	LastUsedAt  *time.Time
	IsActive    bool
	Permissions []string
	Metadata    map[string]string
	UsageCount  int64
}

// APIKeyManager manages API keys
type APIKeyManager struct {
	keys   map[string]*APIKey
	mu     sync.RWMutex
	prefix string
}

// NewAPIKeyManager creates a new API key manager
func NewAPIKeyManager(prefix string) *APIKeyManager {
	if prefix == "" {
		prefix = "kiro"
	}
	
	return &APIKeyManager{
		keys:   make(map[string]*APIKey),
		prefix: prefix,
	}
}

// GenerateKey generates a new API key
func (m *APIKeyManager) GenerateKey(name, userID string, expiresIn *time.Duration, permissions []string) (*APIKey, error) {
	// Generate random bytes
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}
	
	// Encode to base64
	keySecret := base64.RawURLEncoding.EncodeToString(randomBytes)
	
	// Format: {prefix}-{secret}
	key := fmt.Sprintf("%s-%s", m.prefix, keySecret)
	
	// Create API key object
	apiKey := &APIKey{
		Key:         key,
		Name:        name,
		UserID:      userID,
		CreatedAt:   time.Now(),
		IsActive:    true,
		Permissions: permissions,
		Metadata:    make(map[string]string),
		UsageCount:  0,
	}
	
	// Set expiration if provided
	if expiresIn != nil {
		expiresAt := time.Now().Add(*expiresIn)
		apiKey.ExpiresAt = &expiresAt
	}
	
	// Store key
	m.mu.Lock()
	m.keys[key] = apiKey
	m.mu.Unlock()
	
	return apiKey, nil
}

// ValidateKey validates an API key and returns the key object
func (m *APIKeyManager) ValidateKey(key string) (*APIKey, error) {
	m.mu.RLock()
	apiKey, exists := m.keys[key]
	m.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("invalid API key")
	}
	
	// Check if active
	if !apiKey.IsActive {
		return nil, fmt.Errorf("API key is inactive")
	}
	
	// Check expiration
	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		return nil, fmt.Errorf("API key has expired")
	}
	
	// Update last used time and usage count
	m.mu.Lock()
	now := time.Now()
	apiKey.LastUsedAt = &now
	apiKey.UsageCount++
	m.mu.Unlock()
	
	return apiKey, nil
}

// RevokeKey revokes an API key
func (m *APIKeyManager) RevokeKey(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	apiKey, exists := m.keys[key]
	if !exists {
		return fmt.Errorf("API key not found")
	}
	
	apiKey.IsActive = false
	return nil
}

// DeleteKey deletes an API key
func (m *APIKeyManager) DeleteKey(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.keys[key]; !exists {
		return fmt.Errorf("API key not found")
	}
	
	delete(m.keys, key)
	return nil
}

// ListKeys lists all API keys for a user
func (m *APIKeyManager) ListKeys(userID string) []*APIKey {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var keys []*APIKey
	for _, apiKey := range m.keys {
		if userID == "" || apiKey.UserID == userID {
			keys = append(keys, apiKey)
		}
	}
	
	return keys
}

// GetKey gets an API key by key string
func (m *APIKeyManager) GetKey(key string) (*APIKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	apiKey, exists := m.keys[key]
	if !exists {
		return nil, fmt.Errorf("API key not found")
	}
	
	return apiKey, nil
}

// UpdateKey updates an API key's metadata
func (m *APIKeyManager) UpdateKey(key string, name string, metadata map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	apiKey, exists := m.keys[key]
	if !exists {
		return fmt.Errorf("API key not found")
	}
	
	if name != "" {
		apiKey.Name = name
	}
	
	if metadata != nil {
		for k, v := range metadata {
			apiKey.Metadata[k] = v
		}
	}
	
	return nil
}

// GetStats returns statistics about API keys
func (m *APIKeyManager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	totalKeys := len(m.keys)
	activeKeys := 0
	expiredKeys := 0
	totalUsage := int64(0)
	
	for _, apiKey := range m.keys {
		if apiKey.IsActive {
			activeKeys++
		}
		if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
			expiredKeys++
		}
		totalUsage += apiKey.UsageCount
	}
	
	return map[string]interface{}{
		"total_keys":   totalKeys,
		"active_keys":  activeKeys,
		"expired_keys": expiredKeys,
		"total_usage":  totalUsage,
	}
}

// CleanupExpiredKeys removes expired keys
func (m *APIKeyManager) CleanupExpiredKeys() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	count := 0
	now := time.Now()
	
	for key, apiKey := range m.keys {
		if apiKey.ExpiresAt != nil && now.After(*apiKey.ExpiresAt) {
			delete(m.keys, key)
			count++
		}
	}
	
	return count
}

// MaskKey returns a masked version of the API key for display
func MaskKey(key string) string {
	if len(key) <= 12 {
		return "***"
	}
	
	// Show prefix and last 4 characters
	prefix := key[:8]
	suffix := key[len(key)-4:]
	
	return fmt.Sprintf("%s...%s", prefix, suffix)
}
