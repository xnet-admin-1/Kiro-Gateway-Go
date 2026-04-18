# Identity Center Requirement for Q Developer Pro

**Important: Q Developer Pro requires AWS IAM Identity Center**

---

## Overview

Amazon Q Developer Pro is an enterprise service that requires AWS IAM Identity Center (formerly AWS SSO) for authentication. This is a **mandatory requirement** - you cannot use Q Developer Pro without Identity Center.

### Why Identity Center is Required

1. **Enterprise Authentication:** Q Developer Pro is designed for organizations and requires centralized identity management
2. **Security:** Identity Center provides MFA, SSO, and centralized access control
3. **Compliance:** Meets enterprise security and compliance requirements
4. **Audit Trail:** All access is logged through CloudTrail for compliance

---

## What This Means for Headless Environments

### Initial Setup (One-Time Human Interaction Required)

**You cannot completely avoid human interaction** for the initial setup:

1. ✅ **One-time browser authentication** is required to obtain tokens
2. ✅ After initial setup, tokens can be refreshed automatically
3. ✅ Refresh tokens last ~90 days before re-authentication needed
4. ✅ Headless operation works for 90 days between authentications

### Workflow

```
┌─────────────────────────────────────────────────────────────┐
│                  Initial Setup (Human)                       │
│                                                              │
│  1. Enable Identity Center                                  │
│  2. Get Profile ARN                                         │
│  3. Authenticate via browser (device code flow)             │
│  4. Extract tokens                                          │
│  5. Store tokens securely                                   │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│              Automated Operation (Headless)                  │
│                                                              │
│  • Gateway uses stored tokens                               │
│  • Tokens auto-refresh (access token: 1 hour)              │
│  • AWS CLI refreshes in background                          │
│  • No human interaction for 90 days                         │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│          Re-authentication (Every 90 Days)                   │
│                                                              │
│  • Refresh token expires                                    │
│  • Human must re-authenticate via browser                   │
│  • New tokens extracted                                     │
│  • Automated operation resumes                              │
└─────────────────────────────────────────────────────────────┘
```

---

## Setup Process

### Step 1: Enable Identity Center

**One-time setup per AWS account:**

```bash
# Via AWS CLI
aws sso-admin create-instance \
  --instance-name "MyOrganization" \
  --region us-east-1
```

**Or via AWS Console:**
1. Go to AWS IAM Identity Center
2. Click "Enable"
3. Choose identity source (AWS Directory, Active Directory, or External IdP)

### Step 2: Get Profile ARN

**Required for Q Developer Pro access:**

```bash
# Option 1: Using Q CLI
qchat profile

# Option 2: Using AWS CLI
aws codewhisperer list-profiles --region us-east-1

# Option 3: AWS Console
# Go to Amazon Q Developer → Settings → Profiles
```

### Step 3: Initial Authentication

**Requires human interaction (one-time):**

```bash
# Run setup script
./scripts/setup-identity-center.sh

# Or manually:
aws sso login --profile YOUR_PROFILE
```

**What happens:**
1. Device code is displayed
2. Browser opens to verification URL
3. You enter the device code
4. You authenticate with Identity Center credentials
5. Tokens are stored in `~/.aws/sso/cache/`

### Step 4: Extract Tokens

```bash
# Find SSO cache file
CACHE_FILE=$(ls -t ~/.aws/sso/cache/*.json | head -1)

# Extract tokens
ACCESS_TOKEN=$(jq -r '.accessToken' "$CACHE_FILE")
REFRESH_TOKEN=$(jq -r '.refreshToken' "$CACHE_FILE")
EXPIRES_AT=$(jq -r '.expiresAt' "$CACHE_FILE")
```

### Step 5: Configure for Headless

```bash
# Create .env file
cat > .env <<EOF
AUTH_TYPE=cli_db
CLI_DB_PATH=$CACHE_FILE
PROFILE_ARN=arn:aws:codewhisperer:us-east-1:123456789012:profile/YOUR_PROFILE_ID
AWS_REGION=us-east-1
EOF
```

---

## Token Lifecycle

### Access Token
- **Lifetime:** ~1 hour
- **Refresh:** Automatic (via AWS CLI or gateway)
- **Storage:** `~/.aws/sso/cache/*.json`
- **Usage:** Every API request

### Refresh Token
- **Lifetime:** ~90 days
- **Refresh:** Requires re-authentication
- **Storage:** `~/.aws/sso/cache/*.json`
- **Usage:** To obtain new access tokens

### Timeline

```
Day 0:  Initial authentication (human)
        ↓
Day 1-89: Automated operation
        • Access tokens refresh automatically
        • No human interaction needed
        ↓
Day 90: Refresh token expires
        • Human must re-authenticate
        • New tokens obtained
        ↓
Day 91-179: Automated operation resumes
```

---

## Handling Token Expiration

### Monitoring

**Set up alerts for token expiration:**

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
    # Send alert (email, Slack, PagerDuty, etc.)
