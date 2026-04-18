package oidc

import (
	"testing"
	"time"
)

func TestClientRegistration_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		reg      *ClientRegistration
		expected bool
	}{
		{
			name: "not expired",
			reg: &ClientRegistration{
				ClientID:              "test-client",
				ClientSecret:          "test-secret",
				ClientSecretExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
				ExpiresAt:             time.Now().Add(1 * time.Hour),
			},
			expected: false,
		},
		{
			name: "expired",
			reg: &ClientRegistration{
				ClientID:              "test-client",
				ClientSecret:          "test-secret",
				ClientSecretExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
				ExpiresAt:             time.Now().Add(-1 * time.Hour),
			},
			expected: true,
		},
		{
			name: "no expiry",
			reg: &ClientRegistration{
				ClientID:              "test-client",
				ClientSecret:          "test-secret",
				ClientSecretExpiresAt: 0,
			},
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.reg.IsExpired(); got != tt.expected {
				t.Errorf("IsExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestOIDCError_String(t *testing.T) {
	tests := []struct {
		name     string
		err      *OIDCError
		expected string
	}{
		{
			name: "with description",
			err: &OIDCError{
				Error:            "invalid_grant",
				ErrorDescription: "The refresh token is invalid",
			},
			expected: "invalid_grant: The refresh token is invalid",
		},
		{
			name: "without description",
			err: &OIDCError{
				Error: "authorization_pending",
			},
			expected: "authorization_pending",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.String(); got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}
