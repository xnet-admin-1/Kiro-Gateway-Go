package auth

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/yourusername/kiro-gateway-go/internal/storage"
)

const (
	tokenStoreKey = "sso_oidc_token"
)

// StoredToken contains all token and client information
type StoredToken struct {
	// Access token from OIDC
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
	
	// Client registration
	ClientID         string    `json:"client_id"`
	ClientSecret     string    `json:"client_secret"`
	ClientExpiresAt  time.Time `json:"client_expires_at,omitempty"`
	
	// SSO configuration
	Region    string `json:"region"`
	StartURL  string `json:"start_url"`
	AccountID string `json:"account_id"`
	RoleName  string `json:"role_name"`
	
	// Metadata
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// IsExpired checks if the access token has expired
func (t *StoredToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// NeedsRefresh checks if the token should be refreshed (within 5 min of expiry)
func (t *StoredToken) NeedsRefresh() bool {
	return time.Until(t.ExpiresAt) < 5*time.Minute
}

// IsClientExpired checks if the client registration has expired
func (t *StoredToken) IsClientExpired() bool {
	if t.ClientExpiresAt.IsZero() {
		return false // No expiry
	}
	return time.Now().After(t.ClientExpiresAt)
}

// TokenStore manages secure storage of OIDC tokens
type TokenStore struct {
	storage storage.Store
}

// NewTokenStore creates a new token store
func NewTokenStore(store storage.Store) *TokenStore {
	return &TokenStore{
		storage: store,
	}
}

// SaveToken saves a token to secure storage
func (s *TokenStore) SaveToken(token *StoredToken) error {
	token.UpdatedAt = time.Now()
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now()
	}
	
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}
	
	if err := s.storage.Set(tokenStoreKey, data); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}
	
	return nil
}

// LoadToken loads a token from secure storage
func (s *TokenStore) LoadToken() (*StoredToken, error) {
	data, err := s.storage.Get(tokenStoreKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load token: %w", err)
	}
	
	var token StoredToken
	if err := json.Unmarshal([]byte(data), &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}
	
	return &token, nil
}

// DeleteToken removes the token from storage
func (s *TokenStore) DeleteToken() error {
	if err := s.storage.Delete(tokenStoreKey); err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}
	return nil
}

// HasToken checks if a token exists in storage
func (s *TokenStore) HasToken() bool {
	_, err := s.storage.Get(tokenStoreKey)
	return err == nil
}
