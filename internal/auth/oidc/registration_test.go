package oidc

import (
	"context"
	"testing"
	"time"
)

func TestDeviceRegistration_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		reg      *DeviceRegistration
		expected bool
	}{
		{
			name: "not expired",
			reg: &DeviceRegistration{
				ClientSecretExpiresAt: time.Now().Add(time.Hour),
			},
			expected: false,
		},
		{
			name: "expired",
			reg: &DeviceRegistration{
				ClientSecretExpiresAt: time.Now().Add(-time.Hour),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.reg.IsExpired()
			if result != tt.expected {
				t.Errorf("IsExpired() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRegister(t *testing.T) {
	// Create a mock client
	mockClient := &mockOIDCClient{}
	
	client := &Client{
		ssooidc: mockClient,
		region:  "us-east-1",
	}

	ctx := context.Background()
	startURL := "https://test.awsapps.com/start"

	reg, err := Register(ctx, client, startURL)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if reg == nil {
		t.Fatal("Register() returned nil registration")
	}

	if reg.ClientID != "test-client-id" {
		t.Errorf("ClientID = %s, want test-client-id", reg.ClientID)
	}

	if reg.ClientSecret != "test-client-secret" {
		t.Errorf("ClientSecret = %s, want test-client-secret", reg.ClientSecret)
	}

	if reg.Region != "us-east-1" {
		t.Errorf("Region = %s, want us-east-1", reg.Region)
	}

	if reg.StartURL != startURL {
		t.Errorf("StartURL = %s, want %s", reg.StartURL, startURL)
	}
}

func TestDeviceRegistration_SaveAndLoad(t *testing.T) {
	reg := &DeviceRegistration{
		ClientID:              "test-client-id",
		ClientSecret:          "test-client-secret",
		ClientSecretExpiresAt: time.Now().Add(time.Hour),
		Region:                "us-east-1",
		StartURL:              "https://test.awsapps.com/start",
	}

	// Save registration
	err := reg.Save()
	if err != nil {
		t.Skipf("Skipping keychain test (keychain not available): %v", err)
	}

	// Load registration
	loaded, err := LoadRegistrationFromStore(reg.Region, reg.StartURL)
	if err != nil {
		t.Fatalf("LoadRegistrationFromStore() error = %v", err)
	}

	if loaded.ClientID != reg.ClientID {
		t.Errorf("Loaded ClientID = %s, want %s", loaded.ClientID, reg.ClientID)
	}

	if loaded.ClientSecret != reg.ClientSecret {
		t.Errorf("Loaded ClientSecret = %s, want %s", loaded.ClientSecret, reg.ClientSecret)
	}

	// Clean up
	_ = reg.Delete()
}
