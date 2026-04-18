package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/yourusername/kiro-gateway-go/internal/auth/credentials"
	"github.com/yourusername/kiro-gateway-go/internal/auth/sigv4"
	"github.com/yourusername/kiro-gateway-go/internal/storage"
)

// AuthMode represents the authentication mode
type AuthMode int

const (
	// AuthModeBearerToken uses bearer token authentication
	AuthModeBearerToken AuthMode = iota
	// AuthModeSigV4 uses AWS Signature Version 4 authentication
	AuthModeSigV4
)

// AuthType represents the type of authentication
type AuthType string

const (
	// AuthTypeDesktop uses Kiro Desktop authentication
	AuthTypeDesktop AuthType = "desktop"
	// AuthTypeCLIDB uses CLI database authentication
	AuthTypeCLIDB AuthType = "cli_db"
	// AuthTypeOIDC uses OIDC authentication
	AuthTypeOIDC AuthType = "oidc"
	// AuthTypeAutomatedOIDC uses fully automated OIDC authentication with browser automation
	AuthTypeAutomatedOIDC AuthType = "automated_oidc"
	// AuthTypeHeadless uses headless OIDC authentication (no AWS CLI required)
	AuthTypeHeadless AuthType = "headless"
)

// Config holds the configuration for AuthManager
type Config struct {
	// AuthType specifies the authentication type (desktop, cli_db, oidc, automated_oidc, headless)
	AuthType string
	
	// EnableSigV4 enables AWS Signature Version 4 authentication
	EnableSigV4 bool
	
	// UseSSOCredentials enables SSO-derived IAM credentials for SigV4 mode
	// When true, converts bearer tokens to IAM credentials via SSO API
	// Required for Q Developer mode with SigV4
	UseSSOCredentials bool
	
	// SSOAccountID is the AWS account ID for SSO credential retrieval
	// Required when UseSSOCredentials is true
	SSOAccountID string
	
	// SSORoleName is the SSO role name for credential retrieval
	// Required when UseSSOCredentials is true
	SSORoleName string
	
	// AWSRegion specifies the AWS region (default: us-east-1)
	AWSRegion string
	
	// AWSService specifies the AWS service name (default: codewhisperer)
	AWSService string
	
	// APIHost specifies the API host
	APIHost string
	
	// ProfileARN specifies the profile ARN for Identity Center
	ProfileARN string
	
	// KiroDBPath specifies the path to Kiro Desktop database
	KiroDBPath string
	
	// CLIDBPath specifies the path to CLI database
	CLIDBPath string
	
	// OIDCClientID specifies the OIDC client ID
	OIDCClientID string
	
	// OIDCClientSecret specifies the OIDC client secret
	OIDCClientSecret string
	
	// OIDCTokenURL specifies the OIDC token URL
	OIDCTokenURL string
	
	// OIDCRefreshToken specifies the OIDC refresh token
	OIDCRefreshToken string
	
	// Headless Mode Configuration
	HeadlessMode    bool   // Enable headless OIDC authentication
	SSOStartURL     string // SSO start URL for headless mode
	SSORegion       string // SSO region for headless mode
	SSOClientID     string // Pre-registered OIDC client ID (optional)
	SSOClientSecret string // Pre-registered OIDC client secret (optional)
	SSOClientExpiry int64  // Client secret expiry timestamp (optional)
	
	// Browser Automation Configuration
	AutomateAuth   bool   // Enable browser automation for headless mode
	SSOUsername    string // IAM Identity Center username for automation
	SSOPassword    string // IAM Identity Center password for automation
	MFATOTPSecret  string // TOTP secret for automated MFA (base32 encoded)
	
	// TokenStore specifies the token storage backend
	TokenStore storage.Store
}

