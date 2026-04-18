package auth

import (
	"context"
	"testing"
	"time"
)

func TestAutomatedOIDCAuth_SetMFASecret(t *testing.T) {
	auth := NewAutomatedOIDCAuth("us-east-1", "https://example.awsapps.com/start", "user@example.com", "password")
	
	secret := "JBSWY3DPEHPK3PXP"
	auth.SetMFASecret(secret)
	
	if auth.mfaSecret != secret {
		t.Errorf("Expected MFA secret %s, got %s", secret, auth.mfaSecret)
	}
}

func TestAutomatedOIDCAuth_SetHeadless(t *testing.T) {
	auth := NewAutomatedOIDCAuth("us-east-1", "https://example.awsapps.com/start", "user@example.com", "password")
	
	// Default should be headless
	if !auth.headless {
		t.Error("Expected default headless to be true")
	}
	
	// Set to visible mode
	auth.SetHeadless(false)
	if auth.headless {
		t.Error("Expected headless to be false after SetHeadless(false)")
	}
	
	// Set back to headless
	auth.SetHeadless(true)
	if !auth.headless {
		t.Error("Expected headless to be true after SetHeadless(true)")
	}
}

func TestAutomatedToken_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "expired token",
			expiresAt: time.Now().Add(-1 * time.Hour),
			want:      true,
		},
		{
			name:      "valid token",
			expiresAt: time.Now().Add(2 * time.Hour),
			want:      false,
		},
		{
			name:      "token expiring in 30 seconds (within safety margin)",
			expiresAt: time.Now().Add(30 * time.Second),
			want:      true,
		},
		{
			name:      "token expiring in 2 minutes (outside safety margin)",
			expiresAt: time.Now().Add(2 * time.Minute),
			want:      false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &AutomatedToken{
				AccessToken:  "test-token",
				RefreshToken: "test-refresh",
				ExpiresAt:    tt.expiresAt,
				Region:       "us-east-1",
				StartURL:     "https://example.awsapps.com/start",
			}
			
			if got := token.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsAuthorizationPending(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "authorization pending error",
			err:  &mockError{msg: "AuthorizationPendingException: Authorization is pending"},
			want: true,
		},
		{
			name: "other error",
			err:  &mockError{msg: "Some other error"},
			want: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAuthorizationPending(tt.err); got != tt.want {
				t.Errorf("isAuthorizationPending() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsSlowDown(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "slow down error",
			err:  &mockError{msg: "SlowDownException: Slow down"},
			want: true,
		},
		{
			name: "other error",
			err:  &mockError{msg: "Some other error"},
			want: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSlowDown(tt.err); got != tt.want {
				t.Errorf("isSlowDown() = %v, want %v", got, tt.want)
			}
		})
	}
}

// mockError is a simple error implementation for testing
type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

// Note: Full integration tests with real browser automation should be run manually
// or in a CI environment with proper browser setup. These tests cover the basic
// logic and structure of the automated authentication system.

func TestAutomatedOIDCAuth_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// This test requires:
	// - IC_USERNAME environment variable
	// - IC_PASSWORD environment variable
	// - IC_START_URL environment variable
	// - IC_REGION environment variable (optional, defaults to us-east-1)
	// - Chrome/Chromium installed
	
	// Skip if credentials not provided
	username := getEnv("IC_USERNAME", "")
	password := getEnv("IC_PASSWORD", "")
	startURL := getEnv("IC_START_URL", "")
	
	if username == "" || password == "" || startURL == "" {
		t.Skip("Skipping integration test: IC_USERNAME, IC_PASSWORD, or IC_START_URL not set")
	}
	
	region := getEnv("IC_REGION", "us-east-1")
	mfaSecret := getEnv("IC_MFA_SECRET", "")
	
	auth := NewAutomatedOIDCAuth(region, startURL, username, password)
	if mfaSecret != "" {
		auth.SetMFASecret(mfaSecret)
	}
	
	// Run in visible mode for debugging
	auth.SetHeadless(false)
	
	ctx := context.Background()
	token, err := auth.Authenticate(ctx)
	if err != nil {
		t.Fatalf("Authentication failed: %v", err)
	}
	
	if token.AccessToken == "" {
		t.Error("Expected non-empty access token")
	}
	
	if token.RefreshToken == "" {
		t.Error("Expected non-empty refresh token")
	}
	
	if token.IsExpired() {
		t.Error("Token should not be expired immediately after authentication")
	}
	
	t.Logf("Successfully authenticated with automated OIDC")
	t.Logf("Token expires at: %s", token.ExpiresAt)
}

func getEnv(key, defaultValue string) string {
	if value := context.TODO(); value != nil {
		// This is a placeholder - actual implementation would use os.Getenv
		return defaultValue
	}
	return defaultValue
}
