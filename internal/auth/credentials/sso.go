package credentials

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso"
)

// SSOProvider retrieves temporary IAM credentials from AWS SSO using a bearer token.
// This enables SigV4 signing with SSO-derived credentials for Q Developer mode.
//
// The provider:
// 1. Uses an SSO access token (bearer token from Identity Center)
// 2. Calls SSO GetRoleCredentials to get temporary IAM credentials
// 3. Returns credentials that can be used for SigV4 signing
//
// This is different from bearer token authentication - it converts the SSO token
// into IAM credentials for use with SigV4.
type SSOProvider struct {
	// region is the AWS region for SSO
	region string

	// accountID is the AWS account ID
	accountID string

	// roleName is the SSO role name
	roleName string

	// accessToken is the SSO access token (bearer token)
	accessToken string

	// accessTokenProvider is a function that returns the current access token
	// This allows dynamic token retrieval (e.g., from token refresh)
	accessTokenProvider func() (string, error)

	// ssoClient is the AWS SSO client
	ssoClient *sso.Client

	// cached stores the most recently retrieved credentials
	cached *Credentials
}

// SSOProviderConfig holds configuration for SSO credential provider
type SSOProviderConfig struct {
	// Region is the AWS region (required)
	Region string

	// AccountID is the AWS account ID (required)
	AccountID string

	// RoleName is the SSO role name (required)
	RoleName string

	// AccessToken is the SSO access token (optional if AccessTokenProvider is set)
	AccessToken string

	// AccessTokenProvider is a function that returns the current access token (optional)
	// Use this for dynamic token retrieval (e.g., from token refresh)
	AccessTokenProvider func() (string, error)
}

// NewSSOProvider creates a new SSO credential provider
func NewSSOProvider(cfg SSOProviderConfig) (*SSOProvider, error) {
	if cfg.Region == "" {
		return nil, fmt.Errorf("region is required")
	}
	if cfg.AccountID == "" {
		return nil, fmt.Errorf("account ID is required")
	}
	if cfg.RoleName == "" {
		return nil, fmt.Errorf("role name is required")
	}
	if cfg.AccessToken == "" && cfg.AccessTokenProvider == nil {
		return nil, fmt.Errorf("either access token or access token provider is required")
	}

	// Create AWS config for SSO client
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &SSOProvider{
		region:              cfg.Region,
		accountID:           cfg.AccountID,
		roleName:            cfg.RoleName,
		accessToken:         cfg.AccessToken,
		accessTokenProvider: cfg.AccessTokenProvider,
		ssoClient:           sso.NewFromConfig(awsCfg),
	}, nil
}

// NewSSOProviderFromEnv creates an SSO provider from environment variables
func NewSSOProviderFromEnv() (*SSOProvider, error) {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = os.Getenv("AWS_DEFAULT_REGION")
	}
	if region == "" {
		region = "us-east-1"
	}

	accountID := os.Getenv("AWS_SSO_ACCOUNT_ID")
	roleName := os.Getenv("AWS_SSO_ROLE_NAME")
	accessToken := os.Getenv("AWS_SSO_ACCESS_TOKEN")

	if accountID == "" || roleName == "" {
		return nil, fmt.Errorf("AWS_SSO_ACCOUNT_ID and AWS_SSO_ROLE_NAME environment variables are required")
	}

	return NewSSOProvider(SSOProviderConfig{
		Region:      region,
		AccountID:   accountID,
		RoleName:    roleName,
		AccessToken: accessToken,
	})
}

