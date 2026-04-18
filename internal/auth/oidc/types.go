package oidc

import "time"

// DeviceAuthResponse contains the response from StartDeviceAuthorization
type DeviceAuthResponse struct {
	DeviceCode              string `json:"deviceCode"`
	UserCode                string `json:"userCode"`
	VerificationUri         string `json:"verificationUri"`
	VerificationUriComplete string `json:"verificationUriComplete"`
	ExpiresIn               int    `json:"expiresIn"`
	Interval                int    `json:"interval"`
}

// TokenResponse contains the response from CreateToken
type TokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken,omitempty"`
	IdToken      string `json:"idToken,omitempty"`
	TokenType    string `json:"tokenType"`
	ExpiresIn    int    `json:"expiresIn"`
}

// ClientRegistration contains client registration details
type ClientRegistration struct {
	ClientID              string    `json:"clientId"`
	ClientSecret          string    `json:"clientSecret"`
	ClientIDIssuedAt      int64     `json:"clientIdIssuedAt"`
	ClientSecretExpiresAt int64     `json:"clientSecretExpiresAt"`
	ExpiresAt             time.Time `json:"-"` // Computed from ClientSecretExpiresAt
}

// IsExpired checks if the client registration has expired
func (c *ClientRegistration) IsExpired() bool {
	if c.ClientSecretExpiresAt == 0 {
		return false // No expiry
	}
	return time.Now().After(c.ExpiresAt)
}

// GrantType constants for CreateToken
const (
	GrantTypeDeviceCode      = "urn:ietf:params:oauth:grant-type:device_code"
	GrantTypeRefreshToken    = "refresh_token"
	GrantTypeAuthorizationCode = "authorization_code"
)

// OIDC error codes
const (
	ErrAuthorizationPending = "authorization_pending"
	ErrSlowDown            = "slow_down"
	ErrExpiredToken        = "expired_token"
	ErrAccessDenied        = "access_denied"
	ErrInvalidGrant        = "invalid_grant"
	ErrInvalidClient       = "invalid_client"
)

// OIDCError represents an error from the OIDC service
type OIDCError struct {
	ErrorCode        string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// Error implements the error interface
func (e *OIDCError) Error() string {
	if e.ErrorDescription != "" {
		return e.ErrorCode + ": " + e.ErrorDescription
	}
	return e.ErrorCode
}

func (e *OIDCError) String() string {
	return e.Error()
}
