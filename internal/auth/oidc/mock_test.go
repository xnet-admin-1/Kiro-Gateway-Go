package oidc

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
)

// mockOIDCClient implements OIDCClientInterface for testing
type mockOIDCClient struct {
	registerClientFunc            func(ctx context.Context, params *ssooidc.RegisterClientInput, optFns ...func(*ssooidc.Options)) (*ssooidc.RegisterClientOutput, error)
	createTokenFunc               func(ctx context.Context, params *ssooidc.CreateTokenInput, optFns ...func(*ssooidc.Options)) (*ssooidc.CreateTokenOutput, error)
	startDeviceAuthorizationFunc  func(ctx context.Context, params *ssooidc.StartDeviceAuthorizationInput, optFns ...func(*ssooidc.Options)) (*ssooidc.StartDeviceAuthorizationOutput, error)
}

func (m *mockOIDCClient) RegisterClient(ctx context.Context, params *ssooidc.RegisterClientInput, optFns ...func(*ssooidc.Options)) (*ssooidc.RegisterClientOutput, error) {
	if m.registerClientFunc != nil {
		return m.registerClientFunc(ctx, params, optFns...)
	}
	
	// Default mock response
	return &ssooidc.RegisterClientOutput{
		ClientId:              aws.String("test-client-id"),
		ClientSecret:          aws.String("test-client-secret"),
		ClientSecretExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}, nil
}

func (m *mockOIDCClient) CreateToken(ctx context.Context, params *ssooidc.CreateTokenInput, optFns ...func(*ssooidc.Options)) (*ssooidc.CreateTokenOutput, error) {
	if m.createTokenFunc != nil {
		return m.createTokenFunc(ctx, params, optFns...)
	}
	
	// Default mock response
	return &ssooidc.CreateTokenOutput{
		AccessToken:  aws.String("test-access-token"),
		RefreshToken: aws.String("test-refresh-token"),
		ExpiresIn:    3600,
		TokenType:    aws.String("Bearer"),
	}, nil
}

func (m *mockOIDCClient) StartDeviceAuthorization(ctx context.Context, params *ssooidc.StartDeviceAuthorizationInput, optFns ...func(*ssooidc.Options)) (*ssooidc.StartDeviceAuthorizationOutput, error) {
	if m.startDeviceAuthorizationFunc != nil {
		return m.startDeviceAuthorizationFunc(ctx, params, optFns...)
	}
	
	// Default mock response
	return &ssooidc.StartDeviceAuthorizationOutput{
		DeviceCode:              aws.String("test-device-code"),
		UserCode:                aws.String("ABCD-1234"),
		VerificationUri:         aws.String("https://device.sso.us-east-1.amazonaws.com/"),
		VerificationUriComplete: aws.String("https://device.sso.us-east-1.amazonaws.com/?user_code=TEST-CODE"),
		ExpiresIn:               600,
		Interval:                5,
	}, nil
}
