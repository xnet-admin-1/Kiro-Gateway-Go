package credentials

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// IMDSProvider retrieves credentials from EC2 instance metadata service.
// It supports IMDSv2 (token-based authentication).
type IMDSProvider struct {
	httpClient *http.Client
}

// IMDSCredentialsResponse represents the response from IMDS endpoint.
type IMDSCredentialsResponse struct {
	AccessKeyID     string    `json:"AccessKeyId"`
	SecretAccessKey string    `json:"SecretAccessKey"`
	Token           string    `json:"Token"`
	Expiration      time.Time `json:"Expiration"`
}

// NewIMDSProvider creates a new IMDS credential provider.
func NewIMDSProvider() *IMDSProvider {
	return &IMDSProvider{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Retrieve retrieves credentials from EC2 instance metadata service.
func (p *IMDSProvider) Retrieve(ctx context.Context) (*Credentials, error) {
	// Step 1: Get IMDSv2 token
	token, err := p.getIMDSToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get IMDS token: %w", err)
	}

	// Step 2: Get IAM role name
	roleName, err := p.getIAMRoleName(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get IAM role name: %w", err)
	}

	// Step 3: Get credentials for the role
	creds, err := p.getCredentials(ctx, token, roleName)
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	return creds, nil
}

// IsExpired returns false as this provider doesn't cache credentials.
func (p *IMDSProvider) IsExpired() bool {
	return false
}

// getIMDSToken gets an IMDSv2 token.
func (p *IMDSProvider) getIMDSToken(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "PUT", "http://169.254.169.254/latest/api/token", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("X-aws-ec2-metadata-token-ttl-seconds", "21600")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("IMDS token request returned status %d", resp.StatusCode)
	}

	token, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(token), nil
}

// getIAMRoleName gets the IAM role name from IMDS.
func (p *IMDSProvider) getIAMRoleName(ctx context.Context, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "http://169.254.169.254/latest/meta-data/iam/security-credentials/", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("X-aws-ec2-metadata-token", token)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("IMDS role request returned status %d", resp.StatusCode)
	}

	roleNameBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	roleName := strings.TrimSpace(string(roleNameBytes))
	if roleName == "" {
		return "", fmt.Errorf("no IAM role found")
	}

	return roleName, nil
}

// getCredentials gets the credentials for the specified role.
func (p *IMDSProvider) getCredentials(ctx context.Context, token, roleName string) (*Credentials, error) {
	url := fmt.Sprintf("http://169.254.169.254/latest/meta-data/iam/security-credentials/%s", roleName)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-aws-ec2-metadata-token", token)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IMDS credentials request returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var credsResp IMDSCredentialsResponse
	if err := json.Unmarshal(body, &credsResp); err != nil {
		return nil, fmt.Errorf("failed to parse credentials response: %w", err)
	}

	if credsResp.AccessKeyID == "" || credsResp.SecretAccessKey == "" {
		return nil, fmt.Errorf("incomplete credentials from IMDS")
	}

	return NewTemporaryCredentials(
		credsResp.AccessKeyID,
		credsResp.SecretAccessKey,
		credsResp.Token,
		"imds",
		credsResp.Expiration,
	), nil
}
