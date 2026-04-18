# Setup script for fully automated Identity Center authentication (Windows)
# This script stores credentials securely in Windows Credential Manager

$ErrorActionPreference = "Stop"

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "Automated Identity Center Setup" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "This script will store your Identity Center credentials"
Write-Host "securely in Windows Credential Manager for automated authentication."
Write-Host ""
Write-Host "WARNING: This enables fully automated authentication" -ForegroundColor Yellow
Write-Host "         without human interaction. Ensure this complies with" -ForegroundColor Yellow
Write-Host "         your organization's security policies." -ForegroundColor Yellow
Write-Host ""

# Collect credentials
$START_URL = Read-Host "Identity Center Start URL"
$USERNAME = Read-Host "Username (email)"
$PASSWORD = Read-Host "Password" -AsSecureString
$PASSWORD_TEXT = [Runtime.InteropServices.Marshal]::PtrToStringAuto([Runtime.InteropServices.Marshal]::SecureStringToBSTR($PASSWORD))
$MFA_SECRET = Read-Host "MFA Secret (optional, press Enter to skip)"
$AWS_REGION = Read-Host "AWS Region [us-east-1]"
if ([string]::IsNullOrWhiteSpace($AWS_REGION)) {
    $AWS_REGION = "us-east-1"
}

Write-Host ""
Write-Host "Storing credentials securely..." -ForegroundColor Green

# Create temporary Go program to store credentials
$tempFile = [System.IO.Path]::GetTempFileName() + ".go"
@'
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/zalando/go-keyring"
)

type Credentials struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	MFASecret string `json:"mfa_secret,omitempty"`
	StartURL  string `json:"start_url"`
	Region    string `json:"region"`
}

func main() {
	creds := &Credentials{
		Username:  os.Getenv("IC_USERNAME"),
		Password:  os.Getenv("IC_PASSWORD"),
		MFASecret: os.Getenv("IC_MFA_SECRET"),
		StartURL:  os.Getenv("IC_START_URL"),
		Region:    os.Getenv("IC_REGION"),
	}

	data, err := json.Marshal(creds)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal credentials: %v\n", err)
		os.Exit(1)
	}

	if err := keyring.Set("kiro-gateway-identity-center", "credentials", string(data)); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to store credentials: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Credentials stored securely in Windows Credential Manager")
}
'@ | Out-File -FilePath $tempFile -Encoding UTF8

# Store credentials using temporary program
$env:IC_USERNAME = $USERNAME
$env:IC_PASSWORD = $PASSWORD_TEXT
$env:IC_MFA_SECRET = $MFA_SECRET
$env:IC_START_URL = $START_URL
$env:IC_REGION = $AWS_REGION

go run $tempFile

# Clean up
Remove-Item $tempFile -Force
Remove-Variable PASSWORD_TEXT

# Generate API key
$API_KEY = -join ((48..57) + (97..102) | Get-Random -Count 64 | ForEach-Object {[char]$_})

# Create or update .env file
if (Test-Path .env) {
    Write-Host ""
    Write-Host "WARNING: .env file already exists" -ForegroundColor Yellow
    $overwrite = Read-Host "Overwrite? (y/N)"
    if ($overwrite -ne "y" -and $overwrite -ne "Y") {
        Write-Host "Keeping existing .env file"
        Write-Host "Add these lines manually:"
        Write-Host ""
        Write-Host "AUTH_TYPE=automated_oidc"
        Write-Host "AWS_REGION=$AWS_REGION"
        Write-Host ""
        exit 0
    }
}

$envContent = @"
# Kiro Gateway - Fully Automated Configuration
# Generated: $(Get-Date)

# Server Configuration
PORT=8090
PROXY_API_KEY=$API_KEY

# Automated Authentication
AUTH_TYPE=automated_oidc
AWS_REGION=$AWS_REGION

# Logging
LOG_LEVEL=info
DEBUG=false

# Optional: Run browser in visible mode for debugging
# AUTOMATED_AUTH_HEADLESS=false
"@

$envContent | Out-File -FilePath .env -Encoding UTF8

Write-Host ""
Write-Host "✓ Setup complete!" -ForegroundColor Green
Write-Host ""
Write-Host "Configuration saved to .env"
Write-Host "Credentials stored securely in Windows Credential Manager"
Write-Host ""
Write-Host "Next steps:"
Write-Host "1. Build the gateway: make build"
Write-Host "2. Start the gateway: .\kiro-gateway.exe"
Write-Host "3. The gateway will automatically authenticate on startup"
Write-Host ""
Write-Host "To remove stored credentials:"
Write-Host "  .\scripts\remove-automated-auth.ps1"
Write-Host ""
Write-Host "Security Notes:" -ForegroundColor Yellow
Write-Host "  - Credentials are stored in Windows Credential Manager (encrypted)"
Write-Host "  - Browser automation runs in headless mode by default"
Write-Host "  - Ensure this complies with your security policies"
Write-Host "  - Consider using IAM roles in production when possible"
Write-Host ""