// AuthManager manages authentication for API requests
type AuthManager struct {
	// authType specifies the authentication type
	authType AuthType
	
	// authMode specifies the authentication mode (bearer token or SigV4)
	authMode AuthMode
	
	// region specifies the AWS region
	region string
	
	// service specifies the AWS service name
	service string
	
	// bearerResolver resolves bearer tokens
	bearerResolver *BearerResolver
	
	// credentialChain provides AWS credentials
	credentialChain *credentials.Chain
	
	// ssoProvider provides SSO-derived IAM credentials
	ssoProvider *credentials.SSOProvider
	
	// useSSOCredentials indicates whether to use SSO-derived credentials for SigV4
	useSSOCredentials bool
	
	// ssoAccountID is the AWS account ID for SSO
	ssoAccountID string
	
	// ssoRoleName is the SSO role name
	ssoRoleName string
	
	// tokenStore stores tokens securely
	tokenStore storage.Store
	
	// token holds the current bearer token
	token string
	
	// tokenExp holds the token expiration time
	tokenExp time.Time
	
	// profileArn holds the profile ARN
	profileArn string
	
	// kiroDBPath holds the path to Kiro Desktop database
	kiroDBPath string
	
	// cliDBPath holds the path to CLI database
	cliDBPath string
	
	// oidcClientID holds the OIDC client ID
	oidcClientID string
	
	// oidcClientSecret holds the OIDC client secret
	oidcClientSecret string
	
	// oidcTokenURL holds the OIDC token URL
	oidcTokenURL string
	
	// oidcRefreshToken holds the OIDC refresh token
	oidcRefreshToken string
	
	// headlessAuth manages headless OIDC authentication
	headlessAuth *HeadlessAuthManager
	
	// mu protects concurrent access to mutable fields
	mu sync.RWMutex
}

