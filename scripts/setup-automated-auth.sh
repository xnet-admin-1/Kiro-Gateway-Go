#!/bin/bash
# Setup script for fully automated Identity Center authentication
# This script stores credentials securely in OS keychain for browser automation

set -e

echo "========================================="
echo "Automated Identity Center Setup"
echo "========================================="
echo ""
echo "This script will store your Identity Center credentials"
echo "securely in your OS keychain for automated authentication."
echo ""
echo "⚠️  WARNING: This enables fully automated authentication"
echo "    without human interaction. Ensure this complies with"
echo "    your organization's security policies."
echo ""

# Collect credentials
read -p "Identity Center Start URL: " START_URL
read -p "Username (email): " USERNAME
read -sp "Password: " PASSWORD
echo ""
read -sp "MFA Secret (optional, press Enter to skip): " MFA_SECRET
echo ""
read -p "AWS Region [us-east-1]: " AWS_REGION
AWS_REGION=${AWS_REGION:-us-east-1}

echo ""
echo "Storing credentials securely..."

# Create temporary Go program to store credentials
cat > /tmp/store-creds.go <<'EOF'
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

	fmt.Println("✓ Credentials stored securely in OS keychain")
}
EOF

# Store credentials using temporary program
IC_USERNAME="$USERNAME" \
IC_PASSWORD="$PASSWORD" \
IC_MFA_SECRET="$MFA_SECRET" \
IC_START_URL="$START_URL" \
IC_REGION="$AWS_REGION" \
go run /tmp/store-creds.go

# Clean up temporary file
rm /tmp/store-creds.go

# Generate API key for gateway
API_KEY=$(openssl rand -hex 32)

# Create or update .env file
if [ -f .env ]; then
	echo ""
	echo "⚠️  .env file already exists"
	read -p "Overwrite? (y/N): " OVERWRITE
	if [ "$OVERWRITE" != "y" ] && [ "$OVERWRITE" != "Y" ]; then
		echo "Keeping existing .env file"
		echo "Add these lines manually:"
		echo ""
		echo "AUTH_TYPE=automated_oidc"
		echo "AWS_REGION=$AWS_REGION"
		echo ""
		exit 0
	fi
fi

cat > .env <<EOF
# Kiro Gateway - Fully Automated Configuration
# Generated: $(date)

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
EOF

echo ""
echo "✓ Setup complete!"
echo ""
echo "Configuration saved to .env"
echo "Credentials stored securely in OS keychain"
echo ""
echo "Next steps:"
echo "1. Build the gateway: make build"
echo "2. Start the gateway: ./kiro-gateway"
echo "3. The gateway will automatically authenticate on startup"
echo ""
echo "To remove stored credentials:"
echo "  ./scripts/remove-automated-auth.sh"
echo ""
echo "⚠️  Security Notes:"
echo "  - Credentials are stored in OS keychain (encrypted)"
echo "  - Browser automation runs in headless mode by default"
echo "  - Ensure this complies with your security policies"
echo "  - Consider using IAM roles in production when possible"
echo ""
