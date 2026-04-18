#!/bin/bash
# Get TOTP code from running Kiro Gateway container

CONTAINER_NAME="${1:-kiro-gateway-go-kiro-gateway-1}"
GATEWAY_URL="${2:-http://localhost:8080}"
API_KEY="${3:-${PROXY_API_KEY}}"

echo -e "\033[36mRetrieving TOTP code from Kiro Gateway...\033[0m"
echo ""

# Check if API key is provided
if [ -z "$API_KEY" ]; then
    echo -e "\033[31mError: API key not provided\033[0m"
    echo ""
    echo -e "\033[33mProvide API key via:\033[0m"
    echo -e "\033[90m  1. Parameter: ./get-totp.sh <container> <url> <api-key>\033[0m"
    echo -e "\033[90m  2. Environment: export PROXY_API_KEY='YOUR_KEY'\033[0m"
    echo -e "\033[90m  3. From container: docker exec <container> cat /app/.kiro/api-keys/*.json\033[0m"
    exit 1
fi

# Try to get TOTP from the gateway endpoint with authentication
if response=$(curl -s -H "Authorization: Bearer $API_KEY" "$GATEWAY_URL/totp" 2>/dev/null); then
    code=$(echo "$response" | jq -r '.code' 2>/dev/null)
    expires_in=$(echo "$response" | jq -r '.expires_in' 2>/dev/null)
    timestamp=$(echo "$response" | jq -r '.timestamp' 2>/dev/null)
    error=$(echo "$response" | jq -r '.error.message' 2>/dev/null)
    
    if [ -n "$code" ] && [ "$code" != "null" ]; then
        echo -e "\033[32mTOTP Code:\033[0m \033[43;30m $code \033[0m"
        echo ""
        echo -e "\033[90mExpires in: $expires_in seconds\033[0m"
        echo -e "\033[90mTimestamp: $timestamp\033[0m"
        echo ""
        echo -e "\033[36mUse this code to authenticate other clients/devices\033[0m"
        exit 0
    elif [ -n "$error" ] && [ "$error" != "null" ]; then
        echo -e "\033[31mAuthentication failed: $error\033[0m"
        echo ""
        echo -e "\033[33mGet a valid API key:\033[0m"
        echo -e "\033[90m  docker exec $CONTAINER_NAME cat /app/.kiro/api-keys/*.json\033[0m"
    fi
fi

echo -e "\033[31mFailed to retrieve TOTP code from gateway endpoint\033[0m"
echo ""
echo -e "\033[33mTrying to get code directly from container...\033[0m"

# Fallback: exec into container and curl the endpoint with auth
if container_response=$(docker exec "$CONTAINER_NAME" sh -c "curl -s -H 'Authorization: Bearer $API_KEY' http://localhost:8080/totp" 2>/dev/null); then
    code=$(echo "$container_response" | jq -r '.code' 2>/dev/null)
    expires_in=$(echo "$container_response" | jq -r '.expires_in' 2>/dev/null)
    timestamp=$(echo "$container_response" | jq -r '.timestamp' 2>/dev/null)
    
    if [ -n "$code" ] && [ "$code" != "null" ]; then
        echo -e "\033[32mTOTP Code:\033[0m \033[43;30m $code \033[0m"
        echo ""
        echo -e "\033[90mExpires in: $expires_in seconds\033[0m"
        echo -e "\033[90mTimestamp: $timestamp\033[0m"
        exit 0
    fi
fi

echo -e "\033[31mFailed to retrieve TOTP code from container\033[0m"
echo ""
echo -e "\033[33mMake sure:\033[0m"
echo -e "\033[90m  1. Container is running: docker ps\033[0m"
echo -e "\033[90m  2. API key is valid\033[0m"
echo -e "\033[90m  3. MFA_TOTP_SECRET is configured\033[0m"
echo -e "\033[90m  4. Gateway is healthy: curl http://localhost:8080/health\033[0m"
exit 1
