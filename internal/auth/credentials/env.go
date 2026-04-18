package credentials

import (
	"context"
	"errors"
	"os"
)

// EnvProvider retrieves credentials from environment variables.
// It reads AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, and AWS_SESSION_TOKEN.
type EnvProvider struct{}

// NewEnvProvider creates a new environment variable credential provider.
func NewEnvProvider() *EnvProvider {
	return &EnvProvider{}
}

// Retrieve retrieves credentials from environment variables.
func (p *EnvProvider) Retrieve(ctx context.Context) (*Credentials, error) {
	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	sessionToken := os.Getenv("AWS_SESSION_TOKEN")

	if accessKeyID == "" {
		return nil, errors.New("AWS_ACCESS_KEY_ID not found in environment")
	}

	if secretAccessKey == "" {
		return nil, errors.New("AWS_SECRET_ACCESS_KEY not found in environment")
	}

	return NewCredentials(accessKeyID, secretAccessKey, sessionToken, "environment"), nil
}

// IsExpired returns false as environment credentials are static.
func (p *EnvProvider) IsExpired() bool {
	return false
}
