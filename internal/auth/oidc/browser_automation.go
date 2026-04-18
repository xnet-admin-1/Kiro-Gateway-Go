package oidc

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/pquerna/otp/totp"
)

// BrowserAutomator handles automated browser-based OIDC authorization
type BrowserAutomator struct {
	browser *rod.Browser
	page    *rod.Page
}

// NewBrowserAutomator creates a new browser automator
func NewBrowserAutomator() *BrowserAutomator {
	return &BrowserAutomator{}
}

// detectChromePath detects the Chrome/Chromium binary path for different environments
func detectChromePath() string {
	// Check environment variable first (for Docker/Alpine)
	if chromeBin := os.Getenv("CHROME_BIN"); chromeBin != "" {
		if _, err := os.Stat(chromeBin); err == nil {
			return chromeBin
		}
	}
	
	// Common Chrome/Chromium paths for different environments
	paths := []string{
		"/usr/bin/chromium-browser",     // Alpine Linux
		"/usr/bin/chromium",              // Some Linux distros
		"/usr/bin/google-chrome",         // Debian/Ubuntu
		"/usr/bin/google-chrome-stable",  // Debian/Ubuntu
		"/usr/bin/chrome",                // Generic
		"/snap/bin/chromium",             // Snap package
	}
	
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	
	// Return empty string to use launcher's default detection
	return ""
}

// isDebugMode checks if debug logging is enabled
func isDebugMode() bool {
	debug := os.Getenv("DEBUG")
	return debug == "true" || debug == "1"
}

