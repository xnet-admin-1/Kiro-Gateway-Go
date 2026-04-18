package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// OIDC auth implementation

type oidcTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func (a *AuthManager) loadOIDCToken(ctx context.Context) error {
	// Use refresh token to get access token
	return a.refreshOIDCToken(ctx)
}

func (a *AuthManager) refreshOIDCToken(ctx context.Context) error {
	if a.oidcRefreshToken == "" {
		return fmt.Errorf("OIDC refresh token not configured")
	}
	
	// Prepare token request
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", a.oidcRefreshToken)
	data.Set("client_id", a.oidcClientID)
	
	if a.oidcClientSecret != "" {
		data.Set("client_secret", a.oidcClientSecret)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", a.oidcTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var tokenResp oidcTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to parse token response: %w", err)
	}
	
	// Parse JWT to get expiration
	token, _, err := new(jwt.Parser).ParseUnverified(tokenResp.AccessToken, jwt.MapClaims{})
	if err != nil {
		return fmt.Errorf("failed to parse JWT: %w", err)
	}
	
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("invalid JWT claims")
	}
	
	exp, ok := claims["exp"].(float64)
	if !ok {
		// Fallback to expires_in
		a.tokenExp = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	} else {
		a.tokenExp = time.Unix(int64(exp), 0)
	}
	
	a.token = tokenResp.AccessToken
	
	// Update refresh token if provided
	if tokenResp.RefreshToken != "" {
		a.oidcRefreshToken = tokenResp.RefreshToken
	}
	
	return nil
}
