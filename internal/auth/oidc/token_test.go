package oidc

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
)

func TestToken_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		token    *Token
		expected bool
	}{
		{
			name: "not expired",
			token: &Token{
				ExpiresAt: time.Now().Add(2 * time.Minute),
			},
			expected: false,
		},
		{
			name: "expired",
			token: &Token{
				ExpiresAt: time.Now().Add(-time.Hour),
			},
			expected: true,
		},
		{
			name: "expires within safety margin",
			token: &Token{
				ExpiresAt: time.Now().Add(30 * time.Second),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.token.IsExpired()
			if result != tt.expected {
				t.Errorf("IsExpired() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestToken_Refresh(t *testing.T) {
	mockClient := &mockOIDCClient{
		createTokenFunc: func(ctx context.Context, params *ssooidc.CreateTokenInput, optFns ...func(*ssooidc.Options)) (*ssooidc.CreateTokenOutput, error) {
			// Verify refresh token grant type
			if *params.GrantType != "refresh_token" {
				t.Errorf("Expected grant_type refresh_token, got %s", *params.GrantType)
			}
			
			return &ssooidc.CreateTokenOutput{
				AccessToken:  aws.String("new-access-token"),
				RefreshToken: aws.String("new-refresh-token"),
				ExpiresIn:    3600,
			}, nil
		},
	}
	
	client := &Client{
		ssooidc: mockClient,
		region:  "us-east-1",
	}

	reg := &DeviceRegistration{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Region:       "us-east-1",
		StartURL:     "https://test.awsapps.com/start",
	}

	token := &Token{
		AccessToken:  "old-access-token",
		RefreshToken: "old-refresh-token",
		ExpiresAt:    time.Now().Add(-time.Hour),
		Region:       "us-east-1",
		StartURL:     "https://test.awsapps.com/start",
	}

	ctx := context.Background()
	newToken, err := token.Refresh(ctx, client, reg)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	if newToken == nil {
		t.Fatal("Refresh() returned nil token")
	}

	if newToken.AccessToken != "new-access-token" {
		t.Errorf("AccessToken = %s, want new-access-token", newToken.AccessToken)
	}

	if newToken.RefreshToken != "new-refresh-token" {
		t.Errorf("RefreshToken = %s, want new-refresh-token", newToken.RefreshToken)
	}

	if newToken.Region != token.Region {
		t.Errorf("Region = %s, want %s", newToken.Region, token.Region)
	}

	if newToken.StartURL != token.StartURL {
		t.Errorf("StartURL = %s, want %s", newToken.StartURL, token.StartURL)
	}
}

func TestToken_RefreshWithoutRefreshToken(t *testing.T) {
	mockClient := &mockOIDCClient{}
	
	client := &Client{
		ssooidc: mockClient,
		region:  "us-east-1",
	}

	reg := &DeviceRegistration{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Region:       "us-east-1",
		StartURL:     "https://test.awsapps.com/start",
	}

	token := &Token{
		AccessToken:  "access-token",
		RefreshToken: "", // No refresh token
		ExpiresAt:    time.Now().Add(-time.Hour),
		Region:       "us-east-1",
		StartURL:     "https://test.awsapps.com/start",
	}

	ctx := context.Background()
	_, err := token.Refresh(ctx, client, reg)
	if err == nil {
		t.Fatal("Refresh() should have returned error for missing refresh token")
	}

	expectedError := "no refresh token available"
	if err.Error() != expectedError {
		t.Errorf("Expected error %q, got %q", expectedError, err.Error())
	}
}

func TestToken_SaveAndLoad(t *testing.T) {
	token := &Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
		Region:       "us-east-1",
		StartURL:     "https://test.awsapps.com/start",
	}

	// Save token
	err := token.Save()
	if err != nil {
		t.Skipf("Skipping keychain test (keychain not available): %v", err)
	}

	// Load token
	loaded, err := LoadTokenFromStore(token.Region, token.StartURL)
	if err != nil {
		t.Fatalf("LoadTokenFromStore() error = %v", err)
	}

	if loaded.AccessToken != token.AccessToken {
		t.Errorf("Loaded AccessToken = %s, want %s", loaded.AccessToken, token.AccessToken)
	}

	if loaded.RefreshToken != token.RefreshToken {
		t.Errorf("Loaded RefreshToken = %s, want %s", loaded.RefreshToken, token.RefreshToken)
	}

	// Clean up
	_ = token.Delete()
}