// AutomateDeviceAuthorization automates the device authorization flow in a browser
// This eliminates the need for manual user interaction
func (b *BrowserAutomator) AutomateDeviceAuthorization(ctx context.Context, authResp *DeviceAuthResponse, username, password, totpSecret string) error {
	log.Println("[BROWSER] Starting automated browser authorization...")
	
	// Launch browser in headless mode with crash reporter disabled
	l := launcher.New().
		Headless(true). // Set to true for production
		NoSandbox(true).
		Set("disable-crash-reporter").      // Disable crash reporter
		Set("disable-breakpad").             // Disable breakpad crash reporting
		Set("disable-dev-shm-usage").        // Overcome limited resource problems
		Set("disable-gpu").                  // Disable GPU hardware acceleration
		Set("no-first-run").                 // Skip first run wizards
		Set("no-default-browser-check").     // Skip default browser check
		Set("disable-background-networking"). // Disable background networking
		Set("crash-dumps-dir", "/tmp").      // Set crash dumps directory to /tmp
		Set("window-size", "1920,1080").     // Set window size to 1920x1080
		UserDataDir("/tmp/chrome-data")      // Use /tmp for user data (writable)
	
	// Detect Chrome path for Alpine/Docker environments
	chromePath := detectChromePath()
	if chromePath != "" {
		if isDebugMode() {
			log.Printf("[BROWSER] Using Chrome at: %s\n", chromePath)
		}
		l = l.Bin(chromePath)
	}
	
	url, err := l.Launch()
	if err != nil {
		return fmt.Errorf("failed to launch browser: %w", err)
	}
	
	// Connect to browser
	b.browser = rod.New().ControlURL(url).MustConnect()
	defer b.cleanup()
	
	// Create new page
	b.page = b.browser.MustPage()
	
	// Set viewport size to ensure everything is visible
	b.page.MustSetViewport(1920, 1080, 1, false)
	
	// Navigate to verification URL with code
	targetURL := authResp.VerificationUriComplete
	if targetURL == "" {
		targetURL = authResp.VerificationUri
	}
	
	log.Printf("[BROWSER] Navigating to: %s\n", "[SSO_LOGIN_PAGE]")
	if err := b.page.Navigate(targetURL); err != nil {
		return fmt.Errorf("failed to navigate: %w", err)
	}
	log.Println("[BROWSER] Navigation completed")
	
	// Wait for initial page load with timeout
	log.Println("[BROWSER] Waiting for page load...")
	waitErr := b.page.Timeout(15 * time.Second).WaitLoad()
	if waitErr != nil {
		log.Printf("[BROWSER] WaitLoad error (continuing anyway): %v\n", waitErr)
	} else {
		log.Println("[BROWSER] Page load completed")
	}
	
	// Wait for React app to render and any redirects to complete
	// AWS SSO does authentication flow redirects before showing login
	log.Println("[BROWSER] Waiting 10 seconds for React SPA and redirects...")
	time.Sleep(10 * time.Second)
	log.Println("[BROWSER] Wait completed")
	
	// Get current URL to see if we were redirected
	currentURL := b.page.MustInfo().URL
	if isDebugMode() {
		log.Printf("[BROWSER] Current URL: %s\n", currentURL)
	} else {
		log.Println("[BROWSER] Reached authentication page")
	}
	
	// Take screenshot for debugging
	if isDebugMode() {
		log.Println("[BROWSER] Taking screenshot...")
		b.takeScreenshot("01-after-navigation")
		log.Println("[BROWSER] Screenshot taken")
	}
	
	// Handle cookie consent banner and device code confirmation
	// These are combined because clicking "Confirm and continue" happens
	// after accepting cookies on the same page
	log.Println("[BROWSER] Checking for cookie consent banner...")
	deviceCodeConfirmed, err := b.handleCookieConsent()
	if err != nil {
		log.Printf("[BROWSER] Cookie consent handling: %v\n", err)
		// Continue anyway - not a critical error
	}
	
	// If device code wasn't confirmed in cookie consent handler, try separately
	if !deviceCodeConfirmed {
		log.Println("[BROWSER] Looking for device code confirmation button...")
		if err := b.confirmDeviceCode(); err != nil {
			// Device code confirmation failed - this might mean we're already on the login page
			// Check if we're on the login page (has username field or "Next" button)
			log.Printf("[BROWSER] Device code confirmation failed: %v\n", err)
			log.Println("[BROWSER] Checking if we're on the login page instead...")
			
			onLoginPage, checkErr := b.isOnLoginPage()
			if checkErr != nil {
				log.Printf("[BROWSER] Error checking for login page: %v\n", checkErr)
			}
			
			if onLoginPage {
				log.Println("[BROWSER] Already on login page - device code was auto-confirmed or not required")
				// Continue with login flow below
			} else {
				// Not on login page and device code confirmation failed
				if isDebugMode() {
					b.takeScreenshot("01b-device-code-error")
				}
				return fmt.Errorf("failed to confirm device code: %w", err)
			}
		}
	}
	
	// Wait for redirect after device code confirmation
	log.Println("[BROWSER] Waiting for redirect after device code confirmation...")
	time.Sleep(5 * time.Second)
	
	if isDebugMode() {
		b.takeScreenshot("01c-after-device-confirm")
	}
	
	// Check if we're already on the authorization approval page
	// This happens when the user is already authenticated (e.g., from a previous session)
	log.Println("[BROWSER] Checking if already on authorization approval page...")
	onApprovalPage, err := b.isOnApprovalPage()
	if err != nil {
		log.Printf("[BROWSER] Error checking for approval page: %v\n", err)
	}
	
	if onApprovalPage {
		log.Println("[BROWSER] Already on authorization approval page - skipping sign-in and MFA")
		// Skip directly to approval
		log.Println("[BROWSER] Approving authorization...")
		if err := b.approveAuthorization(); err != nil {
			if isDebugMode() {
				b.takeScreenshot("06-approval-error")
			}
			return fmt.Errorf("failed to approve authorization: %w", err)
		}
		
		if isDebugMode() {
			b.takeScreenshot("07-after-approval")
		}
		
		log.Println("[BROWSER] Waiting for AWS to process approval...")
		time.Sleep(3 * time.Second)
		
		log.Println("[BROWSER] Automated authorization completed successfully!")
		return nil
	}
	
	// Not on approval page yet, proceed with sign-in flow
	log.Println("[BROWSER] Not on approval page - proceeding with sign-in...")
	log.Println("[BROWSER] Handling sign-in...")
	if err := b.handleSignIn(username, password); err != nil {
		if isDebugMode() {
			b.takeScreenshot("02-signin-error")
		}
		return fmt.Errorf("failed to sign in: %w", err)
	}
	
	// Wait for sign-in to process and next page to load
	log.Println("[BROWSER] Waiting for sign-in to process...")
	time.Sleep(5 * time.Second)
	if isDebugMode() {
		b.takeScreenshot("03-after-signin")
	}
	
	// Check if MFA is required
	log.Println("[BROWSER] Checking for MFA...")
	if err := b.handleMFA(totpSecret); err != nil {
		if isDebugMode() {
			b.takeScreenshot("04-mfa-error")
		}
		return fmt.Errorf("failed to handle MFA: %w", err)
	}
	
	// Wait for MFA to process
	log.Println("[BROWSER] Waiting for MFA to process...")
	time.Sleep(5 * time.Second)
	if isDebugMode() {
		b.takeScreenshot("05-after-mfa")
	}
	
	// Approve authorization
	log.Println("[BROWSER] Approving authorization...")
	if err := b.approveAuthorization(); err != nil {
		if isDebugMode() {
			b.takeScreenshot("06-approval-error")
		}
		return fmt.Errorf("failed to approve authorization: %w", err)
	}
	
	if isDebugMode() {
		b.takeScreenshot("07-after-approval")
	}
	
	// Wait for AWS to process the approval on the backend
	log.Println("[BROWSER] Waiting for AWS to process approval...")
	time.Sleep(3 * time.Second)
	
	log.Println("[BROWSER] Automated authorization completed successfully!")
	return nil
}

