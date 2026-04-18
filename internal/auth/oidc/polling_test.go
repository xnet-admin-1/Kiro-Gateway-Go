package oidc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
)

func TestPollToken_Success(t *testing.T) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	token, err := PollToken(ctx, client, reg, "test-device-code", 1)
	if err != nil {
		t.Fatalf("PollToken() error = %v", err)
	}

	if token == nil {
		t.Fatal("PollToken() returned nil token")
	}

	if token.AccessToken != "test-access-token" {
		t.Errorf("AccessToken = %s, want test-access-token", token.AccessToken)
	}

	if token.RefreshToken != "test-refresh-token" {
		t.Errorf("RefreshToken = %s, want test-refresh-token", token.RefreshToken)
	}

	if token.Region != "us-east-1" {
		t.Errorf("Region = %s, want us-east-1", token.Region)
	}
}

func TestPollToken_AuthorizationPending(t *testing.T) {
	callCount := 0
	mockClient := &mockOIDCClient{
		createTokenFunc: func(ctx context.Context, params *ssooidc.CreateTokenInput, optFns ...func(*ssooidc.Options)) (*ssooidc.CreateTokenOutput, error) {
			callCount++
			if callCount < 3 {
				return nil, errors.New("authorization_pending")
			}
			return &ssooidc.CreateTokenOutput{
				AccessToken:  aws.String("test-access-token"),
				RefreshToken: aws.String("test-refresh-token"),
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	token, err := PollToken(ctx, client, reg, "test-device-code", 1)
	if err != nil {
		t.Fatalf("PollToken() error = %v", err)
	}

	if token == nil {
		t.Fatal("PollToken() returned nil token")
	}

	if callCount < 3 {
		t.Errorf("Expected at least 3 calls, got %d", callCount)
	}
}

func TestPollToken_ContextCanceled(t *testing.T) {
	mockClient := &mockOIDCClient{
		createTokenFunc: func(ctx context.Context, params *ssooidc.CreateTokenInput, optFns ...func(*ssooidc.Options)) (*ssooidc.CreateTokenOutput, error) {
			// Always return authorization_pending to force timeout
			return nil, errors.New("authorization_pending")
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

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := PollToken(ctx, client, reg, "test-device-code", 1)
	if err == nil {
		t.Fatal("PollToken() should have returned context deadline exceeded error")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}
}
