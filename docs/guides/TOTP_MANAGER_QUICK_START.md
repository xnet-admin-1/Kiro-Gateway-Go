# TOTP Manager Quick Start Guide

This guide will help you get started with the Kiro Gateway TOTP Manager desktop application.

## What is TOTP Manager?

TOTP Manager is a desktop GUI application that allows you to:
- Retrieve TOTP codes from your deployed Kiro Gateway
- Manage API keys for gateway access
- Auto-refresh TOTP codes for continuous access
- Copy codes to clipboard with one click

## Prerequisites

- Kiro Gateway deployed and running (container or standalone)
- Admin API key from the gateway
- Windows, Linux, or macOS

## Installation

### Option 1: Build from Source

```powershell
# Windows
.\scripts\build-totp-manager.ps1

# Linux/Mac
./scripts/build-totp-manager.sh
```

The executable will be created in the `dist/` directory.

### Option 2: Download Pre-built Binary

Download the latest release for your platform from the releases page.

## Initial Setup

### Step 1: Get Your Admin API Key

The admin API key is generated when the gateway first starts.

**From container logs:**
```powershell
docker logs kiro-YOUR_API_KEY_HERE | Select-String "ADMIN API KEY"
```

**From container filesystem:**
```powershell
docker exec kiro-YOUR_API_KEY_HERE cat /app/.kiro/api-keys/*.json
```

**From local filesystem (standalone):**
```powershell
Get-Content .kiro\api-keys\*.json
```

Copy the API key value.

### Step 2: Launch TOTP Manager

```powershell
# Windows
.\dist\totp-manager.exe

# Linux/Mac
./dist/totp-manager
```

### Step 3: Configure the Application

1. Click on the "Configuration" tab
2. Enter your gateway URL:
   - Local: `http://localhost:8080`
   - Remote: `http://your-server:8080`
3. Paste your admin API key
4. Click "Save Configuration"

## Using TOTP Manager

### Getting TOTP Codes

1. Go to the "TOTP" tab
2. Click "Get TOTP Code"
3. The current 6-digit code will be displayed
4. Click "Copy to Clipboard" to copy the code
5. Use the code in your AWS SSO MFA prompt

### Auto-Refresh

Enable "Auto-refresh" to automatically update the TOTP code every 5 seconds. This is useful when you need continuous access to fresh codes.

### Creating API Keys

1. Go to the "API Keys" tab
2. Click "Create New API Key"
3. Fill in the form:
   - **Name**: Descriptive name (e.g., "laptop-access")
   - **User ID**: User identifier (e.g., "john-laptop")
   - **Permissions**: Select appropriate permissions
     - **Read**: View TOTP codes and gateway status
     - **Write**: Modify gateway configuration
     - **Admin**: Full access including key management
4. Click "Create"
5. **IMPORTANT**: Copy the generated key immediately - it won't be shown again!
6. The key is automatically saved to your configuration

### Managing API Keys

**View Keys:**
- All created keys are listed in the "API Keys" tab
- Each entry shows the key name and user ID

**Copy Key:**
- Click "Copy" next to any key to copy it to clipboard

**Delete Key:**
- Click "Delete" next to any key
- Confirm the deletion
- Note: This only removes the key from local config; it remains valid on the gateway

## Common Use Cases

### Use Case 1: Authenticating Other Devices

When you need to authenticate AWS SSO on another device:

1. Open TOTP Manager on your main computer
2. Click "Get TOTP Code"
3. Copy the code
4. Enter it in the MFA prompt on your other device

### Use Case 2: Continuous Authentication

When working with multiple AWS sessions:

1. Enable "Auto-refresh" in TOTP Manager
2. Keep the window open
3. Fresh codes are available every 30 seconds
4. Copy codes as needed for new sessions

### Use Case 3: Team Access

When sharing gateway access with team members:

1. Create a new API key for each team member
2. Set appropriate permissions (usually "read" for TOTP access)
3. Share the key securely (encrypted message, password manager)
4. Team members configure their TOTP Manager with their key

## Configuration File Location

The application stores configuration in:

**Windows:**
```
%APPDATA%\kiro-totp-manager\totp-manager-config.json
```

**Linux:**
```
~/.config/kiro-totp-manager/totp-manager-config.json
```

**macOS:**
```
~/Library/Application Support/kiro-totp-manager/totp-manager-config.json
```

## Troubleshooting

### "Gateway URL not configured"

**Solution**: Go to Configuration tab and enter your gateway URL.

### "Authentication failed"

**Possible causes:**
- Invalid API key
- Expired API key
- Gateway not running

**Solution**: 
1. Verify gateway is running: `curl http://localhost:8080/health`
2. Get a fresh admin API key from container logs
3. Update the key in Configuration tab

### "Failed to get TOTP: connection refused"

**Possible causes:**
- Gateway not running
- Wrong URL
- Firewall blocking connection

**Solution**:
1. Check gateway status: `docker ps`
2. Verify URL is correct
3. Test connection: `curl http://localhost:8080/health`
4. Check firewall rules

### "TOTP secret not configured"

**Cause**: Gateway doesn't have the TOTP secret configured.

**Solution**: Set `MFA_TOTP_SECRET` environment variable in your gateway configuration:

```yaml
# docker-compose.yml
services:
  kiro-gateway:
    environment:
      - MFA_TOTP_SECRET=YOUR_BASE32_SECRET
```

### Codes Don't Work in AWS SSO

**Possible causes:**
- Time synchronization issue
- Wrong TOTP secret
- Code expired

**Solution**:
1. Check system time is accurate
2. Verify TOTP secret in gateway configuration
3. Get a fresh code (codes expire after 30 seconds)

## Security Best Practices

1. **Protect the Admin Key**: Store it securely, don't share it
2. **Use Least Privilege**: Create API keys with minimal required permissions
3. **Rotate Keys**: Periodically create new keys and delete old ones
4. **Secure the Config File**: The config file contains sensitive data
5. **Use HTTPS**: If accessing remote gateway, use HTTPS
6. **Monitor Access**: Check gateway logs for unusual activity

## Advanced Usage

### Remote Gateway Access

To access a remote gateway:

1. Ensure gateway is accessible over network
2. Use HTTPS if possible: `https://gateway.example.com`
3. Configure firewall to allow your IP
4. Use VPN for additional security

### Multiple Gateways

To manage multiple gateways:

1. Create separate API keys for each gateway
2. Switch between configurations as needed
3. Consider using different TOTP Manager instances

### Scripting

The TOTP Manager can be used alongside scripts:

```powershell
# Get TOTP code via API
$apiKey = "your-api-key"
$response = Invoke-RestMethod -Uri "http://localhost:8080/totp" `
    -Headers @{"Authorization" = "Bearer $apiKey"}
$code = $response.code

# Use in automation
Write-Host "Current TOTP: $code"
```

## Next Steps

- [TOTP Code Retrieval Guide](TOTP_CODE_RETRIEVAL.md) - Detailed API documentation
- [TOTP MFA Setup](../TOTP_MFA_SETUP.md) - Configure TOTP in AWS
- [API Key Management](../API_KEY_MANAGEMENT.md) - Advanced key management
- [Docker Deployment](../DOCKER_DEPLOYMENT.md) - Deploy gateway in production

## Support

For issues or questions:
1. Check the troubleshooting section above
2. Review the gateway logs: `docker logs kiro-YOUR_API_KEY_HERE`
3. Check the TOTP Manager README: `cmd/totp-manager/README.md`
