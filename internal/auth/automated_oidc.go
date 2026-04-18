package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/pquerna/otp/totp"
)

// AutomatedOIDCAuth handles fully automated OIDC authentication using browser automation
type AutomatedOIDCAuth struct {
	region      string
	startURL    string
	username    string
	password    string
	mfaSecret   string // Optional: TOTP secret for MFA
	client      *ssooidc.Client
	headless    bool   // Run browser in headless mode
}

// NewAutomatedOIDCAuth creates a new automated OIDC authenticator
func NewAutomatedOIDCAuth(region, startURL, username, password string) *AutomatedOIDCAuth {
	return &AutomatedOIDCAuth{
		region:   region,
		startURL: startURL,
		username: username,
		password: password,
		client:   ssooidc.New(ssooidc.Options{Region: region}),
		headless: true, // Default to headless
	}
}

// SetMFASecret sets the TOTP secret for MFA
func (a *AutomatedOIDCAuth) SetMFASecret(secret string) {
	a.mfaSecret = secret
}

// SetHeadless sets whether to run browser in headless mode
func (a *AutomatedOIDCAuth) SetHeadless(headless bool) {
	a.headless = headless
}

// Authenticate performs fully automated authentication
func (a *AutomatedOIDCAuth) Authenticate(ctx context.Context) (*AutomatedToken, error) {
	// Step 1: Register client
	reg, err := a.registerClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to register client: %w", err)
	}

	// Step 2: Start device authorization
	auth, err := a.startDeviceAuthorization(ctx, reg)
	if err != nil {
		return nil, fmt.Errorf("failed to start device authorization: %w", err)
	}

	// Step 3: Automate browser to complete authorization
	if err := a.automateDeviceCodeFlow(ctx, auth); err != nil {
		return nil, fmt.Errorf("failed to automate device code flow: %w", err)
	}

	// Step 4: Poll for token
	token, err := a.pollForToken(ctx, reg, auth)
	if err != nil {
		return nil, fmt.Errorf("failed to poll for token: %w", err)
	}

	return token, nil
}

// registerClient registers OIDC client
func (a *AutomatedOIDCAuth) registerClient(ctx context.Context) (*AutomatedDeviceRegistration, error) {
	input := &ssooidc.RegisterClientInput{
		ClientName: aws.String("kiro-gateway-automated"),
		ClientType: aws.String("public"),
		Scopes:     []string{"codewhisperer:completions", "codewhisperer:analysis", "codewhisperer:conversations"},
	}

	output, err := a.client.RegisterClient(ctx, input)
	if err != nil {
		return nil, err
	}

	return &AutomatedDeviceRegistration{
		ClientID:              *output.ClientId,
		ClientSecret:          *output.ClientSecret,
		ClientSecretExpiresAt: time.Unix(output.ClientSecretExpiresAt, 0),
		Region:                a.region,
		StartURL:              a.startURL,
	}, nil
}

// startDeviceAuthorization starts device authorization flow
func (a *AutomatedOIDCAuth) startDeviceAuthorization(ctx context.Context, reg *AutomatedDeviceRegistration) (*AutomatedDeviceAuthorization, error) {
	input := &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     aws.String(reg.ClientID),
		ClientSecret: aws.String(reg.ClientSecret),
		StartUrl:     aws.String(a.startURL),
	}

	output, err := a.client.StartDeviceAuthorization(ctx, input)
	if err != nil {
		return nil, err
	}

	return &AutomatedDeviceAuthorization{
		DeviceCode:              *output.DeviceCode,
		UserCode:                *output.UserCode,
		VerificationURI:         *output.VerificationUri,
		VerificationURIComplete: aws.ToString(output.VerificationUriComplete),
		ExpiresIn:               output.ExpiresIn,
		Interval:                output.Interval,
	}, nil
}

