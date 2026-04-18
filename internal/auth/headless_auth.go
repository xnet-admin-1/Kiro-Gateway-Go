package auth

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/pquerna/otp/totp"
	"github.com/yourusername/kiro-gateway-go/internal/auth/credentials"
	"github.com/yourusername/kiro-gateway-go/internal/auth/oidc"
	"github.com/yourusername/kiro-gateway-go/internal/storage"
)

// HeadlessAuthManager manages headless SSO OIDC authentication
type HeadlessAuthManager struct {
	oidcClient *oidc.Client
	tokenStore *TokenStore
	
	// Configuration
	region    string
	startURL  string
	accountID string
	roleName  string
	
	// Browser automation
	automateAuth bool
	username     string
	password     string
	totpSecret   string // TOTP secret for automated MFA
	
	// State
	mu          sync.RWMutex
	credentials *credentials.Credentials
	credExpiry  time.Time

	// Device flow state (exposed to admin panel)
	deviceFlowMu       sync.RWMutex
	deviceFlowActive   bool
	deviceUserCode     string
	deviceVerifyURL    string
	deviceExpiresAt    time.Time
	deviceFlowDone     chan struct{}
	onDeviceFlowUpdate func() // callback when state changes
	
	// Pre-registered client (optional)
	preRegisteredClientID     string
	preRegisteredClientSecret string
	preRegisteredClientExpiry int64
}

// HeadlessAuthConfig contains configuration for headless authentication
type HeadlessAuthConfig struct {
	Region    string
	StartURL  string
	AccountID string
	RoleName  string
	
	// Optional: Pre-registered client
	ClientID     string
	ClientSecret string
	ClientExpiry int64
	
	// Optional: Browser automation credentials
	AutomateAuth bool   // Enable browser automation
	Username     string // IAM Identity Center username
	Password     string // IAM Identity Center password
	TOTPSecret   string // TOTP secret for automated MFA (base32 encoded)
}

// NewHeadlessAuthManager creates a new headless auth manager
func NewHeadlessAuthManager(config HeadlessAuthConfig, store storage.Store) *HeadlessAuthManager {
	oidcClient := oidc.NewClient(config.Region, config.StartURL)
	
	// Set pre-registered client if provided
	if config.ClientID != "" && config.ClientSecret != "" {
		oidcClient.SetClientRegistration(config.ClientID, config.ClientSecret, config.ClientExpiry)
	}
	
	return &HeadlessAuthManager{
		oidcClient:                oidcClient,
		tokenStore:                NewTokenStore(store),
		region:                    config.Region,
		startURL:                  config.StartURL,
		accountID:                 config.AccountID,
		roleName:                  config.RoleName,
		automateAuth:              config.AutomateAuth,
		username:                  config.Username,
		password:                  config.Password,
		totpSecret:                config.TOTPSecret,
		preRegisteredClientID:     config.ClientID,
		preRegisteredClientSecret: config.ClientSecret,
		preRegisteredClientExpiry: config.ClientExpiry,
	}
}

// Initialize sets up headless authentication
func (m *HeadlessAuthManager) Initialize(ctx context.Context) error {
	log.Println("[HEADLESS] Initializing headless authentication...")
	
	// Try to load existing token
	token, err := m.tokenStore.LoadToken()
	if err == nil {
		log.Println("[HEADLESS] Found existing token")
		
		// Validate token configuration matches
		if token.Region != m.region || token.StartURL != m.startURL ||
			token.AccountID != m.accountID || token.RoleName != m.roleName {
			log.Println("[HEADLESS] Token configuration mismatch, re-authenticating...")
			return m.performDeviceFlow(ctx)
		}
		
		// Check if token is expired
		if token.IsExpired() {
			log.Println("[HEADLESS] Token expired, attempting refresh...")
			if err := m.refreshAccessToken(ctx, token); err != nil {
				log.Printf("[HEADLESS] Token refresh failed: %v, re-authenticating...\n", err)
				return m.performDeviceFlow(ctx)
			}
		}
		
		// Get AWS credentials
		return m.refreshAWSCredentials(ctx, token)
	}
	
	// No existing token, perform device flow
	log.Println("[HEADLESS] No existing token found, starting device flow...")
	return m.performDeviceFlow(ctx)
}

