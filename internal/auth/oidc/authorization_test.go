package oidc

import (
	"context"
	"testing"
	"time"
)

func TestStartDeviceAuthorization(t *testing.T) {
	mockClient := &mockOIDCClient{}
	
	client := &Client{
		region:  "us-east-1",
		ssooidc: mockClient,
	}

	reg := &DeviceRegistration{
		ClientID:              "test-client-id",
		ClientSecret:          "test-client-secret",
		ClientSecretExpiresAt: time.Now().Add(time.Hour),
		Region:                "us-east-1",
		StartURL:              "https://test.awsapps.com/start",
	}

	ctx := context.Background()
	startURL := "https://test.awsapps.com/start"

	auth, err := StartDeviceAuthorization(ctx, client, reg, startURL)
	if err != nil {
		t.Fatalf("StartDeviceAuthorization() error = %v", err)
	}

	if auth == nil {
		t.Fatal("StartDeviceAuthorization() returned nil authorization")
	}

	if auth.DeviceCode != "test-device-code" {
		t.Errorf("DeviceCode = %s, want test-device-code", auth.DeviceCode)
	}

	if auth.UserCode != "ABCD-1234" {
		t.Errorf("UserCode = %s, want ABCD-1234", auth.UserCode)
	}

	if auth.VerificationURI != "https://device.sso.us-east-1.amazonaws.com/" {
		t.Errorf("VerificationURI = %s, want https://device.sso.us-east-1.amazonaws.com/", auth.VerificationURI)
	}

	if auth.ExpiresIn != 600 {
		t.Errorf("ExpiresIn = %d, want 600", auth.ExpiresIn)
	}

	if auth.Interval != 5 {
		t.Errorf("Interval = %d, want 5", auth.Interval)
	}
}
