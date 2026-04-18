package auth

import (
	"context"
	"testing"
	"time"
)

// TestBearerResolver_Comprehensive provides comprehensive table-driven tests for bearer token resolution
func TestBearerResolver_Comprehensive(t *testing.T) {
	tests := []struct {
		name        string
		authType    AuthType
		authMode    AuthMode
		token       string
		tokenExp    time.Time
		wantToken   string
		wantErr     bool
		description string
	}{
		{
			name:        "valid desktop token with plenty of time",
			authType:    AuthTypeDesktop,
			authMode:    AuthModeBearerToken,
			token:       "desktop-token-123",
			tokenExp:    time.Now().Add(10 * time.Minute),
			wantToken:   "desktop-token-123",
			wantErr:     false,
			description: "Desktop token with 10+ minutes remaining should be returned as-is",
		},
		{
			name:        "valid OIDC token with plenty of time",
			authType:    AuthTypeOIDC,
			authMode:    AuthModeBearerToken,
			token:       "oidc-token-456",
			tokenExp:    time.Now().Add(15 * time.Minute),
			wantToken:   "oidc-token-456",
			wantErr:     false,
			description: "OIDC token with 15+ minutes remaining should be returned as-is",
		},
		{
			name:        "valid CLI DB token with plenty of time",
			authType:    AuthTypeCLIDB,
			authMode:    AuthModeBearerToken,
			token:       "cli-db-token-789",
			tokenExp:    time.Now().Add(20 * time.Minute),
			wantToken:   "cli-db-token-789",
			wantErr:     false,
			description: "CLI DB token with 20+ minutes remaining should be returned as-is",
		},
		{
			name:        "token with exactly 2 minutes remaining",
			authType:    AuthTypeDesktop,
			authMode:    AuthModeBearerToken,
			token:       "token-2min",
			tokenExp:    time.Now().Add(2 * time.Minute),
			wantToken:   "token-2min",
			wantErr:     false,
			description: "Token with exactly 2 minutes remaining should be valid",
		},
		{
			name:        "token with 90 seconds remaining",
			authType:    AuthTypeDesktop,
			authMode:    AuthModeBearerToken,
			token:       "token-90s",
			tokenExp:    time.Now().Add(90 * time.Second),
			wantToken:   "token-90s",
			wantErr:     false,
			description: "Token with 90 seconds remaining should be valid",
		},
		{
			name:        "token expires within safety margin (59 seconds)",
			authType:    AuthTypeDesktop,
			authMode:    AuthModeBearerToken,
			token:       "expires-soon-59s",
			tokenExp:    time.Now().Add(59 * time.Second),
			wantToken:   "",
			wantErr:     true,
			description: "Token expiring within 1 minute should trigger refresh (fails without backend)",
		},
		{
			name:        "token expires within safety margin (30 seconds)",
			authType:    AuthTypeDesktop,
			authMode:    AuthModeBearerToken,
			token:       "expires-soon-30s",
			tokenExp:    time.Now().Add(30 * time.Second),
			wantToken:   "",
			wantErr:     true,
			description: "Token expiring in 30 seconds should trigger refresh (fails without backend)",
		},
		{
			name:        "token expires exactly at safety margin (60 seconds)",
			authType:    AuthTypeDesktop,
			authMode:    AuthModeBearerToken,
			token:       "expires-exactly-60s",
			tokenExp:    time.Now().Add(60 * time.Second),
			wantToken:   "",
			wantErr:     true,
			description: "Token at exact safety margin should trigger refresh",
		},
		{
			name:        "already expired token (1 minute ago)",
			authType:    AuthTypeDesktop,
			authMode:    AuthModeBearerToken,
			token:       "expired-1min",
			tokenExp:    time.Now().Add(-1 * time.Minute),
			wantToken:   "",
			wantErr:     true,
			description: "Token expired 1 minute ago should trigger refresh (fails without backend)",
		},
		{
			name:        "already expired token (1 hour ago)",
			authType:    AuthTypeDesktop,
			authMode:    AuthModeBearerToken,
			token:       "expired-1hour",
			tokenExp:    time.Now().Add(-1 * time.Hour),
			wantToken:   "",
			wantErr:     true,
			description: "Token expired 1 hour ago should trigger refresh (fails without backend)",
		},
		{
			name:        "empty token with valid expiration",
			authType:    AuthTypeDesktop,
			authMode:    AuthModeBearerToken,
			token:       "",
			tokenExp:    time.Now().Add(10 * time.Minute),
			wantToken:   "",
			wantErr:     true,
			description: "Empty token should cause error regardless of expiration",
		},
		{
			name:        "whitespace-only token",
			authType:    AuthTypeDesktop,
			authMode:    AuthModeBearerToken,
			token:       "   ",
			tokenExp:    time.Now().Add(10 * time.Minute),
			wantToken:   "   ",
			wantErr:     false,
			description: "Whitespace-only token should be treated as valid (trimming is not our responsibility)",
		},
		{
			name:        "very long token",
			authType:    AuthTypeDesktop,
			authMode:    AuthModeBearerToken,
			token:       string(make([]byte, 1000)),
			tokenExp:    time.Now().Add(10 * time.Minute),
			wantToken:   string(make([]byte, 1000)),
			wantErr:     false,
			description: "Very long token should be handled correctly",
		},
		{
			name:        "token with special characters",
			authType:    AuthTypeDesktop,
			authMode:    AuthModeBearerToken,
			token:       "token-with-special-chars!@#$%^&*()",
			tokenExp:    time.Now().Add(10 * time.Minute),
			wantToken:   "token-with-special-chars!@#$%^&*()",
			wantErr:     false,
			description: "Token with special characters should be preserved",
		},
		{
			name:        "SigV4 auth mode (should not use bearer resolver)",
			authType:    AuthTypeDesktop,
			authMode:    AuthModeSigV4,
			token:       "sigv4-token",
			tokenExp:    time.Now().Add(10 * time.Minute),
			wantToken:   "sigv4-token",
			wantErr:     false,
			description: "SigV4 mode should still return token (resolver doesn't check mode)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create auth manager with test configuration
			am := &AuthManager{
				authType: tt.authType,
				authMode: tt.authMode,
				token:    tt.token,
				tokenExp: tt.tokenExp,
			}

			resolver := NewBearerResolver(am)
			ctx := context.Background()
			
			token, err := resolver.GetToken(ctx)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("GetToken() error = %v, wantErr %v (%s)", err, tt.wantErr, tt.description)
				return
			}
			
			if !tt.wantErr && token != tt.wantToken {
				t.Errorf("GetToken() = %q, want %q (%s)", token, tt.wantToken, tt.description)
			}
		})
	}
}