// enterDeviceCode enters the device code if not auto-filled
func (b *BrowserAutomator) enterDeviceCode(code string) error {
	// Try to find code input field
	// Common selectors for AWS SSO device code input
	selectors := []string{
		"input[name='user_code']",
		"input[id='user_code']",
		"input[type='text']",
		"input.code-input",
	}
	
	for _, selector := range selectors {
		elem := b.page.MustElement(selector)
		if elem != nil {
			if err := elem.Input(code); err != nil {
				continue
			}
			
			// Click submit button
			submitSelectors := []string{
				"button[type='submit']",
				"input[type='submit']",
				"button.submit",
				"button:contains('Submit')",
				"button:contains('Continue')",
			}
			
			for _, submitSel := range submitSelectors {
				submitBtn := b.page.MustElement(submitSel)
				if submitBtn != nil {
					submitBtn.MustClick()
					time.Sleep(2 * time.Second)
					return nil
				}
			}
		}
	}
	
	return fmt.Errorf("could not find device code input field")
}

// confirmDeviceCode clicks the "Confirm and continue" button on the device code page
func (b *BrowserAutomator) confirmDeviceCode() error {
	log.Println("[BROWSER] Looking for 'Confirm and continue' button...")
	
	// Get all buttons
	buttons, err := b.page.Elements("button")
	if err != nil {
		return fmt.Errorf("failed to get buttons: %w", err)
	}
	
	log.Printf("[BROWSER] Found %d buttons on page\n", len(buttons))
	
	// Look for "Confirm and continue" button
	var confirmButton *rod.Element
	for i, button := range buttons {
		buttonText, _ := button.Text()
		buttonID, _ := button.Property("id")
		
		idStr := buttonID.String()
		textLower := strings.ToLower(buttonText)
		
		log.Printf("[BROWSER] Button[%d]: id=%s, text='%s'\n", i, idStr, buttonText)
		
		// Look for "Confirm and continue" button (device code verification)
		if (strings.Contains(textLower, "confirm") && strings.Contains(textLower, "continue")) ||
			idStr == "cli_verification_btn" {
			confirmButton = button
			log.Printf("[BROWSER] Found device code confirmation button[%d]: '%s'\n", i, buttonText)
			break
		}
	}
	
	if confirmButton == nil {
		return fmt.Errorf("could not find 'Confirm and continue' button")
	}
	
	// Click button
	log.Println("[BROWSER] Clicking 'Confirm and continue' button...")
	_, err = confirmButton.Eval("() => this.click()")
	if err != nil {
		return fmt.Errorf("failed to click confirmation button: %w", err)
	}
	
	log.Println("[BROWSER] Device code confirmed successfully")
	return nil
}

