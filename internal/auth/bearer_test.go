package auth

import (
	"context"
	"testing"
	"time"
)

func TestBearerResolver_GetToken_Valid(t *testing.T) {
	// Create auth manager with valid token
	am := &AuthManager{
		authType: AuthTypeDesktop,
		authMode: AuthModeBearerToken,
		token:    "valid-token-123",
		tokenExp: time.Now().Add(10 * time.Minute),
	}

	resolver := NewBearerResolver(am)

	ctx := context.Background()
	token, err := resolver.GetToken(ctx)
	if err != nil {
		t.Fatalf("GetToken() error = %v, want nil", err)
	}

	if token != "valid-token-123" {
		t.Errorf("GetToken() = %q, want %q", token, "valid-token-123")
	}
}

func TestBearerResolver_GetToken_Expired(t *testing.T) {
	// Create auth manager with expired token
	am := &AuthManager{
		authType: AuthTypeDesktop,
		authMode: AuthModeBearerToken,
		token:    "expired-token",
		tokenExp: time.Now().Add(-1 * time.Minute), // Already expired
	}

	resolver := NewBearerResolver(am)

	ctx := context.Background()
	_, err := resolver.GetToken(ctx)
	
	// We expect an error because refresh will fail (no real backend)
	if err == nil {
		t.Error("GetToken() with expired token should return error when refresh fails")
	}
}

func TestBearerResolver_GetToken_ExpiresSoon(t *testing.T) {
	// Create auth manager with token that expires within 1 minute
	am := &AuthManager{
		authType: AuthTypeDesktop,
		authMode: AuthModeBearerToken,
		token:    "expires-soon-token",
		tokenExp: time.Now().Add(30 * time.Second), // Expires in 30 seconds
	}

	resolver := NewBearerResolver(am)

	ctx := context.Background()
	_, err := resolver.GetToken(ctx)
	
	// We expect an error because refresh will fail (no real backend)
	// The important thing is that it attempted to refresh
	if err == nil {
		t.Error("GetToken() with token expiring soon should attempt refresh")
	}
}

func TestBearerResolver_IsExpired(t *testing.T) {
	tests := []struct {
		name       string
		tokenExp   time.Time
		wantExpired bool
	}{
		{
			name:       "valid token",
			tokenExp:   time.Now().Add(10 * time.Minute),
			wantExpired: false,
		},
		{
			name:       "expires soon (within 1 minute)",
			tokenExp:   time.Now().Add(30 * time.Second),
			wantExpired: true,
		},
		{
			name:       "already expired",
			tokenExp:   time.Now().Add(-1 * time.Minute),
			wantExpired: true,
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

			if got := resolver.IsExpired(); got != tt.wantExpired {
				t.Errorf("IsExpired() = %v, want %v", got, tt.wantExpired)
			}
		})
	}
}

func TestBearerResolver_GetExpiration(t *testing.T) {
	expectedExp := time.Now().Add(1 * time.Hour)
	
	am := &AuthManager{
		authType: AuthTypeDesktop,
		authMode: AuthModeBearerToken,
		token:    "test-token",
		tokenExp: expectedExp,
	}

	resolver := NewBearerResolver(am)

	gotExp := resolver.GetExpiration()
	
	// Allow 1 second difference due to timing
	if gotExp.Sub(expectedExp).Abs() > time.Second {
		t.Errorf("GetExpiration() = %v, want %v", gotExp, expectedExp)
	}
}

func TestBearerResolver_ConcurrentAccess(t *testing.T) {
	am := &AuthManager{
		authType: AuthTypeDesktop,
		authMode: AuthModeBearerToken,
		token:    "concurrent-token",
		tokenExp: time.Now().Add(10 * time.Minute),
	}

	resolver := NewBearerResolver(am)
	ctx := context.Background()

	// Test concurrent access to GetToken
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = resolver.GetToken(ctx)
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// If we get here without deadlock or race condition, test passes
}

func TestNewBearerResolver(t *testing.T) {
	am := &AuthManager{
		authType: AuthTypeDesktop,
		authMode: AuthModeBearerToken,
	}

	resolver := NewBearerResolver(am)

	if resolver == nil {
		t.Fatal("NewBearerResolver() returned nil")
	}

	if resolver.authManager != am {
		t.Error("NewBearerResolver() did not set authManager correctly")
	}
}
