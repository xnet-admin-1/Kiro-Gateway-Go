package credentials

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNewSSOProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  SSOProviderConfig
		wantErr bool
	}{
		{
			name: "valid config with access token",
			config: SSOProviderConfig{
				Region:      "us-east-1",
				AccountID:   "123456789012",
				RoleName:    "MyRole",
				AccessToken: "test-token",
			},
			wantErr: false,
		},
		{
			name: "valid config with token provider",
			config: SSOProviderConfig{
				Region:    "us-east-1",
				AccountID: "123456789012",
				RoleName:  "MyRole",
				AccessTokenProvider: func() (string, error) {
					return "test-token", nil
				},
			},
			wantErr: false,
		},
		{
			name: "missing region",
			config: SSOProviderConfig{
				AccountID:   "123456789012",
				RoleName:    "MyRole",
				AccessToken: "test-token",
			},
			wantErr: true,
		},
		{
			name: "missing account ID",
			config: SSOProviderConfig{
				Region:      "us-east-1",
				RoleName:    "MyRole",
				AccessToken: "test-token",
			},
			wantErr: true,
		},
		{
			name: "missing role name",
			config: SSOProviderConfig{
				Region:      "us-east-1",
				AccountID:   "123456789012",
				AccessToken: "test-token",
			},
			wantErr: true,
		},
		{
			name: "missing access token and provider",
			config: SSOProviderConfig{
				Region:    "us-east-1",
				AccountID: "123456789012",
				RoleName:  "MyRole",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewSSOProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSSOProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && provider == nil {
				t.Error("Expected non-nil provider")
			}
		})
	}
}

func TestNewSSOProviderFromEnv(t *testing.T) {
	// Save original env vars
	origRegion := os.Getenv("AWS_REGION")
	origAccountID := os.Getenv("AWS_SSO_ACCOUNT_ID")
	origRoleName := os.Getenv("AWS_SSO_ROLE_NAME")
	origAccessToken := os.Getenv("AWS_SSO_ACCESS_TOKEN")

	// Restore env vars after test
	defer func() {
		os.Setenv("AWS_REGION", origRegion)
		os.Setenv("AWS_SSO_ACCOUNT_ID", origAccountID)
		os.Setenv("AWS_SSO_ROLE_NAME", origRoleName)
		os.Setenv("AWS_SSO_ACCESS_TOKEN", origAccessToken)
	}()

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "valid environment variables",
			setup: func() {
				os.Setenv("AWS_REGION", "us-east-1")
				os.Setenv("AWS_SSO_ACCOUNT_ID", "123456789012")
				os.Setenv("AWS_SSO_ROLE_NAME", "MyRole")
				os.Setenv("AWS_SSO_ACCESS_TOKEN", "test-token")
			},
			wantErr: false,
		},
		{
			name: "missing account ID",
			setup: func() {
				os.Setenv("AWS_REGION", "us-east-1")
				os.Unsetenv("AWS_SSO_ACCOUNT_ID")
				os.Setenv("AWS_SSO_ROLE_NAME", "MyRole")
			},
			wantErr: true,
		},
		{
			name: "missing role name",
			setup: func() {
				os.Setenv("AWS_REGION", "us-east-1")
				os.Setenv("AWS_SSO_ACCOUNT_ID", "123456789012")
				os.Unsetenv("AWS_SSO_ROLE_NAME")
			},
			wantErr: true,
		},
		{
			name: "default region",
			setup: func() {
				os.Unsetenv("AWS_REGION")
				os.Unsetenv("AWS_DEFAULT_REGION")
				os.Setenv("AWS_SSO_ACCOUNT_ID", "123456789012")
				os.Setenv("AWS_SSO_ROLE_NAME", "MyRole")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			provider, err := NewSSOProviderFromEnv()
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSSOProviderFromEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && provider == nil {
				t.Error("Expected non-nil provider")
			}
		})
	}
}