// handleCookieConsent handles AWS cookie consent banner if present
// Returns true if device code was confirmed, false otherwise
func (b *BrowserAutomator) handleCookieConsent() (bool, error) {
	log.Println("[BROWSER] Looking for cookie consent banner...")
	
	// Wait a moment for banner to appear
	time.Sleep(2 * time.Second)
	
	// Check if there's an iframe (cookie consent might be in an iframe)
	iframes, err := b.page.Elements("iframe")
	if err == nil && len(iframes) > 0 {
		log.Printf("[BROWSER] Found %d iframes on page\n", len(iframes))
		for i, iframe := range iframes {
			src, _ := iframe.Property("src")
			srcStr := src.String()
			log.Printf("[BROWSER] Iframe[%d]: src=%s\n", i, srcStr)
		}
	}
	
	// Look for AWS cookie consent buttons
	// Common patterns: "Accept all", "Reject all", "Customize", etc.
	buttons, err := b.page.Elements("button")
	if err != nil {
		return false, fmt.Errorf("failed to get buttons: %w", err)
	}
	
	log.Printf("[BROWSER] Found %d buttons on page\n", len(buttons))
	
	// Look for cookie consent accept button
	var acceptButton *rod.Element
	for i, button := range buttons {
		buttonText, _ := button.Text()
		buttonID, _ := button.Property("id")
		buttonClass, _ := button.Property("class")
		
		idStr := buttonID.String()
		classStr := buttonClass.String()
		textLower := strings.ToLower(buttonText)
		
		log.Printf("[BROWSER] Button[%d]: id=%s, class=%s, text='%s'\n", i, idStr, classStr, buttonText)
		
		// Look for "Accept" button (AWS cookie consent uses simple "Accept" text)
		// Skip other buttons like "Confirm and continue" which are for device code
		// Also skip "Decline" to avoid clicking the wrong button
		if (textLower == "accept" || 
			textLower == "accept all" || 
			textLower == "accept cookies" ||
			strings.Contains(idStr, "awsccc-cb-btn-accept") ||
			strings.Contains(classStr, "awsccc-u-btn-accept")) &&
			!strings.Contains(textLower, "decline") {
			acceptButton = button
			log.Printf("[BROWSER] Found cookie consent accept button[%d]: '%s'\n", i, buttonText)
			break
		}
	}
	
	if acceptButton == nil {
		log.Println("[BROWSER] No cookie consent banner found - continuing")
		return false, nil
	}
	
	// Click accept button
	log.Println("[BROWSER] Clicking cookie consent accept button...")
	_, err = acceptButton.Eval("() => this.click()")
	if err != nil {
		return false, fmt.Errorf("failed to click accept button: %w", err)
	}
	
	log.Println("[BROWSER] Cookie consent accepted successfully")
	
	// Wait for banner to disappear and page to stabilize
	log.Println("[BROWSER] Waiting for banner to disappear...")
	time.Sleep(3 * time.Second)
	
	// Now look for "Confirm and continue" button (device code confirmation)
	// This appears after accepting cookies
	log.Println("[BROWSER] Looking for 'Confirm and continue' button...")
	buttons, err = b.page.Elements("button")
	if err != nil {
		return false, fmt.Errorf("failed to get buttons after cookie consent: %w", err)
	}
	
	var confirmButton *rod.Element
	for i, button := range buttons {
		buttonText, _ := button.Text()
		buttonID, _ := button.Property("id")
		
		idStr := buttonID.String()
		textLower := strings.ToLower(buttonText)
		
		log.Printf("[BROWSER] Post-cookie Button[%d]: id=%s, text='%s'\n", i, idStr, buttonText)
		
		// Look for "Confirm and continue" button (device code verification)
		if (strings.Contains(textLower, "confirm") && strings.Contains(textLower, "continue")) ||
			idStr == "cli_verification_btn" {
			confirmButton = button
			log.Printf("[BROWSER] Found device code confirmation button[%d]: '%s'\n", i, buttonText)
			break
		}
	}
	
	if confirmButton == nil {
		log.Println("[BROWSER] No 'Confirm and continue' button found after cookie consent")
		return false, nil
	}
	
	// Click button
	log.Println("[BROWSER] Clicking 'Confirm and continue' button...")
	_, err = confirmButton.Eval("() => this.click()")
	if err != nil {
		return false, fmt.Errorf("failed to click confirmation button: %w", err)
	}
	
	log.Println("[BROWSER] Device code confirmed successfully")
	time.Sleep(2 * time.Second)
	
	// Now dismiss the cookie banner completely by clicking "Dismiss" button
	log.Println("[BROWSER] Looking for 'Dismiss' button to close cookie banner...")
	buttons, err = b.page.Elements("button")
	if err == nil {
		for i, button := range buttons {
			buttonText, _ := button.Text()
			textLower := strings.ToLower(buttonText)
			
			if textLower == "dismiss" {
				log.Printf("[BROWSER] Found 'Dismiss' button[%d]: '%s'\n", i, buttonText)
				log.Println("[BROWSER] Clicking 'Dismiss' button to close cookie banner...")
				_, err := button.Eval("() => this.click()")
				if err != nil {
					log.Printf("[BROWSER] Warning: failed to click 'Dismiss': %v\n", err)
				} else {
					log.Println("[BROWSER] Cookie banner dismissed successfully")
					time.Sleep(2 * time.Second)
				}
				break
			}
		}
	}
	
	return true, nil
}

