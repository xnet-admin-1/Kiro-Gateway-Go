# Automated Authentication Quick Reference

## Quick Start

### 1. Enable Automated Authentication

Create or update `.env` file:

```bash
# Enable headless mode
HEADLESS_MODE=true

# SSO Configuration
SSO_START_URL=https://xnetinc.awsapps.com/start
SSO_REGION=us-east-1
SSO_ACCOUNT_ID=096305372922
SSO_ROLE_NAME=AdministratorAccess

# Enable SigV4 and Q Developer
AMAZON_Q_SIGV4=true
Q_USE_SENDMESSAGE=true

# Enable Browser Automation (FULLY AUTOMATED)
AUTOMATE_AUTH=true
SSO_USERNAME=your-username
SSO_PASSWORD=your-password
```

### 2. Run Gateway

```bash
./kiro-gateway.exe
```

That's it! The gateway will:
1. Register OIDC client
2. Get device code
3. Launch headless browser
4. Automatically sign in
5. Automatically approve
6. Get AWS credentials
7. Start serving requests

## Configuration Options

### Manual Mode (Display Code)
```bash
HEADLESS_MODE=true
AUTOMATE_AUTH=false  # or omit
```
Gateway displays device code and URL for manual authorization.

### Automated Mode (Fully Hands-Free)
```bash
HEADLESS_MODE=true
AUTOMATE_AUTH=true
SSO_USERNAME=your-username
SSO_PASSWORD=your-password
```
Gateway handles everything automatically.

## Credential Storage

### Option 1: Environment Variables (Testing)
```bash
SSO_USERNAME=your-username
SSO_PASSWORD=your-password
```

### Option 2: OS Keychain (Production)
Store credentials in Windows Credential Manager or macOS Keychain.
Gateway retrieves them at runtime using the storage layer.

### Option 3: Secrets Manager (Enterprise)
Store in AWS Secrets Manager, retrieve at startup.

## Troubleshooting

### Browser Automation Fails
- Check credentials are correct
- Verify SSO start URL is accessible
- Check logs for specific error
- Gateway automatically falls back to manual mode

### Token Expired
- Gateway automatically refreshes tokens
- Background refresh runs every minute
- Tokens refresh 5 minutes before expiry

### No Credentials Available
- Check HEADLESS_MODE is enabled
- Verify SSO configuration is correct
- Check token storage is accessible

## Testing

### Test Manual Flow
```bash
HEADLESS_MODE=true AUTOMATE_AUTH=false ./kiro-gateway.exe
```

### Test Automated Flow
```bash
HEADLESS_MODE=true AUTOMATE_AUTH=true SSO_USERNAME=user SSO_PASSWORD=pass ./kiro-gateway.exe
```

### Test SigV4 with SSO
```bash
# After authentication, test API call
curl -X POST http://localhost:8090/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test-headless-key" \
  -d '{"model":"anthropic.claude-3-5-sonnet-20241022-v2:0","messages":[{"role":"user","content":"Hello"}]}'
```

## Architecture Flow

```
Start Gateway
    ↓
Load Config (AUTOMATE_AUTH=true)
    ↓
Initialize HeadlessAuthManager
    ↓
Register OIDC Client
    ↓
Start Device Authorization
    ↓
Launch Headless Browser (go-rod)
    ↓
Navigate to Verification URL
    ↓
Auto-Enter Username & Password
    ↓
Auto-Click Approve
    ↓
Poll for Token (gateway side)
    ↓
Receive Access Token
    ↓
Exchange for AWS Credentials (SSO API)
    ↓
Start Background Refresh
    ↓
Ready to Serve Requests (SigV4 signed)
```

## Security Best Practices

1. **Never commit credentials** to version control
2. **Use .env files** for local development only
3. **Use OS keychain** for production deployments
4. **Rotate credentials** regularly
5. **Monitor access logs** for unauthorized usage
6. **Use least privilege** IAM roles

## See Also

- [Headless Authentication Guide](HEADLESS_AUTHENTICATION.md)
- [Headless Quick Start](HEADLESS_docs/guides/docs/guides/docs/guides/docs/guides/QUICK_START.md)
- [Full Implementation Details](../.archive/implementation-summaries/.archive/implementation-summaries/FULLY_AUTOMATED_AUTH_IMPLEMENTATION.md)