// performDeviceFlow executes the complete device authorization flow
func (m *HeadlessAuthManager) performDeviceFlow(ctx context.Context) error {
	log.Println("[HEADLESS] Starting device authorization flow...")
	
	// 1. Register client (or use cached)
	registration, err := m.oidcClient.RegisterClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to register client: %w", err)
	}
	log.Printf("[HEADLESS] Client registered: %s\n", registration.ClientID)
	
	// 2. Start device authorization
	authResp, err := m.oidcClient.StartDeviceAuthorization(ctx)
	if err != nil {
		return fmt.Errorf("failed to start device authorization: %w", err)
	}
	
	// 3. Display instructions to user (or automate)
	if m.shouldAutomateAuth() {
		log.Println("[HEADLESS] Automating browser authorization...")
		if err := m.automateAuthorization(ctx, authResp); err != nil {
			log.Printf("[HEADLESS] Browser automation failed: %v\n", err)
			log.Println("[HEADLESS] Falling back to manual authorization...")
			m.displayAuthInstructions(authResp)
		} else {
			log.Println("[HEADLESS] Browser automation successful, waiting for token...")
		}
	} else {
		m.displayAuthInstructions(authResp)
	}
	
	// 4. Poll for token
	log.Println("[HEADLESS] Waiting for user authorization...")
	tokenResp, err := m.oidcClient.PollForToken(ctx, authResp.DeviceCode, authResp.Interval)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}
	
	log.Println("[HEADLESS] [SUCCESS] Authorization successful!")
	
	// 5. Save token
	token := &StoredToken{
		AccessToken:     tokenResp.AccessToken,
		RefreshToken:    tokenResp.RefreshToken,
		TokenType:       tokenResp.TokenType,
		ExpiresAt:       time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		ClientID:        registration.ClientID,
		ClientSecret:    registration.ClientSecret,
		ClientExpiresAt: registration.ExpiresAt,
		Region:          m.region,
		StartURL:        m.startURL,
		AccountID:       m.accountID,
		RoleName:        m.roleName,
	}
	
	if err := m.tokenStore.SaveToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}
	
	// 6. Get AWS credentials
	return m.refreshAWSCredentials(ctx, token)
}