// handleSignIn handles the IAM Identity Center sign-in
// AWS IAM Identity Center uses a two-step login:
// 1. Enter username and click Next
// 2. Enter password and click Sign in
func (b *BrowserAutomator) handleSignIn(username, password string) error {
	log.Println("[BROWSER] Starting handleSignIn (two-step flow)...")
	
	// STEP 1: Enter username
	log.Println("[BROWSER] === STEP 1: Username ===")
	log.Println("[BROWSER] Getting all input elements...")
	inputs, err := b.page.Elements("input")
	if err != nil {
		return fmt.Errorf("failed to get input elements: %w", err)
	}
	
	log.Printf("[BROWSER] Found %d input elements\n", len(inputs))
	
	if len(inputs) == 0 {
		log.Println("[BROWSER] No inputs found, waiting 5 more seconds...")
		time.Sleep(5 * time.Second)
		
		inputs, err = b.page.Elements("input")
		if err != nil {
			return fmt.Errorf("failed to get input elements (retry): %w", err)
		}
		log.Printf("[BROWSER] After retry: Found %d input elements\n", len(inputs))
	}
	
	// Find username field (type=text)
	var usernameField *rod.Element
	for i, input := range inputs {
		inputType, _ := input.Property("type")
		inputName, _ := input.Property("name")
		inputID, _ := input.Property("id")
		
		typeStr := inputType.String()
		nameStr := inputName.String()
		idStr := inputID.String()
		
		log.Printf("[BROWSER] Input[%d]: type=%s, name=%s, id=%s\n", i, typeStr, nameStr, idStr)
		
		// Skip cookie consent checkboxes
		if strings.Contains(idStr, "awsccc-") || strings.Contains(nameStr, "awsccc-") {
			log.Printf("[BROWSER] Skipping cookie consent checkbox: Input[%d]\n", i)
			continue
		}
		
		// Look for text input (not hidden)
		if typeStr == "text" || typeStr == "email" {
			usernameField = input
			log.Printf("[BROWSER] Selected username field: Input[%d]\n", i)
			break
		}
	}
	
	if usernameField == nil {
		return fmt.Errorf("could not find username field")
	}
	
	// Enter username
	log.Println("[BROWSER] Entering username...")
	if err := usernameField.Input(username); err != nil {
		return fmt.Errorf("failed to enter username: %w", err)
	}
	log.Println("[BROWSER] Username entered successfully")
	
	time.Sleep(1 * time.Second)
	
	// Find and click Next button
	log.Println("[BROWSER] Looking for Next button...")
	buttons, err := b.page.Elements("button")
	if err != nil {
		return fmt.Errorf("failed to get button elements: %w", err)
	}
	
	log.Printf("[BROWSER] Found %d button elements\n", len(buttons))
	
	var nextButton *rod.Element
	for i, button := range buttons {
		buttonType, _ := button.Property("type")
		buttonText, _ := button.Text()
		
		typeStr := buttonType.String()
		textLower := strings.ToLower(buttonText)
		
		log.Printf("[BROWSER] Button[%d]: type=%s, text=%s\n", i, typeStr, buttonText)
		
		// Look for Next button or submit button
		if typeStr == "submit" || strings.Contains(textLower, "next") || strings.Contains(textLower, "continue") {
			nextButton = button
			log.Printf("[BROWSER] Selected Next button: Button[%d]\n", i)
			break
		}
	}
	
	if nextButton == nil {
		return fmt.Errorf("could not find Next button")
	}
	
	// Click Next button using JavaScript (more reliable)
	log.Println("[BROWSER] Clicking Next button...")
	_, err = nextButton.Eval("() => this.click()")
	if err != nil {
		return fmt.Errorf("failed to click Next button: %w", err)
	}
	log.Println("[BROWSER] Next button clicked successfully")
	
	// Wait for password page to load
	log.Println("[BROWSER] Waiting for password page to load...")
	time.Sleep(3 * time.Second)
	
	// STEP 2: Enter password
	log.Println("[BROWSER] === STEP 2: Password ===")
	log.Println("[BROWSER] Looking for password field...")
	inputs, err = b.page.Elements("input")
	if err != nil {
		return fmt.Errorf("failed to get input elements for password: %w", err)
	}
	
	log.Printf("[BROWSER] Found %d input elements on password page\n", len(inputs))
	
	var passwordField *rod.Element
	for i, input := range inputs {
		inputType, _ := input.Property("type")
		typeStr := inputType.String()
		
		log.Printf("[BROWSER] Password page - Input[%d]: type=%s\n", i, typeStr)
		
		if typeStr == "password" {
			passwordField = input
			log.Printf("[BROWSER] Found password field: Input[%d]\n", i)
			break
		}
	}
	
	if passwordField == nil {
		return fmt.Errorf("could not find password field on password page")
	}
	
	// Enter password
	log.Println("[BROWSER] Entering password...")
	if err := passwordField.Input(password); err != nil {
		return fmt.Errorf("failed to enter password: %w", err)
	}
	log.Println("[BROWSER] Password entered successfully")
	
	time.Sleep(1 * time.Second)
	
	// Check "This is a trusted device" checkbox
	log.Println("[BROWSER] Looking for 'trusted device' checkbox...")
	
	// Try to find the checkbox by searching for the label text
	// AWS uses labels next to checkboxes, so we'll look for text containing "trusted device"
	labels, err := b.page.Elements("label")
	if err == nil && len(labels) > 0 {
		log.Printf("[BROWSER] Found %d label elements\n", len(labels))
		
		for i, label := range labels {
			labelText, _ := label.Text()
			labelTextLower := strings.ToLower(labelText)
			
			log.Printf("[BROWSER] Label[%d]: '%s'\n", i, labelText)
			
			// Look for "trusted device" or "trust this device" in label text
			if strings.Contains(labelTextLower, "trusted device") || strings.Contains(labelTextLower, "trust this device") {
				log.Printf("[BROWSER] Found 'trusted device' label[%d]: '%s'\n", i, labelText)
				
				// Try to click the label (which should toggle the checkbox)
				log.Println("[BROWSER] Clicking label to check checkbox...")
				_, err := label.Eval("() => this.click()")
				if err != nil {
					log.Printf("[BROWSER] Warning: failed to click label: %v\n", err)
				} else {
					log.Println("[BROWSER] Trusted device checkbox checked successfully")
				}
				break
			}
		}
	} else {
		log.Println("[BROWSER] No labels found or error getting labels")
	}
	
	time.Sleep(1 * time.Second)
	
	// Find and click Sign in button
	log.Println("[BROWSER] Looking for Sign in button...")
	buttons, err = b.page.Elements("button")
	if err != nil {
		return fmt.Errorf("failed to get button elements for sign in: %w", err)
	}
	
	log.Printf("[BROWSER] Found %d button elements on password page\n", len(buttons))
	
	var signInButton *rod.Element
	for i, button := range buttons {
		buttonType, _ := button.Property("type")
		buttonText, _ := button.Text()
		
		typeStr := buttonType.String()
		textLower := strings.ToLower(buttonText)
		
		log.Printf("[BROWSER] Button[%d]: type=%s, text=%s\n", i, typeStr, buttonText)
		
		// Look for Sign in button or submit button
		if typeStr == "submit" || strings.Contains(textLower, "sign in") || strings.Contains(textLower, "login") {
			signInButton = button
			log.Printf("[BROWSER] Selected Sign in button: Button[%d]\n", i)
			break
		}
	}
	
	if signInButton == nil {
		return fmt.Errorf("could not find Sign in button")
	}
	
	// Click Sign in button using JavaScript (more reliable)
	log.Println("[BROWSER] Clicking Sign in button...")
	_, err = signInButton.Eval("() => this.click()")
	if err != nil {
		return fmt.Errorf("failed to click Sign in button: %w", err)
	}
	log.Println("[BROWSER] Sign in button clicked successfully")
	
	return nil
}

