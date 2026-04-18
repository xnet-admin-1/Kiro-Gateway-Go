package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/pquerna/otp/totp"
)

// TOTPResponse represents the TOTP code response
type TOTPResponse struct {
	Code      string `json:"code"`
	ExpiresIn int    `json:"expires_in"` // seconds until code expires
	Timestamp string `json:"timestamp"`
}

// TOTPHandler handles TOTP code generation requests
func TOTPHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get TOTP secret from environment or Docker secrets
	totpSecret := os.Getenv("MFA_TOTP_SECRET")
	if totpSecret == "" {
		totpSecret = os.Getenv("SSO_MFA_TOTP_SECRET")
	}
	if totpSecret == "" {
		if data, err := os.ReadFile("/run/secrets/mfa_totp_secret"); err == nil {
			totpSecret = string(data)
		}
	}

	if totpSecret == "" {
		http.Error(w, "TOTP secret not configured", http.StatusServiceUnavailable)
		return
	}

	// Generate TOTP code
	code, err := totp.GenerateCode(totpSecret, time.Now())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate TOTP code: %v", err), http.StatusInternalServerError)
		return
	}

	// Calculate expiration (TOTP codes typically expire every 30 seconds)
	now := time.Now()
	secondsInPeriod := now.Unix() % 30
	expiresIn := 30 - int(secondsInPeriod)

	// Create response
	response := TOTPResponse{
		Code:      code,
		ExpiresIn: expiresIn,
		Timestamp: now.Format(time.RFC3339),
	}

	// Send JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
