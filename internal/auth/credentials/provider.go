// Package credentials provides AWS credential providers for the credential chain.
// It implements the standard AWS credential provider interface with support for
// environment variables, profiles, web identity tokens, ECS, and IMDS.
package credentials

import (
    "context"
    "time"
)

// Provider represents a credential provider that can retrieve AWS credentials.
// Providers are used in a chain to resolve credentials from multiple sources.
type Provider interface {
    // Retrieve returns AWS credentials or an error if credentials cannot be obtained.
    // The context can be used to cancel the operation.
    Retrieve(ctx context.Context) (*Credentials, error)
    
    // IsExpired returns true if the provider's cached credentials are expired.
    // This is used to determine if credentials need to be refreshed.
    IsExpired() bool
}

// Credentials represents AWS credentials with optional expiration.
// These credentials can be used for AWS API authentication.
type Credentials struct {
    // AccessKeyID is the AWS access key ID
    AccessKeyID     string
    
    // SecretAccessKey is the AWS secret access key
    SecretAccessKey string
    
    // SessionToken is the AWS session token (for temporary credentials)
    SessionToken    string
    
    // Source identifies which provider supplied these credentials
    Source          string
    
    // CanExpire indicates if these credentials can expire
    CanExpire       bool
    
    // Expires is the expiration time for temporary credentials
    Expires         time.Time
}

// IsExpired returns true if the credentials are expired.
// Static credentials (CanExpire=false) never expire.
func (c *Credentials) IsExpired() bool {
    return c.CanExpire && time.Now().After(c.Expires)
}