// TestBearerResolver_IsExpired_Comprehensive provides comprehensive tests for token expiration checking
func TestBearerResolver_IsExpired_Comprehensive(t *testing.T) {
	tests := []struct {
		name        string
		tokenExp    time.Time
		wantExpired bool
		description string
	}{
		{
			name:        "token valid for 10 minutes",
			tokenExp:    time.Now().Add(10 * time.Minute),
			wantExpired: false,
			description: "Token with 10 minutes remaining should not be expired",
		},
		{
			name:        "token valid for 5 minutes",
			tokenExp:    time.Now().Add(5 * time.Minute),
			wantExpired: false,
			description: "Token with 5 minutes remaining should not be expired",
		},
		{
			name:        "token valid for 2 minutes",
			tokenExp:    time.Now().Add(2 * time.Minute),
			wantExpired: false,
			description: "Token with 2 minutes remaining should not be expired",
		},
		{
			name:        "token valid for 90 seconds",
			tokenExp:    time.Now().Add(90 * time.Second),
			wantExpired: false,
			description: "Token with 90 seconds remaining should not be expired",
		},
		{
			name:        "token valid for exactly 61 seconds",
			tokenExp:    time.Now().Add(61 * time.Second),
			wantExpired: false,
			description: "Token with 61 seconds remaining should not be expired",
		},
		{
			name:        "token valid for exactly 60 seconds (safety margin)",
			tokenExp:    time.Now().Add(60 * time.Second),
			wantExpired: true,
			description: "Token at exact safety margin (60s) should be considered expired",
		},
		{
			name:        "token valid for 59 seconds",
			tokenExp:    time.Now().Add(59 * time.Second),
			wantExpired: true,
			description: "Token with 59 seconds remaining should be considered expired",
		},
		{
			name:        "token valid for 30 seconds",
			tokenExp:    time.Now().Add(30 * time.Second),
			wantExpired: true,
			description: "Token with 30 seconds remaining should be considered expired",
		},
		{
			name:        "token valid for 1 second",
			tokenExp:    time.Now().Add(1 * time.Second),
			wantExpired: true,
			description: "Token with 1 second remaining should be considered expired",
		},
		{
			name:        "token expires now",
			tokenExp:    time.Now(),
			wantExpired: true,
			description: "Token expiring now should be considered expired",
		},
		{
			name:        "token expired 1 second ago",
			tokenExp:    time.Now().Add(-1 * time.Second),
			wantExpired: true,
			description: "Token expired 1 second ago should be considered expired",
		},
		{
			name:        "token expired 1 minute ago",
			tokenExp:    time.Now().Add(-1 * time.Minute),
			wantExpired: true,
			description: "Token expired 1 minute ago should be considered expired",
		},
		{
			name:        "token expired 1 hour ago",
			tokenExp:    time.Now().Add(-1 * time.Hour),
			wantExpired: true,
			description: "Token expired 1 hour ago should be considered expired",
		},
		{
			name:        "token expired 1 day ago",
			tokenExp:    time.Now().Add(-24 * time.Hour),
			wantExpired: true,
			description: "Token expired 1 day ago should be considered expired",
		},
		{
			name:        "zero time (uninitialized)",
			tokenExp:    time.Time{},
			wantExpired: true,
			description: "Zero time should be considered expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			am := &AuthManager{
				authType: AuthTypeDesktop,
				authMode: AuthModeBearerToken,
				token:    "test-token",
				tokenExp: tt.tokenExp,
			}

			resolver := NewBearerResolver(am)

			got := resolver.IsExpired()
			if got != tt.wantExpired {
				t.Errorf("IsExpired() = %v, want %v (%s)", got, tt.wantExpired, tt.description)
			}
		})
	}
}

