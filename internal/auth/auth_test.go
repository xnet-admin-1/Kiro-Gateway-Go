package auth

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/yourusername/kiro-gateway-go/internal/auth/credentials"
)

func TestAuthManager_GetAuthMode(t *testing.T) {
	tests := []struct {
		name        string
		enableSigV4 bool
		wantMode    AuthMode
	}{
		{
			name:        "bearer token mode",
			enableSigV4: false,
			wantMode:    AuthModeBearerToken,
		},
		{
			name:        "sigv4 mode",
			enableSigV4: true,
			wantMode:    AuthModeSigV4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				AuthType:    string(AuthTypeDesktop),
				EnableSigV4: tt.enableSigV4,
				APIHost:     "test.example.com",
			}

			am, err := NewAuthManager(cfg)
			if err != nil && !tt.enableSigV4 {
				// Bearer token mode requires initial token load, which will fail in tests
				// This is expected for this unit test
				t.Skip("Skipping test that requires token load")
			}

			if am != nil && am.GetAuthMode() != tt.wantMode {
				t.Errorf("GetAuthMode() = %v, want %v", am.GetAuthMode(), tt.wantMode)
			}
		})
	}
}

func TestAuthManager_SetAuthMode(t *testing.T) {
	// Create a minimal auth manager for testing
	am := &AuthManager{
		authType: AuthTypeDesktop,
		authMode: AuthModeBearerToken,
		region:   "us-east-1",
		service:  "codewhisperer",
	}

	// Test switching to SigV4 mode
	am.SetAuthMode(AuthModeSigV4)
	if am.GetAuthMode() != AuthModeSigV4 {
		t.Errorf("SetAuthMode(SigV4) failed, got mode %v", am.GetAuthMode())
	}
	if am.credentialChain == nil {
		t.Error("SetAuthMode(SigV4) did not initialize credential chain")
	}

	// Test switching back to bearer token mode
	am.SetAuthMode(AuthModeBearerToken)
	if am.GetAuthMode() != AuthModeBearerToken {
		t.Errorf("SetAuthMode(BearerToken) failed, got mode %v", am.GetAuthMode())
	}
	if am.bearerResolver == nil {
		t.Error("SetAuthMode(BearerToken) did not initialize bearer resolver")
	}
}