// handleMFA handles MFA if required
// If totpSecret is provided, automatically generates and enters TOTP code
// Otherwise, waits for manual entry
func (b *BrowserAutomator) handleMFA(totpSecret string) error {
	log.Println("[BROWSER] Checking if MFA is required...")
	
	// Wait a moment for page to load
	time.Sleep(2 * time.Second)
	
	// Check if there's an MFA input field
	inputs, err := b.page.Elements("input")
	if err != nil {
		log.Println("[BROWSER] Could not get inputs, assuming no MFA required")
		return nil
	}
	
	log.Printf("[BROWSER] Found %d input elements\n", len(inputs))
	
	// Look for MFA code input (usually type=text with specific patterns)
	var mfaField *rod.Element
	for i, input := range inputs {
		inputType, _ := input.Property("type")
		inputName, _ := input.Property("name")
		inputID, _ := input.Property("id")
		placeholder, _ := input.Property("placeholder")
		
		typeStr := inputType.String()
		nameStr := inputName.String()
		idStr := inputID.String()
		placeholderStr := placeholder.String()
		
		log.Printf("[BROWSER] MFA check - Input[%d]: type=%s, name=%s, id=%s, placeholder=%s\n", 
			i, typeStr, nameStr, idStr, placeholderStr)
		
		// Look for MFA-related fields
		nameLower := strings.ToLower(nameStr)
		idLower := strings.ToLower(idStr)
		placeholderLower := strings.ToLower(placeholderStr)
		
		if strings.Contains(nameLower, "mfa") || strings.Contains(nameLower, "code") ||
			strings.Contains(nameLower, "token") || strings.Contains(nameLower, "otp") ||
			strings.Contains(idLower, "mfa") || strings.Contains(idLower, "code") ||
			strings.Contains(idLower, "token") || strings.Contains(idLower, "otp") ||
			strings.Contains(placeholderLower, "code") || strings.Contains(placeholderLower, "token") {
			mfaField = input
			log.Printf("[BROWSER] Found MFA field: Input[%d]\n", i)
			break
		}
	}
	
	if mfaField == nil {
		log.Println("[BROWSER] No MFA field found, skipping MFA step")
		return nil
	}
	
	// MFA is required
	if totpSecret != "" {
		// Automated TOTP
		log.Println("[BROWSER] MFA required - generating TOTP code...")
		
		code, err := generateTOTPCode(totpSecret)
		if err != nil {
			return fmt.Errorf("failed to generate TOTP code: %w", err)
		}
		
		if isDebugMode() {
			log.Printf("[BROWSER] Generated TOTP code: %s\n", code)
		} else {
			log.Println("[BROWSER] Generated TOTP code")
		}
		log.Println("[BROWSER] Entering TOTP code...")
		
		if err := mfaField.Input(code); err != nil {
			return fmt.Errorf("failed to enter TOTP code: %w", err)
		}
		
		log.Println("[BROWSER] TOTP code entered successfully")
		
		// Find and click submit button
		time.Sleep(1 * time.Second)
		
		buttons, err := b.page.Elements("button")
		if err == nil {
			for _, button := range buttons {
				buttonType, _ := button.Property("type")
				buttonText, _ := button.Text()
				
				typeStr := buttonType.String()
				textLower := strings.ToLower(buttonText)
				
				if typeStr == "submit" || strings.Contains(textLower, "verify") || 
					strings.Contains(textLower, "submit") || strings.Contains(textLower, "continue") {
					log.Printf("[BROWSER] Clicking MFA submit button: '%s'\n", buttonText)
					_, err := button.Eval("() => this.click()")
					if err != nil {
						log.Printf("[BROWSER] Warning: failed to click submit button: %v\n", err)
					} else {
						log.Println("[BROWSER] MFA submit button clicked successfully")
					}
					break
				}
			}
		}
		
		log.Println("[BROWSER] Automated TOTP MFA completed")
	} else {
		// Manual MFA entry
		log.Println("[BROWSER] [WARNING] MFA REQUIRED - Browser will stay open for manual MFA entry")
		log.Println("[BROWSER] Please enter your MFA code in the browser window...")
		log.Println("[BROWSER] Waiting 60 seconds for manual MFA entry...")
		
		// Wait for user to enter MFA manually
		time.Sleep(60 * time.Second)
		
		log.Println("[BROWSER] Continuing after MFA wait period...")
	}
	
	return nil
}

