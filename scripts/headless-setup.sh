#!/bin/bash
# Headless Authentication Setup Script
# Automates the setup of Kiro Gateway for headless environments

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
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

# Check if running in AWS environment
check_aws_environment() {
    print_header "Checking AWS Environment"
    
    # Check for EC2 instance metadata
    if curl -s -m 2 http://169.254.169.254/latest/meta-data/ > /dev/null 2>&1; then
        print_success "Running on EC2 instance"
        return 0
    fi
    
    # Check for ECS task metadata
    if [ -n "$AWS_CONTAINER_CREDENTIALS_RELATIVE_URI" ]; then
        print_success "Running in ECS container"
        return 0
    fi
    
    # Check for Lambda environment
    if [ -n "$AWS_LAMBDA_FUNCTION_NAME" ]; then
        print_success "Running in Lambda function"
        return 0
    fi
    
    # Check for Kubernetes with IRSA
    if [ -n "$AWS_WEB_IDENTITY_TOKEN_FILE" ] && [ -n "$AWS_ROLE_ARN" ]; then
        print_success "Running in Kubernetes with IRSA"
        return 0
    fi
    
    print_warning "Not running in AWS environment"
    return 1
}

# Check for AWS credentials
check_aws_credentials() {
    print_header "Checking AWS Credentials"
    
    if aws sts get-caller-identity > /dev/null 2>&1; then
        IDENTITY=$(aws sts get-caller-identity)
        ACCOUNT=$(echo "$IDENTITY" | jq -r '.Account')
        ARN=$(echo "$IDENTITY" | jq -r '.Arn')
        
        print_success "AWS credentials found"
        print_info "Account: $ACCOUNT"
        print_info "Identity: $ARN"
        return 0
    else
        print_error "No AWS credentials found"
        return 1
    fi
}

# Setup Method 1: IAM + SigV4 (Recommended)
setup_sigv4() {
    print_header "Setting up IAM + SigV4 Authentication"
    
    # Check for AWS credentials
    if ! check_aws_credentials; then
        print_error "AWS credentials required for SigV4 authentication"
        print_info "Please configure AWS credentials using one of:"
        print_info "  1. aws configure"
        print_info "  2. IAM role (EC2/ECS/Lambda)"
        print_info "  3. Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)"
        return 1
    fi
    
    # Create .env file
    cat > .env <<EOF
# Kiro Gateway - Headless Configuration (SigV4)
# Generated: $(date)

# Server Configuration
PORT=8090
PROXY_API_KEY=${PROXY_API_KEY:-$(openssl rand -hex 32)}

# Authentication - SigV4 with IAM credentials
AMAZON_Q_SIGV4=true
AWS_REGION=${AWS_REGION:-us-east-1}

# Use Q Developer endpoint with SigV4
Q_USE_SENDMESSAGE=true

# Profile ARN (if using Identity Center)
${PROFILE_ARN:+PROFILE_ARN=$PROFILE_ARN}

# Logging
LOG_LEVEL=info
DEBUG=false

# Retry Configuration
MAX_RETRIES=3
MAX_BACKOFF=30s
CONNECT_TIMEOUT=10s
READ_TIMEOUT=30s
OPERATION_TIMEOUT=60s
EOF
    
    print_success "Created .env file with SigV4 configuration"
    print_info "API Key: $(grep PROXY_API_KEY .env | cut -d= -f2)"
    
    return 0
}

