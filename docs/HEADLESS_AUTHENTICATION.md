# Headless Authentication Guide

**Complete guide for automating Kiro Gateway authentication in headless/CI/CD environments**

---

## Table of Contents

1. [Overview](#overview)
2. [NEW: Headless SSO OIDC (No AWS CLI Required)](#new-headless-sso-oidc-no-aws-cli-required)
3. [Recommended Methods](#recommended-methods)
4. [Method 1: IAM Credentials with SigV4](#method-1-iam-credentials-with-sigv4-recommended)
5. [Method 2: Pre-Generated Bearer Token](#method-2-pre-generated-bearer-token)
6. [Method 3: AWS SSO with Refresh Token](#method-3-aws-sso-with-refresh-token)
7. [Method 4: Service Account Credentials](#method-4-service-account-credentials)
8. [Method 5: Headless SSO OIDC (Self-Contained)](#method-5-headless-sso-oidc-self-contained)
9. [CI/CD Integration Examples](#cicd-integration-examples)
10. [Security Best Practices](#security-best-practices)
11. [Troubleshooting](#troubleshooting)

---

## NEW: Headless SSO OIDC (No AWS CLI Required)

**🎉 We now support fully self-contained headless authentication!**

The new **Headless SSO OIDC** mode eliminates the AWS CLI dependency by implementing the complete OAuth 2.0 device authorization flow directly in the gateway.

### Key Features

✅ **No AWS CLI Required** - Fully self-contained implementation  
✅ **Automatic Token Refresh** - Tokens refresh automatically in background  
✅ **Docker-Friendly** - Works perfectly in containers  
✅ **One-Time Setup** - User authorizes once, then fully automated  
✅ **Secure Storage** - Tokens stored in encrypted keychain/database  
✅ **Production Ready** - Supports pre-registered OIDC clients  

### Quick Start

```bash
# 1. Configure headless mode
cat > .env <<EOF
HEADLESS_MODE=true
SSO_START_URL=https://my-portal.awsapps.com/start
SSO_REGION=us-east-1
SSO_ACCOUNT_ID=123456789012
SSO_ROLE_NAME=MyRole
AMAZON_Q_SIGV4=true
EOF

# 2. Start gateway (first time - requires user authorization)
./kiro-gateway

# Gateway will display:
# ╔════════════════════════════════════════════════════════════╗
# ║          AWS IAM Identity Center Authentication           ║
# ╠════════════════════════════════════════════════════════════╣
# ║  Please visit: https://device.sso.us-east-1.amazonaws.com ║
# ║  And enter code: ABCD-1234                                ║
# ╚════════════════════════════════════════════════════════════╝

# 3. Authorize in browser (one-time)
# Visit the URL and enter the code

# 4. Gateway continues automatically
# Tokens are stored securely and refresh automatically

# 5. Subsequent runs - fully automated (no user interaction)
./kiro-gateway
```

See [Method 5: Headless SSO OIDC](#method-5-headless-sso-oidc-self-contained) for complete details.

---

## Overview

Headless authentication enables Kiro Gateway to run without human interaction in:
- **CI/CD pipelines** (GitHub Actions, GitLab CI, Jenkins)
- **Docker containers** (production deployments)
- **Kubernetes clusters** (with IRSA or service accounts)
- **Lambda functions** (serverless deployments)
- **Automated testing** (integration tests, load tests)
- **Scheduled jobs** (cron, batch processing)

### ⚠️ Important: Q Developer Pro Requirements

**Q Developer Pro requires AWS IAM Identity Center (formerly AWS SSO):**

- ✅ You must have an **Identity Center instance** configured
- ✅ You need a **Q Developer Pro subscription** (not free tier)
- ✅ You must obtain a **Profile ARN** from Identity Center
- ✅ Initial authentication requires **one-time human interaction** (device code flow)
- ✅ After initial setup, tokens can be refreshed automatically

**For headless environments:**
1. Perform initial authentication on a workstation (one-time)
2. Extract and securely store the refresh token
3. Use refresh token for automated token renewal (no human interaction)

See [Identity Center Setup](#identity-center-setup-for-headless) for detailed instructions.

### Key Requirements

✅ **No browser interaction** (after initial setup)  
✅ **No manual token entry** (after initial setup)  
✅ **Automatic token refresh**  
✅ **Secure credential storage**  
✅ **Audit logging**  
✅ **Failure recovery**

---

## Recommended Methods

| Method | Use Case | Security | Complexity | Auto-Refresh | AWS CLI Required |
|--------|----------|----------|------------|--------------|------------------|
| **Headless OIDC** | Docker, CI/CD, Production | ⭐⭐⭐⭐⭐ | Low | ✅ Yes | ❌ No |
| **IAM + SigV4** | AWS environments | ⭐⭐⭐⭐⭐ | Low | ✅ Yes | ❌ No |
| **SSO Refresh Token** | Corporate environments | ⭐⭐⭐⭐ | Medium | ✅ Yes | ✅ Yes |
| **Bearer Token** | Development, testing | ⭐⭐⭐ | Very Low | ❌ No | ✅ Yes |
| **Service Account** | Multi-tenant, SaaS | ⭐⭐⭐⭐ | Medium | ✅ Yes | ❌ No |

**Recommendation:** 
- **For Docker/Containers**: Use **Headless OIDC** (no AWS CLI needed)
- **For AWS Environments**: Use **IAM + SigV4** (automatic credentials)
- **For Development**: Use **Bearer Token** or **SSO Refresh Token**

**Note:** All methods require AWS IAM Identity Center for Q Developer Pro access.

---

## Identity Center Setup for Headless

**Q Developer Pro requires AWS IAM Identity Center.** Here's how to set it up for headless environments.

### Prerequisites

1. **AWS IAM Identity Center enabled** in your AWS account
2. **Q Developer Pro subscription** (not free tier)
3. **Profile ARN** from Identity Center

### Step 1: Enable Identity Center

If not already enabled:

```bash
# Enable Identity Center (one-time setup)
aws sso-admin create-instance \
  --instance-name "MyOrganization" \
  --region us-east-1

# Note the Instance ARN - you'll need this
```

**Or use AWS Console:**
1. Go to AWS IAM Identity Center
2. Click "Enable"
3. Choose your identity source (AWS Directory, Active Directory, or External IdP)

### Step 2: Get Your Profile ARN

**Option A: Using Q CLI**
```bash
# Install Q CLI if not already installed
# See: https://docs.aws.amazon.com/amazonq/latest/qdeveloper-ug/command-line-getting-started-installing.html

# Login and get profile
qchat auth login
qchat profile

# Output will show your Profile ARN:
# arn:aws:codewhisperer:us-east-1:123456789012:profile/abc123def456
```

**Option B: Using AWS Console**
1. Go to Amazon Q Developer console
2. Navigate to "Settings" → "Profiles"
3. Copy your Profile ARN

**Option C: Using AWS CLI**
```bash
# List available profiles
aws codewhisperer list-profiles --region us-east-1

# Output:
# {
#   "profiles": [
#     {
#       "profileArn": "arn:aws:codewhisperer:us-east-1:123456789012:profile/abc123",
#       "profileName": "MyProfile"
#     }
#   ]
# }
```

### Step 3: Initial Authentication (One-Time)

**This step requires human interaction but only needs to be done once.**

```bash
# Method 1: Using AWS CLI
aws sso login --profile YOUR_PROFILE

# This will:
# 1. Display a device code
# 2. Open browser to verification URL
# 3. Prompt you to enter the code
# 4. Store tokens in ~/.aws/sso/cache/

# Method 2: Using Q CLI
qchat auth login

# This will:
# 1. Display a device code and URL
# 2. Wait for you to authenticate in browser
# 3. Store tokens locally
```

**What happens:**
1. Device code is generated
2. You visit the verification URL in a browser
3. You enter the device code
4. You authenticate with your Identity Center credentials
5. Tokens are stored locally:
   - Access token (expires in ~1 hour)
   - Refresh token (expires in ~90 days)

### Step 4: Extract Tokens for Headless Use

After initial authentication, extract tokens for automated use:

```bash
# Find the SSO cache file
SSO_CACHE_DIR="$HOME/.aws/sso/cache"
CACHE_FILE=$(ls -t "$SSO_CACHE_DIR"/*.json | head -1)

# Extract tokens
ACCESS_TOKEN=$(jq -r '.accessToken' "$CACHE_FILE")
REFRESH_TOKEN=$(jq -r '.refreshToken' "$CACHE_FILE")
EXPIRES_AT=$(jq -r '.expiresAt' "$CACHE_FILE")

echo "Access Token: $ACCESS_TOKEN"
echo "Refresh Token: $REFRESH_TOKEN"
echo "Expires At: $EXPIRES_AT"

# Store refresh token securely (see Security section)
# The refresh token is what enables headless operation
```

### Step 5: Store Tokens Securely

**For Production (Recommended):**

```bash
# Option A: AWS Secrets Manager
aws secretsmanager create-secret \
  --name kiro-gateway/identity-center-tokens \
  --secret-string "{
    \"refresh_token\": \"$REFRESH_TOKEN\",
    \"profile_arn\": \"arn:aws:codewhisperer:us-east-1:123456789012:profile/abc123\",
    \"region\": \"us-east-1\"
  }"

# Option B: AWS Systems Manager Parameter Store
aws ssm put-parameter \
  --name /kiro-gateway/refresh-token \
  --value "$REFRESH_TOKEN" \
  --type SecureString

aws ssm put-parameter \
  --name /kiro-gateway/profile-arn \
  --value "arn:aws:codewhisperer:us-east-1:123456789012:profile/abc123" \
  --type String
```

**For Development:**

```bash
# Store in .env file (add to .gitignore!)
cat > .env <<EOF
AUTH_TYPE=cli_db
CLI_DB_PATH=$CACHE_FILE
PROFILE_ARN=arn:aws:codewhisperer:us-east-1:123456789012:profile/abc123
AWS_REGION=us-east-1
EOF
```

### Step 6: Configure Gateway for Headless

**Method 1: Using SSO Cache File (Simplest)**

```bash
# .env
AUTH_TYPE=cli_db
CLI_DB_PATH=/path/to/.aws/sso/cache/token.json
PROFILE_ARN=arn:aws:codewhisperer:us-east-1:123456789012:profile/abc123
AWS_REGION=us-east-1
```

**How it works:**
- Gateway reads tokens from SSO cache file
- AWS CLI refreshes tokens in background
- Gateway reloads refreshed tokens automatically
- No manual intervention needed

**Method 2: Using SigV4 with Identity Center**

```bash
# .env
AMAZON_Q_SIGV4=true
Q_USE_SENDMESSAGE=true
AWS_REGION=us-east-1
AWS_PROFILE=your-identity-center-profile
PROFILE_ARN=arn:aws:codewhisperer:us-east-1:123456789012:profile/abc123
```

**How it works:**
- Gateway uses AWS credentials from profile
- Profile is configured to use Identity Center
- Credentials automatically refresh via Identity Center
- Works with IAM roles and temporary credentials

### Step 7: Verify Setup

```bash
# Start gateway
./kiro-gateway

# Test health endpoint
curl http://localhost:8090/health

# Test chat endpoint
curl -X POST http://localhost:8090/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "claude-3-5-sonnet-20241022-v2",
    "messages": [{"role": "user", "content": "What is AWS Lambda?"}],
    "max_tokens": 100
  }'
```

### Token Refresh Strategy

**Automatic Refresh (Recommended):**

The gateway automatically refreshes tokens using one of these methods:

1. **CLI DB Method:**
   - AWS CLI monitors SSO cache
   - Refreshes tokens before expiration
   - Gateway reloads from cache file
   - No gateway code changes needed

2. **SigV4 Method:**
   - AWS SDK handles credential refresh
   - Uses Identity Center for authentication
   - Transparent to gateway
   - Works with IAM roles

**Manual Refresh (Fallback):**

If automatic refresh fails:

```bash
# Re-authenticate with Identity Center
aws sso login --profile YOUR_PROFILE

# Or using Q CLI
qchat auth login

# Gateway will pick up new tokens automatically
```

### Refresh Token Expiration

**Important:** Refresh tokens expire after ~90 days.

**To handle expiration:**

1. **Monitor token expiration:**
```bash
# Check expiration date
jq -r '.expiresAt' ~/.aws/sso/cache/*.json

# Set up monitoring alert 7 days before expiration
```

2. **Automate expiration monitoring:**
```bash
#!/bin/bash
# scripts/check-token-expiration.sh

CACHE_FILE=$(ls -t ~/.aws/sso/cache/*.json | head -1)
EXPIRES_AT=$(jq -r '.expiresAt' "$CACHE_FILE")
EXPIRES_EPOCH=$(date -d "$EXPIRES_AT" +%s)
NOW_EPOCH=$(date +%s)
DAYS_UNTIL_EXPIRY=$(( ($EXPIRES_EPOCH - $NOW_EPOCH) / 86400 ))

if [ $DAYS_UNTIL_EXPIRY -lt 7 ]; then
    echo "WARNING: Token expires in $DAYS_UNTIL_EXPIRY days"
    echo "Please re-authenticate: aws sso login --profile YOUR_PROFILE"
    # Send alert (email, Slack, PagerDuty, etc.)
fi
```

3. **Set up cron job:**
```bash
# Check token expiration daily
0 9 * * * /path/to/check-token-expiration.sh
```

### Identity Center Best Practices

**Security:**
- ✅ Use MFA for Identity Center authentication
- ✅ Store refresh tokens in Secrets Manager
- ✅ Rotate refresh tokens every 60 days
- ✅ Use separate profiles per environment
- ✅ Enable CloudTrail logging for Identity Center

**Automation:**
- ✅ Monitor token expiration
- ✅ Alert 7 days before expiration
- ✅ Document re-authentication process
- ✅ Test token refresh regularly
- ✅ Have fallback authentication method

**Compliance:**
- ✅ Audit token usage via CloudTrail
- ✅ Review access logs monthly
- ✅ Implement least-privilege policies
- ✅ Document who has access
- ✅ Regular access reviews

---

## Method 1: IAM Credentials with SigV4 (Recommended)

**Best for:** Production deployments, AWS environments, Kubernetes with IRSA

### Overview

Uses AWS IAM credentials instead of bearer tokens. Credentials are automatically rotated by AWS and never expire (for IAM roles).

### Advantages

✅ No manual token management  
✅ Automatic credential rotation  
✅ AWS-native security model  
✅ Works with IAM roles, instance profiles, IRSA  
✅ Audit trail via CloudTrail  
✅ No token expiration issues  

### Configuration

#### Option A: IAM Role (EC2, ECS, Lambda)

```bash
# .env
AMAZON_Q_SIGV4=true
AWS_REGION=us-east-1
# No credentials needed - automatically from instance metadata
```

**How it works:**
1. Gateway detects it's running on AWS (EC2/ECS/Lambda)
2. Retrieves credentials from instance metadata service (IMDS)
3. Credentials automatically rotate before expiration
4. No configuration needed!

#### Option B: IAM User with Access Keys

```bash
# .env
AMAZON_Q_SIGV4=true
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

**Security Note:** Use IAM roles instead of access keys when possible.

#### Option C: Kubernetes with IRSA

```yaml
# kubernetes/serviceaccount.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kiro-gateway
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/kiro-gateway-role
---
# kubernetes/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kiro-gateway
spec:
  template:
    spec:
      serviceAccountName: kiro-gateway
      containers:
      - name: kiro-gateway
        image: kiro-gateway:latest
        env:
        - name: AMAZON_Q_SIGV4
          value: "true"
        - name: AWS_REGION
          value: "us-east-1"
```

**How it works:**
1. EKS injects web identity token into pod
2. Gateway exchanges token for temporary AWS credentials
3. Credentials automatically refresh before expiration

### IAM Policy Required

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "codewhisperer:GenerateRecommendations",
        "codewhisperer:GenerateAssistantResponse"
      ],
      "Resource": "*"
    }
  ]
}
```

### Testing

```bash
# Test with IAM credentials
curl -X POST http://localhost:8090/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "claude-3-5-sonnet-20241022-v2",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

---

## Method 2: Pre-Generated Bearer Token

**Best for:** Development, testing, short-lived environments

### Overview

Use a pre-generated bearer token from AWS SSO. Simple but requires manual renewal.

### Advantages

✅ Simple setup  
✅ No AWS credentials needed  
✅ Works anywhere  

### Disadvantages

❌ Manual token renewal (tokens expire in ~1 hour)  
❌ Not suitable for long-running services  
❌ Security risk if token leaks  

### Setup

#### Step 1: Generate Token

```bash
# Option A: Using AWS CLI
aws sso login --profile YOUR_PROFILE
aws sso get-role-credentials --profile YOUR_PROFILE --role-name YOUR_ROLE --account-id YOUR_ACCOUNT

# Option B: Using Q CLI
qchat auth login
# Token stored in: ~/.aws/sso/cache/*.json
```

#### Step 2: Extract Token

```bash
# Find the SSO cache file
ls -la ~/.aws/sso/cache/

# Extract access token (macOS/Linux)
cat ~/.aws/sso/cache/*.json | jq -r '.accessToken'

# Extract access token (Windows PowerShell)
Get-Content "$env:USERPROFILE\.aws\sso\cache\*.json" | ConvertFrom-Json | Select-Object -ExpandProperty accessToken
```

#### Step 3: Configure Gateway

```bash
# .env
AMAZON_Q_SIGV4=false
AWS_REGION=us-east-1
BEARER_TOKEN=eyJraWQiOiJ...  # Your extracted token
PROFILE_ARN=arn:aws:codewhisperer:us-east-1:123456789012:profile/abc123
```

### Automation Script

```bash
#!/bin/bash
# scripts/get-bearer-token.sh

set -e

# Login to AWS SSO
aws sso login --profile "${AWS_PROFILE:-default}"

# Find SSO cache file
CACHE_FILE=$(ls -t ~/.aws/sso/cache/*.json | head -1)

# Extract token
TOKEN=$(cat "$CACHE_FILE" | jq -r '.accessToken')

# Export for use
export BEARER_TOKEN="$TOKEN"

echo "Bearer token exported to BEARER_TOKEN environment variable"
echo "Token expires at: $(cat "$CACHE_FILE" | jq -r '.expiresAt')"
```

**Usage:**
```bash
source scripts/get-bearer-token.sh
./kiro-gateway
```

---

## Method 3: AWS SSO with Refresh Token

**Best for:** Corporate environments with IAM Identity Center

### Overview

Uses AWS SSO refresh token for automatic token renewal. Refresh tokens are long-lived (90 days).

### Advantages

✅ Automatic token refresh  
✅ Long-lived credentials (90 days)  
✅ Works with IAM Identity Center  
✅ Suitable for long-running services  

### Setup

#### Step 1: Initial Authentication

```bash
# Authenticate with AWS SSO
aws sso login --profile YOUR_PROFILE

# Or use Q CLI
qchat auth login
```

#### Step 2: Extract Refresh Token

```bash
# Find SSO cache file
CACHE_FILE=$(ls -t ~/.aws/sso/cache/*.json | head -1)

# Extract refresh token
REFRESH_TOKEN=$(cat "$CACHE_FILE" | jq -r '.refreshToken')

# Store securely (use secrets manager in production)
echo "$REFRESH_TOKEN" > .refresh_token
chmod 600 .refresh_token
```

#### Step 3: Configure Gateway

```bash
# .env
AUTH_TYPE=cli_db
CLI_DB_PATH=/path/to/.aws/sso/cache/token.json
AWS_REGION=us-east-1
PROFILE_ARN=arn:aws:codewhisperer:us-east-1:123456789012:profile/abc123
```

### Automated Token Refresh

The gateway automatically refreshes tokens using the CLI DB method:

```go
// Gateway checks token expiration on every request
if tokenExpiresWithin(1 * time.Minute) {
    // Reload token from SSO cache file
    // AWS CLI has already refreshed it in background
    token = reloadFromFile()
}
```

### Docker Example

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o kiro-gateway ./cmd/kiro-gateway

FROM alpine:latest
RUN apk add --no-cache ca-certificates aws-cli
WORKDIR /app
COPY --from=builder /app/kiro-gateway .

# Copy SSO cache (mounted as volume)
VOLUME /root/.aws/sso/cache

CMD ["./kiro-gateway"]
```

```bash
# docker-compose.yml
version: '3.8'
services:
  kiro-gateway:
    build: .
    ports:
      - "8090:8090"
    volumes:
      - ~/.aws/sso/cache:/root/.aws/sso/cache:ro
    environment:
      - AUTH_TYPE=cli_db
      - CLI_DB_PATH=/root/.aws/sso/cache/token.json
      - AWS_REGION=us-east-1
      - PROFILE_ARN=arn:aws:codewhisperer:us-east-1:123456789012:profile/abc123
```

---

## Method 4: Service Account Credentials

**Best for:** Multi-tenant SaaS, service-to-service authentication

### Overview

Create dedicated service account credentials for automated access.

### Setup

#### Step 1: Create Service Account

```bash
# Using AWS IAM
aws iam create-user --user-name kiro-gateway-service

# Attach policy
aws iam attach-user-policy \
  --user-name kiro-gateway-service \
  --policy-arn arn:aws:iam::aws:policy/AmazonCodeWhispererFullAccess

# Create access key
aws iam create-access-key --user-name kiro-gateway-service
```

#### Step 2: Store Credentials Securely

```bash
# Option A: AWS Secrets Manager
aws secretsmanager create-secret \
  --name kiro-gateway/credentials \
  --secret-string '{
    "access_key_id": "AKIAIOSFODNN7EXAMPLE",
    "secret_access_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
  }'

# Option B: Environment variables (less secure)
export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

#### Step 3: Configure Gateway

```bash
# .env
AMAZON_Q_SIGV4=true
AWS_REGION=us-east-1
# Credentials from environment or Secrets Manager
```

### Credential Rotation

```bash
#!/bin/bash
# scripts/rotate-credentials.sh

set -e

USER_NAME="kiro-gateway-service"

# List current access keys
KEYS=$(aws iam list-access-keys --user-name "$USER_NAME" --query 'AccessKeyMetadata[*].AccessKeyId' --output text)

# Create new access key
NEW_KEY=$(aws iam create-access-key --user-name "$USER_NAME")
NEW_ACCESS_KEY=$(echo "$NEW_KEY" | jq -r '.AccessKey.AccessKeyId')
NEW_SECRET_KEY=$(echo "$NEW_KEY" | jq -r '.AccessKey.SecretAccessKey')

# Update Secrets Manager
aws secretsmanager update-secret \
  --secret-id kiro-gateway/credentials \
  --secret-string "{
    \"access_key_id\": \"$NEW_ACCESS_KEY\",
    \"secret_access_key\": \"$NEW_SECRET_KEY\"
  }"

# Wait for propagation
sleep 30

# Delete old keys
for KEY in $KEYS; do
  aws iam delete-access-key --user-name "$USER_NAME" --access-key-id "$KEY"
done

echo "Credentials rotated successfully"
```

---

## Method 5: Headless SSO OIDC (Self-Contained)

**Best for:** Docker containers, CI/CD pipelines, any environment without AWS CLI

### Overview

Implements the complete OAuth 2.0 device authorization flow directly in the gateway, eliminating the AWS CLI dependency. This is the most Docker-friendly and CI/CD-friendly method.

### How It Works

```
1. Gateway registers as OIDC client with IAM Identity Center
2. Gateway requests device authorization code
3. User visits URL and enters code (one-time, can be done on any device)
4. Gateway polls for authorization completion
5. Gateway receives access token + refresh token
6. Gateway exchanges access token for AWS credentials
7. Gateway signs requests with SigV4
8. Background worker automatically refreshes tokens before expiration
```

### Advantages

✅ **No AWS CLI dependency** - Fully self-contained  
✅ **Docker-friendly** - Works in minimal containers  
✅ **Automatic refresh** - Tokens refresh in background  
✅ **One-time setup** - User authorizes once, then fully automated  
✅ **Secure storage** - Tokens encrypted at rest  
✅ **Production ready** - Supports pre-registered clients  
✅ **Audit trail** - All operations logged  

### Configuration

#### Basic Setup (Dynamic Client Registration)

```bash
# .env
HEADLESS_MODE=true
SSO_START_URL=https://my-portal.awsapps.com/start
SSO_REGION=us-east-1
SSO_ACCOUNT_ID=123456789012
SSO_ROLE_NAME=MyRole
AMAZON_Q_SIGV4=true
AWS_REGION=us-east-1
```

**How it works:**
- Gateway dynamically registers as OIDC client on first run
- Client credentials cached for future use
- Client registration expires after 90 days (automatically re-registers)

#### Production Setup (Pre-Registered Client)

For production deployments, pre-register your OIDC client:

```bash
# 1. Register client manually (one-time)
aws ssooidc register-client \
  --client-name "kiro-gateway-production" \
  --client-type "public" \
  --scopes "sso:account:access" \
  --region us-east-1

# Output:
# {
#   "clientId": "abc123...",
#   "clientSecret": "xyz789...",
#   "clientSecretExpiresAt": 1234567890
# }

# 2. Configure gateway with pre-registered client
cat > .env <<EOF
HEADLESS_MODE=true
SSO_START_URL=https://my-portal.awsapps.com/start
SSO_REGION=us-east-1
SSO_ACCOUNT_ID=123456789012
SSO_ROLE_NAME=MyRole
SSO_CLIENT_ID=abc123...
SSO_CLIENT_SECRET=xyz789...
SSO_CLIENT_EXPIRY=1234567890
AMAZON_Q_SIGV4=true
AWS_REGION=us-east-1
EOF
```

**Benefits:**
- Faster startup (no client registration needed)
- Better for production (consistent client ID)
- Can be managed centrally

### First-Time Setup

#### Step 1: Start Gateway

```bash
./kiro-gateway
```

#### Step 2: Authorize Device

Gateway will display authorization instructions:

```
╔════════════════════════════════════════════════════════════════════════╗
║                AWS IAM Identity Center Authentication                 ║
╠════════════════════════════════════════════════════════════════════════╣
║                                                                        ║
║  Please visit the following URL to authorize this device:             ║
║                                                                        ║
║  https://device.sso.us-east-1.amazonaws.com/                          ║
║                                                                        ║
║  And enter this code:                                                 ║
║                                                                        ║
║                    ABCD-1234                                          ║
║                                                                        ║
║  Or visit this URL directly (auto-fills code):                        ║
║  https://device.sso.us-east-1.amazonaws.com/?user_code=ABCD-1234     ║
║                                                                        ║
║  ⏳ Waiting for authorization...                                       ║
║  (Code expires in 600 seconds)                                        ║
║                                                                        ║
╚════════════════════════════════════════════════════════════════════════╝
```

#### Step 3: Complete Authorization

1. Visit the URL in any browser (can be on different device)
2. Enter the code shown
3. Sign in with your IAM Identity Center credentials
4. Approve the authorization request

#### Step 4: Gateway Continues Automatically

```
[HEADLESS] ✅ Authorization successful!
[HEADLESS] ✅ Access token refreshed
[HEADLESS] ✅ AWS credentials obtained (expires: 2026-01-24T18:30:00Z)
[HEADLESS] Starting background token refresh...
Starting Kiro Gateway on port 8090
```

### Subsequent Runs

After initial authorization, the gateway runs fully automatically:

```bash
# No user interaction needed!
./kiro-gateway

# Output:
# [HEADLESS] Initializing headless authentication...
# [HEADLESS] Found existing token
# [HEADLESS] ✅ AWS credentials obtained (expires: 2026-01-24T18:30:00Z)
# [HEADLESS] Starting background token refresh...
# Starting Kiro Gateway on port 8090
```

### Docker Deployment

#### Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o kiro-gateway ./cmd/kiro-gateway

FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/kiro-gateway .

# Create directory for token storage
RUN mkdir -p /app/.kiro/tokens

# Volume for persistent token storage
VOLUME /app/.kiro/tokens

CMD ["./kiro-gateway"]
```

#### Docker Compose

```yaml
version: '3.8'

services:
  kiro-gateway:
    build: .
    ports:
      - "8090:8090"
    environment:
      # Headless OIDC configuration
      - HEADLESS_MODE=true
      - SSO_START_URL=https://my-portal.awsapps.com/start
      - SSO_REGION=us-east-1
      - SSO_ACCOUNT_ID=123456789012
      - SSO_ROLE_NAME=MyRole
      
      # Optional: Pre-registered client
      - SSO_CLIENT_ID=${SSO_CLIENT_ID}
      - SSO_CLIENT_SECRET=${SSO_CLIENT_SECRET}
      - SSO_CLIENT_EXPIRY=${SSO_CLIENT_EXPIRY}
      
      # Gateway configuration
      - AMAZON_Q_SIGV4=true
      - AWS_REGION=us-east-1
      - PORT=8090
    
    # Persist tokens across container restarts
    volumes:
      - ./tokens:/app/.kiro/tokens
    
    restart: unless-stopped
    
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8090/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

#### First Run (Interactive)

```bash
# Start container with TTY for initial authorization
docker-compose run --rm kiro-gateway

# Follow authorization instructions
# Tokens will be saved to ./tokens/ directory
```

#### Subsequent Runs (Automated)

```bash
# Start in background - fully automated
docker-compose up -d

# Tokens are loaded from ./tokens/ directory
# No user interaction needed!
```

### Kubernetes Deployment

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kiro-gateway-config
data:
  HEADLESS_MODE: "true"
  SSO_START_URL: "https://my-portal.awsapps.com/start"
  SSO_REGION: "us-east-1"
  SSO_ACCOUNT_ID: "123456789012"
  SSO_ROLE_NAME: "MyRole"
  AMAZON_Q_SIGV4: "true"
  AWS_REGION: "us-east-1"
---
apiVersion: v1
kind: Secret
metadata:
  name: kiro-gateway-oidc-client
type: Opaque
stringData:
  SSO_CLIENT_ID: "abc123..."
  SSO_CLIENT_SECRET: "xyz789..."
  SSO_CLIENT_EXPIRY: "1234567890"
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: kiro-gateway-tokens
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kiro-gateway
spec:
  replicas: 2
  selector:
    matchLabels:
      app: kiro-gateway
  template:
    metadata:
      labels:
        app: kiro-gateway
    spec:
      containers:
      - name: kiro-gateway
        image: kiro-gateway:latest
        ports:
        - containerPort: 8090
        envFrom:
        - configMapRef:
            name: kiro-gateway-config
        - secretRef:
            name: kiro-gateway-oidc-client
        volumeMounts:
        - name: tokens
          mountPath: /app/.kiro/tokens
        livenessProbe:
          httpGet:
            path: /health
            port: 8090
          initialDelaySeconds: 30
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health
            port: 8090
          initialDelaySeconds: 10
          periodSeconds: 10
      volumes:
      - name: tokens
        persistentVolumeClaim:
          claimName: kiro-gateway-tokens
---
apiVersion: v1
kind: Service
metadata:
  name: kiro-gateway
spec:
  selector:
    app: kiro-gateway
  ports:
  - port: 8090
    targetPort: 8090
  type: LoadBalancer
```

**Initial Setup:**
```bash
# 1. Deploy to Kubernetes
kubectl apply -f kubernetes/

# 2. Get pod name
POD=$(kubectl get pods -l app=kiro-gateway -o jsonpath='{.items[0].metadata.name}')

# 3. View authorization instructions
kubectl logs -f $POD

# 4. Complete authorization in browser

# 5. Tokens are persisted in PVC
# Subsequent pod restarts are fully automated
```

### Token Management

#### Token Lifecycle

```
Access Token:  Expires in ~1 hour  → Auto-refreshed by gateway
Refresh Token: Expires in ~90 days → Used to get new access tokens
Client Secret: Expires in ~90 days → Auto-re-registered if needed
AWS Credentials: Expire in ~1 hour → Auto-refreshed from access token
```

#### Automatic Refresh

The gateway automatically handles all token refresh:

```go
// Background worker runs every minute
// Checks token expiration and refreshes if needed

if accessTokenExpiresWithin(5 * time.Minute) {
    // Refresh access token using refresh token
    newToken = oidcClient.RefreshToken(refreshToken)
}

if awsCredentialsExpireWithin(5 * time.Minute) {
    // Get new AWS credentials using access token
    newCreds = ssoClient.GetRoleCredentials(accessToken)
}
```

#### Manual Token Refresh

If needed, you can trigger manual refresh:

```bash
# Delete stored tokens to force re-authentication
rm -rf .kiro/tokens/*

# Restart gateway
./kiro-gateway

# Follow authorization instructions again
```

### Monitoring and Logging

#### Enable Debug Logging

```bash
export LOG_LEVEL=debug
./kiro-gateway
```

#### Log Output

```
[HEADLESS] Initializing headless authentication...
[HEADLESS] Found existing token
[HEADLESS] Token expires at: 2026-01-24T17:30:00Z
[HEADLESS] ✅ AWS credentials obtained (expires: 2026-01-24T18:30:00Z)
[HEADLESS] Starting background token refresh...
[HEADLESS] Background refresh: Token valid for 45 minutes
[HEADLESS] Background refresh: Credentials valid for 55 minutes
```

#### Health Check

```bash
curl http://localhost:8090/health

# Response:
# {
#   "status": "healthy",
#   "auth_mode": "headless",
#   "token_expires_in": "45m",
#   "credentials_expire_in": "55m"
# }
```

### Troubleshooting

#### Issue: "Device code expired"

**Cause:** User didn't authorize within 10 minutes

**Solution:**
```bash
# Restart gateway to get new device code
./kiro-gateway
```

#### Issue: "Refresh token expired"

**Cause:** Refresh token expired after 90 days

**Solution:**
```bash
# Delete tokens and re-authenticate
rm -rf .kiro/tokens/*
./kiro-gateway
# Follow authorization instructions
```

#### Issue: "Client registration expired"

**Cause:** Client secret expired after 90 days

**Solution:**
```bash
# Gateway automatically re-registers
# No action needed - just restart
./kiro-gateway
```

#### Issue: "Cannot connect to OIDC endpoint"

**Cause:** Network connectivity or firewall

**Solution:**
```bash
# Test connectivity
curl https://oidc.us-east-1.amazonaws.com/

# Check firewall rules
# Ensure outbound HTTPS (443) is allowed
```

### Security Considerations

**Token Storage:**
- Tokens stored in encrypted keychain (macOS/Linux)
- Or encrypted SQLite database (fallback)
- File permissions: 0600 (owner read/write only)

**Network Security:**
- All OIDC communication over HTTPS
- TLS certificate validation enabled
- No credentials in logs (redacted)

**Access Control:**
- Tokens scoped to specific AWS account and role
- Automatic token rotation
- Audit trail via CloudTrail

### Comparison with CLI-Based Method

| Feature | Headless OIDC | CLI-Based |
|---------|---------------|-----------|
| AWS CLI Required | ❌ No | ✅ Yes |
| Docker-Friendly | ✅ Yes | ⚠️ Requires CLI in container |
| Container Size | 📦 Small (~20MB) | 📦 Large (~200MB with CLI) |
| Startup Time | ⚡ Fast (~1s) | 🐌 Slow (~5s) |
| Token Refresh | ✅ Built-in | ⚠️ Depends on CLI |
| Production Ready | ✅ Yes | ⚠️ CLI dependency |
| Complexity | 🟢 Low | 🟡 Medium |

---

## CI/CD Integration Examples

### GitHub Actions (Headless OIDC)

```yaml
# .github/workflows/deploy-headless.yml
name: Deploy Kiro Gateway (Headless OIDC)

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Build Gateway
        run: |
          go build -o kiro-gateway ./cmd/kiro-gateway
      
      - name: Configure Headless OIDC
        env:
          SSO_CLIENT_ID: ${{ secrets.SSO_CLIENT_ID }}
          SSO_CLIENT_SECRET: ${{ secrets.SSO_CLIENT_SECRET }}
        run: |
          cat > .env <<EOF
          HEADLESS_MODE=true
          SSO_START_URL=${{ secrets.SSO_START_URL }}
          SSO_REGION=us-east-1
          SSO_ACCOUNT_ID=${{ secrets.SSO_ACCOUNT_ID }}
          SSO_ROLE_NAME=${{ secrets.SSO_ROLE_NAME }}
          SSO_CLIENT_ID=${SSO_CLIENT_ID}
          SSO_CLIENT_SECRET=${SSO_CLIENT_SECRET}
          AMAZON_Q_SIGV4=true
          AWS_REGION=us-east-1
          EOF
      
      - name: Restore Tokens from Cache
        uses: actions/cache@v3
        with:
          path: .kiro/tokens
          key: kiro-tokens-${{ github.sha }}
          restore-keys: |
            kiro-tokens-
      
      - name: Test Gateway
        run: |
          ./kiro-gateway &
          GATEWAY_PID=$!
          sleep 10
          
          # Test health endpoint
          curl -f http://localhost:8090/health
          
          # Test chat endpoint
          curl -X POST http://localhost:8090/v1/chat/completions \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer ${{ secrets.API_KEY }}" \
            -d '{"model":"claude-3-5-sonnet-20241022-v2","messages":[{"role":"user","content":"test"}]}'
          
          kill $GATEWAY_PID
      
      - name: Build Docker Image
        run: |
          docker build -t kiro-gateway:${{ github.sha }} .
      
      - name: Deploy to Production
        run: |
          # Deploy using your preferred method
          # Tokens will be loaded from persistent volume
          echo "Deploying to production..."
```

### GitHub Actions

```yaml
# .github/workflows/deploy.yml
name: Deploy Kiro Gateway

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    permissions:
      id-token: write  # Required for OIDC
      contents: read
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Configure AWS Credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: arn:aws:iam::123456789012:role/github-actions-role
          aws-region: us-east-1
      
      - name: Build Gateway
        run: |
          go build -o kiro-gateway ./cmd/kiro-gateway
      
      - name: Test with SigV4
        env:
          AMAZON_Q_SIGV4: "true"
          AWS_REGION: us-east-1
        run: |
          ./kiro-gateway &
          sleep 5
          curl -X POST http://localhost:8090/v1/chat/completions \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer ${{ secrets.API_KEY }}" \
            -d '{"model":"claude-3-5-sonnet-20241022-v2","messages":[{"role":"user","content":"test"}]}'
      
      - name: Deploy to ECS
        run: |
          # Deploy using AWS CLI or Terraform
          aws ecs update-service --cluster my-cluster --service kiro-gateway --force-new-deployment
```

### GitLab CI

```yaml
# .gitlab-ci.yml
variables:
  AWS_REGION: us-east-1
  AMAZON_Q_SIGV4: "true"

stages:
  - build
  - test
  - deploy

build:
  stage: build
  image: golang:1.21
  script:
    - go build -o kiro-gateway ./cmd/kiro-gateway
  artifacts:
    paths:
      - kiro-gateway

test:
  stage: test
  image: golang:1.21
  services:
    - name: kiro-gateway:latest
  variables:
    AWS_ACCESS_KEY_ID: $AWS_ACCESS_KEY_ID
    AWS_SECRET_ACCESS_KEY: $AWS_SECRET_ACCESS_KEY
  script:
    - ./kiro-gateway &
    - sleep 5
    - curl -X POST http://localhost:8090/v1/chat/completions \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $API_KEY" \
        -d '{"model":"claude-3-5-sonnet-20241022-v2","messages":[{"role":"user","content":"test"}]}'

deploy:
  stage: deploy
  image: amazon/aws-cli
  script:
    - aws ecs update-service --cluster my-cluster --service kiro-gateway --force-new-deployment
  only:
    - main
```

### Jenkins

```groovy
// Jenkinsfile
pipeline {
    agent any
    
    environment {
        AWS_REGION = 'us-east-1'
        AMAZON_Q_SIGV4 = 'true'
        AWS_CREDENTIALS = credentials('aws-credentials-id')
    }
    
    stages {
        stage('Build') {
            steps {
                sh 'go build -o kiro-gateway ./cmd/kiro-gateway'
            }
        }
        
        stage('Test') {
            steps {
                sh '''
                    ./kiro-gateway &
                    GATEWAY_PID=$!
                    sleep 5
                    
                    curl -X POST http://localhost:8090/v1/chat/completions \
                      -H "Content-Type: application/json" \
                      -H "Authorization: Bearer ${API_KEY}" \
                      -d '{"model":"claude-3-5-sonnet-20241022-v2","messages":[{"role":"user","content":"test"}]}'
                    
                    kill $GATEWAY_PID
                '''
            }
        }
        
        stage('Deploy') {
            when {
                branch 'main'
            }
            steps {
                sh 'aws ecs update-service --cluster my-cluster --service kiro-gateway --force-new-deployment'
            }
        }
    }
}
```

### Docker Compose

```yaml
# docker-compose.yml
version: '3.8'

services:
  kiro-gateway:
    build: .
    ports:
      - "8090:8090"
    environment:
      # Method 1: IAM credentials
      - AMAZON_Q_SIGV4=true
      - AWS_REGION=us-east-1
      - AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID}
      - AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY}
      
      # Method 2: Bearer token
      # - AMAZON_Q_SIGV4=false
      # - BEARER_TOKEN=${BEARER_TOKEN}
      # - PROFILE_ARN=${PROFILE_ARN}
    
    # Optional: Mount AWS credentials
    volumes:
      - ~/.aws:/root/.aws:ro
    
    restart: unless-stopped
    
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8090/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

---

## Security Best Practices

### Credential Storage

**DO:**
- ✅ Use AWS Secrets Manager or Parameter Store
- ✅ Use environment variables for CI/CD
- ✅ Rotate credentials regularly (90 days max)
- ✅ Use IAM roles instead of access keys when possible
- ✅ Encrypt credentials at rest
- ✅ Use least-privilege IAM policies

**DON'T:**
- ❌ Commit credentials to version control
- ❌ Store credentials in plain text files
- ❌ Share credentials between environments
- ❌ Use root account credentials
- ❌ Disable credential rotation
- ❌ Log credential values

### Network Security

**DO:**
- ✅ Use HTTPS for all API calls
- ✅ Implement network segmentation
- ✅ Use VPC endpoints for AWS services
- ✅ Enable CloudTrail logging
- ✅ Monitor for unusual access patterns

**DON'T:**
- ❌ Expose gateway to public internet
- ❌ Disable SSL verification
- ❌ Use insecure protocols
- ❌ Allow unrestricted outbound access

### Access Control

**DO:**
- ✅ Use separate credentials per environment
- ✅ Implement IP allowlisting
- ✅ Enable MFA for human access
- ✅ Use service accounts for automation
- ✅ Audit access logs regularly

**DON'T:**
- ❌ Use personal credentials for automation
- ❌ Grant overly broad permissions
- ❌ Skip access reviews
- ❌ Ignore security alerts

---

## Troubleshooting

### Issue: "No credentials found"

**Cause:** Gateway cannot find AWS credentials

**Solution:**
```bash
# Check credential chain
aws sts get-caller-identity

# Verify environment variables
echo $AWS_ACCESS_KEY_ID
echo $AWS_SECRET_ACCESS_KEY

# Check IAM role (EC2/ECS)
curl http://169.254.169.254/latest/meta-data/iam/security-credentials/

# Enable debug logging
export DEBUG=true
export LOG_LEVEL=debug
./kiro-gateway
```

### Issue: "Token expired"

**Cause:** Bearer token has expired

**Solution:**
```bash
# Method 1: Use SigV4 instead (recommended)
export AMAZON_Q_SIGV4=true

# Method 2: Refresh token manually
aws sso login --profile YOUR_PROFILE
# Extract new token and update BEARER_TOKEN

# Method 3: Use CLI DB with auto-refresh
export AUTH_TYPE=cli_db
export CLI_DB_PATH=~/.aws/sso/cache/token.json
```

### Issue: "Access denied"

**Cause:** Insufficient IAM permissions

**Solution:**
```bash
# Check current permissions
aws sts get-caller-identity
aws iam get-user

# Verify required permissions
aws iam simulate-principal-policy \
  --policy-source-arn arn:aws:iam::123456789012:user/kiro-gateway \
  --action-names codewhisperer:GenerateRecommendations \
  --action-names codewhisperer:GenerateAssistantResponse

# Attach required policy
aws iam attach-user-policy \
  --user-name kiro-gateway \
  --policy-arn arn:aws:iam::aws:policy/AmazonCodeWhispererFullAccess
```

### Issue: "Connection timeout"

**Cause:** Network connectivity issues

**Solution:**
```bash
# Test connectivity
curl -v https://codewhisperer.us-east-1.amazonaws.com

# Check VPC endpoints (if using)
aws ec2 describe-vpc-endpoints

# Verify security groups
aws ec2 describe-security-groups --group-ids sg-xxxxx

# Check DNS resolution
nslookup codewhisperer.us-east-1.amazonaws.com
```

---

## Complete Example: Production Deployment

```bash
# 1. Create IAM role for ECS task
aws iam create-role \
  --role-name kiro-gateway-task-role \
  --assume-role-policy-document file://trust-policy.json

# 2. Attach permissions
aws iam attach-role-policy \
  --role-name kiro-gateway-task-role \
  --policy-arn arn:aws:iam::aws:policy/AmazonCodeWhispererFullAccess

# 3. Create ECS task definition
cat > task-definition.json <<EOF
{
  "family": "kiro-gateway",
  "taskRoleArn": "arn:aws:iam::123456789012:role/kiro-gateway-task-role",
  "executionRoleArn": "arn:aws:iam::123456789012:role/ecsTaskExecutionRole",
  "networkMode": "awsvpc",
  "containerDefinitions": [
    {
      "name": "kiro-gateway",
      "image": "123456789012.dkr.ecr.us-east-1.amazonaws.com/kiro-gateway:latest",
      "portMappings": [
        {
          "containerPort": 8090,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "AMAZON_Q_SIGV4",
          "value": "true"
        },
        {
          "name": "AWS_REGION",
          "value": "us-east-1"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/kiro-gateway",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ],
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512"
}
EOF

# 4. Register task definition
aws ecs register-task-definition --cli-input-json file://task-definition.json

# 5. Create ECS service
aws ecs create-service \
  --cluster my-cluster \
  --service-name kiro-gateway \
  --task-definition kiro-gateway \
  --desired-count 2 \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-xxx,subnet-yyy],securityGroups=[sg-xxx],assignPublicIp=DISABLED}"

# 6. Verify deployment
aws ecs describe-services --cluster my-cluster --services kiro-gateway
```

---

## Summary

**For production:** Use **IAM + SigV4** with IAM roles  
**For development:** Use **Bearer Token** or **CLI DB**  
**For CI/CD:** Use **OIDC with GitHub Actions** or **IAM roles**  
**For Kubernetes:** Use **IRSA** (IAM Roles for Service Accounts)

All methods support fully automated, headless operation with no human interaction required.

---

**Last Updated:** January 24, 2026  
**Version:** 1.0.0  
**Status:** Production Ready

