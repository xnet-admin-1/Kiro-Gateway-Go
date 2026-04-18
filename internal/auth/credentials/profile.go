package credentials

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
)

// ProfileProvider retrieves credentials from AWS profile files.
// It uses the AWS SDK's config loader which supports SSO profiles.
type ProfileProvider struct {
	profile string
}

// NewProfileProvider creates a new profile credential provider.
// If profile is empty, it uses the AWS_PROFILE environment variable or "default".
func NewProfileProvider() *ProfileProvider {
	profile := os.Getenv("AWS_PROFILE")
	if profile == "" {
		profile = "default"
	}
	return &ProfileProvider{profile: profile}
}

// NewProfileProviderWithProfile creates a new profile provider with a specific profile name.
func NewProfileProviderWithProfile(profile string) *ProfileProvider {
	return &ProfileProvider{profile: profile}
}

// Retrieve retrieves credentials from AWS profile files using AWS SDK.
// This supports static credentials, SSO profiles, and assume role configurations.
func (p *ProfileProvider) Retrieve(ctx context.Context) (*Credentials, error) {
	// Log the profile being used
	fmt.Printf("ProfileProvider: Loading credentials for profile '%s'\n", p.profile)
	
	// Use AWS SDK's config loader which handles SSO, assume role, etc.
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile(p.profile),
	)
	if err != nil {
		fmt.Printf("ProfileProvider: Failed to load config: %v\n", err)
		return nil, fmt.Errorf("failed to load AWS config for profile %s: %w", p.profile, err)
	}

	fmt.Printf("ProfileProvider: Config loaded, retrieving credentials...\n")
	
	// Retrieve credentials from the config
	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		fmt.Printf("ProfileProvider: Failed to retrieve credentials: %v\n", err)
		return nil, fmt.Errorf("failed to retrieve credentials for profile %s: %w", p.profile, err)
	}

	fmt.Printf("ProfileProvider: Successfully retrieved credentials (AccessKeyID: %s...)\n", creds.AccessKeyID[:10])
	

	// Check if credentials are temporary (have expiration)
	if creds.CanExpire {
		return NewTemporaryCredentials(
			creds.AccessKeyID,
			creds.SecretAccessKey,
			creds.SessionToken,
			"profile",
			creds.Expires,
		), nil
	}

	// Static credentials
	return NewCredentials(
		creds.AccessKeyID,
		creds.SecretAccessKey,
		creds.SessionToken,
		"profile",
	), nil
}

// IsExpired returns false as profile credentials are managed by AWS SDK.
func (p *ProfileProvider) IsExpired() bool {
	return false
}
