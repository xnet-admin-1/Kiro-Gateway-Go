package credentials

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// STSClient interface for STS operations (allows mocking)
type STSClient interface {
	AssumeRoleWithWebIdentity(ctx context.Context, params *sts.AssumeRoleWithWebIdentityInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleWithWebIdentityOutput, error)
}

// WebIdentityProvider retrieves credentials using web identity token.
// It reads AWS_WEB_IDENTITY_TOKEN_FILE and AWS_ROLE_ARN environment variables.
type WebIdentityProvider struct {
	tokenFile   string
	roleARN     string
	sessionName string
	stsClient   STSClient
}

// NewWebIdentityProvider creates a new web identity token credential provider.
func NewWebIdentityProvider() *WebIdentityProvider {
	tokenFile := os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE")
	roleARN := os.Getenv("AWS_ROLE_ARN")
	sessionName := os.Getenv("AWS_ROLE_SESSION_NAME")
	if sessionName == "" {
		sessionName = "kiro-gateway-session"
	}

	return &WebIdentityProvider{
		tokenFile:   tokenFile,
		roleARN:     roleARN,
		sessionName: sessionName,
	}
}

// Retrieve retrieves credentials using web identity token.
func (p *WebIdentityProvider) Retrieve(ctx context.Context) (*Credentials, error) {
	if p.tokenFile == "" {
		return nil, fmt.Errorf("AWS_WEB_IDENTITY_TOKEN_FILE not set")
	}

	if p.roleARN == "" {
		return nil, fmt.Errorf("AWS_ROLE_ARN not set")
	}

	// Read token from file
	tokenBytes, err := os.ReadFile(p.tokenFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read web identity token file: %w", err)
	}
	token := string(tokenBytes)

	// Initialize STS client if not already done
	if p.stsClient == nil {
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}
		p.stsClient = sts.NewFromConfig(cfg)
	}

	// Call AssumeRoleWithWebIdentity
	input := &sts.AssumeRoleWithWebIdentityInput{
		RoleArn:          aws.String(p.roleARN),
		RoleSessionName:  aws.String(p.sessionName),
		WebIdentityToken: aws.String(token),
	}

	result, err := p.stsClient.AssumeRoleWithWebIdentity(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to assume role with web identity: %w", err)
	}

	if result.Credentials == nil {
		return nil, fmt.Errorf("no credentials returned from STS")
	}

	return NewTemporaryCredentials(
		*result.Credentials.AccessKeyId,
		*result.Credentials.SecretAccessKey,
		*result.Credentials.SessionToken,
		"web-identity",
		*result.Credentials.Expiration,
	), nil
}

// IsExpired returns false as this provider doesn't cache credentials.
func (p *WebIdentityProvider) IsExpired() bool {
	return false
}