func TestAuthManager_SignRequest_BearerToken(t *testing.T) {
	am := &AuthManager{
		authType: AuthTypeDesktop,
		authMode: AuthModeBearerToken,
		token:    "test-token-123",
		tokenExp: time.Now().Add(1 * time.Hour),
	}
	am.bearerResolver = NewBearerResolver(am)

	req, err := http.NewRequest("POST", "https://example.com/api", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	ctx := context.Background()
	if err := am.SignRequest(ctx, req, nil); err != nil {
		t.Fatalf("SignRequest() error = %v", err)
	}

	authHeader := req.Header.Get("Authorization")
	expectedHeader := "Bearer test-token-123"
	if authHeader != expectedHeader {
		t.Errorf("Authorization header = %q, want %q", authHeader, expectedHeader)
	}
}

func TestAuthManager_SignRequest_SigV4(t *testing.T) {
	// Create mock credentials
	creds := &credentials.Credentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		Source:          "test",
	}

	// Create a mock provider that returns our test credentials
	mockProvider := &mockCredentialProvider{creds: creds}

	am := &AuthManager{
		authType:        AuthTypeDesktop,
		authMode:        AuthModeSigV4,
		region:          "us-east-1",
		service:         "codewhisperer",
		credentialChain: credentials.NewChain(mockProvider),
	}

	req, err := http.NewRequest("POST", "https://example.com/api", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	ctx := context.Background()
	body := []byte(`{"test":"data"}`)
	if err := am.SignRequest(ctx, req, body); err != nil {
		t.Fatalf("SignRequest() error = %v", err)
	}

	// Verify SigV4 headers are present
	if req.Header.Get("Authorization") == "" {
		t.Error("Authorization header not set")
	}
	if req.Header.Get("X-Amz-Date") == "" {
		t.Error("X-Amz-Date header not set")
	}
	if !containsString(req.Header.Get("Authorization"), "AWS4-HMAC-SHA256") {
		t.Error("Authorization header does not contain AWS4-HMAC-SHA256")
	}
}

func TestAuthManager_GetRegion(t *testing.T) {
	tests := []struct {
		name   string
		region string
		want   string
	}{
		{
			name:   "custom region",
			region: "eu-west-1",
			want:   "eu-west-1",
		},
		{
			name:   "default region",
			region: "",
			want:   "us-east-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				AuthType:    string(AuthTypeDesktop),
				EnableSigV4: true,
				AWSRegion:   tt.region,
			}

			am, err := NewAuthManager(cfg)
			if err != nil {
				t.Fatalf("NewAuthManager() error = %v", err)
			}

			if got := am.GetRegion(); got != tt.want {
				t.Errorf("GetRegion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthManager_GetService(t *testing.T) {
	tests := []struct {
		name    string
		service string
		want    string
	}{
		{
			name:    "custom service",
			service: "q",
			want:    "q",
		},
		{
			name:    "default service",
			service: "",
			want:    "codewhisperer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				AuthType:    string(AuthTypeDesktop),
				EnableSigV4: true,
				AWSService:  tt.service,
			}

			am, err := NewAuthManager(cfg)
			if err != nil {
				t.Fatalf("NewAuthManager() error = %v", err)
			}

			if got := am.GetService(); got != tt.want {
				t.Errorf("GetService() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Mock credential provider for testing
type mockCredentialProvider struct {
	creds *credentials.Credentials
	err   error
}

func (m *mockCredentialProvider) Retrieve(ctx context.Context) (*credentials.Credentials, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.creds, nil
}

func (m *mockCredentialProvider) IsExpired() bool {
	return false
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
func TestAuthManager_GetAuthType(t *testing.T) {
	cfg := Config{
		AuthType:    string(AuthTypeDesktop),
		EnableSigV4: true,
	}

	am, err := NewAuthManager(cfg)
	if err != nil {
		t.Fatalf("NewAuthManager() error = %v", err)
	}

	if got := am.GetAuthType(); got != AuthTypeDesktop {
		t.Errorf("GetAuthType() = %v, want %v", got, AuthTypeDesktop)
	}
}

func TestAuthManager_GetProfileArn(t *testing.T) {
	cfg := Config{
		AuthType:    string(AuthTypeDesktop),
		EnableSigV4: true,
	}

	am, err := NewAuthManager(cfg)
	if err != nil {
		t.Fatalf("NewAuthManager() error = %v", err)
	}

	if got := am.GetProfileArn(); got != "" {
		t.Errorf("GetProfileArn() = %v, want empty string", got)
	}
}

func TestAuthManager_IsTokenExpired(t *testing.T) {
	am := &AuthManager{
		authType: AuthTypeDesktop,
		authMode: AuthModeBearerToken,
		tokenExp: time.Now().Add(2 * time.Minute), // Future expiration
	}

	if am.IsTokenExpired() {
		t.Error("IsTokenExpired() should return false for future expiration")
	}

	// Set expiration within safety margin
	am.tokenExp = time.Now().Add(30 * time.Second)
	if !am.IsTokenExpired() {
		t.Error("IsTokenExpired() should return true for expiration within safety margin")
	}
}

func TestAuthManager_GetTokenExpiration(t *testing.T) {
	expectedExp := time.Now().Add(time.Hour)
	am := &AuthManager{
		authType: AuthTypeDesktop,
		authMode: AuthModeBearerToken,
		tokenExp: expectedExp,
	}

	if got := am.GetTokenExpiration(); !got.Equal(expectedExp) {
		t.Errorf("GetTokenExpiration() = %v, want %v", got, expectedExp)
	}
}

func TestAuthManager_GetToken_BearerMode(t *testing.T) {
	am := &AuthManager{
		authType: AuthTypeDesktop,
		authMode: AuthModeBearerToken,
		token:    "test-token",
		tokenExp: time.Now().Add(time.Hour),
	}
	am.bearerResolver = NewBearerResolver(am)

	ctx := context.Background()
	token, err := am.GetToken(ctx)
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}

	if token != "test-token" {
		t.Errorf("GetToken() = %v, want test-token", token)
	}
}

func TestAuthManager_GetToken_SigV4Mode(t *testing.T) {
	am := &AuthManager{
		authType: AuthTypeDesktop,
		authMode: AuthModeSigV4,
	}

	ctx := context.Background()
	_, err := am.GetToken(ctx)
	if err == nil {
		t.Error("GetToken() should fail in SigV4 mode")
	}
}

func TestAuthManager_RefreshToken(t *testing.T) {
	tests := []struct {
		name     string
		authType AuthType
		wantErr  bool
	}{
		{
			name:     "desktop auth type",
			authType: AuthTypeDesktop,
			wantErr:  true, // Will fail without real token
		},
		{
			name:     "cli db auth type",
			authType: AuthTypeCLIDB,
			wantErr:  true, // Will fail without real database
		},
		{
			name:     "oidc auth type",
			authType: AuthTypeOIDC,
			wantErr:  true, // Will fail without real OIDC setup
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			am := &AuthManager{
				authType: tt.authType,
				authMode: AuthModeBearerToken,
			}

			ctx := context.Background()
			err := am.RefreshToken(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("RefreshToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
func TestAuthManager_SetAuthType(t *testing.T) {
	cfg := Config{
		AuthType:    string(AuthTypeDesktop),
		EnableSigV4: true,
	}

	am, err := NewAuthManager(cfg)
	if err != nil {
		t.Fatalf("NewAuthManager() error = %v", err)
	}

	// Test setting different auth type
	am.SetAuthType(AuthTypeCLIDB)
	if got := am.GetAuthType(); got != AuthTypeCLIDB {
		t.Errorf("SetAuthType(CLIDB) failed, got %v", got)
	}

	// Test setting OIDC auth type
	am.SetAuthType(AuthTypeOIDC)
	if got := am.GetAuthType(); got != AuthTypeOIDC {
		t.Errorf("SetAuthType(OIDC) failed, got %v", got)
	}
}
