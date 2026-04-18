# SSO SigV4 Configuration Guide

**Configure SigV4 signing with SSO-derived IAM credentials for Q Developer mode**

---

## Overview

This guide explains how to configure the gateway to use **SigV4 signing with SSO-derived IAM credentials**. This is required for **Q Developer mode** when you want to use SigV4 instead of bearer tokens.

### Two Authentication Paths

The gateway supports two distinct authentication paths:

#### 1. Bearer Token Mode (CodeWhisperer)
```
Identity Center → OIDC → Bearer Token → API
```
- Uses `Authorization: Bearer <token>` headers
- Standard OIDC/OAuth2 flow
- **Configuration:** `AMAZON_Q_SIGV4=false` (default)

#### 2. SigV4 Mode with SSO Credentials (Q Developer)
```
Identity Center → OIDC → Bearer Token → SSO API → IAM Credentials → SigV4 → API
```
- Converts bearer tokens to temporary IAM credentials
- Uses AWS SigV4 signing
- **Configuration:** `AMAZON_Q_SIGV4=true` + `USE_SSO_CREDENTIALS=true`

---

## When to Use Each Mode

### Use Bearer Token Mode When:
- ✅ Using CodeWhisperer API
- ✅ Standard Identity Center authentication
- ✅ Simpler configuration needed
- ✅ No IAM credential conversion required

### Use SigV4 with SSO Credentials When:
- ✅ Using Q Developer API
- ✅ Need SigV4 signing for compliance/audit
- ✅ Want IAM-style credential management
- ✅ CloudTrail logging required

---

## Configuration

### Environment Variables

#### Required for All Modes

```bash
# AWS Configuration
AWS_REGION=us-east-1

# Identity Center
PROFILE_ARN=arn:aws:codewhisperer:us-east-1:123456789012:profile/abc123

# Authentication Type
AUTH_TYPE=cli_db  # or automated_oidc
```

#### For Bearer Token Mode (CodeWhisperer)

```bash
# Disable SigV4 (use bearer tokens)
AMAZON_Q_SIGV4=false

# CLI DB path for token storage
CLI_DB_PATH=/path/to/.aws/sso/cache/token.json
```

#### For SigV4 Mode with SSO Credentials (Q Developer)

```bash
# Enable SigV4
AMAZON_Q_SIGV4=true

# Enable SSO credential conversion
USE_SSO_CREDENTIALS=true

# SSO Configuration (REQUIRED)
AWS_SSO_ACCOUNT_ID=123456789012
AWS_SSO_ROLE_NAME=YourSSORole

# CLI DB path for bearer token (used to get IAM credentials)
CLI_DB_PATH=/path/to/.aws/sso/cache/token.json
```

---

## Getting SSO Account ID and Role Name

### Method 1: From AWS SSO Configuration

```bash
# List your SSO configuration
aws configure list-profiles

# Get SSO configuration for a profile
aws configure get sso_account_id --profile YOUR_PROFILE
aws configure get sso_role_name --profile YOUR_PROFILE
```

### Method 2: From AWS Console

1. Go to **AWS IAM Identity Center**
2. Navigate to **AWS accounts**
3. Select your account
4. Note the **Account ID**
5. Click on **Permission sets**
6. Note the **Role name** (e.g., `PowerUserAccess`, `AdministratorAccess`)

### Method 3: From AWS CLI

```bash
# Get account ID
aws sts get-caller-identity --query Account --output text

# List available roles
aws iam list-roles --query 'Roles[?contains(RoleName, `SSO`)].RoleName'
```

### Method 4: From SSO Start URL

Your SSO start URL contains hints:
```
https://YOUR-ORG.awsapps.com/start
```

Check your `~/.aws/config`:
```ini
[profile YOUR_PROFILE]
sso_start_url = https://YOUR-ORG.awsapps.com/start
sso_region = us-east-1
sso_account_id = 123456789012
sso_role_name = PowerUserAccess
```

---

## Complete Configuration Examples

### Example 1: CodeWhisperer Mode (Bearer Token)

```bash
# .env
PORT=8080
AWS_REGION=us-east-1

# Bearer Token Mode
AMAZON_Q_SIGV4=false

# Identity Center
PROFILE_ARN=arn:aws:codewhisperer:us-east-1:123456789012:profile/abc123
AUTH_TYPE=cli_db
CLI_DB_PATH=/home/user/.aws/sso/cache/abc123def456.json

# Logging
LOG_LEVEL=info
```

### Example 2: Q Developer Mode (SigV4 with SSO)

```bash
# .env
PORT=8080
AWS_REGION=us-east-1

# SigV4 Mode with SSO Credentials
AMAZON_Q_SIGV4=true
USE_SSO_CREDENTIALS=true

# SSO Configuration
AWS_SSO_ACCOUNT_ID=123456789012
AWS_SSO_ROLE_NAME=PowerUserAccess

# Identity Center
PROFILE_ARN=arn:aws:codewhisperer:us-east-1:123456789012:profile/abc123
AUTH_TYPE=cli_db
CLI_DB_PATH=/home/user/.aws/sso/cache/abc123def456.json

# Logging
LOG_LEVEL=info
```

### Example 3: Automated Auth with SigV4

```bash
# .env
PORT=8080
AWS_REGION=us-east-1

# SigV4 Mode with SSO Credentials
AMAZON_Q_SIGV4=true
USE_SSO_CREDENTIALS=true

# SSO Configuration
AWS_SSO_ACCOUNT_ID=123456789012
AWS_SSO_ROLE_NAME=PowerUserAccess

# Automated Authentication
AUTH_TYPE=automated_oidc
# Credentials stored in OS keychain

# Logging
LOG_LEVEL=info
```

