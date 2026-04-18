package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/yourusername/kiro-gateway-go/internal/models"
)

// AuthManager interface for testing
type AuthManager interface {
	GetRegion() string
	SignRequest(ctx context.Context, req *http.Request, body []byte) error
}

// Client wraps HTTP client with Kiro-specific functionality
type Client struct {
	httpClient              *http.Client
	authManager             AuthManager
	maxRetries              int
	retryClassifier         *RetryClassifier
	interceptorChain        *InterceptorChain
	stalledStreamProtection *StalledStreamProtection
	useQDeveloper           bool   // Use Q endpoint instead of CodeWhisperer
	apiEndpoint             string // Override endpoint (optional)
}

// Config holds client configuration
type Config struct {
	MaxConnections       int
	KeepAliveConnections int
	ConnectionTimeout    time.Duration
	MaxRetries           int
	StalledStreamGrace   time.Duration
	MinStreamSpeed       int64
	AppName              string
	AppVersion           string
	Fingerprint          string
	OptOutTelemetry      bool
	Logger               *log.Logger
	RequestID            string
	UseQDeveloper        bool   // Use Q endpoint instead of CodeWhisperer
	APIEndpoint          string // Override endpoint (optional)
	ProxyURL             string // HTTP/HTTPS/SOCKS5 proxy URL
}

// NewClient creates a new HTTP client
func NewClient(authManager AuthManager, cfg Config) *Client {
	transport := &http.Transport{
		MaxIdleConns:        cfg.MaxConnections,
		MaxIdleConnsPerHost: cfg.KeepAliveConnections,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
	}
	
	// Configure proxy if provided
	if cfg.ProxyURL != "" {
		proxyURL, err := parseProxyURL(cfg.ProxyURL)
		if err != nil {
			log.Printf("Warning: Invalid proxy URL '%s': %v", cfg.ProxyURL, err)
		} else {
			transport.Proxy = http.ProxyURL(proxyURL)
			log.Printf("Using proxy: %s", cfg.ProxyURL)
		}
	}

	httpClient := &http.Client{
		Transport: transport,
		// Don't set Timeout here - use context timeouts instead for better control
		// This allows different timeouts for text vs multimodal requests
	}

	// Create retry classifier
	retryClassifier := NewRetryClassifier()

	// Create interceptor chain
	interceptorChain := NewInterceptorChain()
	
	// Add default interceptors
	if cfg.Logger != nil {
		interceptorChain.Add(NewLoggingInterceptor(cfg.Logger))
		interceptorChain.Add(NewRequestIDInterceptor(cfg.Logger))
	}
	
	if cfg.AppName != "" && cfg.AppVersion != "" {
		interceptorChain.Add(NewUserAgentInterceptor(cfg.AppName, cfg.AppVersion, cfg.Fingerprint))
	}
	
	interceptorChain.Add(NewOptOutInterceptor(cfg.OptOutTelemetry))
	interceptorChain.Add(NewTelemetryInterceptor(cfg.AppName, cfg.AppVersion, !cfg.OptOutTelemetry))
	
	// Add request ID header interceptor if RequestID is provided
	if cfg.RequestID != "" {
		interceptorChain.Add(NewRequestIDHeaderInterceptor(cfg.RequestID))
	}

	// Create stalled stream protection
	gracePeriod := cfg.StalledStreamGrace
	if gracePeriod == 0 {
		gracePeriod = 5 * time.Minute // Default 5 minutes
	}
	
	minSpeed := cfg.MinStreamSpeed
	if minSpeed == 0 {
		minSpeed = 1 // Default 1 byte/second
	}
	
	stalledStreamProtection := NewStalledStreamProtection(gracePeriod, minSpeed)

	return &Client{
		httpClient:              httpClient,
		authManager:             authManager,
		maxRetries:              cfg.MaxRetries,
		retryClassifier:         retryClassifier,
		interceptorChain:        interceptorChain,
		stalledStreamProtection: stalledStreamProtection,
		useQDeveloper:           cfg.UseQDeveloper,
		apiEndpoint:             cfg.APIEndpoint,
	}
}