// automateDeviceCodeFlow automates the browser to complete device code flow
func (a *AutomatedOIDCAuth) automateDeviceCodeFlow(ctx context.Context, auth *AutomatedDeviceAuthorization) error {
	// Launch headless browser
	l := launcher.New().
		Headless(a.headless).
		NoSandbox(true).
		Set("disable-blink-features", "AutomationControlled")

	defer l.Cleanup()

	url := l.MustLaunch()
	browser := rod.New().ControlURL(url).MustConnect()
	defer browser.MustClose()

	// Navigate to verification URL (use complete URL with device code pre-filled)
	verificationURL := auth.VerificationURIComplete
	if verificationURL == "" {
		verificationURL = auth.VerificationURI
	}

	page := browser.MustPage(verificationURL)

	// Wait for page to load
	page.MustWaitLoad()

	// If using non-complete URL, fill in device code
	if auth.VerificationURIComplete == "" {
		if err := a.fillDeviceCode(page, auth.UserCode); err != nil {
			return fmt.Errorf("failed to fill device code: %w", err)
		}
	}

	// Fill in username
	if err := a.fillUsername(page); err != nil {
		return fmt.Errorf("failed to fill username: %w", err)
	}

	// Fill in password
	if err := a.fillPassword(page); err != nil {
		return fmt.Errorf("failed to fill password: %w", err)
	}

	// Click sign in button
	if err := a.clickSignIn(page); err != nil {
		return fmt.Errorf("failed to click sign in: %w", err)
	}

	// Wait for potential MFA page
	time.Sleep(2 * time.Second)

	// Handle MFA if present
	if a.mfaSecret != "" {
		if err := a.handleMFA(page); err != nil {
			return fmt.Errorf("failed to handle MFA: %w", err)
		}
	}

	// Wait for device code confirmation page
	page.MustWaitLoad()

	// Look for "Allow" or "Approve" button
	if err := a.clickAllow(page); err != nil {
		return fmt.Errorf("failed to click allow: %w", err)
	}

	// Wait for success page
	page.MustWaitLoad()

	return nil
}

// fillDeviceCode fills in the device code
func (a *AutomatedOIDCAuth) fillDeviceCode(page *rod.Page, userCode string) error {
	// Try common device code input selectors
	selectors := []string{
		"#device_code",
		"input[name='device_code']",
		"input[type='text']",
		"input[placeholder*='code']",
	}

	for _, selector := range selectors {
		elem, err := page.Element(selector)
		if err == nil {
			elem.MustInput(userCode)
			return nil
		}
	}

	return fmt.Errorf("device code input not found")
}

// fillUsername fills in the username
func (a *AutomatedOIDCAuth) fillUsername(page *rod.Page) error {
	// Try common username input selectors
	selectors := []string{
		"#username",
		"#email",
		"input[name='username']",
		"input[name='email']",
		"input[type='email']",
		"input[placeholder*='username']",
		"input[placeholder*='email']",
	}

	for _, selector := range selectors {
		elem, err := page.Element(selector)
		if err == nil {
			elem.MustInput(a.username)
			return nil
		}
	}

	return fmt.Errorf("username input not found")
}

// fillPassword fills in the password
func (a *AutomatedOIDCAuth) fillPassword(page *rod.Page) error {
	// Try common password input selectors
	selectors := []string{
		"#password",
		"input[name='password']",
		"input[type='password']",
		"input[placeholder*='password']",
	}

	for _, selector := range selectors {
		elem, err := page.Element(selector)
		if err == nil {
			elem.MustInput(a.password)
			return nil
		}
	}

	return fmt.Errorf("password input not found")
}

// clickSignIn clicks the sign in button
func (a *AutomatedOIDCAuth) clickSignIn(page *rod.Page) error {
	// Try common sign in button selectors
	selectors := []string{
		"button[type='submit']",
		"button[name='signIn']",
		"button[name='login']",
		"input[type='submit']",
		"button:contains('Sign in')",
		"button:contains('Log in')",
		"button:contains('Continue')",
	}

	for _, selector := range selectors {
		elem, err := page.Element(selector)
		if err == nil {
			elem.MustClick()
			return nil
		}
	}

	return fmt.Errorf("sign in button not found")
}

