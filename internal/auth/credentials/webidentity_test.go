package credentials

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
)

// mockSTSClient implements a mock STS client for testing
type mockSTSClient struct {
	assumeRoleWithWebIdentityFunc func(ctx context.Context, params *sts.AssumeRoleWithWebIdentityInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleWithWebIdentityOutput, error)
}

func (m *mockSTSClient) AssumeRoleWithWebIdentity(ctx context.Context, params *sts.AssumeRoleWithWebIdentityInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleWithWebIdentityOutput, error) {
	if m.assumeRoleWithWebIdentityFunc != nil {
		return m.assumeRoleWithWebIdentityFunc(ctx, params, optFns...)
	}
	
	// Default successful response
	expiration := time.Now().Add(time.Hour)
	return &sts.AssumeRoleWithWebIdentityOutput{
		Credentials: &types.Credentials{
			AccessKeyId:     aws.String("AKIAIOSFODNN7EXAMPLE"),
			SecretAccessKey: aws.String("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"),
			SessionToken:    aws.String("session-token"),
			Expiration:      &expiration,
		},
	}, nil
}

func TestWebIdentityProvider_Retrieve(t *testing.T) {
	// Create temporary token file
	tempDir, err := os.MkdirTemp("", "web-identity-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tokenFile := filepath.Join(tempDir, "token")
	tokenContent := "test-web-identity-token"
	if err := os.WriteFile(tokenFile, []byte(tokenContent), 0644); err != nil {
		t.Fatalf("Failed to write token file: %v", err)
	}

	tests := []struct {
		name        string
		setupEnv    func()
		cleanupEnv  func()
		mockSTS     *mockSTSClient
		wantErr     bool
		wantSource  string
	}{
		{
			name: "successful web identity token exchange",
			setupEnv: func() {
				os.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", tokenFile)
				os.Setenv("AWS_ROLE_ARN", "arn:aws:iam::123456789012:role/test-role")
				os.Setenv("AWS_ROLE_SESSION_NAME", "test-session")
			},
			cleanupEnv: func() {
				os.Unsetenv("AWS_WEB_IDENTITY_TOKEN_FILE")
				os.Unsetenv("AWS_ROLE_ARN")
				os.Unsetenv("AWS_ROLE_SESSION_NAME")
			},
			mockSTS: &mockSTSClient{},
			wantErr: false,
			wantSource: "web-identity",
		},
		{
			name: "missing token file environment variable",
			setupEnv: func() {
				os.Setenv("AWS_ROLE_ARN", "arn:aws:iam::123456789012:role/test-role")
			},
			cleanupEnv: func() {
				os.Unsetenv("AWS_ROLE_ARN")
			},
			mockSTS: &mockSTSClient{},
			wantErr: true,
		},
		{
			name: "missing role ARN environment variable",
			setupEnv: func() {
				os.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", tokenFile)
			},
			cleanupEnv: func() {
				os.Unsetenv("AWS_WEB_IDENTITY_TOKEN_FILE")
			},
			mockSTS: &mockSTSClient{},
			wantErr: true,
		},
		{
			name: "token file does not exist",
			setupEnv: func() {
				os.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", "/nonexistent/token")
				os.Setenv("AWS_ROLE_ARN", "arn:aws:iam::123456789012:role/test-role")
			},
			cleanupEnv: func() {
				os.Unsetenv("AWS_WEB_IDENTITY_TOKEN_FILE")
				os.Unsetenv("AWS_ROLE_ARN")
			},
			mockSTS: &mockSTSClient{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer tt.cleanupEnv()

			provider := NewWebIdentityProvider()
			// Inject mock STS client
			provider.stsClient = tt.mockSTS

			creds, err := provider.Retrieve(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("WebIdentityProvider.Retrieve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if creds == nil {
					t.Error("WebIdentityProvider.Retrieve() returned nil credentials")
					return
				}

				if creds.Source != tt.wantSource {
					t.Errorf("WebIdentityProvider.Retrieve() source = %v, want %v", creds.Source, tt.wantSource)
				}

				if creds.AccessKeyID == "" {
					t.Error("WebIdentityProvider.Retrieve() returned empty AccessKeyID")
				}

				if creds.SecretAccessKey == "" {
					t.Error("WebIdentityProvider.Retrieve() returned empty SecretAccessKey")
				}

				if creds.SessionToken == "" {
					t.Error("WebIdentityProvider.Retrieve() returned empty SessionToken")
				}

				if !creds.CanExpire {
					t.Error("WebIdentityProvider.Retrieve() credentials should expire")
				}

				if creds.Expires.IsZero() {
					t.Error("WebIdentityProvider.Retrieve() credentials should have expiration time")
				}
			}
		})
	}
}

func TestWebIdentityProvider_IsExpired(t *testing.T) {
	provider := NewWebIdentityProvider()
	if provider.IsExpired() {
		t.Error("WebIdentityProvider.IsExpired() should always return false")
	}
}