// displayAuthInstructions shows user how to authorize the device
// Now includes TOTP helper in a single unified display
func (m *HeadlessAuthManager) displayAuthInstructions(auth *oidc.DeviceAuthResponse) {
	// Check if TOTP secret is available
	totpSecret := m.totpSecret
	if totpSecret == "" {
		totpSecret = os.Getenv("SSO_MFA_TOTP_SECRET")
	}
	if totpSecret == "" {
		if data, err := os.ReadFile("/run/secrets/mfa_totp_secret"); err == nil {
			totpSecret = string(data)
		}
	}
	
	// Define professional styles
	var (
		subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
		highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
		special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
		warning   = lipgloss.AdaptiveColor{Light: "#FFA500", Dark: "#FFB84D"}
		
		boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(highlight).
			Padding(1, 2).
			Width(70)
		
		titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(highlight).
			Align(lipgloss.Center).
			Width(70)
		
		labelStyle = lipgloss.NewStyle().
			Foreground(subtle).
			Bold(true)
		
		urlStyle = lipgloss.NewStyle().
			Foreground(special).
			Underline(true)
		
		codeStyle = lipgloss.NewStyle().
			Foreground(highlight).
			Bold(true).
			Align(lipgloss.Center).
			Width(70).
			Padding(1, 0)
		
		totpCodeStyle = lipgloss.NewStyle().
			Foreground(special).
			Bold(true).
			Align(lipgloss.Center).
			Width(70).
			Padding(1, 0)
		
		timerStyle = lipgloss.NewStyle().
			Foreground(warning).
			Italic(true).
			Align(lipgloss.Center).
			Width(70)
		
		infoStyle = lipgloss.NewStyle().
			Foreground(subtle).
			Italic(true).
			Align(lipgloss.Center).
			Width(70)
		
		dividerStyle = lipgloss.NewStyle().
			Foreground(subtle).
			Align(lipgloss.Center).
			Width(70)
	)
	
	// Build the display
	title := titleStyle.Render("AWS IAM Identity Center Authentication")
	
	urlLabel := labelStyle.Render("Authorization URL:")
	url := urlStyle.Render(auth.VerificationUri)
	
	codeLabel := labelStyle.Render("Device Code:")
	code := codeStyle.Render(auth.UserCode)
	
	var completeSection string
	if auth.VerificationUriComplete != "" {
		completeLabel := labelStyle.Render("Direct Link (auto-fills code):")
		completeURL := urlStyle.Render(auth.VerificationUriComplete)
		completeSection = fmt.Sprintf("\n\n%s\n%s", completeLabel, completeURL)
	}
	
	expiryMsg := fmt.Sprintf("Code expires in %d seconds", auth.ExpiresIn)
	expiry := infoStyle.Render(expiryMsg)
	
	waiting := infoStyle.Render("[*] Waiting for authorization...")
	
	// Build TOTP section if secret is available
	var totpSection string
	if totpSecret != "" {
		// Generate current TOTP code
		currentCode, err := totp.GenerateCode(totpSecret, time.Now())
		if err == nil {
			// Calculate time until code expires
			now := time.Now()
			secondsInPeriod := now.Unix() % 30
			secondsRemaining := 30 - secondsInPeriod
			
			// Generate next code
			nextTime := now.Add(time.Duration(secondsRemaining) * time.Second)
			nextCode, err := totp.GenerateCode(totpSecret, nextTime)
			if err != nil {
				nextCode = "N/A"
			}
			
			divider := dividerStyle.Render("─────────────────────────────────────────────────────────────────────")
			
			totpTitle := labelStyle.Render("MFA Code (TOTP):")
			currentTOTP := totpCodeStyle.Render(currentCode)
			
			timerMsg := fmt.Sprintf("MFA code expires in %d seconds", secondsRemaining)
			timer := timerStyle.Render(timerMsg)
			
			nextLabel := labelStyle.Render("Next Code:")
			nextTOTP := totpCodeStyle.Render(nextCode)
			
			totpSection = fmt.Sprintf(
				"\n\n%s\n\n%s\n%s\n\n%s\n\n%s\n%s",
				divider,
				totpTitle, currentTOTP,
				timer,
				nextLabel, nextTOTP,
			)
			
			// Log for debugging (redacted in production)
			if os.Getenv("DEBUG") == "true" || os.Getenv("DEBUG") == "1" {
				log.Printf("[TOTP] Current code: %s (expires in %ds)\n", currentCode, secondsRemaining)
			} else {
				log.Printf("[TOTP] MFA code generated (expires in %ds)\n", secondsRemaining)
			}
		}
	}
	
	// Combine all sections
	content := fmt.Sprintf(
		"%s\n\n%s\n%s\n\n%s\n%s%s%s\n\n%s\n%s",
		title,
		urlLabel, url,
		codeLabel, code,
		completeSection,
		totpSection,
		waiting, expiry,
	)
	
	// Render the box
	box := boxStyle.Render(content)
	fmt.Printf("\n%s\n\n", box)
}

// refreshAccessToken refreshes the OIDC access token using refresh token
func (m *HeadlessAuthManager) refreshAccessToken(ctx context.Context, token *StoredToken) error {
	if token.RefreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}
	
	log.Println("[HEADLESS] Refreshing access token...")
	
	// Restore client registration
	if token.ClientID != "" && token.ClientSecret != "" {
		m.oidcClient.SetClientRegistration(token.ClientID, token.ClientSecret, token.ClientExpiresAt.Unix())
	}
	
	tokenResp, err := m.oidcClient.RefreshToken(ctx, token.RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}
	
	// Update token
	token.AccessToken = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		token.RefreshToken = tokenResp.RefreshToken
	}
	token.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	
	if err := m.tokenStore.SaveToken(token); err != nil {
		return fmt.Errorf("failed to save refreshed token: %w", err)
	}
	
	log.Println("[HEADLESS] [SUCCESS] Access token refreshed")
	return nil
}