// generateTOTPCode generates a TOTP code from a base32-encoded secret
func generateTOTPCode(secret string) (string, error) {
	// Import at top of file: "github.com/pquerna/otp/totp"
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		return "", fmt.Errorf("failed to generate TOTP code: %w", err)
	}
	return code, nil
}

// approveAuthorization approves the authorization request
// AWS SSO has TWO approval pages:
// 1. Device code approval: "Accept" button to approve the device code
// 2. Access approval: "Allow" button to grant access to the application
func (b *BrowserAutomator) approveAuthorization() error {
	// Wait for authorization page to load and stabilize
	log.Println("[BROWSER] Waiting for authorization page to load...")
	time.Sleep(5 * time.Second)
	
	// APPROVAL 1: Device code approval page
	log.Println("[BROWSER] === APPROVAL 1: Device Code ===")
	if err := b.clickApprovalButton("device code approval"); err != nil {
		return err
	}
	
	// Wait for next page to load
	log.Println("[BROWSER] Waiting for access approval page to load...")
	time.Sleep(5 * time.Second)
	
	// APPROVAL 2: Access approval page
	log.Println("[BROWSER] === APPROVAL 2: Access Grant ===")
	if err := b.clickApprovalButton("access approval"); err != nil {
		// This might fail if page closes immediately, which is OK
		log.Printf("[BROWSER] Access approval click result: %v (page may have closed)\n", err)
	}
	
	// Wait for final redirect/close
	log.Println("[BROWSER] Waiting for final redirect...")
	time.Sleep(3 * time.Second)
	
	// Try to get final state, but don't panic if page is gone
	finalInfo, err := b.page.Info()
	if err == nil {
		if isDebugMode() {
			log.Printf("[BROWSER] Final URL: %s\n", finalInfo.URL)
			log.Printf("[BROWSER] Page title: %s\n", finalInfo.Title)
		} else {
			log.Println("[BROWSER] Authorization page reached")
		}
	} else {
		log.Println("[BROWSER] Page closed after approval - authorization completed!")
	}
	
	return nil
}

// isOnLoginPage checks if we're on the login page
// Returns true if username/email input field or "Next" button is found
func (b *BrowserAutomator) isOnLoginPage() (bool, error) {
	// Check for username/email input field
	inputs, err := b.page.Elements("input")
	if err != nil {
		return false, fmt.Errorf("failed to get inputs: %w", err)
	}
	
	for _, input := range inputs {
		inputType, _ := input.Property("type")
		typeStr := inputType.String()
		
		if typeStr == "text" || typeStr == "email" {
			log.Println("[BROWSER] Found text/email input - on login page")
			return true, nil
		}
	}
	
	// Check for "Next" button (typical of AWS login page)
	buttons, err := b.page.Elements("button")
	if err != nil {
		return false, fmt.Errorf("failed to get buttons: %w", err)
	}
	
	for _, button := range buttons {
		buttonText, _ := button.Text()
		textLower := strings.ToLower(buttonText)
		
		if strings.Contains(textLower, "next") || strings.Contains(textLower, "sign in") {
			log.Printf("[BROWSER] Found '%s' button - on login page\n", buttonText)
			return true, nil
		}
	}
	
	return false, nil
}

// isOnApprovalPage checks if we're on the authorization approval page
// Returns true if "Allow" or "Deny" buttons are found
func (b *BrowserAutomator) isOnApprovalPage() (bool, error) {
	buttons, err := b.page.Elements("button")
	if err != nil {
		return false, fmt.Errorf("failed to get buttons: %w", err)
	}
	
	for _, button := range buttons {
		buttonText, _ := button.Text()
		textLower := strings.ToLower(buttonText)
		
		// Look for approval/deny buttons
		if strings.Contains(textLower, "allow") || 
			strings.Contains(textLower, "deny") ||
			strings.Contains(textLower, "approve") {
			log.Printf("[BROWSER] Found approval page button: '%s'\n", buttonText)
			return true, nil
		}
	}
	
	return false, nil
}

