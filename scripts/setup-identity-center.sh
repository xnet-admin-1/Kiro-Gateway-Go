#!/bin/bash
# Identity Center Setup Script for Q Developer Pro
# One-time setup to obtain tokens for headless operation

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

print_header "AWS IAM Identity Center Setup for Q Developer Pro"
echo ""

# Check for required tools
if ! command -v aws &> /dev/null; then
    print_error "AWS CLI not found"
    print_info "Install from: https://aws.amazon.com/cli/"
    exit 1
fi

if ! command -v jq &> /dev/null; then
    print_error "jq not found"
    print_info "Install with: sudo apt-get install jq (Ubuntu/Debian)"
    print_info "            : brew install jq (macOS)"
    exit 1
fi

print_success "Required tools found"
echo ""

# Step 1: Check if Identity Center is enabled
print_header "Step 1: Checking Identity Center Status"
echo ""

print_info "Checking if Identity Center is enabled..."
if aws sso-admin list-instances --region us-east-1 &> /dev/null; then
    INSTANCES=$(aws sso-admin list-instances --region us-east-1 2>/dev/null)
    if [ -n "$INSTANCES" ]; then
        print_success "Identity Center is enabled"
        INSTANCE_ARN=$(echo "$INSTANCES" | jq -r '.Instances[0].InstanceArn')
        print_info "Instance ARN: $INSTANCE_ARN"
    else
        print_warning "Identity Center may not be enabled"
        print_info "Enable it at: https://console.aws.amazon.com/singlesignon"
    fi
else
    print_warning "Could not check Identity Center status"
    print_info "You may need to enable it at: https://console.aws.amazon.com/singlesignon"
fi

echo ""
read -p "Press Enter to continue..."
echo ""

# Step 2: Get Profile ARN
print_header "Step 2: Getting Q Developer Profile ARN"
echo ""

print_info "Attempting to get your Q Developer profile..."

# Try using Q CLI first
if command -v qchat &> /dev/null; then
    print_info "Q CLI found, attempting to get profile..."
    PROFILE_OUTPUT=$(qchat profile 2>/dev/null || true)
    
    if [ -n "$PROFILE_OUTPUT" ]; then
        PROFILE_ARN=$(echo "$PROFILE_OUTPUT" | grep -oP 'arn:aws:codewhisperer:[^:]+:[^:]+:profile/[a-zA-Z0-9]+' | head -1)
        
        if [ -n "$PROFILE_ARN" ]; then
            print_success "Profile ARN found: $PROFILE_ARN"
        fi
    fi
fi

# If not found, try AWS CLI
if [ -z "$PROFILE_ARN" ]; then
    print_info "Trying AWS CLI..."
    PROFILES=$(aws codewhisperer list-profiles --region us-east-1 2>/dev/null || true)
    
    if [ -n "$PROFILES" ]; then
        PROFILE_ARN=$(echo "$PROFILES" | jq -r '.profiles[0].profileArn' 2>/dev/null || true)
        
        if [ -n "$PROFILE_ARN" ] && [ "$PROFILE_ARN" != "null" ]; then
            print_success "Profile ARN found: $PROFILE_ARN"
        fi
    fi
fi

# If still not found, ask user
if [ -z "$PROFILE_ARN" ] || [ "$PROFILE_ARN" == "null" ]; then
    print_warning "Could not automatically detect Profile ARN"
    echo ""
    print_info "You can find your Profile ARN at:"
    print_info "  1. Amazon Q Developer console → Settings → Profiles"
    print_info "  2. Or run: qchat profile"
    echo ""
    read -p "Enter your Profile ARN: " PROFILE_ARN
fi

if [ -z "$PROFILE_ARN" ]; then
    print_error "Profile ARN is required"
    exit 1
fi

print_success "Profile ARN: $PROFILE_ARN"
echo ""

# Step 3: Authenticate with Identity Center
print_header "Step 3: Authenticating with Identity Center"
echo ""

print_warning "This step requires browser interaction (one-time only)"
print_info "You will be prompted to:"
print_info "  1. Visit a verification URL in your browser"
print_info "  2. Enter a device code"
print_info "  3. Authenticate with your Identity Center credentials"
echo ""

# Ask for profile name
read -p "Enter your AWS profile name (or press Enter for 'default'): " AWS_PROFILE
AWS_PROFILE=${AWS_PROFILE:-default}