// refreshAWSCredentials gets AWS credentials from SSO using the access token
func (m *HeadlessAuthManager) refreshAWSCredentials(ctx context.Context, token *StoredToken) error {
	log.Println("[HEADLESS] Getting AWS credentials from SSO...")
	
	// Use the SSO service to get role credentials
	ssoProvider, err := credentials.NewSSOProvider(credentials.SSOProviderConfig{
		Region:    m.region,
		AccountID: m.accountID,
		RoleName:  m.roleName,
		AccessTokenProvider: func() (string, error) {
			return token.AccessToken, nil
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create SSO provider: %w", err)
	}
	
	// Retrieve credentials
	creds, err := ssoProvider.Retrieve(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve SSO credentials: %w", err)
	}
	
	m.mu.Lock()
	m.credentials = creds
	m.credExpiry = creds.Expires
	m.mu.Unlock()
	
	log.Printf("[HEADLESS] [SUCCESS] AWS credentials obtained (expires: %s)\n", creds.Expires.Format(time.RFC3339))
	return nil
}

// GetCredentials returns current AWS credentials (with auto-refresh)
func (m *HeadlessAuthManager) GetCredentials(ctx context.Context) (*credentials.Credentials, error) {
	m.mu.RLock()
	creds := m.credentials
	expiry := m.credExpiry
	m.mu.RUnlock()
	
	// Check if credentials need refresh (within 5 min of expiry)
	if time.Until(expiry) < 5*time.Minute {
		log.Println("[HEADLESS] AWS credentials expiring soon, refreshing...")
		
		token, err := m.tokenStore.LoadToken()
		if err != nil {
			return nil, fmt.Errorf("failed to load token: %w", err)
		}
		
		// Check if access token needs refresh
		if token.NeedsRefresh() {
			if err := m.refreshAccessToken(ctx, token); err != nil {
				return nil, fmt.Errorf("failed to refresh access token: %w", err)
			}
		}
		
		if err := m.refreshAWSCredentials(ctx, token); err != nil {
			return nil, fmt.Errorf("failed to refresh AWS credentials: %w", err)
		}
		
		m.mu.RLock()
		creds = m.credentials
		m.mu.RUnlock()
	}
	
	if creds == nil {
		return nil, fmt.Errorf("no credentials available")
	}
	
	return creds, nil
}

// StartBackgroundRefresh starts automatic token refresh in the background
func (m *HeadlessAuthManager) StartBackgroundRefresh(ctx context.Context) {
	log.Println("[HEADLESS] Starting background token refresh...")
	
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			log.Println("[HEADLESS] Background refresh stopped")
			return
		case <-ticker.C:
			token, err := m.tokenStore.LoadToken()
			if err != nil {
				continue
			}
			
			// Refresh access token if needed
			if token.NeedsRefresh() && token.RefreshToken != "" {
				if err := m.refreshAccessToken(ctx, token); err != nil {
					log.Printf("[HEADLESS] Background token refresh failed: %v\n", err)
					
					// Check if refresh token is invalid - trigger re-authentication
					if strings.Contains(err.Error(), "invalid_grant") || 
					   strings.Contains(err.Error(), "Invalid refresh token") {
						log.Println("[HEADLESS] Refresh token invalid, triggering re-authentication...")
						
						// Perform device flow in background
						go func() {
							reAuthCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
							defer cancel()
							
							if err := m.performDeviceFlow(reAuthCtx); err != nil {
								log.Printf("[HEADLESS] Automatic re-authentication failed: %v\n", err)
								log.Println("[HEADLESS] Manual intervention required - please restart gateway")
							} else {
								log.Println("[HEADLESS] Automatic re-authentication successful!")
							}
						}()
					}
					
					continue
				}
			}
			
			// Refresh AWS credentials if needed
			m.mu.RLock()
			needsRefresh := time.Until(m.credExpiry) < 5*time.Minute
			m.mu.RUnlock()
			
			if needsRefresh {
				if err := m.refreshAWSCredentials(ctx, token); err != nil {
					log.Printf("[HEADLESS] Background credential refresh failed: %v\n", err)
				}
			}
		}
	}
}


// shouldAutomateAuth checks if browser automation should be used
func (m *HeadlessAuthManager) shouldAutomateAuth() bool {
	return m.automateAuth && m.username != "" && m.password != ""
}

// automateAuthorization uses browser automation to complete the authorization
func (m *HeadlessAuthManager) automateAuthorization(ctx context.Context, authResp *oidc.DeviceAuthResponse) error {
	automator := oidc.NewBrowserAutomator()
	
	// Try automation with retries
	return automator.AutomateDeviceAuthorizationWithRetry(
		ctx,
		authResp,
		m.username,
		m.password,
		m.totpSecret,
		3, // max retries
	)
}

// GetAccessToken returns the SSO access token
// This is used for CodeWhisperer APIs that require bearer token authentication
func (m *HeadlessAuthManager) GetAccessToken(ctx context.Context) (string, error) {
	token, err := m.tokenStore.LoadToken()
	if err != nil {
		return "", fmt.Errorf("failed to load token: %w", err)
	}
	
	// Check if token is expired
	if token.IsExpired() {
		// Try to refresh
		if err := m.refreshAccessToken(ctx, token); err != nil {
			return "", fmt.Errorf("failed to refresh token: %w", err)
		}
		
		// Reload token after refresh
		token, err = m.tokenStore.LoadToken()
		if err != nil {
			return "", fmt.Errorf("failed to reload token after refresh: %w", err)
		}
	}
	
	return token.AccessToken, nil
}

