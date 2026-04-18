package oidc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Client handles SSO OIDC operations
type Client struct {
	region       string
	startURL     string
	httpClient   *http.Client
	
	// Cached client registration
	registration *ClientRegistration
}

// NewClient creates a new OIDC client
func NewClient(region, startURL string) *Client {
	return &Client{
		region:   region,
		startURL: startURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetClientRegistration sets a pre-registered client
func (c *Client) SetClientRegistration(clientID, clientSecret string, expiresAt int64) {
	c.registration = &ClientRegistration{
		ClientID:              clientID,
		ClientSecret:          clientSecret,
		ClientSecretExpiresAt: expiresAt,
		ExpiresAt:             time.Unix(expiresAt, 0),
	}
}

// RegisterClient registers the application with IAM Identity Center
func (c *Client) RegisterClient(ctx context.Context) (*ClientRegistration, error) {
	// Check if we have a valid cached registration
	if c.registration != nil && !c.registration.IsExpired() {
		return c.registration, nil
	}
	
	endpoint := fmt.Sprintf("https://oidc.%s.amazonaws.com/client/register", c.region)
	
	reqBody := map[string]interface{}{
		"clientName": "kiro-gateway",
		"clientType": "public",
		"scopes":     []string{"sso:account:access", "codewhisperer:completions", "codewhisperer:analysis", "codewhisperer:conversations"},
	}
	
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to register client: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		var oidcErr OIDCError
		if err := json.Unmarshal(body, &oidcErr); err == nil {
			return nil, fmt.Errorf("OIDC error: %s", oidcErr.String())
		}
		return nil, fmt.Errorf("registration failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var registration ClientRegistration
	if err := json.Unmarshal(body, &registration); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	// Compute expiry time
	if registration.ClientSecretExpiresAt > 0 {
		registration.ExpiresAt = time.Unix(registration.ClientSecretExpiresAt, 0)
	}
	
	c.registration = &registration
	return &registration, nil
}

// StartDeviceAuthorization initiates the device authorization flow
func (c *Client) StartDeviceAuthorization(ctx context.Context) (*DeviceAuthResponse, error) {
	// Ensure we have a client registration
	if c.registration == nil || c.registration.IsExpired() {
		if _, err := c.RegisterClient(ctx); err != nil {
			return nil, fmt.Errorf("failed to register client: %w", err)
		}
	}
	
	endpoint := fmt.Sprintf("https://oidc.%s.amazonaws.com/device_authorization", c.region)
	
	reqBody := map[string]string{
		"clientId":     c.registration.ClientID,
		"clientSecret": c.registration.ClientSecret,
		"startUrl":     c.startURL,
	}
	
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to start device authorization: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		var oidcErr OIDCError
		if err := json.Unmarshal(body, &oidcErr); err == nil {
			return nil, fmt.Errorf("OIDC error: %s", oidcErr.String())
		}
		return nil, fmt.Errorf("device authorization failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var authResp DeviceAuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	return &authResp, nil
}

// CreateToken creates an access token using device code or refresh token
func (c *Client) CreateToken(ctx context.Context, grantType, code string) (*TokenResponse, error) {
	if c.registration == nil {
		return nil, fmt.Errorf("client not registered")
	}
	
	endpoint := fmt.Sprintf("https://oidc.%s.amazonaws.com/token", c.region)
	
	reqBody := map[string]string{
		"clientId":     c.registration.ClientID,
		"clientSecret": c.registration.ClientSecret,
		"grantType":    grantType,
	}
	
	// Add the appropriate code based on grant type
	switch grantType {
	case GrantTypeDeviceCode:
		reqBody["deviceCode"] = code
	case GrantTypeRefreshToken:
		reqBody["refreshToken"] = code
	case GrantTypeAuthorizationCode:
		reqBody["code"] = code
	default:
		return nil, fmt.Errorf("unsupported grant type: %s", grantType)
	}
	
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create token: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	// Handle OIDC errors (including authorization_pending)
	if resp.StatusCode != http.StatusOK {
		var oidcErr OIDCError
		if err := json.Unmarshal(body, &oidcErr); err == nil {
			// Return the error code so caller can handle authorization_pending
			return nil, &oidcErr
		}
		return nil, fmt.Errorf("token creation failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	return &tokenResp, nil
}

// PollForToken polls the token endpoint until authorization is complete
func (c *Client) PollForToken(ctx context.Context, deviceCode string, interval int) (*TokenResponse, error) {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()
	
	pollCount := 0
	log.Printf("[OIDC] Starting token polling (interval: %ds)...\n", interval)
	
	for {
		select {
		case <-ctx.Done():
			log.Println("[OIDC] Polling cancelled by context")
			return nil, ctx.Err()
		case <-ticker.C:
			pollCount++
			log.Printf("[OIDC] Poll attempt #%d...\n", pollCount)
			
			token, err := c.CreateToken(ctx, GrantTypeDeviceCode, deviceCode)
			if err != nil {
				// Check if it's an OIDC error
				if oidcErr, ok := err.(*OIDCError); ok {
					switch oidcErr.ErrorCode {
					case ErrAuthorizationPending:
						// Keep polling
						log.Printf("[OIDC] Authorization pending, continuing to poll...\n")
						continue
					case ErrSlowDown:
						// Increase polling interval
						log.Printf("[OIDC] Slow down requested, increasing interval...\n")
						ticker.Reset(time.Duration(interval+5) * time.Second)
						continue
					case ErrExpiredToken:
						log.Println("[OIDC] Device code expired")
						return nil, fmt.Errorf("device code expired, please start over")
					case ErrAccessDenied:
						log.Println("[OIDC] Authorization denied by user")
						return nil, fmt.Errorf("authorization denied by user")
					default:
						log.Printf("[OIDC] OIDC error: %s\n", oidcErr.String())
						return nil, fmt.Errorf("OIDC error: %s", oidcErr.String())
					}
				}
				log.Printf("[OIDC] Polling error: %v\n", err)
				return nil, err
			}
			
			// Success!
			log.Println("[OIDC] [SUCCESS] Token received successfully!")
			return token, nil
		}
	}
}

// RefreshToken refreshes an access token using a refresh token
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	return c.CreateToken(ctx, GrantTypeRefreshToken, refreshToken)
}