// NewAuthManager creates a new AuthManager with the given configuration
func NewAuthManager(cfg Config) (*AuthManager, error) {
	// Set defaults
	region := cfg.AWSRegion
	if region == "" {
		region = "us-east-1"
	}
	
	service := cfg.AWSService
	if service == "" {
		service = "codewhisperer"
	}
	
	authType := AuthType(cfg.AuthType)
	if authType == "" {
		authType = AuthTypeDesktop
	}
	
	// Determine auth mode
	authMode := AuthModeBearerToken
	if cfg.EnableSigV4 {
		authMode = AuthModeSigV4
	}
	
	am := &AuthManager{
		authType:          authType,
		authMode:          authMode,
		region:            region,
		service:           service,
		tokenStore:        cfg.TokenStore,
		profileArn:        cfg.ProfileARN,
		kiroDBPath:        cfg.KiroDBPath,
		cliDBPath:         cfg.CLIDBPath,
		oidcClientID:      cfg.OIDCClientID,
		oidcClientSecret:  cfg.OIDCClientSecret,
		oidcTokenURL:      cfg.OIDCTokenURL,
		oidcRefreshToken:  cfg.OIDCRefreshToken,
		useSSOCredentials: cfg.UseSSOCredentials,
		ssoAccountID:      cfg.SSOAccountID,
		ssoRoleName:       cfg.SSORoleName,
	}
	
	// Check if headless mode is enabled
	if cfg.HeadlessMode {
		// Validate headless configuration
		if cfg.SSOStartURL == "" {
			return nil, fmt.Errorf("SSO start URL is required for headless mode")
		}
		if cfg.SSOAccountID == "" || cfg.SSORoleName == "" {
			return nil, fmt.Errorf("SSO account ID and role name are required for headless mode")
		}
		
		// Override auth type to headless
		am.authType = AuthTypeHeadless
		
		// Initialize headless auth manager
		headlessConfig := HeadlessAuthConfig{
			Region:       cfg.SSORegion,
			StartURL:     cfg.SSOStartURL,
			AccountID:    cfg.SSOAccountID,
			RoleName:     cfg.SSORoleName,
			ClientID:     cfg.SSOClientID,
			ClientSecret: cfg.SSOClientSecret,
			ClientExpiry: cfg.SSOClientExpiry,
			AutomateAuth: cfg.AutomateAuth,
			Username:     cfg.SSOUsername,
			Password:     cfg.SSOPassword,
			TOTPSecret:   cfg.MFATOTPSecret,
		}
		
		if headlessConfig.Region == "" {
			headlessConfig.Region = region
		}
		
		am.headlessAuth = NewHeadlessAuthManager(headlessConfig, cfg.TokenStore)
		
		// Initialize headless auth
		ctx := context.Background()
		if err := am.headlessAuth.InitializeNonBlocking(ctx); err != nil {
			return nil, fmt.Errorf("failed to initialize headless authentication: %w", err)
		}
		
		// Start background refresh
		go am.headlessAuth.StartBackgroundRefresh(context.Background())
		
		// For SigV4 mode, we'll use credentials from headless auth
		if authMode == AuthModeSigV4 {
			// Create a credential provider that uses headless auth
			am.credentialChain = credentials.NewDefaultChain()
		}
		
		return am, nil
	}
	
	// Initialize based on auth mode
	if authMode == AuthModeSigV4 {
		// Check if we should use SSO-derived credentials
		if cfg.UseSSOCredentials {
			// Validate SSO configuration
			if cfg.SSOAccountID == "" || cfg.SSORoleName == "" {
				return nil, fmt.Errorf("SSO account ID and role name are required when UseSSOCredentials is true")
			}
			
			// IMPORTANT: Load bearer token first, as SSO provider needs it
			// Even though we're in SigV4 mode, we need the bearer token to convert to IAM credentials
			ctx := context.Background()
			if err := am.loadInitialToken(ctx); err != nil {
				// Log warning but don't fail - token may be loaded later
				fmt.Printf("Warning: Failed to load initial bearer token for SSO conversion: %v\n", err)
			}
			
			// Initialize SSO provider with token provider function
			// This allows dynamic token retrieval from bearer token auth
			ssoProvider, err := credentials.NewSSOProvider(credentials.SSOProviderConfig{
				Region:    region,
				AccountID: cfg.SSOAccountID,
				RoleName:  cfg.SSORoleName,
				AccessTokenProvider: func() (string, error) {
					// Get current bearer token
					am.mu.RLock()
					token := am.token
					am.mu.RUnlock()
					
					if token == "" {
						return "", fmt.Errorf("no bearer token available for SSO credential conversion")
					}
					return token, nil
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create SSO provider: %w", err)
			}
			
			am.ssoProvider = ssoProvider
			
			// Create credential chain with SSO provider
			am.credentialChain = credentials.NewDefaultChainWithSSO(ssoProvider)
		} else {
			// Use standard credential chain (IAM credentials)
			am.credentialChain = credentials.NewDefaultChain()
		}
	} else {
		// Initialize bearer resolver for bearer token mode
		am.bearerResolver = NewBearerResolver(am)
		
		// Load initial token based on auth type
		ctx := context.Background()
		if err := am.loadInitialToken(ctx); err != nil {
			// Don't fail on initial load error - token may be loaded later
			// or refreshed on first use
		}
	}
	
	return am, nil
}

// loadInitialToken loads the initial token based on auth type
func (am *AuthManager) loadInitialToken(ctx context.Context) error {
	// Check for BEARER_TOKEN environment variable first
	if envToken := os.Getenv("BEARER_TOKEN"); envToken != "" {
		am.mu.Lock()
		am.token = envToken
		// Set expiration to 24 hours from now if not specified
		am.tokenExp = time.Now().Add(24 * time.Hour)
		am.mu.Unlock()
		return nil
	}
	
	switch am.authType {
	case AuthTypeDesktop:
		return am.loadDesktopToken(ctx)
	case AuthTypeCLIDB:
		return am.loadCLIDBToken(ctx)
	case AuthTypeOIDC:
		return am.loadOIDCToken(ctx)
	case AuthTypeAutomatedOIDC:
		return am.loadAutomatedOIDCToken(ctx)
	case AuthTypeHeadless:
		// Headless auth is already initialized in NewAuthManager
		return nil
	default:
		return fmt.Errorf("unknown auth type: %s", am.authType)
	}
}

// GetAuthMode returns the current authentication mode
func (am *AuthManager) GetAuthMode() AuthMode {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.authMode
}
// GetAuthType returns the current authentication type
func (am *AuthManager) GetAuthType() AuthType {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.authType
}

// SetAuthType sets the authentication type
func (am *AuthManager) SetAuthType(authType AuthType) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.authType = authType
}
// SetAuthMode sets the authentication mode and initializes required components
func (am *AuthManager) SetAuthMode(mode AuthMode) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	am.authMode = mode
	
	if mode == AuthModeSigV4 {
		// Initialize credential chain if not already initialized
		if am.credentialChain == nil {
			am.credentialChain = credentials.NewDefaultChain()
		}
	} else {
		// Initialize bearer resolver if not already initialized
		if am.bearerResolver == nil {
			am.bearerResolver = NewBearerResolver(am)
		}
	}
}