func TestSSOProvider_SetAccessToken(t *testing.T) {
	provider, err := NewSSOProvider(SSOProviderConfig{
		Region:      "us-east-1",
		AccountID:   "123456789012",
		RoleName:    "MyRole",
		AccessToken: "initial-token",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Set cached credentials
	provider.cached = &Credentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		SessionToken:    "token",
		CanExpire:       true,
		Expires:         time.Now().Add(time.Hour),
		Source:          "SSOProvider",
	}

	// Verify cached credentials exist
	if provider.cached == nil {
		t.Error("Expected cached credentials")
	}

	// Set new access token
	provider.SetAccessToken("new-token")

	// Verify access token was updated
	if provider.accessToken != "new-token" {
		t.Errorf("Access token not updated: got %s, want new-token", provider.accessToken)
	}

	// Verify cached credentials were invalidated
	if provider.cached != nil {
		t.Error("Expected cached credentials to be invalidated")
	}
}

func TestSSOProvider_SetAccessTokenProvider(t *testing.T) {
	provider, err := NewSSOProvider(SSOProviderConfig{
		Region:      "us-east-1",
		AccountID:   "123456789012",
		RoleName:    "MyRole",
		AccessToken: "initial-token",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Set cached credentials
	provider.cached = &Credentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		SessionToken:    "token",
		CanExpire:       true,
		Expires:         time.Now().Add(time.Hour),
		Source:          "SSOProvider",
	}

	// Set new token provider
	provider.SetAccessTokenProvider(func() (string, error) {
		return "provider-token", nil
	})

	// Verify provider was updated
	if provider.accessTokenProvider == nil {
		t.Error("Expected token provider to be set")
	}

	// Verify cached credentials were invalidated
	if provider.cached != nil {
		t.Error("Expected cached credentials to be invalidated")
	}
}

func TestSSOProvider_IsExpired(t *testing.T) {
	provider, err := NewSSOProvider(SSOProviderConfig{
		Region:      "us-east-1",
		AccountID:   "123456789012",
		RoleName:    "MyRole",
		AccessToken: "test-token",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// No cached credentials - should be expired
	if !provider.IsExpired() {
		t.Error("Expected provider to be expired with no cached credentials")
	}

	// Set valid cached credentials
	provider.cached = &Credentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		SessionToken:    "token",
		CanExpire:       true,
		Expires:         time.Now().Add(time.Hour),
		Source:          "SSOProvider",
	}

	// Should not be expired
	if provider.IsExpired() {
		t.Error("Expected provider not to be expired with valid cached credentials")
	}

	// Set expired cached credentials
	provider.cached.Expires = time.Now().Add(-time.Hour)

	// Should be expired
	if !provider.IsExpired() {
		t.Error("Expected provider to be expired with expired cached credentials")
	}
}

func TestSSOProvider_Invalidate(t *testing.T) {
	provider, err := NewSSOProvider(SSOProviderConfig{
		Region:      "us-east-1",
		AccountID:   "123456789012",
		RoleName:    "MyRole",
		AccessToken: "test-token",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Set cached credentials
	provider.cached = &Credentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		SessionToken:    "token",
		CanExpire:       true,
		Expires:         time.Now().Add(time.Hour),
		Source:          "SSOProvider",
	}

	// Verify cached credentials exist
	if provider.cached == nil {
		t.Error("Expected cached credentials")
	}

	// Invalidate
	provider.Invalidate()

	// Verify cached credentials were cleared
	if provider.cached != nil {
		t.Error("Expected cached credentials to be nil after invalidation")
	}
}

// Note: Integration tests with real AWS SSO API should be run separately
// with proper AWS credentials and SSO configuration
func TestSSOProvider_Retrieve_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test requires:
	// - AWS_SSO_ACCOUNT_ID environment variable
	// - AWS_SSO_ROLE_NAME environment variable
	// - AWS_SSO_ACCESS_TOKEN environment variable
	// - Valid AWS SSO configuration

	accountID := os.Getenv("AWS_SSO_ACCOUNT_ID")
	roleName := os.Getenv("AWS_SSO_ROLE_NAME")
	accessToken := os.Getenv("AWS_SSO_ACCESS_TOKEN")

	if accountID == "" || roleName == "" || accessToken == "" {
		t.Skip("Skipping integration test: AWS_SSO_ACCOUNT_ID, AWS_SSO_ROLE_NAME, or AWS_SSO_ACCESS_TOKEN not set")
	}

	provider, err := NewSSOProvider(SSOProviderConfig{
		Region:      "us-east-1",
		AccountID:   accountID,
		RoleName:    roleName,
		AccessToken: accessToken,
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()
	creds, err := provider.Retrieve(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve credentials: %v", err)
	}

	if creds.AccessKeyID == "" {
		t.Error("Expected non-empty access key ID")
	}

	if creds.SecretAccessKey == "" {
		t.Error("Expected non-empty secret access key")
	}

	if creds.SessionToken == "" {
		t.Error("Expected non-empty session token")
	}

	if creds.IsExpired() {
		t.Error("Credentials should not be expired immediately after retrieval")
	}

	t.Logf("Successfully retrieved SSO credentials")
	t.Logf("Access Key ID: %s", creds.AccessKeyID[:10]+"...")
	t.Logf("Expires at: %s", creds.Expires)
}