// Post makes a POST request to Kiro API with retry logic
func (c *Client) Post(ctx context.Context, endpoint string, payload interface{}) (*http.Response, error) {
	url := c.buildURL(endpoint)
	
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}
	
	// Determine target operation from endpoint
	target := c.determineTarget(endpoint)
	
	return c.doWithRetry(ctx, "POST", url, body, false, target)
}

// PostStream makes a streaming POST request to Kiro API with retry logic
func (c *Client) PostStream(ctx context.Context, endpoint string, payload interface{}) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}
	
	// Use bearer token auth for SendMessage (AI_EDITOR origin requires it)
	token, err := c.getBearerToken(ctx)
	if err != nil {
		log.Printf("[WARN] Bearer token unavailable: %v — falling back to SigV4", err)
		url := c.buildURL("/generateAssistantResponse")
		target := "AmazonQDeveloperStreamingService.GenerateAssistantResponse"
		return c.doWithRetry(ctx, "POST", url, body, true, target)
	}

	// Bearer token path uses /generateAssistantResponse endpoint
	sendURL := fmt.Sprintf("https://q.%s.amazonaws.com/generateAssistantResponse", c.authManager.GetRegion())

	req, err := http.NewRequestWithContext(ctx, "POST", sendURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("User-Agent", "aws-sdk-js/1.0.27 ua/2.1 os/linux lang/js md/nodejs#22.21.1 api/codewhispererstreaming#1.0.27 m/E KiroIDE-0.7.45")
	req.Header.Set("x-amz-user-agent", "aws-sdk-js/1.0.27 KiroIDE-0.7.45")
	req.Header.Set("x-amzn-codewhisperer-optout", "true")
	req.Header.Set("x-amzn-kiro-agent-mode", "vibe")

	log.Printf("[HTTP] POST %s (Bearer)", sendURL)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	return resp, nil
}

// determineTarget determines the X-Amz-Target header value based on endpoint
func (c *Client) determineTarget(endpoint string) string {
	if !c.useQDeveloper {
		return "" // CodeWhisperer mode doesn't use target header
	}
	
	// Map endpoints to their target operations
	// Note: Q Developer API uses SendMessage, not GenerateAssistantResponse
	switch endpoint {
	case "/generateAssistantResponse":
		return "AmazonQDeveloperStreamingService.SendMessage"
	case "/sendMessage":
		return "AmazonQDeveloperStreamingService.SendMessage"
	case "/":
		// When calling root with SigV4, use SendMessage
		return "AmazonQDeveloperStreamingService.SendMessage"
	default:
		// Default to SendMessage
		return "AmazonQDeveloperStreamingService.SendMessage"
	}
}

// buildURL constructs the full API URL based on mode and configuration
func (c *Client) buildURL(endpoint string) string {
	// Use custom endpoint if provided
	if c.apiEndpoint != "" {
		return c.apiEndpoint + endpoint
	}
	
	// Select endpoint based on mode
	var baseURL string
	if c.useQDeveloper {
		// QDeveloper mode: use q.{region}.amazonaws.com
		baseURL = "https://q." + c.authManager.GetRegion() + ".amazonaws.com"
	} else {
		// CodeWhisperer mode: use codewhisperer.{region}.amazonaws.com
		baseURL = "https://codewhisperer." + c.authManager.GetRegion() + ".amazonaws.com"
	}
	
	return baseURL + endpoint
}

