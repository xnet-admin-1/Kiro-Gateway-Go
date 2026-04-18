// Package credentials provides AWS credential provider implementations.
package credentials

import (
	"context"
	"fmt"
	"sync"
)

// Chain implements a credential provider chain that tries multiple providers
// in order until one succeeds. This follows the AWS SDK credential chain pattern.
//
// The standard AWS credential chain order is:
// 1. Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_SESSION_TOKEN)
// 2. AWS profile files (~/.aws/credentials and ~/.aws/config)
// 3. Web Identity Token (AWS_WEB_IDENTITY_TOKEN_FILE, AWS_ROLE_ARN)
// 4. ECS container credentials (AWS_CONTAINER_CREDENTIALS_RELATIVE_URI)
// 5. EC2 instance metadata (IMDS)
//
// Credentials are cached until they expire to avoid repeated provider calls.
type Chain struct {
	// providers is the list of credential providers to try in order
	providers []Provider

	// cached stores the most recently retrieved credentials
	cached *Credentials

	// mu protects concurrent access to cached credentials
	mu sync.RWMutex
}

// NewChain creates a new credential provider chain with the given providers.
// Providers are tried in the order they are provided.
func NewChain(providers ...Provider) *Chain {
	return &Chain{
		providers: providers,
	}
}

// Retrieve attempts to retrieve credentials from the provider chain.
// It tries each provider in order until one succeeds or all fail.
// Credentials are cached until they expire.
func (c *Chain) Retrieve(ctx context.Context) (*Credentials, error) {
	// Check if we have valid cached credentials
	c.mu.RLock()
	if c.cached != nil && !c.cached.IsExpired() {
		cached := c.cached.Copy()
		c.mu.RUnlock()
		return cached, nil
	}
	c.mu.RUnlock()

	// Try each provider in order
	var lastErr error
	for _, provider := range c.providers {
		creds, err := provider.Retrieve(ctx)
		if err != nil {
			lastErr = err
			continue
		}

		if creds != nil && !creds.IsExpired() && creds.AccessKeyID != "" && creds.SecretAccessKey != "" {
			// Cache the credentials
			c.mu.Lock()
			c.cached = creds.Copy()
			c.mu.Unlock()
			return creds, nil
		}
	}

	// All providers failed
	if lastErr != nil {
		return nil, fmt.Errorf("credential chain: all providers failed, last error: %w", lastErr)
	}
	return nil, fmt.Errorf("credential chain: no valid credentials found")
}

// IsExpired checks if the cached credentials are expired.
// Returns true if there are no cached credentials or they are expired.
func (c *Chain) IsExpired() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.cached == nil {
		return true
	}
	return c.cached.IsExpired()
}

// Invalidate clears the cached credentials, forcing the next Retrieve call
// to fetch fresh credentials from the provider chain.
func (c *Chain) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cached = nil
}

// AddProvider adds a provider to the end of the chain.
func (c *Chain) AddProvider(provider Provider) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.providers = append(c.providers, provider)
}

// NewDefaultChain creates a new credential chain with the standard AWS providers
// in the correct order: Environment → Profile → WebIdentity → ECS → IMDS.
func NewDefaultChain() *Chain {
	return NewChain(
		NewEnvProvider(),
		NewProfileProvider(),
		NewWebIdentityProvider(),
		NewECSProvider(),
		NewIMDSProvider(),
	)
}

// NewDefaultChainWithSSO creates a new credential chain that includes SSO provider
// The SSO provider is added at the beginning of the chain for priority
// Order: SSO → Environment → Profile → WebIdentity → ECS → IMDS
func NewDefaultChainWithSSO(ssoProvider *SSOProvider) *Chain {
	return NewChain(
		ssoProvider,
		NewEnvProvider(),
		NewProfileProvider(),
		NewWebIdentityProvider(),
		NewECSProvider(),
		NewIMDSProvider(),
	)
}
