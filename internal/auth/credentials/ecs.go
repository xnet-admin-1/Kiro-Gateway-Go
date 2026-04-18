package credentials

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// ECSProvider retrieves credentials from ECS container metadata.
// It reads AWS_CONTAINER_CREDENTIALS_RELATIVE_URI environment variable.
type ECSProvider struct {
	httpClient *http.Client
}

// ECSCredentialsResponse represents the response from ECS metadata endpoint.
type ECSCredentialsResponse struct {
	AccessKeyID     string    `json:"AccessKeyId"`
	SecretAccessKey string    `json:"SecretAccessKey"`
	Token           string    `json:"Token"`
	Expiration      time.Time `json:"Expiration"`
}

// NewECSProvider creates a new ECS credential provider.
func NewECSProvider() *ECSProvider {
	return &ECSProvider{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Retrieve retrieves credentials from ECS metadata endpoint.
func (p *ECSProvider) Retrieve(ctx context.Context) (*Credentials, error) {
	relativeURI := os.Getenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI")
	if relativeURI == "" {
		return nil, fmt.Errorf("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI not set")
	}

	// ECS metadata endpoint
	endpoint := "http://169.254.170.2" + relativeURI

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call ECS metadata endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ECS metadata endpoint returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var credsResp ECSCredentialsResponse
	if err := json.Unmarshal(body, &credsResp); err != nil {
		return nil, fmt.Errorf("failed to parse credentials response: %w", err)
	}

	if credsResp.AccessKeyID == "" || credsResp.SecretAccessKey == "" {
		return nil, fmt.Errorf("incomplete credentials from ECS metadata")
	}

	return NewTemporaryCredentials(
		credsResp.AccessKeyID,
		credsResp.SecretAccessKey,
		credsResp.Token,
		"ecs",
		credsResp.Expiration,
	), nil
}

// IsExpired returns false as this provider doesn't cache credentials.
func (p *ECSProvider) IsExpired() bool {
	return false
}