// doWithRetry executes HTTP request with retry logic
func (c *Client) doWithRetry(ctx context.Context, method, url string, body []byte, stream bool, target string) (*http.Response, error) {
	var lastErr error
	var lastResp *http.Response
	
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		// Debug: Log request details
		fmt.Printf("[DEBUG] Request URL: %s\n", url)
		fmt.Printf("[DEBUG] Request method: %s\n", method)
		fmt.Printf("[DEBUG] X-Amz-Target: %s\n", target)
		fmt.Printf("[DEBUG] UseQDeveloper: %v\n", c.useQDeveloper)
		
		// Debug: Log request body (only in DEBUG mode)
		if os.Getenv("DEBUG") == "true" || os.Getenv("DEBUG") == "1" {
			if len(body) > 0 && len(body) < 10000 {
				fmt.Printf("[DEBUG] Request body: %s\n", string(body))
			} else if len(body) > 0 {
				fmt.Printf("[DEBUG] Request body length: %d bytes (too large to print)\n", len(body))
				// Save to file for inspection
				if err := os.WriteFile("debug-request.json", body, 0644); err == nil {
					fmt.Printf("[DEBUG] Request body saved to debug-request.json\n")
				}
			}
		} else {
			if len(body) > 0 {
				fmt.Printf("[DEBUG] Request body: [REDACTED - %d bytes] (set DEBUG=true to view)\n", len(body))
			}
		}
		
		// Create request
		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		
		// Set headers BEFORE signing (required for canonical request)
		
		// For Q Developer with SigV4, use JSON-RPC style headers
		// This is based on the Amazon Q CLI implementation
		// AWS ALWAYS returns event-stream format, even for non-streaming requests
		if c.useQDeveloper {
			// Q Developer API uses JSON-RPC style with X-Amz-Target header
			req.Header.Set("Content-Type", "application/x-amz-json-1.0")
			// Use the target passed in, or default to SendMessage for backward compatibility
			if target != "" {
				req.Header.Set("X-Amz-Target", target)
			} else {
				req.Header.Set("X-Amz-Target", "AmazonQDeveloperStreamingService.SendMessage")
			}
		} else {
			// CodeWhisperer uses standard JSON
			req.Header.Set("Content-Type", "application/json")
		}
		
		if stream {
			req.Header.Set("Accept", "text/event-stream")
		}
		
		// Sign request with auth manager (after headers are set)
		if err := c.authManager.SignRequest(ctx, req, body); err != nil {
			return nil, fmt.Errorf("failed to sign request: %w", err)
		}
		
		// Apply interceptors before request
		if err := c.interceptorChain.BeforeRequest(req); err != nil {
			return nil, fmt.Errorf("interceptor error: %w", err)
		}
		
		// Execute request
		resp, err := c.httpClient.Do(req)
		
		// Apply interceptors after response
		if interceptorErr := c.interceptorChain.AfterResponse(resp, err); interceptorErr != nil {
			// Log interceptor error but don't fail the request
			if c.interceptorChain != nil {
				// Interceptor error is logged internally
			}
		}
		
		if err != nil {
			lastErr = err
			lastResp = nil
			
			// Use retry classifier to determine if we should retry
			action := c.retryClassifier.ClassifyError(err, resp)
			if !c.retryClassifier.ShouldRetry(action) || attempt >= c.maxRetries-1 {
				break
			}
			
			backoff := calculateBackoff(attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				continue
			}
		}
		
		lastResp = resp
		
		// Read response body for classification if needed
		var respBody []byte
		if resp.StatusCode >= 400 {
			respBody, _ = io.ReadAll(resp.Body)
			resp.Body.Close()
			// Log error response for debugging
			fmt.Printf("[DEBUG] Error response (status %d): %s\n", resp.StatusCode, string(respBody))
			// Recreate response with new body reader
			resp.Body = io.NopCloser(bytes.NewReader(respBody))
		}
		
		// Use retry classifier to determine if we should retry
		action := c.retryClassifier.ClassifyErrorWithBody(nil, resp, respBody)
		if !c.retryClassifier.ShouldRetry(action) || attempt >= c.maxRetries-1 {
			// For streaming responses, wrap with stalled stream protection
			if stream && resp.StatusCode < 400 {
				resp.Body = io.NopCloser(c.stalledStreamProtection.WrapReader(ctx, resp.Body))
			}
			return resp, nil
		}
		
		// Close response body before retry
		resp.Body.Close()
		
		backoff := calculateBackoffFromResponse(resp, attempt)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
			continue
		}
	}
	
	if lastErr != nil {
		return nil, fmt.Errorf("request failed after %d attempts: %w", c.maxRetries, lastErr)
	}
	
	if lastResp != nil {
		return lastResp, nil
	}
	
	return nil, fmt.Errorf("request failed after %d attempts", c.maxRetries)
}

