package auth

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestManager_AuthModes(t *testing.T) {
	cfg := Config{
		AuthType:    string(AuthTypeDesktop),
		EnableSigV4: false,
	}
	
	manager, err := NewAuthManager(cfg)
	if err != nil {
		// Expected to fail without real token, but we can still test mode
		t.Logf("NewAuthManager error (expected): %v", err)
	}
	
	// Create a minimal manager for testing
	manager = &AuthManager{
		authType: AuthTypeDesktop,
		authMode: AuthModeBearerToken,
		region:   "us-east-1",
		service:  "codewhisperer",
	}
	
	// Test default mode
	mode := manager.GetAuthMode()
	if mode != AuthModeBearerToken {
		t.Errorf("Default auth mode = %v, want %v", mode, AuthModeBearerToken)
	}
	
	// Test mode switching
	manager.SetAuthMode(AuthModeSigV4)
	mode = manager.GetAuthMode()
	if mode != AuthModeSigV4 {
		t.Errorf("Auth mode after switch = %v, want %v", mode, AuthModeSigV4)
	}
}

func TestManager_SignRequest(t *testing.T) {
	// Create a minimal manager with bearer token
	manager := &AuthManager{
		authType: AuthTypeDesktop,
		authMode: AuthModeBearerToken,
		region:   "us-east-1",
		service:  "codewhisperer",
		token:    "test-token",
		tokenExp: time.Now().Add(1 * time.Hour),
	}
	manager.bearerResolver = NewBearerResolver(manager)
	
	req := &http.Request{
		Method: "GET",
		Header: make(http.Header),
	}
	
	ctx := context.Background()
	err := manager.SignRequest(ctx, req, []byte{})
	if err != nil {
		t.Errorf("SignRequest() error = %v", err)
	}
	
	// Verify Authorization header is set
	auth := req.Header.Get("Authorization")
	if auth == "" {
		t.Error("Authorization header not set")
	}
	if auth != "Bearer test-token" {
		t.Errorf("Authorization header = %q, want %q", auth, "Bearer test-token")
	}
}

func TestManager_GetToken(t *testing.T) {
	// Create a minimal manager with bearer token
	manager := &AuthManager{
		authType: AuthTypeDesktop,
		authMode: AuthModeBearerToken,
		region:   "us-east-1",
		service:  "codewhisperer",
		token:    "test-token",
		tokenExp: time.Now().Add(1 * time.Hour),
	}
	manager.bearerResolver = NewBearerResolver(manager)
	
	token, err := manager.GetToken(context.Background())
	if err != nil {
		t.Errorf("GetToken() error = %v", err)
	}
	
	if token == "" {
		t.Error("GetToken() returned empty token")
	}
	
	if token != "test-token" {
		t.Errorf("GetToken() = %q, want %q", token, "test-token")
	}
}

func TestManager_ThreadSafety(t *testing.T) {
	// Create a minimal manager with bearer token
	manager := &AuthManager{
		authType: AuthTypeDesktop,
		authMode: AuthModeBearerToken,
		region:   "us-east-1",
		service:  "codewhisperer",
		token:    "test-token",
		tokenExp: time.Now().Add(1 * time.Hour),
	}
	manager.bearerResolver = NewBearerResolver(manager)
	
	// Test concurrent access
	done := make(chan bool, 10)
	errors := make(chan error, 10)
	
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			
			_, err := manager.GetToken(context.Background())
			if err != nil {
				errors <- err
				return
			}
			
			req := &http.Request{
				Method: "GET",
				Header: make(http.Header),
			}
			
			err = manager.SignRequest(context.Background(), req, []byte{})
			if err != nil {
				errors <- err
			}
		}()
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	close(errors)
	for err := range errors {
		t.Errorf("Concurrent operation error = %v", err)
	}
}
