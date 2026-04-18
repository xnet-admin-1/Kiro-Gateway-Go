# TOTP MFA Setup for Automated Authentication

This guide explains how to set up TOTP (Time-based One-Time Password) MFA for fully automated headless authentication.

## Overview

For truly hands-free automated authentication, you need to configure TOTP-based MFA instead of hardware keys (YubiKey, biometric) or SMS. The gateway can automatically generate TOTP codes during the authentication flow.

## Prerequisites

- IAM Identity Center account with TOTP MFA configured
- TOTP secret key (obtained during MFA setup)

## Setting Up TOTP MFA

### 1. Configure TOTP in IAM Identity Center

1. Log in to AWS IAM Identity Center
2. Go to **My Profile** → **Multi-factor authentication**
3. Click **Register MFA device**
4. Select **Authenticator app**
5. **IMPORTANT**: When the QR code is displayed, click **Show secret key**
6. Copy the secret key (base32-encoded string like `JBSWY3DPEHPK3PXP`)
7. Complete the setup by entering two consecutive TOTP codes

### 2. Store the TOTP Secret

The TOTP secret can be stored in:

#### Option A: Environment Variable (Recommended for Development)

```powershell
# Windows PowerShell
$env:MFA_TOTP_SECRET = "JBSWY3DPEHPK3PXP"
```

```bash
# Linux/Mac
export MFA_TOTP_SECRET="JBSWY3DPEHPK3PXP"
```

#### Option B: .env File (Recommended for Production)

Add to your `.env` file:

```env
# Automated Authentication with TOTP MFA
AUTOMATE_AUTH=true
SSO_USERNAME=your-username
SSO_PASSWORD=your-password
MFA_TOTP_SECRET=JBSWY3DPEHPK3PXP
```

#### Option C: Windows Credential Manager (Most Secure)

```powershell
# Store TOTP secret in Credential Manager
cmdkey /generic:"kiro-gateway-totp" /user:"totp-secret" /pass:"JBSWY3DPEHPK3PXP"

# Retrieve in script
$cred = cmdkey /list:"kiro-gateway-totp"
$env:MFA_TOTP_SECRET = $cred.Password
```

### 3. Test TOTP Generation

You can verify your TOTP secret works correctly:

```powershell
# The gateway will log the generated code during authentication
# Look for: [BROWSER] Generated TOTP code: 123456
```

## Configuration

### Complete Automated Auth Configuration

```env
# Headless Mode
HEADLESS_MODE=true
SSO_START_URL=https://your-org.awsapps.com/start
SSO_REGION=us-east-1
AWS_SSO_ACCOUNT_ID=123456789012
AWS_SSO_ROLE_NAME=AdministratorAccess

# Browser Automation
AUTOMATE_AUTH=true
SSO_USERNAME=service-account
SSO_PASSWORD=your-secure-password

# TOTP MFA (NEW)
MFA_TOTP_SECRET=JBSWY3DPEHPK3PXP
```

## How It Works

1. **Browser automation starts** and enters username/password
2. **MFA page detected** - gateway checks for MFA input field
3. **TOTP code generated** using the secret and current time
4. **Code entered automatically** and submitted
5. **Authentication completes** without manual intervention

## Security Considerations

### TOTP Secret Storage

The TOTP secret is **highly sensitive** - it's equivalent to your MFA device:

- ✅ **DO**: Store in secure credential managers (Windows Credential Manager, AWS Secrets Manager)
- ✅ **DO**: Use environment variables for development/testing
- ✅ **DO**: Rotate secrets periodically
- ❌ **DON'T**: Commit secrets to version control
- ❌ **DON'T**: Share secrets between accounts
- ❌ **DON'T**: Use the same secret for multiple environments

### Service Accounts

For production automation:

1. **Create dedicated service accounts** - don't use personal admin accounts
2. **Use least-privilege roles** - only grant necessary permissions
3. **Enable CloudTrail logging** - monitor service account activity
4. **Rotate credentials regularly** - update passwords and TOTP secrets
5. **Use "trusted device" checkbox** - reduces MFA prompts

## Troubleshooting

### TOTP Code Rejected

**Symptoms**: "Invalid MFA code" error

**Solutions**:
- Verify the secret is correct (no spaces, correct base32 encoding)
- Check system time is synchronized (TOTP is time-based)
- Ensure secret hasn't been rotated in IAM Identity Center

### Time Synchronization

TOTP codes are time-sensitive (30-second windows):

```powershell
# Windows - Sync time
w32tm /resync

# Linux - Sync time
sudo ntpdate -s time.nist.gov
```

### Manual Fallback

If TOTP automation fails, the gateway falls back to manual entry:

```
[BROWSER] ⚠️ MFA REQUIRED - Browser will stay open for manual MFA entry
[BROWSER] Please enter your MFA code in the browser window...
[BROWSER] Waiting 60 seconds for manual MFA entry...
```

## Example: Complete Setup Script

```powershell
# setup-totp-auth.ps1

# Store credentials in Windows Credential Manager
cmdkey /generic:"kiro-gateway-sso" /user:"service-account" /pass:"SecurePassword123!"
cmdkey /generic:"kiro-gateway-totp" /user:"totp-secret" /pass:"JBSWY3DPEHPK3PXP"

# Create .env file
@"
HEADLESS_MODE=true
SSO_START_URL=https://myorg.awsapps.com/start
SSO_REGION=us-east-1
AWS_SSO_ACCOUNT_ID=123456789012
AWS_SSO_ROLE_NAME=PowerUserAccess

AUTOMATE_AUTH=true
SSO_USERNAME=service-account
SSO_PASSWORD=SecurePassword123!
MFA_TOTP_SECRET=JBSWY3DPEHPK3PXP
"@ | Out-File -FilePath .env -Encoding UTF8

Write-Host "✅ TOTP authentication configured successfully!"
Write-Host "Run: .\kiro-gateway.exe"
```

## Comparison: Manual vs TOTP MFA

| Feature | Manual MFA | TOTP MFA |
|---------|-----------|----------|
| **Automation** | Requires manual code entry | Fully automated |
| **MFA Type** | Any (YubiKey, SMS, TOTP) | TOTP only |
| **Setup Complexity** | Simple | Requires secret storage |
| **Security** | High (hardware key best) | High (if secret secured) |
| **Use Case** | Admin accounts | Service accounts |
| **Headless Support** | Limited (60s window) | Full support |

## Next Steps

1. ✅ Configure TOTP MFA in IAM Identity Center
2. ✅ Store TOTP secret securely
3. ✅ Test automated authentication
4. ✅ Monitor logs for successful TOTP generation
5. ✅ Set up credential rotation schedule

## Related Documentation

- [Headless Authentication Guide](HEADLESS_AUTHENTICATION.md)
- [Automated Auth Quick Reference](AUTOMATED_AUTH_QUICK_REFERENCE.md)
- [Security Best Practices](SECURITY.md)