// calculateBackoff calculates exponential backoff with jitter
func calculateBackoff(attempt int) time.Duration {
	baseMs := 2000 * (1 << attempt) // 2s, 4s, 8s, 16s...
	jitterMs := int(float64(baseMs) * 0.2)
	totalMs := baseMs + jitterMs
	
	// Cap at 30 seconds
	if totalMs > 30000 {
		totalMs = 30000
	}
	
	return time.Duration(totalMs) * time.Millisecond
}

// calculateBackoffFromResponse extracts retry-after header if present
func calculateBackoffFromResponse(resp *http.Response, attempt int) time.Duration {
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter != "" {
		var seconds int
		if _, err := fmt.Sscanf(retryAfter, "%d", &seconds); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	
	// Fallback to exponential backoff
	return calculateBackoff(attempt)
}

// Close closes the HTTP client
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

// ReadBody reads and closes response body
func ReadBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// RetryClient is a simple HTTP client with retry logic for testing
type RetryClient struct {
	HTTPClient *http.Client
	MaxRetries int
	BaseDelay  time.Duration
}

// Do executes an HTTP request with retry logic
func (r *RetryClient) Do(req *http.Request) (*http.Response, error) {
	var lastErr error
	var lastResp *http.Response
	
	for attempt := 0; attempt <= r.MaxRetries; attempt++ {
		// Clone request for retry
		reqClone := req.Clone(req.Context())
		
		// Execute request
		resp, err := r.HTTPClient.Do(reqClone)
		
		if err != nil {
			lastErr = err
			lastResp = nil
		} else {
			lastResp = resp
			lastErr = nil
			
			// Check if we should retry based on status code
			if resp.StatusCode < 500 && resp.StatusCode != 429 {
				// Success or client error - don't retry
				return resp, nil
			}
			
			// Server error or throttling - close body and retry
			resp.Body.Close()
		}
		
		// Don't sleep after last attempt
		if attempt < r.MaxRetries {
			delay := r.BaseDelay * time.Duration(1<<attempt) // Exponential backoff
			time.Sleep(delay)
		}
	}
	
	if lastErr != nil {
		return nil, lastErr
	}
	
	return lastResp, nil
}

// ListAvailableProfiles discovers available profiles from AWS Q Developer API
func (c *Client) ListAvailableProfiles(ctx context.Context) (*models.ListAvailableProfilesResponse, error) {
	url := fmt.Sprintf("https://q.%s.amazonaws.com/ListAvailableProfiles", c.authManager.GetRegion())

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader([]byte("{}")))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	token, err := c.getBearerToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get bearer token: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result models.ListAvailableProfilesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ListAvailableModels calls the AWS Q Developer ListAvailableModels API
// This fetches the list of models available to the authenticated user
// IMPORTANT: Based on Python gateway implementation, this uses:
// - GET request (not POST!)
// - Bearer token authentication (not SigV4!)
// - Query parameters: origin and profileArn
// - Endpoint: https://q.{region}.amazonaws.com/ListAvailableModels
func (c *Client) ListAvailableModels(ctx context.Context, profileArn string) (*models.ListAvailableModelsResponse, error) {
	// Build URL with query parameters
	url := fmt.Sprintf("https://q.%s.amazonaws.com/ListAvailableModels", c.authManager.GetRegion())
	
	fmt.Printf("[DEBUG] ListAvailableModels - Using Q Developer endpoint: %s\n", url)
	fmt.Printf("[DEBUG] ListAvailableModels - Using Bearer token authentication\n")
	
	// Create GET request (not POST!)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Add query parameters
	q := req.URL.Query()
	q.Add("origin", "AI_EDITOR")
	if profileArn != "" {
		q.Add("profileArn", profileArn)
	}
	req.URL.RawQuery = q.Encode()
	
	// Get bearer token from auth manager
	token, err := c.getBearerToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get bearer token: %w", err)
	}
	
	// Set headers for Q Developer API with bearer token
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")
	
	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	
	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	// Parse response
	var result models.ListAvailableModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &result, nil
}