// handleMFA handles MFA if required
func (a *AutomatedOIDCAuth) handleMFA(page *rod.Page) error {
	// Check if MFA page is present
	selectors := []string{
		"#mfa_code",
		"#totp_code",
		"input[name='mfa_code']",
		"input[name='totp']",
		"input[placeholder*='code']",
	}

	var mfaInput *rod.Element
	for _, selector := range selectors {
		elem, err := page.Element(selector)
		if err == nil {
			mfaInput = elem
			break
		}
	}

	if mfaInput == nil {
		// No MFA required
		return nil
	}

	// Generate TOTP code
	totpCode, err := totp.GenerateCode(a.mfaSecret, time.Now())
	if err != nil {
		return fmt.Errorf("failed to generate TOTP code: %w", err)
	}

	// Fill in MFA code
	mfaInput.MustInput(totpCode)

	// Submit MFA
	submitSelectors := []string{
		"button[type='submit']",
		"button[name='verify']",
		"button:contains('Verify')",
		"button:contains('Submit')",
	}

	for _, selector := range submitSelectors {
		elem, err := page.Element(selector)
		if err == nil {
			elem.MustClick()
			break
		}
	}

	// Wait for next page
	page.MustWaitLoad()

	return nil
}

// clickAllow clicks the allow/approve button
func (a *AutomatedOIDCAuth) clickAllow(page *rod.Page) error {
	// Try common allow/approve button selectors
	selectors := []string{
		"button[name='allow']",
		"button[name='approve']",
		"button:contains('Allow')",
		"button:contains('Approve')",
		"button:contains('Authorize')",
		"button:contains('Continue')",
	}

	for _, selector := range selectors {
		elem, err := page.Element(selector)
		if err == nil {
			elem.MustClick()
			return nil
		}
	}

	return fmt.Errorf("allow button not found")
}

// pollForToken polls for access token
func (a *AutomatedOIDCAuth) pollForToken(ctx context.Context, reg *AutomatedDeviceRegistration, auth *AutomatedDeviceAuthorization) (*AutomatedToken, error) {
	interval := time.Duration(auth.Interval) * time.Second
	timeout := time.Duration(auth.ExpiresIn) * time.Second
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		input := &ssooidc.CreateTokenInput{
			ClientId:     aws.String(reg.ClientID),
			ClientSecret: aws.String(reg.ClientSecret),
			GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
			DeviceCode:   aws.String(auth.DeviceCode),
		}

		output, err := a.client.CreateToken(ctx, input)
		if err != nil {
			// Check if authorization is pending
			if isAuthorizationPending(err) {
				time.Sleep(interval)
				continue
			}

			// Check if we need to slow down
			if isSlowDown(err) {
				interval = interval * 2
				time.Sleep(interval)
				continue
			}

			return nil, err
		}

		// Success! Return token
		return &AutomatedToken{
			AccessToken:  *output.AccessToken,
			RefreshToken: aws.ToString(output.RefreshToken),
			ExpiresAt:    time.Now().Add(time.Duration(output.ExpiresIn) * time.Second),
			Region:       a.region,
			StartURL:     a.startURL,
		}, nil
	}

	return nil, fmt.Errorf("device code flow timed out")
}

// Helper functions
func isAuthorizationPending(err error) bool {
	return strings.Contains(err.Error(), "AuthorizationPendingException")
}

func isSlowDown(err error) bool {
	return strings.Contains(err.Error(), "SlowDownException")
}

// AutomatedDeviceRegistration represents device registration for automated auth
type AutomatedDeviceRegistration struct {
	ClientID              string
	ClientSecret          string
	ClientSecretExpiresAt time.Time
	Region                string
	StartURL              string
}

// AutomatedDeviceAuthorization represents device authorization for automated auth
type AutomatedDeviceAuthorization struct {
	DeviceCode              string
	UserCode                string
	VerificationURI         string
	VerificationURIComplete string
	ExpiresIn               int32
	Interval                int32
}

// AutomatedToken represents an authentication token from automated auth
type AutomatedToken struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	Region       string
	StartURL     string
}

// IsExpired checks if the token is expired (with 1-minute safety margin)
func (t *AutomatedToken) IsExpired() bool {
	return time.Now().Add(time.Minute).After(t.ExpiresAt)
}