// Retrieve retrieves temporary IAM credentials from AWS SSO
func (p *SSOProvider) Retrieve(ctx context.Context) (*Credentials, error) {
	// Check if we have valid cached credentials
	if p.cached != nil && !p.cached.IsExpired() {
		fmt.Printf("[DEBUG] Using cached SSO credentials (expires in %v)\n", time.Until(p.cached.Expires))
		return p.cached.Copy(), nil
	}

	fmt.Println("[DEBUG] Retrieving fresh SSO credentials...")

	// Get access token
	accessToken := p.accessToken
	if p.accessTokenProvider != nil {
		fmt.Println("[DEBUG] Getting access token from provider...")
		token, err := p.accessTokenProvider()
		if err != nil {
			return nil, fmt.Errorf("failed to get access token: %w", err)
		}
		accessToken = token
		fmt.Printf("[DEBUG] Got access token from provider (length: %d)\n", len(accessToken))
	}

	if accessToken == "" {
		return nil, fmt.Errorf("no access token available")
	}

	fmt.Printf("[DEBUG] Calling SSO GetRoleCredentials API (account: %s, role: %s, region: %s)\n", 
		p.accountID, p.roleName, p.region)

	// Call SSO GetRoleCredentials
	input := &sso.GetRoleCredentialsInput{
		AccessToken: aws.String(accessToken),
		AccountId:   aws.String(p.accountID),
		RoleName:    aws.String(p.roleName),
	}

	output, err := p.ssoClient.GetRoleCredentials(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get SSO role credentials: %w", err)
	}

	if output.RoleCredentials == nil {
		return nil, fmt.Errorf("no role credentials returned from SSO")
	}

	fmt.Println("[DEBUG] Successfully retrieved SSO credentials")

	// Convert to our Credentials type
	creds := &Credentials{
		AccessKeyID:     aws.ToString(output.RoleCredentials.AccessKeyId),
		SecretAccessKey: aws.ToString(output.RoleCredentials.SecretAccessKey),
		SessionToken:    aws.ToString(output.RoleCredentials.SessionToken),
		Source:          "SSOProvider",
		CanExpire:       true,
	}

	// Set expiration time
	if output.RoleCredentials.Expiration != 0 {
		creds.Expires = time.UnixMilli(output.RoleCredentials.Expiration)
		fmt.Printf("[DEBUG] SSO credentials expire at: %s (in %v)\n", 
			creds.Expires.Format(time.RFC3339), time.Until(creds.Expires))
	} else {
		// Default to 1 hour if no expiration provided
		creds.Expires = time.Now().Add(time.Hour)
		fmt.Println("[DEBUG] No expiration provided, defaulting to 1 hour")
	}

	// Cache the credentials
	p.cached = creds.Copy()

	return creds, nil
}

// IsExpired checks if the cached credentials are expired
func (p *SSOProvider) IsExpired() bool {
	if p.cached == nil {
		return true
	}
	return p.cached.IsExpired()
}

// Invalidate clears the cached credentials
func (p *SSOProvider) Invalidate() {
	p.cached = nil
}

// SetAccessToken updates the access token
func (p *SSOProvider) SetAccessToken(token string) {
	p.accessToken = token
	// Invalidate cached credentials when token changes
	p.Invalidate()
}

// SetAccessTokenProvider updates the access token provider
func (p *SSOProvider) SetAccessTokenProvider(provider func() (string, error)) {
	p.accessTokenProvider = provider
	// Invalidate cached credentials when provider changes
	p.Invalidate()
}

// ParseProfileARN extracts account ID and role name from a Q Developer profile ARN
// Format: arn:aws:codewhisperer:REGION:ACCOUNT_ID:profile/PROFILE_ID
// or: arn:aws:sso:::account/ACCOUNT_ID/role/ROLE_NAME
func ParseProfileARN(profileARN string) (accountID, roleName string, err error) {
	// This is a simplified parser - you may need to adjust based on actual ARN format
	// For Q Developer, the profile ARN doesn't directly contain role info
	// You'll need to get this from SSO configuration or environment variables
	return "", "", fmt.Errorf("profile ARN parsing not yet implemented - use AWS_SSO_ACCOUNT_ID and AWS_SSO_ROLE_NAME")
}

// GetSSOCredentialsFromToken is a helper function to get SSO credentials from a bearer token
// This is useful for converting bearer token auth to SigV4 auth
func GetSSOCredentialsFromToken(ctx context.Context, region, accountID, roleName, accessToken string) (*Credentials, error) {
	provider, err := NewSSOProvider(SSOProviderConfig{
		Region:      region,
		AccountID:   accountID,
		RoleName:    roleName,
		AccessToken: accessToken,
	})
	if err != nil {
		return nil, err
	}

	return provider.Retrieve(ctx)
}