---

## How It Works

### Bearer Token Flow

```
1. Gateway loads bearer token from CLI DB or automated auth
2. Token used directly in Authorization header
3. Request sent to API with bearer token
```

### SigV4 with SSO Flow

```
1. Gateway loads bearer token from CLI DB or automated auth
2. Bearer token sent to AWS SSO GetRoleCredentials API
3. SSO returns temporary IAM credentials:
   - Access Key ID
   - Secret Access Key
   - Session Token
   - Expiration time
4. IAM credentials used to sign request with SigV4
5. Signed request sent to API
6. Credentials cached until expiration
```

---

## Credential Caching

### Bearer Tokens
- Cached in memory
- Refreshed automatically when expired
- 1-minute safety margin

### SSO-Derived IAM Credentials
- Cached in memory
- Typically valid for 1 hour
- Automatically refreshed when expired
- New bearer token → new IAM credentials

---

## Troubleshooting

### "SSO account ID and role name are required"

**Cause:** `USE_SSO_CREDENTIALS=true` but missing SSO configuration

**Solution:**
```bash
export AWS_SSO_ACCOUNT_ID=123456789012
export AWS_SSO_ROLE_NAME=PowerUserAccess
```

### "No bearer token available for SSO credential conversion"

**Cause:** Bearer token not loaded or expired

**Solution:**
1. Check `AUTH_TYPE` is set correctly
2. Verify `CLI_DB_PATH` points to valid token file
3. Re-authenticate: `aws sso login --profile YOUR_PROFILE`

### "Failed to get SSO role credentials"

**Cause:** Invalid SSO configuration or expired bearer token

**Solution:**
1. Verify account ID and role name are correct
2. Check bearer token is valid
3. Ensure SSO role has necessary permissions
4. Re-authenticate if needed

### "Access denied" errors

**Cause:** SSO role lacks required permissions

**Solution:**
1. Check role permissions in IAM Identity Center
2. Ensure role has Q Developer access
3. Verify profile ARN is correct

---

## Security Considerations

### Bearer Token Storage
- Tokens stored in CLI DB (encrypted by OS)
- Or in OS keychain (automated auth)
- Never logged or exposed

### IAM Credential Handling
- Credentials cached in memory only
- Never written to disk
- Automatically cleared on expiration
- Refreshed transparently

### Audit Trail
- SigV4 requests logged in CloudTrail
- Bearer token requests may not appear in CloudTrail
- Use SigV4 mode for compliance requirements

---

## Migration Guide

### From Bearer Token to SigV4 with SSO

1. **Get SSO configuration:**
   ```bash
   aws configure get sso_account_id --profile YOUR_PROFILE
   aws configure get sso_role_name --profile YOUR_PROFILE
   ```

2. **Update .env file:**
   ```bash
   # Add these lines
   AMAZON_Q_SIGV4=true
   USE_SSO_CREDENTIALS=true
   AWS_SSO_ACCOUNT_ID=YOUR_ACCOUNT_ID
   AWS_SSO_ROLE_NAME=YOUR_ROLE_NAME
   ```

3. **Restart gateway:**
   ```bash
   ./kiro-gateway
   ```

4. **Verify mode:**
   ```bash
   # Check logs for "SigV4 mode enabled with SSO credentials"
   ```

### From SigV4 with SSO to Bearer Token

1. **Update .env file:**
   ```bash
   # Change or remove these lines
   AMAZON_Q_SIGV4=false
   # Remove USE_SSO_CREDENTIALS
   # Remove AWS_SSO_ACCOUNT_ID
   # Remove AWS_SSO_ROLE_NAME
   ```

2. **Restart gateway:**
   ```bash
   ./kiro-gateway
   ```

---

## Testing

### Test Bearer Token Mode

```bash
# Start gateway
AMAZON_Q_SIGV4=false ./kiro-gateway

# Make request
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"claude-3-5-sonnet-20241022-v2","messages":[{"role":"user","content":"Hello"}]}'
```

### Test SigV4 with SSO Mode

```bash
# Start gateway
AMAZON_Q_SIGV4=true \
USE_SSO_CREDENTIALS=true \
AWS_SSO_ACCOUNT_ID=123456789012 \
AWS_SSO_ROLE_NAME=PowerUserAccess \
./kiro-gateway

# Make request (same as above)
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"claude-3-5-sonnet-20241022-v2","messages":[{"role":"user","content":"Hello"}]}'
```

---

## Summary

**Key Points:**

1. **Two modes:** Bearer Token (CodeWhisperer) vs SigV4 with SSO (Q Developer)
2. **Configuration:** Set `USE_SSO_CREDENTIALS=true` for SigV4 with SSO
3. **Required:** `AWS_SSO_ACCOUNT_ID` and `AWS_SSO_ROLE_NAME` for SSO mode
4. **Automatic:** Credential conversion and caching handled by gateway
5. **Flexible:** Can switch modes by changing environment variables

**Environment Variables Summary:**

| Variable | Bearer Mode | SigV4 with SSO Mode |
|----------|-------------|---------------------|
| `AMAZON_Q_SIGV4` | `false` | `true` |
| `USE_SSO_CREDENTIALS` | not set | `true` |
| `AWS_SSO_ACCOUNT_ID` | not needed | **required** |
| `AWS_SSO_ROLE_NAME` | not needed | **required** |
| `AUTH_TYPE` | required | required |
| `CLI_DB_PATH` or automated | required | required |

---

**Last Updated:** January 24, 2026  
**Version:** 1.0.0