// clickApprovalButton finds and clicks an approval button on the current page
func (b *BrowserAutomator) clickApprovalButton(stage string) error {
	// Get current URL
	pageInfo, err := b.page.Info()
	if err != nil {
		return fmt.Errorf("failed to get page info: %w", err)
	}
	
	if isDebugMode() {
		log.Printf("[BROWSER] Current URL: %s\n", pageInfo.URL)
	} else {
		log.Printf("[BROWSER] On %s page\n", stage)
	}
	
	// Get all buttons on the page
	log.Println("[BROWSER] Searching for approve/allow/confirm button...")
	buttons, err := b.page.Elements("button")
	if err != nil {
		return fmt.Errorf("failed to get button elements: %w", err)
	}
	
	log.Printf("[BROWSER] Found %d button elements\n", len(buttons))
	
	// Look for approve/allow/authorize/accept/confirm button
	var approveButton *rod.Element
	var buttonText string
	
	// Log ALL buttons to see what's available
	for i, button := range buttons {
		text, _ := button.Text()
		buttonType, _ := button.Property("type")
		visible, _ := button.Visible()
		
		typeStr := buttonType.String()
		
		log.Printf("[BROWSER] Button[%d]: type=%s, visible=%v, text='%s'\n", i, typeStr, visible, text)
		
		// Only consider visible buttons
		if !visible {
			continue
		}
		
		textLower := strings.ToLower(text)
		
		// Skip deny/cancel buttons to avoid clicking them
		if strings.Contains(textLower, "deny") || strings.Contains(textLower, "cancel") || strings.Contains(textLower, "reject") {
			log.Printf("[BROWSER] Skipping deny/cancel button[%d]: '%s'\n", i, text)
			continue
		}
		
		// Check if button text contains approval keywords (positive actions only)
		if strings.Contains(textLower, "accept") ||
			strings.Contains(textLower, "approve") ||
			strings.Contains(textLower, "allow") ||
			strings.Contains(textLower, "authorize") ||
			strings.Contains(textLower, "confirm") ||
			strings.Contains(textLower, "continue") ||
			(typeStr == "submit" && len(text) > 0 && !strings.Contains(textLower, "deny")) {
			approveButton = button
			buttonText = text
			log.Printf("[BROWSER] Found approval button[%d]: '%s'\n", i, text)
			break
		}
	}
	
	if approveButton == nil {
		return fmt.Errorf("no approval button found for %s", stage)
	}
	
	// Click approve button using JavaScript
	log.Printf("[BROWSER] Clicking '%s' button for %s...\n", buttonText, stage)
	
	_, err = approveButton.Eval("() => this.click()")
	if err != nil {
		return fmt.Errorf("failed to click button: %w", err)
	}
	
	log.Printf("[BROWSER] '%s' button clicked successfully\n", buttonText)
	return nil
}

// takeScreenshot takes a screenshot for debugging (only in debug mode)
func (b *BrowserAutomator) takeScreenshot(name string) {
	if !isDebugMode() {
		return // Skip screenshots in production
	}
	
	if b.page == nil {
		return
	}
	
	// Create screenshots directory if it doesn't exist
	screenshotDir := "logs/screenshots"
	if err := os.MkdirAll(screenshotDir, 0755); err != nil {
		log.Printf("[BROWSER] Warning: failed to create screenshot directory: %v\n", err)
		return
	}
	
	// Take screenshot
	filename := filepath.Join(screenshotDir, fmt.Sprintf("%s.png", name))
	screenshot, err := b.page.Screenshot(true, nil)
	if err != nil {
		log.Printf("[BROWSER] Warning: failed to take screenshot: %v\n", err)
		return
	}
	
	if err := os.WriteFile(filename, screenshot, 0644); err != nil {
		log.Printf("[BROWSER] Warning: failed to save screenshot: %v\n", err)
		return
	}
	
	log.Printf("[BROWSER] Screenshot saved: %s\n", filename)
}

// cleanup closes the browser
func (b *BrowserAutomator) cleanup() {
	if b.page != nil {
		b.page.Close()
	}
	if b.browser != nil {
		b.browser.Close()
	}
}

// AutomateDeviceAuthorizationWithRetry attempts automated authorization with retries
func (b *BrowserAutomator) AutomateDeviceAuthorizationWithRetry(ctx context.Context, authResp *DeviceAuthResponse, username, password, totpSecret string, maxRetries int) error {
	var lastErr error
	
	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			log.Printf("[BROWSER] Retry attempt %d/%d...\n", i+1, maxRetries)
			time.Sleep(time.Duration(i) * 2 * time.Second)
		}
		
		err := b.AutomateDeviceAuthorization(ctx, authResp, username, password, totpSecret)
		if err == nil {
			return nil
		}
		
		lastErr = err
		log.Printf("[BROWSER] Attempt %d failed: %v\n", i+1, err)
	}
	
	return fmt.Errorf("all %d attempts failed, last error: %w", maxRetries, lastErr)
}