// GetRegion returns the AWS region
func (am *AuthManager) GetRegion() string {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.region
}

// GetHeadlessAuth returns the headless auth manager (nil if not in headless mode)
func (am *AuthManager) GetHeadlessAuth() *HeadlessAuthManager {
	return am.headlessAuth
}

// GetService returns the AWS service name
func (am *AuthManager) GetService() string {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.service
}

// GetProfileArn returns the profile ARN
func (am *AuthManager) GetProfileArn() string {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.profileArn
}

// SetProfileArn sets the profile ARN (used by auto-discovery)
func (am *AuthManager) SetProfileArn(arn string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.profileArn = arn
}

// GetToken returns a valid bearer token
func (am *AuthManager) GetToken(ctx context.Context) (string, error) {
	if am.GetAuthMode() != AuthModeBearerToken {
		return "", fmt.Errorf("GetToken is only available in bearer token mode")
	}
	
	if am.bearerResolver == nil {
		return "", fmt.Errorf("bearer resolver not initialized")
	}
	
	return am.bearerResolver.GetToken(ctx)
}

// GetBearerToken returns a bearer token for CodeWhisperer API calls
// This works in both bearer token mode and SigV4 mode (using SSO access token)
func (am *AuthManager) GetBearerToken(ctx context.Context) (string, error) {
	// In bearer token mode, use the regular token
	if am.GetAuthMode() == AuthModeBearerToken {
		return am.GetToken(ctx)
	}
	
	// In SigV4 mode with headless auth, use the SSO access token
	if am.authType == AuthTypeHeadless && am.headlessAuth != nil {
		return am.GetSSOAccessToken(ctx)
	}
	
	return "", fmt.Errorf("bearer token not available in current auth mode")
}

// GetSSOAccessToken returns the SSO access token from headless auth
// This is used for CodeWhisperer APIs that require bearer token authentication
func (am *AuthManager) GetSSOAccessToken(ctx context.Context) (string, error) {
	if am.authType != AuthTypeHeadless || am.headlessAuth == nil {
		return "", fmt.Errorf("SSO access token only available in headless mode")
	}
	
	return am.headlessAuth.GetAccessToken(ctx)
}

// SignRequest signs an HTTP request based on the current authentication mode
func (am *AuthManager) SignRequest(ctx context.Context, req *http.Request, body []byte) error {
	return am.SignRequestWithService(ctx, req, body, "")
}