// DeviceFlowState returns the current device flow state for the admin panel
func (m *HeadlessAuthManager) DeviceFlowState() map[string]interface{} {
	m.deviceFlowMu.RLock()
	defer m.deviceFlowMu.RUnlock()
	return map[string]interface{}{
		"active":     m.deviceFlowActive,
		"user_code":  m.deviceUserCode,
		"verify_url": m.deviceVerifyURL,
		"expires_at": m.deviceExpiresAt,
	}
}

// SetDeviceFlowCallback sets a callback for when device flow state changes
func (m *HeadlessAuthManager) SetDeviceFlowCallback(fn func()) {
	m.deviceFlowMu.Lock()
	m.onDeviceFlowUpdate = fn
	m.deviceFlowMu.Unlock()
}

// InitializeNonBlocking starts auth - returns immediately if device flow is needed
func (m *HeadlessAuthManager) InitializeNonBlocking(ctx context.Context) error {
	log.Println("[HEADLESS] Initializing headless authentication...")

	token, err := m.tokenStore.LoadToken()
	if err == nil && token != nil {
		log.Println("[HEADLESS] Found existing token")
		if token.Region != m.region || token.StartURL != m.startURL {
			log.Println("[HEADLESS] Token configuration mismatch, need re-auth")
		} else if time.Now().After(token.ExpiresAt) {
			log.Println("[HEADLESS] Token expired, attempting refresh...")
			if err := m.refreshAccessToken(ctx, token); err != nil {
				log.Printf("[HEADLESS] Token refresh failed: %v\n", err)
			} else {
				return m.refreshAWSCredentials(ctx, token)
			}
		} else {
			return m.refreshAWSCredentials(ctx, token)
		}
	}

	log.Println("[HEADLESS] No valid token. Run 'login' in admin terminal to authenticate.")
	return nil
}

// StartLogin begins the OIDC device flow. Returns URL and code, or error.
func (m *HeadlessAuthManager) StartLogin(ctx context.Context) (url string, code string, err error) {
	m.deviceFlowMu.Lock()
	if m.deviceFlowActive {
		url = m.deviceVerifyURL
		code = m.deviceUserCode
		m.deviceFlowMu.Unlock()
		return url, code, nil
	}
	m.deviceFlowMu.Unlock()

	log.Println("[HEADLESS] Starting device authorization flow...")

	registration, regErr := m.oidcClient.RegisterClient(ctx)
	if regErr != nil {
		return "", "", fmt.Errorf("failed to register client: %w", regErr)
	}

	authResp, authErr := m.oidcClient.StartDeviceAuthorization(ctx)
	if authErr != nil {
		return "", "", fmt.Errorf("failed to start device authorization: %w", authErr)
	}

	verifyURL := authResp.VerificationUriComplete
	if verifyURL == "" {
		verifyURL = authResp.VerificationUri
	}

	// Set state
	m.deviceFlowMu.Lock()
	m.deviceFlowActive = true
	m.deviceUserCode = authResp.UserCode
	m.deviceVerifyURL = verifyURL
	m.deviceExpiresAt = time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second)
	cb := m.onDeviceFlowUpdate
	m.deviceFlowMu.Unlock()

	if cb != nil {
		cb()
	}

	// Poll for token in background
	go func() {
		tokenResp, pollErr := m.oidcClient.PollForToken(ctx, authResp.DeviceCode, authResp.Interval)

		m.deviceFlowMu.Lock()
		m.deviceFlowActive = false
		m.deviceUserCode = ""
		m.deviceVerifyURL = ""
		m.deviceFlowMu.Unlock()

		if pollErr != nil {
			log.Printf("[HEADLESS] Device flow failed: %v\n", pollErr)
			if cb != nil {
				cb()
			}
			return
		}

		log.Println("[HEADLESS] [SUCCESS] Authorization successful!")

		token := &StoredToken{
			AccessToken:     tokenResp.AccessToken,
			RefreshToken:    tokenResp.RefreshToken,
			TokenType:       tokenResp.TokenType,
			ExpiresAt:       time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
			ClientID:        registration.ClientID,
			ClientSecret:    registration.ClientSecret,
			ClientExpiresAt: registration.ExpiresAt,
			Region:          m.region,
			StartURL:        m.startURL,
			AccountID:       m.accountID,
			RoleName:        m.roleName,
		}

		if saveErr := m.tokenStore.SaveToken(token); saveErr != nil {
			log.Printf("[HEADLESS] Failed to save token: %v\n", saveErr)
		}

		if credErr := m.refreshAWSCredentials(ctx, token); credErr != nil {
			log.Printf("[HEADLESS] Failed to get AWS credentials: %v\n", credErr)
		}

		// Start background refresh
		go m.StartBackgroundRefresh(ctx)

		if cb != nil {
			cb()
		}
	}()

	return verifyURL, authResp.UserCode, nil
}