// getBearerToken gets a bearer token for CodeWhisperer API calls
// This is needed because some CodeWhisperer APIs (like ListAvailableModels) use bearer token auth
func (c *Client) getBearerToken(ctx context.Context) (string, error) {
	// Check if auth manager supports bearer token retrieval
	type tokenProvider interface {
		GetBearerToken(ctx context.Context) (string, error)
	}
	
	if provider, ok := c.authManager.(tokenProvider); ok {
		return provider.GetBearerToken(ctx)
	}
	
	// Fallback: try to get token from headless auth
	// In headless mode, we have an SSO access token that can be used as bearer token
	type headlessProvider interface {
		GetSSOAccessToken(ctx context.Context) (string, error)
	}
	
	if provider, ok := c.authManager.(headlessProvider); ok {
		return provider.GetSSOAccessToken(ctx)
	}
	
	return "", fmt.Errorf("auth manager does not support bearer token retrieval")
}

// signRequestWithService signs a request with a specific service name override
// This is needed for APIs like ListAvailableModels that use CodeWhisperer service
// even when the client is in Q Developer mode
func (c *Client) signRequestWithService(ctx context.Context, req *http.Request, body []byte, service string) error {
	// We need to sign with a different service than the default
	// The auth manager interface needs to support this
	
	// Check if auth manager supports service override
	type serviceOverrider interface {
		SignRequestWithService(ctx context.Context, req *http.Request, body []byte, service string) error
	}
	
	if overrider, ok := c.authManager.(serviceOverrider); ok {
		return overrider.SignRequestWithService(ctx, req, body, service)
	}
	
	// Fallback: use default SignRequest (will use wrong service but better than nothing)
	// This will work if the auth manager is already configured with codewhisperer service
	return c.authManager.SignRequest(ctx, req, body)
}


// parseProxyURL parses and validates a proxy URL
// Supports: http://host:port, https://host:port, socks5://host:port
// Also supports authentication: http://user:pass@host:port
func parseProxyURL(proxyURL string) (*url.URL, error) {
	// If no protocol specified, default to http://
	if !strings.Contains(proxyURL, "://") {
		proxyURL = "http://" + proxyURL
	}
	
	parsedURL, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse proxy URL: %w", err)
	}
	
	// Validate scheme
	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != "http" && scheme != "https" && scheme != "socks5" {
		return nil, fmt.Errorf("unsupported proxy scheme: %s (supported: http, https, socks5)", scheme)
	}
	
	// Validate host
	if parsedURL.Host == "" {
		return nil, fmt.Errorf("proxy URL missing host")
	}
	
	return parsedURL, nil
}

// ReadBodyBytes reads body bytes from io.ReadCloser
func ReadBodyBytes(body io.ReadCloser) ([]byte, error) {
	defer body.Close()
	return io.ReadAll(body)
}

// NewReadCloser creates a new ReadCloser from bytes
func NewReadCloser(data []byte) io.ReadCloser {
	return io.NopCloser(bytes.NewReader(data))
}
