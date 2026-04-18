// +build integration

package auth

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestAuthManager_BearerTokenMode_EndToEnd tests bearer token authentication end-to-end
func TestAuthManager_BearerTokenMode_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// This test requires a valid bearer token setup
	// In a real scenario, you would have a test environment with valid credentials
	t.Skip("Requires valid bearer token credentials")

	cfg := Config{
		AuthType:    string(AuthTypeOIDC),
		EnableSigV4: false,
		APIHost:     "codewhisperer.us-east-1.amazonaws.com",
	}

	am, err := NewAuthManager(cfg)
	if err != nil {
		t.Fatalf("NewAuthManager() error = %v", err)
	}

	// Test getting a token
	ctx := context.Background()
	token, err := am.GetToken(ctx)
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}

	if token == "" {
		t.Error("GetToken() returned empty token")
	}

	// Test signing a request
	req, err := http.NewRequest("POST", "https://codewhisperer.us-east-1.amazonaws.com/api", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	if err := am.SignRequest(ctx, req, nil); err != nil {
		t.Fatalf("SignRequest() error = %v", err)
	}

	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		t.Error("Authorization header not set")
	}
	if len(authHeader) < 10 || authHeader[:7] != "Bearer " {
		t.Errorf("Authorization header format incorrect: %s", authHeader)
	}
}

