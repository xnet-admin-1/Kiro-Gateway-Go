package auth

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// BearerResolver resolves bearer tokens with automatic refresh
type BearerResolver struct {
	// authManager is the parent auth manager
	authManager *AuthManager
	
	// mu protects concurrent access to token operations
	mu sync.RWMutex
}

// NewBearerResolver creates a new BearerResolver
func NewBearerResolver(manager *AuthManager) *BearerResolver {
	return &BearerResolver{
		authManager: manager,
	}
}

// GetToken returns a valid bearer token, refreshing if necessary
func (r *BearerResolver) GetToken(ctx context.Context) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Get current token expiration
	expiration := r.authManager.GetTokenExpiration()
	log.Printf("[DEBUG] Token expiration: %s (in %v)", expiration.Format(time.RFC3339), time.Until(expiration))
	
	// Check if token is expired or expiring soon (within 1 minute)
	if r.IsExpired() {
		log.Printf("[DEBUG] Token is expired or expiring within 1 minute, triggering refresh")
		
		// Attempt to refresh the token
		if err := r.authManager.RefreshToken(ctx); err != nil {
			log.Printf("[ERROR] Failed to refresh token: %v", err)
			return "", fmt.Errorf("failed to refresh token: %w", err)
		}
		
		// Get new expiration
		newExpiration := r.authManager.GetTokenExpiration()
		log.Printf("[INFO] Token refreshed successfully, new expiration: %s (in %v)", newExpiration.Format(time.RFC3339), time.Until(newExpiration))
	} else {
		log.Printf("[DEBUG] Token is still valid, no refresh needed")
	}
	
	// Return the current token
	r.authManager.mu.RLock()
	token := r.authManager.token
	r.authManager.mu.RUnlock()
	
	if token == "" {
		return "", fmt.Errorf("no token available")
	}
	
	return token, nil
}

// IsExpired checks if the token is expired or expiring soon (within 1 minute)
func (r *BearerResolver) IsExpired() bool {
	return r.authManager.IsTokenExpired()
}

// GetExpiration returns the token expiration time
func (r *BearerResolver) GetExpiration() time.Time {
	return r.authManager.GetTokenExpiration()
}
