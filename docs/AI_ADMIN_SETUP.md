# AI Admin Account Setup for Q Developer Pro

## Overview
This guide walks through setting up the `ai-admin` IAM Identity Center user for automated Q Developer Pro access with TOTP MFA.

## Prerequisites
- AWS account with IAM Identity Center enabled
- Administrator access to IAM Identity Center
- `ai-admin` AWS CLI profile configured (already done ✓)

## Step 1: Create IAM Identity Center User

1. **Open IAM Identity Center Console**
   ```powershell
   # Open in browser
   Start-Process "https://console.aws.amazon.com/singlesignon/home?region=us-east-1"
   ```

2. **Navigate to Users**
   - Click "Users" in the left navigation
   - Click "Add user"

3. **Create User**
   - **Username**: `ai-admin`
   - **Email**: Use a dedicated email for automation (e.g., `ai-admin@xnetinc.com`)
   - **First name**: `AI`
   - **Last name**: `Admin`
   - **Display name**: `AI Admin`
   - Click "Next"

4. **Add to Groups** (Optional)
   - You can add to existing groups or skip for now
   - Click "Next"

5. **Review and Create**
   - Review the details
   - Click "Add user"

## Step 2: Subscribe ai-admin to Q Developer Pro

1. **Open Amazon Q Developer Console**
   ```powershell
   Start-Process "https://console.aws.amazon.com/amazonq/home?region=us-east-1"
   ```

2. **Navigate to Subscriptions**
   - Click "Subscriptions" in the left navigation
   - Click "Subscribe users"

3. **Subscribe ai-admin**
   - Search for "ai-admin"
   - Select the user
   - Click "Assign"

4. **Verify Subscription**
   - The user should appear in the subscriptions list
   - Status will be "Pending" until first activation

## Step 3: Set Up TOTP MFA

### Option A: Manual Setup (Recommended for First Time)

1. **Access the Activation Email**
   - Check the email inbox for `ai-admin@xnetinc.com`
   - Look for "Activate Your Amazon Q Developer Pro Subscription"
   - Click the activation link

2. **Complete Initial Setup**
   - Set a password for the account
   - You'll be prompted to set up MFA

3. **Configure TOTP MFA**
   - Choose "Authenticator app" as MFA method
   - Scan the QR code with an authenticator app (Google Authenticator, Authy, etc.)
   - **IMPORTANT**: Save the secret key (the text version under the QR code)
   - Enter the 6-digit code from your authenticator app
   - Complete the setup

4. **Save the TOTP Secret**
   - The secret key is what you'll use in the `MFA_TOTP_SECRET` environment variable
   - Store it securely (e.g., in AWS Secrets Manager or a password manager)

### Option B: Add TOTP to Existing Account

If the account already exists and uses a different MFA method:

1. **Sign in to AWS Access Portal**
   ```
   https://xnetinc.awsapps.com/start
   ```

2. **Go to MFA Settings**
   - Click your username in the top right
   - Click "MFA devices"
   - Click "Register MFA device"

3. **Add Authenticator App**
   - Choose "Authenticator app"
   - Scan QR code or enter secret key manually
   - **IMPORTANT**: Save the secret key
   - Enter verification code
   - Click "Add MFA"

## Step 4: Configure Environment Variables

Update your `.env` file with the TOTP secret:

```bash
# Headless OIDC Mode Configuration
PORT=8090
PROXY_API_KEY=test-headless-key

# Enable headless mode
HEADLESS_MODE=true

# SSO Configuration
SSO_START_URL=https://xnetinc.awsapps.com/start
SSO_REGION=us-east-1
SSO_ACCOUNT_ID=096305372922
SSO_ROLE_NAME=AdministratorAccess

# IMPORTANT: Add these for ai-admin automation
AUTOMATION_USERNAME=ai-admin
AUTOMATION_PASSWORD=<password-you-set>
MFA_TOTP_SECRET=<secret-key-from-authenticator-setup>

# Enable SigV4 authentication
AMAZON_Q_SIGV4=true
AWS_REGION=us-east-1

# Use Q Developer endpoint
Q_USE_SENDMESSAGE=true

# Profile ARN for Identity Center
PROFILE_ARN=arn:aws:codewhisperer:us-east-1:096305372922:profile/VREYVEXNNH3H

# Enable debug logging
LOG_LEVEL=debug
DEBUG=true
```

## Step 5: Test the Setup

1. **Build the Gateway**
   ```powershell
   go build -o kiro-gateway.exe ./cmd/kiro-gateway
   ```

2. **Run with Headless Mode**
   ```powershell
   # Load environment variables
   Get-Content .env | ForEach-Object {
       if ($_ -match '^([^#][^=]+)=(.*)$') {
           [Environment]::SetEnvironmentVariable($matches[1], $matches[2], 'Process')
       }
   }
   
   # Run the gateway
   .\kiro-gateway.exe
   ```

3. **Watch for Success**
   - The gateway should automatically:
     1. Navigate to the SSO login page
     2. Enter username and password
     3. Check "This is a trusted device"
     4. Generate and enter TOTP code automatically
     5. Approve the device
     6. Allow access
     7. Obtain AWS credentials
     8. Start the gateway

4. **Test Q Developer**
   ```powershell
   # Test the gateway
   .\scripts\test_automated_auth_simple.ps1
   ```

## Step 6: Switch to Headless Mode (Production)

Once everything works with visible browser:

1. **Update browser_automation.go**
   ```go
   // Change this line:
   browser := rod.New().ControlURL(launcher.MustLaunch()).MustConnect()
   
   // To this:
   launcher := launcher.New().Headless(true)  // Enable headless
   browser := rod.New().ControlURL(launcher.MustLaunch()).MustConnect()
   ```

2. **Rebuild and Test**
   ```powershell
   go build -o kiro-gateway.exe ./cmd/kiro-gateway
   .\kiro-gateway.exe
   ```

## Security Best Practices

1. **Store Secrets Securely**
   - Use AWS Secrets Manager for production
   - Never commit secrets to git
   - Rotate credentials regularly

2. **Limit Account Permissions**
   - Only grant necessary permissions to ai-admin
   - Use least-privilege principle
   - Monitor account activity

3. **Enable Audit Logging**
   - CloudTrail for AWS API calls
   - IAM Identity Center sign-in logs
   - Gateway application logs

## Troubleshooting

### TOTP Code Not Working
- Ensure system time is synchronized (TOTP is time-based)
- Verify the secret key is correct
- Check if the authenticator app is generating 6-digit codes

### Browser Automation Fails
- Check screenshots in `logs/screenshots/`
- Review gateway logs for error messages
- Ensure all selectors are correct

### Token Refresh Issues
- Check token expiry times
- Verify SSO configuration is correct
- Ensure network connectivity to AWS

## Cost
- **Q Developer Pro**: $19/month per user
- **ai-admin subscription**: $19/month
- **Total**: $19/month for automated access

## Next Steps
1. Complete the setup above
2. Test thoroughly in development
3. Deploy to production with headless mode
4. Set up monitoring and alerting
5. Document the process for your team