// SignRequestWithService signs an HTTP request with an optional service override
// If service is empty, uses the default service from auth manager configuration
func (am *AuthManager) SignRequestWithService(ctx context.Context, req *http.Request, body []byte, serviceOverride string) error {
	mode := am.GetAuthMode()
	
	switch mode {
	case AuthModeBearerToken:
		// Get bearer token and add to Authorization header
		token, err := am.GetToken(ctx)
		if err != nil {
			return fmt.Errorf("failed to get bearer token: %w", err)
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		return nil
		
	case AuthModeSigV4:
		// Determine which service to use for signing
		service := am.GetService()
		if serviceOverride != "" {
			service = serviceOverride
		}
		
		// Check if using headless auth
		if am.authType == AuthTypeHeadless && am.headlessAuth != nil {
			// Get credentials from headless auth
			creds, err := am.headlessAuth.GetCredentials(ctx)
			if err != nil {
				return fmt.Errorf("failed to get headless credentials: %w", err)
			}
			
			// Create signer with service override and sign request
			signer := sigv4.NewSigner(creds, am.GetRegion(), service)
			if err := signer.SignRequest(req, body); err != nil {
				return fmt.Errorf("failed to sign request: %w", err)
			}
			return nil
		}
		
		// Get credentials from chain
		am.mu.RLock()
		chain := am.credentialChain
		am.mu.RUnlock()
		
		if chain == nil {
			return fmt.Errorf("credential chain not initialized")
		}
		
		creds, err := chain.Retrieve(ctx)
		if err != nil {
			return fmt.Errorf("failed to retrieve credentials: %w", err)
		}
		
		// Create signer with service override and sign request
		signer := sigv4.NewSigner(creds, am.GetRegion(), service)
		if err := signer.SignRequest(req, body); err != nil {
			return fmt.Errorf("failed to sign request: %w", err)
		}
		return nil
		
	default:
		return fmt.Errorf("unknown auth mode: %d", mode)
	}
}

// RefreshToken refreshes the bearer token based on auth type
func (am *AuthManager) RefreshToken(ctx context.Context) error {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	switch am.authType {
	case AuthTypeDesktop:
		return am.refreshDesktopToken(ctx)
	case AuthTypeCLIDB:
		return am.refreshCLIDBToken(ctx)
	case AuthTypeOIDC:
		return am.refreshOIDCToken(ctx)
	case AuthTypeAutomatedOIDC:
		return am.refreshAutomatedOIDCToken(ctx)
	case AuthTypeHeadless:
		// Headless auth handles refresh automatically
		return nil
	default:
		return fmt.Errorf("unknown auth type: %s", am.authType)
	}
}

// IsTokenExpired checks if the current token is expired (with 1-minute safety margin)
func (am *AuthManager) IsTokenExpired() bool {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	// Consider token expired if it expires within 1 minute (inclusive)
	// Using AfterOrEqual logic: now + 1 minute >= expiration time
	return !time.Now().Add(time.Minute).Before(am.tokenExp)
}

// GetTokenExpiration returns the token expiration time
func (am *AuthManager) GetTokenExpiration() time.Time {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.tokenExp
}

// loadAutomatedOIDCToken loads token using automated OIDC authentication
func (am *AuthManager) loadAutomatedOIDCToken(ctx context.Context) error {
	// Try to load existing token from storage first
	if am.tokenStore != nil {
		// Implementation would load from storage
		// For now, trigger automated authentication
	}

	// Perform automated authentication
	return am.AuthenticateAutomated(ctx)
}

// refreshAutomatedOIDCToken refreshes token using automated OIDC authentication
func (am *AuthManager) refreshAutomatedOIDCToken(ctx context.Context) error {
	// For automated OIDC, we re-authenticate to get a fresh token
	return am.AuthenticateAutomated(ctx)
}

// AuthenticateAutomated performs fully automated Identity Center authentication
// using browser automation. Credentials are retrieved from OS keychain.
func (am *AuthManager) AuthenticateAutomated(ctx context.Context) error {
	// Retrieve credentials from vault
	vault := NewCredentialVault()
	creds, err := vault.Retrieve()
	if err != nil {
		return fmt.Errorf("failed to retrieve credentials from keychain: %w", err)
	}

	// Create automated authenticator
	auth := NewAutomatedOIDCAuth(
		creds.Region,
		creds.StartURL,
		creds.Username,
		creds.Password,
	)

	// Set MFA secret if available
	if creds.MFASecret != "" {
		auth.SetMFASecret(creds.MFASecret)
	}

	// Perform automated authentication
	token, err := auth.Authenticate(ctx)
	if err != nil {
		return fmt.Errorf("automated authentication failed: %w", err)
	}

	// Store token
	am.mu.Lock()
	am.token = token.AccessToken
	am.tokenExp = token.ExpiresAt
	am.mu.Unlock()

	// Save token to storage if available
	if am.tokenStore != nil {
		// Convert AutomatedToken to storage format
		// Implementation depends on storage interface
	}

	return nil
}

// SetupAutomatedAuth stores credentials in OS keychain for automated authentication
func SetupAutomatedAuth(username, password, mfaSecret, startURL, region string) error {
	vault := NewCredentialVault()

	creds := &StoredCredentials{
		Username:  username,
		Password:  password,
		MFASecret: mfaSecret,
		StartURL:  startURL,
		Region:    region,
	}

	if err := vault.Store(creds); err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}

	return nil
}

// RemoveAutomatedAuth removes stored credentials from OS keychain
func RemoveAutomatedAuth() error {
	vault := NewCredentialVault()
	if err := vault.Delete(); err != nil {
		return fmt.Errorf("failed to delete credentials: %w", err)
	}
	return nil
}

// HasAutomatedAuth checks if automated auth credentials are stored
func HasAutomatedAuth() bool {
	vault := NewCredentialVault()
	return vault.Exists()
}
