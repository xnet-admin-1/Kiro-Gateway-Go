package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	
	"github.com/yourusername/kiro-gateway-go/internal/logging"
)

// PostStreamRaw sends a POST request with raw bytes (no marshaling)
// This is used for the Direct API endpoint to ensure pure passthrough
func (c *Client) PostStreamRaw(ctx context.Context, endpoint string, body []byte) (*http.Response, error) {
	// Build URL
	var url string
	if c.apiEndpoint != "" {
		url = c.apiEndpoint + endpoint
	} else if c.useQDeveloper {
		url = fmt.Sprintf("https://q.%s.amazonaws.com%s", c.authManager.GetRegion(), endpoint)
	} else {
		url = fmt.Sprintf("https://codewhisperer.%s.amazonaws.com%s", c.authManager.GetRegion(), endpoint)
	}
	
	logging.DebugLog("PostStreamRaw - URL: %s", url)
	logging.DebugLog("PostStreamRaw - Body length: %d bytes", len(body))
	
	// Create request with raw body
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "AmazonQDeveloperStreamingService.SendMessage")
	
	// Add authentication (SigV4 or Bearer token)
	if err := c.authManager.SignRequest(ctx, req, body); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}
	
	logging.DebugLog("PostStreamRaw - Request signed successfully")
	
	// Save request body to debug file if debug mode is enabled
	if logging.IsDebugEnabled() {
		debugPath := "logs/debug-request-raw.json"
		if err := os.WriteFile(debugPath, body, 0644); err == nil {
			logging.DebugLog("PostStreamRaw - Request body saved to %s", debugPath)
		}
	}
	
	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	
	logging.DebugLog("PostStreamRaw - Response status: %d %s", resp.StatusCode, resp.Status)
	
	// If error response, read and log body
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		logging.DebugLog("PostStreamRaw - Error response body: %s", string(bodyBytes))
		
		// Recreate response with new body reader
		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}
	
	return resp, nil
}
