// Package credentials provides AWS credential provider implementations.
package credentials

import (
	"fmt"
	"time"
)

// NewCredentials creates a new Credentials instance with the provided values.
// This is a convenience function for creating static credentials.
func NewCredentials(accessKeyID, secretAccessKey, sessionToken, source string) *Credentials {
	return &Credentials{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		SessionToken:    sessionToken,
		Source:          source,
		CanExpire:       false,
		Expires:         time.Time{},
	}
}

// NewTemporaryCredentials creates a new Credentials instance with expiration.
// This is used for temporary credentials from STS, ECS, or IMDS.
func NewTemporaryCredentials(accessKeyID, secretAccessKey, sessionToken, source string, expires time.Time) *Credentials {
	return &Credentials{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		SessionToken:    sessionToken,
		Source:          source,
		CanExpire:       true,
		Expires:         expires,
	}
}

// String returns a string representation of the credentials (without exposing secrets).
// This is safe to use in logs and error messages.
func (c *Credentials) String() string {
	if c == nil {
		return "Credentials{nil}"
	}

	hasAccessKey := c.AccessKeyID != ""
	hasSecretKey := c.SecretAccessKey != ""
	hasSessionToken := c.SessionToken != ""

	return fmt.Sprintf("Credentials{Source: %s, HasAccessKey: %v, HasSecretKey: %v, HasSessionToken: %v, CanExpire: %v, Expires: %v}",
		c.Source, hasAccessKey, hasSecretKey, hasSessionToken, c.CanExpire, c.Expires)
}

// Copy creates a deep copy of the credentials.
func (c *Credentials) Copy() *Credentials {
	if c == nil {
		return nil
	}
	return &Credentials{
		AccessKeyID:     c.AccessKeyID,
		SecretAccessKey: c.SecretAccessKey,
		SessionToken:    c.SessionToken,
		Source:          c.Source,
		CanExpire:       c.CanExpire,
		Expires:         c.Expires,
	}
}