// TestAuthManager_SigV4Mode_EndToEnd tests SigV4 authentication end-to-end
func TestAuthManager_SigV4Mode_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Set up test AWS credentials
	os.Setenv("AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
	defer os.Unsetenv("AWS_ACCESS_KEY_ID")
	defer os.Unsetenv("AWS_SECRET_ACCESS_KEY")

	cfg := Config{
		AuthType:    string(AuthTypeDesktop),
		EnableSigV4: true,
		AWSRegion:   "us-east-1",
		AWSService:  "codewhisperer",
		APIHost:     "codewhisperer.us-east-1.amazonaws.com",
	}

	am, err := NewAuthManager(cfg)
	if err != nil {
		t.Fatalf("NewAuthManager() error = %v", err)
	}

	// Verify auth mode
	if am.GetAuthMode() != AuthModeSigV4 {
		t.Errorf("GetAuthMode() = %v, want %v", am.GetAuthMode(), AuthModeSigV4)
	}

	// Test signing a request
	ctx := context.Background()
	req, err := http.NewRequest("POST", "https://codewhisperer.us-east-1.amazonaws.com/api", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	body := []byte(`{"test":"data"}`)
	if err := am.SignRequest(ctx, req, body); err != nil {
		t.Fatalf("SignRequest() error = %v", err)
	}

	// Verify SigV4 headers
	if req.Header.Get("Authorization") == "" {
		t.Error("Authorization header not set")
	}
	if req.Header.Get("X-Amz-Date") == "" {
		t.Error("X-Amz-Date header not set")
	}

	authHeader := req.Header.Get("Authorization")
	if len(authHeader) < 20 || authHeader[:17] != "AWS4-HMAC-SHA256 " {
		t.Errorf("Authorization header format incorrect: %s", authHeader)
	}
}

// TestAuthManager_ModeSwitching tests switching between authentication modes
func TestAuthManager_ModeSwitching(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Set up test AWS credentials
	os.Setenv("AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
	defer os.Unsetenv("AWS_ACCESS_KEY_ID")
	defer os.Unsetenv("AWS_SECRET_ACCESS_KEY")

	// Start with SigV4 mode
	cfg := Config{
		AuthType:    string(AuthTypeDesktop),
		EnableSigV4: true,
		AWSRegion:   "us-east-1",
		AWSService:  "codewhisperer",
	}

	am, err := NewAuthManager(cfg)
	if err != nil {
		t.Fatalf("NewAuthManager() error = %v", err)
	}

	// Verify SigV4 mode
	if am.GetAuthMode() != AuthModeSigV4 {
		t.Errorf("Initial mode = %v, want %v", am.GetAuthMode(), AuthModeSigV4)
	}

	// Switch to bearer token mode
	am.SetAuthMode(AuthModeBearerToken)
	am.token = "test-token-123"
	am.tokenExp = time.Now().Add(1 * time.Hour)

	if am.GetAuthMode() != AuthModeBearerToken {
		t.Errorf("After switch mode = %v, want %v", am.GetAuthMode(), AuthModeBearerToken)
	}

	// Test signing with bearer token
	ctx := context.Background()
	req, err := http.NewRequest("POST", "https://example.com/api", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	if err := am.SignRequest(ctx, req, nil); err != nil {
		t.Fatalf("SignRequest() error = %v", err)
	}

	authHeader := req.Header.Get("Authorization")
	if authHeader != "Bearer test-token-123" {
		t.Errorf("Authorization header = %q, want %q", authHeader, "Bearer test-token-123")
	}

	// Switch back to SigV4 mode
	am.SetAuthMode(AuthModeSigV4)

	if am.GetAuthMode() != AuthModeSigV4 {
		t.Errorf("After second switch mode = %v, want %v", am.GetAuthMode(), AuthModeSigV4)
	}

	// Test signing with SigV4
	req2, err := http.NewRequest("POST", "https://example.com/api", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	if err := am.SignRequest(ctx, req2, []byte(`{"test":"data"}`)); err != nil {
		t.Fatalf("SignRequest() error = %v", err)
	}

	authHeader2 := req2.Header.Get("Authorization")
	if len(authHeader2) < 17 || authHeader2[:17] != "AWS4-HMAC-SHA256 " {
		t.Errorf("Authorization header format incorrect: %s", authHeader2)
	}
}

// TestAuthManager_CredentialChain tests the credential chain integration
func TestAuthManager_CredentialChain(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Test with environment variables
	os.Setenv("AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
	defer os.Unsetenv("AWS_ACCESS_KEY_ID")
	defer os.Unsetenv("AWS_SECRET_ACCESS_KEY")

	cfg := Config{
		AuthType:    string(AuthTypeDesktop),
		EnableSigV4: true,
		AWSRegion:   "us-east-1",
		AWSService:  "codewhisperer",
	}

	am, err := NewAuthManager(cfg)
	if err != nil {
		t.Fatalf("NewAuthManager() error = %v", err)
	}

	// Retrieve credentials
	ctx := context.Background()
	creds, err := am.credentialChain.Retrieve(ctx)
	if err != nil {
		t.Fatalf("credentialChain.Retrieve() error = %v", err)
	}

	if creds.AccessKeyID != "AKIAIOSFODNN7EXAMPLE" {
		t.Errorf("AccessKeyID = %s, want AKIAIOSFODNN7EXAMPLE", creds.AccessKeyID)
	}

	if creds.Source != "environment" {
		t.Errorf("Source = %s, want environment", creds.Source)
	}
}

// TestAuthManager_TokenRefresh tests automatic token refresh
func TestAuthManager_TokenRefresh(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// This test requires a valid token refresh setup
	t.Skip("Requires valid token refresh credentials")

	am := &AuthManager{
		authType: AuthTypeDesktop,
		authMode: AuthModeBearerToken,
		token:    "old-token",
		tokenExp: time.Now().Add(30 * time.Second), // Expires soon
	}
	am.bearerResolver = NewBearerResolver(am)

	ctx := context.Background()

	// First call should trigger refresh
	_, err := am.bearerResolver.GetToken(ctx)
	if err != nil {
		// Expected to fail without real backend
		t.Logf("Token refresh failed as expected: %v", err)
	}
}

// TestAuthManager_ConcurrentRequests tests concurrent request signing
func TestAuthManager_ConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	os.Setenv("AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
	defer os.Unsetenv("AWS_ACCESS_KEY_ID")
	defer os.Unsetenv("AWS_SECRET_ACCESS_KEY")

	cfg := Config{
		AuthType:    string(AuthTypeDesktop),
		EnableSigV4: true,
		AWSRegion:   "us-east-1",
		AWSService:  "codewhisperer",
	}

	am, err := NewAuthManager(cfg)
	if err != nil {
		t.Fatalf("NewAuthManager() error = %v", err)
	}

	ctx := context.Background()
	done := make(chan bool)
	errors := make(chan error, 10)

	// Sign 10 requests concurrently
	for i := 0; i < 10; i++ {
		go func(id int) {
			req, err := http.NewRequest("POST", "https://example.com/api", nil)
			if err != nil {
				errors <- err
				done <- true
				return
			}

			body := []byte(`{"test":"data"}`)
			if err := am.SignRequest(ctx, req, body); err != nil {
				errors <- err
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	close(errors)
	for err := range errors {
		t.Errorf("Concurrent request signing error: %v", err)
	}
}