print_info "Starting authentication..."
echo ""

# Authenticate
if aws sso login --profile "$AWS_PROFILE"; then
    print_success "Authentication successful!"
else
    print_error "Authentication failed"
    exit 1
fi

echo ""

# Step 4: Extract tokens
print_header "Step 4: Extracting Tokens"
echo ""

SSO_CACHE_DIR="$HOME/.aws/sso/cache"

if [ ! -d "$SSO_CACHE_DIR" ]; then
    print_error "SSO cache directory not found: $SSO_CACHE_DIR"
    exit 1
fi

CACHE_FILE=$(ls -t "$SSO_CACHE_DIR"/*.json 2>/dev/null | head -1)

if [ -z "$CACHE_FILE" ]; then
    print_error "No SSO cache file found"
    exit 1
fi

print_info "Found cache file: $CACHE_FILE"

# Extract tokens
ACCESS_TOKEN=$(jq -r '.accessToken' "$CACHE_FILE" 2>/dev/null)
REFRESH_TOKEN=$(jq -r '.refreshToken' "$CACHE_FILE" 2>/dev/null)
EXPIRES_AT=$(jq -r '.expiresAt' "$CACHE_FILE" 2>/dev/null)

if [ -z "$ACCESS_TOKEN" ] || [ "$ACCESS_TOKEN" == "null" ]; then
    print_error "Could not extract access token"
    exit 1
fi

if [ -z "$REFRESH_TOKEN" ] || [ "$REFRESH_TOKEN" == "null" ]; then
    print_warning "No refresh token found (may not be needed)"
fi

print_success "Tokens extracted successfully"
print_info "Access token expires at: $EXPIRES_AT"
echo ""

# Step 5: Save configuration
print_header "Step 5: Saving Configuration"
echo ""

# Create .env file
cat > .env <<EOF
# Kiro Gateway - Identity Center Configuration
# Generated: $(date)

# Server Configuration
PORT=8090
PROXY_API_KEY=$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | xxd -p)

# Identity Center Authentication
AUTH_TYPE=cli_db
CLI_DB_PATH=$CACHE_FILE
PROFILE_ARN=$PROFILE_ARN
AWS_REGION=us-east-1

# AWS Profile
AWS_PROFILE=$AWS_PROFILE

# Logging
LOG_LEVEL=info
DEBUG=false
EOF

print_success "Configuration saved to .env"
echo ""

# Display summary
print_header "Setup Complete!"
echo ""
print_success "Identity Center authentication configured"
print_info "Profile ARN: $PROFILE_ARN"
print_info "AWS Profile: $AWS_PROFILE"
print_info "Token expires: $EXPIRES_AT"
echo ""

# Show next steps
print_header "Next Steps"
echo ""
echo "1. Start the gateway:"
echo "   ./kiro-gateway"
echo ""
echo "2. Test the health endpoint:"
echo "   curl http://localhost:8090/health"
echo ""
echo "3. Test a chat request:"
echo "   curl -X POST http://localhost:8090/v1/chat/completions \\"
echo "     -H 'Content-Type: application/json' \\"
echo "     -H 'Authorization: Bearer YOUR_API_KEY' \\"
echo "     -d '{\"model\":\"claude-3-5-sonnet-20241022-v2\",\"messages\":[{\"role\":\"user\",\"content\":\"What is AWS Lambda?\"}]}'"
echo ""

# Token refresh reminder
print_header "Important: Token Refresh"
echo ""
print_warning "Tokens will be automatically refreshed by AWS CLI"
print_info "Refresh tokens expire after ~90 days"
print_info "Set up monitoring to alert before expiration"
echo ""
print_info "To re-authenticate when needed:"
echo "  aws sso login --profile $AWS_PROFILE"
echo ""

# Security reminder
print_header "Security Reminder"
echo ""
print_warning "The .env file contains sensitive information"
print_info "Add .env to .gitignore"
print_info "For production, use AWS Secrets Manager:"
echo ""
echo "  aws secretsmanager create-secret \\"
echo "    --name kiro-gateway/identity-center-tokens \\"
echo "    --secret-string '{\"refresh_token\":\"...\",\"profile_arn\":\"$PROFILE_ARN\"}'"
echo ""

print_success "Setup complete! You're ready to use Kiro Gateway with Q Developer Pro."