# Setup Method 2: Bearer Token
setup_bearer_token() {
    print_header "Setting up Bearer Token Authentication"
    
    # Check if AWS CLI is configured
    if ! command -v aws &> /dev/null; then
        print_error "AWS CLI not found. Please install AWS CLI first."
        return 1
    fi
    
    # Try to get token from SSO cache
    SSO_CACHE_DIR="$HOME/.aws/sso/cache"
    if [ -d "$SSO_CACHE_DIR" ]; then
        CACHE_FILE=$(ls -t "$SSO_CACHE_DIR"/*.json 2>/dev/null | head -1)
        
        if [ -n "$CACHE_FILE" ]; then
            print_info "Found SSO cache file: $CACHE_FILE"
            
            # Extract token
            ACCESS_TOKEN=$(jq -r '.accessToken' "$CACHE_FILE" 2>/dev/null)
            EXPIRES_AT=$(jq -r '.expiresAt' "$CACHE_FILE" 2>/dev/null)
            
            if [ -n "$ACCESS_TOKEN" ] && [ "$ACCESS_TOKEN" != "null" ]; then
                print_success "Extracted bearer token from SSO cache"
                print_info "Token expires at: $EXPIRES_AT"
                
                # Create .env file
                cat > .env <<EOF
# Kiro Gateway - Headless Configuration (Bearer Token)
# Generated: $(date)

# Server Configuration
PORT=8090
PROXY_API_KEY=${PROXY_API_KEY:-$(openssl rand -hex 32)}

# Authentication - Bearer Token
AMAZON_Q_SIGV4=false
AWS_REGION=${AWS_REGION:-us-east-1}
BEARER_TOKEN=$ACCESS_TOKEN

# Profile ARN (required for bearer token mode)
${PROFILE_ARN:+PROFILE_ARN=$PROFILE_ARN}

# Logging
LOG_LEVEL=info
DEBUG=false
EOF
                
                print_success "Created .env file with bearer token"
                print_warning "Token will expire at: $EXPIRES_AT"
                print_warning "You will need to refresh the token before expiration"
                
                return 0
            fi
        fi
    fi
    
    print_error "No valid SSO token found"
    print_info "Please run: aws sso login --profile YOUR_PROFILE"
    return 1
}

# Setup Method 3: CLI DB with Auto-Refresh
setup_cli_db() {
    print_header "Setting up CLI DB Authentication"
    
    # Check for SSO cache
    SSO_CACHE_DIR="$HOME/.aws/sso/cache"
    if [ ! -d "$SSO_CACHE_DIR" ]; then
        print_error "AWS SSO cache directory not found"
        print_info "Please run: aws sso login --profile YOUR_PROFILE"
        return 1
    fi
    
    # Find SSO cache file
    CACHE_FILE=$(ls -t "$SSO_CACHE_DIR"/*.json 2>/dev/null | head -1)
    
    if [ -z "$CACHE_FILE" ]; then
        print_error "No SSO cache file found"
        print_info "Please run: aws sso login --profile YOUR_PROFILE"
        return 1
    fi
    
    print_success "Found SSO cache file: $CACHE_FILE"
    
    # Create .env file
    cat > .env <<EOF
# Kiro Gateway - Headless Configuration (CLI DB)
# Generated: $(date)

# Server Configuration
PORT=8090
PROXY_API_KEY=${PROXY_API_KEY:-$(openssl rand -hex 32)}

# Authentication - CLI DB with auto-refresh
AUTH_TYPE=cli_db
CLI_DB_PATH=$CACHE_FILE
AWS_REGION=${AWS_REGION:-us-east-1}

# Use CodeWhisperer endpoint with bearer token
Q_USE_SENDMESSAGE=false

# Profile ARN (required)
${PROFILE_ARN:+PROFILE_ARN=$PROFILE_ARN}

# Logging
LOG_LEVEL=info
DEBUG=false
EOF
    
    print_success "Created .env file with CLI DB configuration"
    print_info "Token will be automatically refreshed by AWS CLI"
    
    return 0
}

# Main menu
show_menu() {
    print_header "Kiro Gateway - Headless Authentication Setup"
    echo ""
    echo "Select authentication method:"
    echo ""
    echo "1) IAM + SigV4 (Recommended for production)"
    echo "   - Uses AWS IAM credentials"
    echo "   - Automatic credential rotation"
    echo "   - Works with IAM roles, instance profiles, IRSA"
    echo ""
    echo "2) Bearer Token (Simple, but requires manual renewal)"
    echo "   - Uses pre-generated bearer token"
    echo "   - Token expires in ~1 hour"
    echo "   - Good for testing"
    echo ""
    echo "3) CLI DB (Auto-refresh, requires AWS CLI)"
    echo "   - Uses AWS SSO cache"
    echo "   - Automatic token refresh"
    echo "   - Good for development"
    echo ""
    echo "4) Exit"
    echo ""
}

# Main script
main() {
    # Check for required tools
    if ! command -v jq &> /dev/null; then
        print_error "jq is required but not installed"
        print_info "Install with: sudo apt-get install jq (Ubuntu/Debian)"
        print_info "            : brew install jq (macOS)"
        exit 1
    fi
    
    # Check AWS environment
    check_aws_environment || true
    
    # Show menu
    while true; do
        show_menu
        read -p "Enter choice [1-4]: " choice
        
        case $choice in
            1)
                if setup_sigv4; then
                    print_success "Setup complete!"
                    print_info "Start gateway with: ./kiro-gateway"
                    break
                fi
                ;;
            2)
                if setup_bearer_token; then
                    print_success "Setup complete!"
                    print_info "Start gateway with: ./kiro-gateway"
                    break
                fi
                ;;
            3)
                if setup_cli_db; then
                    print_success "Setup complete!"
                    print_info "Start gateway with: ./kiro-gateway"
                    break
                fi
                ;;
            4)
                print_info "Exiting..."
                exit 0
                ;;
            *)
                print_error "Invalid choice. Please select 1-4."
                ;;
        esac
        
        echo ""
        read -p "Press Enter to continue..."
        clear
    done
    
    # Test configuration
    echo ""
    read -p "Would you like to test the configuration? (y/n): " test_choice
    
    if [ "$test_choice" = "y" ] || [ "$test_choice" = "Y" ]; then
        print_header "Testing Configuration"
        
        # Build gateway if needed
        if [ ! -f "./kiro-gateway" ]; then
            print_info "Building gateway..."
            go build -o kiro-gateway ./cmd/kiro-gateway
            print_success "Gateway built successfully"
        fi
        
        # Start gateway in background
        print_info "Starting gateway..."
        ./kiro-gateway &
        GATEWAY_PID=$!
        
        # Wait for startup
        sleep 5
        
        # Test health endpoint
        if curl -s http://localhost:8090/health > /dev/null; then
            print_success "Gateway is running"
            
            # Get API key
            API_KEY=$(grep PROXY_API_KEY .env | cut -d= -f2)
            
            # Test chat endpoint
            print_info "Testing chat endpoint..."
            RESPONSE=$(curl -s -X POST http://localhost:8090/v1/chat/completions \
                -H "Content-Type: application/json" \
                -H "Authorization: Bearer $API_KEY" \
                -d '{
                    "model": "claude-3-5-sonnet-20241022-v2",
                    "messages": [{"role": "user", "content": "Say hello"}],
                    "max_tokens": 50
                }')
            
            if echo "$RESPONSE" | jq -e '.choices[0].message.content' > /dev/null 2>&1; then
                print_success "Chat endpoint working!"
                print_info "Response: $(echo "$RESPONSE" | jq -r '.choices[0].message.content')"
            else
                print_error "Chat endpoint test failed"
                print_info "Response: $RESPONSE"
            fi
        else
            print_error "Gateway failed to start"
        fi
        
        # Stop gateway
        print_info "Stopping gateway..."
        kill $GATEWAY_PID 2>/dev/null || true
        wait $GATEWAY_PID 2>/dev/null || true
        
        print_success "Test complete"
    fi
    
    echo ""
    print_success "All done! Your gateway is ready for headless operation."
    print_info "Configuration saved to: .env"
    print_info "Start gateway with: ./kiro-gateway"
}

# Run main function
main