// TestBearerResolver_GetExpiration_Comprehensive provides comprehensive tests for expiration time retrieval
func TestBearerResolver_GetExpiration_Comprehensive(t *testing.T) {
	tests := []struct {
		name        string
		tokenExp    time.Time
		description string
	}{
		{
			name:        "future expiration",
			tokenExp:    time.Now().Add(1 * time.Hour),
			description: "Future expiration should be returned accurately",
		},
		{
			name:        "past expiration",
			tokenExp:    time.Now().Add(-1 * time.Hour),
			description: "Past expiration should be returned accurately",
		},
		{
			name:        "current time expiration",
			tokenExp:    time.Now(),
			description: "Current time expiration should be returned accurately",
		},
		{
			name:        "zero time expiration",
			tokenExp:    time.Time{},
			description: "Zero time expiration should be returned as-is",
		},
		{
			name:        "far future expiration",
			tokenExp:    time.Now().Add(365 * 24 * time.Hour),
			description: "Far future expiration should be handled correctly",
		},
		{
			name:        "far past expiration",
			tokenExp:    time.Now().Add(-365 * 24 * time.Hour),
			description: "Far past expiration should be handled correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			am := &AuthManager{
				authType: AuthTypeDesktop,
				authMode: AuthModeBearerToken,
				token:    "test-token",
				tokenExp: tt.tokenExp,
			}

			resolver := NewBearerResolver(am)
			got := resolver.GetExpiration()
			
			// Allow 1 second difference due to timing in tests
			diff := got.Sub(tt.tokenExp)
			if diff < 0 {
				diff = -diff
			}
			
			if diff > time.Second {
				t.Errorf("GetExpiration() = %v, want %v (diff: %v) (%s)", 
					got, tt.tokenExp, diff, tt.description)
			}
		})
	}
}

// TestNewBearerResolver_Comprehensive provides comprehensive tests for bearer resolver creation
func TestNewBearerResolver_Comprehensive(t *testing.T) {
	tests := []struct {
		name        string
		authManager *AuthManager
		wantNil     bool
		description string
	}{
		{
			name: "valid auth manager",
			authManager: &AuthManager{
				authType: AuthTypeDesktop,
				authMode: AuthModeBearerToken,
				token:    "test-token",
				tokenExp: time.Now().Add(1 * time.Hour),
			},
			wantNil:     false,
			description: "Valid auth manager should create resolver",
		},
		{
			name: "auth manager with empty token",
			authManager: &AuthManager{
				authType: AuthTypeDesktop,
				authMode: AuthModeBearerToken,
				token:    "",
				tokenExp: time.Now().Add(1 * time.Hour),
			},
			wantNil:     false,
			description: "Auth manager with empty token should still create resolver",
		},
		{
			name: "auth manager with expired token",
			authManager: &AuthManager{
				authType: AuthTypeDesktop,
				authMode: AuthModeBearerToken,
				token:    "expired-token",
				tokenExp: time.Now().Add(-1 * time.Hour),
			},
			wantNil:     false,
			description: "Auth manager with expired token should still create resolver",
		},
		{
			name: "auth manager with SigV4 mode",
			authManager: &AuthManager{
				authType: AuthTypeDesktop,
				authMode: AuthModeSigV4,
				token:    "test-token",
				tokenExp: time.Now().Add(1 * time.Hour),
			},
			wantNil:     false,
			description: "Auth manager with SigV4 mode should still create resolver",
		},
		{
			name: "auth manager with OIDC type",
			authManager: &AuthManager{
				authType: AuthTypeOIDC,
				authMode: AuthModeBearerToken,
				token:    "oidc-token",
				tokenExp: time.Now().Add(1 * time.Hour),
			},
			wantNil:     false,
			description: "Auth manager with OIDC type should create resolver",
		},
		{
			name: "auth manager with CLI DB type",
			authManager: &AuthManager{
				authType: AuthTypeCLIDB,
				authMode: AuthModeBearerToken,
				token:    "cli-token",
				tokenExp: time.Now().Add(1 * time.Hour),
			},
			wantNil:     false,
			description: "Auth manager with CLI DB type should create resolver",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewBearerResolver(tt.authManager)
			
			if (resolver == nil) != tt.wantNil {
				t.Errorf("NewBearerResolver() = %v, wantNil %v (%s)", 
					resolver, tt.wantNil, tt.description)
				return
			}
			
			if resolver != nil {
				if resolver.authManager != tt.authManager {
					t.Errorf("NewBearerResolver() authManager = %v, want %v (%s)", 
						resolver.authManager, tt.authManager, tt.description)
				}
			}
		})
	}
}
