# Headless OIDC Quick Start Guide

**Get started with headless authentication in 5 minutes**

---

## What is Headless OIDC?

Headless OIDC mode allows Kiro Gateway to authenticate with AWS IAM Identity Center **without requiring the AWS CLI**. Perfect for:

- 🐳 Docker containers
- 🔄 CI/CD pipelines
- ☁️ Cloud deployments
- 🤖 Automated environments

---

## Prerequisites

✅ AWS IAM Identity Center enabled  
✅ Q Developer Pro subscription  
✅ SSO start URL (e.g., `https://my-portal.awsapps.com/start`)  
✅ AWS account ID and role name  

---

## Quick Start

### Step 1: Configure Environment

```bash
cat > .env <<EOF
# Enable headless mode
HEADLESS_MODE=true

# IAM Identity Center configuration
SSO_START_URL=https://my-portal.awsapps.com/start
SSO_REGION=us-east-1
SSO_ACCOUNT_ID=123456789012
SSO_ROLE_NAME=MyRole

# Enable SigV4 authentication
AMAZON_Q_SIGV4=true
AWS_REGION=us-east-1
EOF
```

### Step 2: Start Gateway

```bash
./kiro-gateway
```

### Step 3: Authorize Device (One-Time)

Gateway will display:

```
╔════════════════════════════════════════════════════════════╗
║          AWS IAM Identity Center Authentication           ║
╠════════════════════════════════════════════════════════════╣
║  Please visit: https://device.sso.us-east-1.amazonaws.com ║
║  And enter code: ABCD-1234                                ║
╚════════════════════════════════════════════════════════════╝
```

1. Visit the URL in any browser
2. Enter the code shown
3. Sign in with your Identity Center credentials
4. Approve the authorization

### Step 4: Done!

Gateway continues automatically:

```
[HEADLESS] ✅ Authorization successful!
[HEADLESS] ✅ AWS credentials obtained
[HEADLESS] Starting background token refresh...
Starting Kiro Gateway on port 8090
```

### Step 5: Test It

```bash
curl -X POST http://localhost:8090/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "claude-3-5-sonnet-20241022-v2",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

---

## Docker Quick Start

### Step 1: Create docker-compose.yml

```yaml
version: '3.8'

services:
  kiro-gateway:
    image: kiro-gateway:latest
    ports:
      - "8090:8090"
    environment:
      - HEADLESS_MODE=true
      - SSO_START_URL=https://my-portal.awsapps.com/start
      - SSO_REGION=us-east-1
      - SSO_ACCOUNT_ID=123456789012
      - SSO_ROLE_NAME=MyRole
      - AMAZON_Q_SIGV4=true
      - AWS_REGION=us-east-1
    volumes:
      - ./tokens:/app/.kiro/tokens
    restart: unless-stopped
```

### Step 2: First Run (Interactive)

```bash
# Start with TTY for authorization
docker-compose run --rm kiro-gateway

# Follow authorization instructions
# Tokens saved to ./tokens/
```

### Step 3: Subsequent Runs (Automated)

```bash
# Start in background - fully automated!
docker-compose up -d

# No user interaction needed
```

---

## Production Setup

For production, pre-register your OIDC client:

```bash
# 1. Register client
aws ssooidc register-client \
  --client-name "kiro-gateway-prod" \
  --client-type "public" \
  --scopes "sso:account:access" \
  --region us-east-1

# 2. Add to .env
cat >> .env <<EOF
SSO_CLIENT_ID=abc123...
SSO_CLIENT_SECRET=xyz789...
SSO_CLIENT_EXPIRY=1234567890
EOF

# 3. Store in secrets manager
aws secretsmanager create-secret \
  --name kiro-gateway/oidc-client \
  --secret-string "{
    \"client_id\": \"abc123...\",
    \"client_secret\": \"xyz789...\"
  }"
```

---

## Troubleshooting

### "Device code expired"

**Solution:** Restart gateway to get new code

```bash
./kiro-gateway
```

### "Refresh token expired"

**Solution:** Re-authenticate (happens after 90 days)

```bash
rm -rf .kiro/tokens/*
./kiro-gateway
```

### "Cannot connect to OIDC endpoint"

**Solution:** Check network connectivity

```bash
curl https://oidc.us-east-1.amazonaws.com/
```

---

## Next Steps

- 📖 Read [Complete Headless Guide](HEADLESS_AUTHENTICATION.md)
- 🔒 Review [Security Best Practices](SECURITY.md)
- 🚀 See [CI/CD Integration Examples](HEADLESS_AUTHENTICATION.md#cicd-integration-examples)
- 🐳 Learn [Docker Deployment](HEADLESS_AUTHENTICATION.md#docker-deployment)

---

## Key Benefits

✅ **No AWS CLI** - Fully self-contained  
✅ **Auto-refresh** - Tokens refresh automatically  
✅ **One-time setup** - Authorize once, then automated  
✅ **Docker-friendly** - Works in minimal containers  
✅ **Production ready** - Secure and reliable  

---

**Last Updated:** January 24, 2026  
**Version:** 1.0.0