// WaitForDeviceFlow blocks until the device flow completes (if active)
func (m *HeadlessAuthManager) WaitForDeviceFlow() {
	m.deviceFlowMu.RLock()
	done := m.deviceFlowDone
	m.deviceFlowMu.RUnlock()
	if done != nil {
		<-done
	}
}

// IsAuthenticated returns true if we have valid credentials
func (m *HeadlessAuthManager) IsAuthenticated() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.credentials != nil && time.Now().Before(m.credExpiry)
}

// performDeviceFlowWithState is like performDeviceFlow but exposes state
func (m *HeadlessAuthManager) performDeviceFlowWithState(ctx context.Context) error {
	log.Println("[HEADLESS] Starting device authorization flow...")

	registration, err := m.oidcClient.RegisterClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to register client: %w", err)
	}
	log.Printf("[HEADLESS] Client registered: %s\n", registration.ClientID)

	authResp, err := m.oidcClient.StartDeviceAuthorization(ctx)
	if err != nil {
		return fmt.Errorf("failed to start device authorization: %w", err)
	}

	// Expose device flow state
	m.deviceFlowMu.Lock()
	m.deviceFlowActive = true
	m.deviceUserCode = authResp.UserCode
	m.deviceVerifyURL = authResp.VerificationUriComplete
	if m.deviceVerifyURL == "" {
		m.deviceVerifyURL = authResp.VerificationUri
	}
	m.deviceExpiresAt = time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second)
	cb := m.onDeviceFlowUpdate
	m.deviceFlowMu.Unlock()

	if cb != nil {
		cb()
	}

	m.displayAuthInstructions(authResp)

	// If browser automation is configured, try it
	if m.shouldAutomateAuth() {
		log.Println("[HEADLESS] Automating browser authorization...")
		if err := m.automateAuthorization(ctx, authResp); err != nil {
			log.Printf("[HEADLESS] Browser automation failed: %v\n", err)
			log.Println("[HEADLESS] Waiting for manual authorization via admin panel...")
		} else {
			log.Println("[HEADLESS] Browser automation successful, waiting for token...")
		}
	}

	// Poll for token
	tokenResp, err := m.oidcClient.PollForToken(ctx, authResp.DeviceCode, authResp.Interval)
	if err != nil {
		m.deviceFlowMu.Lock()
		m.deviceFlowActive = false
		m.deviceFlowMu.Unlock()
		if cb != nil {
			cb()
		}
		return fmt.Errorf("failed to get token: %w", err)
	}

	log.Println("[HEADLESS] [SUCCESS] Authorization successful!")

	// Clear device flow state
	m.deviceFlowMu.Lock()
	m.deviceFlowActive = false
	m.deviceUserCode = ""
	m.deviceVerifyURL = ""
	m.deviceFlowMu.Unlock()

	token := &StoredToken{
		AccessToken:     tokenResp.AccessToken,
		RefreshToken:    tokenResp.RefreshToken,
		TokenType:       tokenResp.TokenType,
		ExpiresAt:       time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		ClientID:        registration.ClientID,
		ClientSecret:    registration.ClientSecret,
		ClientExpiresAt: registration.ExpiresAt,
		Region:          m.region,
		StartURL:        m.startURL,
		AccountID:       m.accountID,
		RoleName:        m.roleName,
	}

	if err := m.tokenStore.SaveToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	err = m.refreshAWSCredentials(ctx, token)
	if cb != nil {
		cb()
	}
	return err
}

// GetTokenStore returns the token store for external management
func (m *HeadlessAuthManager) GetTokenStore() *TokenStore {
	return m.tokenStore
}

// ClearCredentials clears in-memory credentials and stored token
func (m *HeadlessAuthManager) ClearCredentials() {
	m.mu.Lock()
	m.credentials = nil
	m.credExpiry = time.Time{}
	m.mu.Unlock()
	m.tokenStore.DeleteToken()
}