fi
```

**Set up cron job:**
```bash
# Check daily at 9 AM
0 9 * * * /path/to/check-token-expiration.sh
```

### Re-authentication Process

**When tokens expire:**

1. **Automated alert** notifies team
2. **Designated person** re-authenticates:
   ```bash
   aws sso login --profile YOUR_PROFILE
   ```
3. **Tokens are updated** in SSO cache
4. **Gateway picks up** new tokens automatically
5. **Operation resumes** without restart

---

## Production Recommendations

### 1. Designated Authentication Person

- Assign specific team member(s) for re-authentication
- Document the process
- Set up calendar reminders (every 80 days)
- Have backup person in case primary is unavailable

### 2. Token Storage

**Development:**
```bash
# Store in .env file (add to .gitignore)
AUTH_TYPE=cli_db
CLI_DB_PATH=/path/to/.aws/sso/cache/token.json
```

**Production:**
```bash
# Store in AWS Secrets Manager
aws secretsmanager create-secret \
  --name kiro-gateway/identity-center-tokens \
  --secret-string "{
    \"refresh_token\": \"...\",
    \"profile_arn\": \"arn:aws:codewhisperer:us-east-1:123456789012:profile/abc123\"
  }"
```

### 3. Monitoring and Alerts

**Set up CloudWatch alarms:**
- Token expiration (7 days before)
- Authentication failures
- Unusual access patterns

**Alert channels:**
- Email to team
- Slack/Teams notification
- PagerDuty for critical services

### 4. Documentation

**Maintain runbook with:**
- Identity Center setup instructions
- Profile ARN location
- Re-authentication procedure
- Escalation contacts
- Troubleshooting steps

---

## Alternatives (Not Recommended)

### Can I avoid Identity Center?

**No.** Q Developer Pro requires Identity Center. There are no alternatives.

### Can I fully automate authentication?

**No.** The device code flow requires human interaction by design for security.

### Can I use IAM users instead?

**No.** Q Developer Pro only works with Identity Center authentication.

### Can I extend token lifetime?

**No.** Token lifetimes are set by AWS and cannot be changed:
- Access token: ~1 hour
- Refresh token: ~90 days

---

## Comparison with Other Services

| Service | Auth Method | Headless Support |
|---------|-------------|------------------|
| **Q Developer Pro** | Identity Center | ⚠️ Partial (90-day cycle) |
| **Q Developer Free** | AWS Builder ID | ⚠️ Partial (90-day cycle) |
| **Bedrock** | IAM credentials | ✅ Full (no expiration) |
| **CodeWhisperer** | IAM credentials | ✅ Full (no expiration) |

**Key Difference:** Q Developer Pro's Identity Center requirement means you cannot achieve 100% headless operation indefinitely. Re-authentication is required every 90 days.

---

## Best Practices

### For CI/CD

1. **Use dedicated CI/CD profile** in Identity Center
2. **Store tokens in CI/CD secrets** (GitHub Secrets, GitLab Variables)
3. **Monitor token expiration** with automated checks
4. **Document re-authentication** in team wiki
5. **Test token refresh** regularly

### For Production

1. **Use separate profiles** per environment (dev, staging, prod)
2. **Store tokens in Secrets Manager** with encryption
3. **Enable CloudTrail logging** for audit trail
4. **Implement monitoring** for token expiration
5. **Have backup authentication** method ready

### For Development

1. **Use personal Identity Center profile**
2. **Store tokens locally** in SSO cache
3. **Re-authenticate as needed** (every 90 days)
4. **Test with production-like setup** before deploying

---

## Troubleshooting

### "Identity Center not enabled"

**Solution:**
```bash
# Enable via AWS Console
# Go to: https://console.aws.amazon.com/singlesignon

# Or via CLI
aws sso-admin create-instance \
  --instance-name "MyOrganization" \
  --region us-east-1
```

### "Profile ARN not found"

**Solution:**
1. Ensure Q Developer Pro subscription is active
2. Check in AWS Console: Amazon Q Developer → Settings → Profiles
3. Or run: `qchat profile`

### "Token expired"

**Solution:**
```bash
# Re-authenticate
aws sso login --profile YOUR_PROFILE

# Gateway will pick up new tokens automatically
```

### "Device code expired"

**Solution:**
- Device codes expire after 15 minutes
- Start authentication process again
- Complete within 15 minutes

---

## Summary

**Key Points:**

1. ✅ Identity Center is **mandatory** for Q Developer Pro
2. ⚠️ Initial authentication **requires human interaction**
3. ✅ After setup, **90 days of automated operation**
4. ⚠️ Re-authentication **required every 90 days**
5. ✅ Tokens **auto-refresh** between re-authentications

**Bottom Line:**

You can achieve **mostly headless** operation with Q Developer Pro, but you cannot completely eliminate human interaction. Plan for re-authentication every 90 days.

---

## Additional Resources

- [Identity Center Setup Guide](docs/IDENTITY_CENTER_SETUP.md)
- [Headless Authentication Guide](docs/HEADLESS_AUTHENTICATION.md)
- [Token Refresh Implementation](.archive/implementation-summaries/TOKEN_REFRESH_IMPLEMENTATION.md)
- [AWS Identity Center Documentation](https://docs.aws.amazon.com/singlesignon/)

---

**Last Updated:** January 24, 2026  
**Status:** Production Guidance  
**Version:** 1.0.0
