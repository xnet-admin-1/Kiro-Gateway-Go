# TOTP Code Retrieval Guide

This guide explains how to retrieve TOTP (Time-based One-Time Password) codes from the Kiro Gateway for authenticating other clients and devices.

## Overview

Kiro Gateway includes built-in TOTP generation for AWS SSO MFA authentication. The same TOTP secret used by the gateway for automated authentication can be accessed via HTTP endpoint to authenticate other clients and devices.

## Quick Start

### Using Helper Scripts

**PowerShell (Windows):**
```powershell
# Set API key in environment
$env:PROXY_API_KEY = "your-api-key-here"

# Run script
.\scripts\get-totp.ps1

# Or provide API key directly
.\scripts\get-totp.ps1 -ApiKey "your-api-key-here"
```

**Bash (Linux/WSL/Mac):**
```bash
# Set API key in environment
export PROXY_API_KEY="your-api-key-here"

# Run script
./scripts/get-totp.sh

# Or provide API key as parameter
./scripts/get-totp.sh kiro-YOUR_API_KEY_HERE http://localhost:8080 "your-api-key-here"
```

### Using curl

```bash
curl -H "Authorization: Bearer YOUR_API_KEY" http://localhost:8080/totp
```

## HTTP Endpoint

### GET /totp

Returns the current TOTP code with expiration information. **Requires authentication.**

**Request:**
```bash
curl -H "Authorization: Bearer YOUR_API_KEY" http://localhost:8080/totp
```

**Response:**
```json
{
  "code": "123456",
  "expires_in": 25,
  "timestamp": "2026-01-26T19:50:00Z"
}
```

**Response Fields:**
- `code` (string): The 6-digit TOTP code
- `expires_in` (integer): Seconds until the code expires (typically 0-30)
- `timestamp` (string): Current server timestamp in RFC3339 format

**Status Codes:**
- `200 OK`: Successfully generated TOTP code
- `401 Unauthorized`: Missing or invalid API key
- `429 Too Many Requests`: Rate limit exceeded
- `503 Service Unavailable`: TOTP secret not configured

## Configuration

The TOTP secret must be configured for the endpoint to work.

### Environment Variables

```bash
# Primary variable
MFA_TOTP_SECRET=YOUR_BASE32_SECRET

# Alternative variable (for compatibility)
SSO_MFA_TOTP_SECRET=YOUR_BASE32_SECRET
```

### Docker Secrets

```bash
# Create secret
echo "YOUR_BASE32_SECRET" | docker secret create mfa_totp_secret -

# Gateway reads from /run/secrets/mfa_totp_secret
```

### docker-compose.yml

```yaml
services:
  kiro-gateway:
    environment:
      - MFA_TOTP_SECRET=${MFA_TOTP_SECRET}
    # OR use secrets
    secrets:
      - mfa_totp_secret

secrets:
  mfa_totp_secret:
    external: true
```

## Use Cases

### Authenticating Other Clients

When you need to authenticate other AWS SSO clients or devices:

1. Run the helper script to get the current TOTP code
2. Enter the code in the MFA prompt of your other client
3. The code is valid for 30 seconds

### Manual Authentication

If automated authentication fails:

1. Check the gateway logs for the authentication URL
2. Open the URL in a browser
3. Use the helper script to get a fresh TOTP code
4. Enter the code in the MFA prompt

### Testing and Debugging

The endpoint is useful for:
- Verifying TOTP secret is correctly configured
- Testing MFA flows
- Debugging authentication issues
- Synchronizing multiple clients

## Helper Scripts

### PowerShell Script

**Location:** `scripts/get-totp.ps1`

**Usage:**
```powershell
# Default usage
.\scripts\get-totp.ps1

# Custom container name
.\scripts\get-totp.ps1 -ContainerName my-gateway

# Custom gateway URL
.\scripts\get-totp.ps1 -GatewayURL http://192.168.1.100:8080
```

**Features:**
- Tries gateway endpoint first
- Falls back to container exec if needed
- Color-coded output
- Error handling with helpful messages

### Bash Script

**Location:** `scripts/get-totp.sh`

**Usage:**
```bash
# Default usage
./scripts/get-totp.sh

# Custom container name
./scripts/get-totp.sh my-gateway

# Custom gateway URL
./scripts/get-totp.sh kiro-gateway http://192.168.1.100:8080
```

**Features:**
- Tries gateway endpoint first
- Falls back to container exec if needed
- Color-coded output (requires jq)
- Error handling with helpful messages

## Examples

### Get Code and Copy to Clipboard

**Windows PowerShell:**
```powershell
(Invoke-RestMethod http://localhost:8080/totp).code | Set-Clipboard
```

**Linux (with xclip):**
```bash
curl -s http://localhost:8080/totp | jq -r '.code' | xclip -selection clipboard
```

**Mac:**
```bash
curl -s http://localhost:8080/totp | jq -r '.code' | pbcopy
```

### Watch for Code Changes

**Linux/Mac:**
```bash
watch -n 1 'curl -s http://localhost:8080/totp | jq -r .code'
```

**Windows PowerShell:**
```powershell
while ($true) {
    Clear-Host
    $response = Invoke-RestMethod http://localhost:8080/totp
    Write-Host "Code: $($response.code) (expires in $($response.expires_in)s)"
    Start-Sleep -Seconds 1
}
```

### Use in Scripts

```bash
# Get code in variable
TOTP_CODE=$(curl -s http://localhost:8080/totp | jq -r '.code')

# Use in automation
if [ -n "$TOTP_CODE" ]; then
    echo "Authenticating with code: $TOTP_CODE"
    # Use code in your authentication flow
fi
```

### From Docker Container

```bash
# Direct curl from inside container
docker exec kiro-YOUR_API_KEY_HERE curl -s http://localhost:8080/totp

# Extract just the code
docker exec kiro-YOUR_API_KEY_HERE curl -s http://localhost:8080/totp | jq -r '.code'
```

## Security Considerations

### Authentication Required

The `/totp` endpoint **requires API key authentication** for security:
- Prevents unauthorized access to TOTP codes
- Uses the same authentication as other gateway endpoints
- Supports both PROXY_API_KEY and managed API keys
- Rate limiting applied to prevent abuse

### Getting an API Key

**From container logs (initial admin key):**
```bash
docker logs kiro-YOUR_API_KEY_HERE | grep "ADMIN API KEY"
```

**From container filesystem:**
```bash
docker exec kiro-YOUR_API_KEY_HERE cat /app/.kiro/api-keys/*.json
```

**Create new API key (requires admin key):**
```bash
curl -X POST http://localhost:8080/v1/api-keys \
  -H "Authorization: Bearer ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "totp-access",
    "user_id": "totp-user",
    "permissions": ["read"]
  }'
```

### Rate Limiting

The endpoint is protected by rate limiting to prevent:
- Brute force attacks
- Excessive requests
- Resource exhaustion

Default limits apply (same as health check endpoints).

### Network Access

In production deployments:
- The endpoint is only accessible from localhost by default
- Use firewall rules to restrict access if needed
- Consider using Docker networks for isolation
- Don't expose the endpoint publicly

### Best Practices

1. **Secure the TOTP secret**: Store in Docker secrets or encrypted environment variables
2. **Limit network access**: Use firewall rules or Docker networks
3. **Monitor usage**: Check logs for unusual access patterns
4. **Rotate secrets**: Periodically update TOTP secrets
5. **Use HTTPS**: If accessing remotely, use HTTPS with valid certificates

## Troubleshooting

### Error: "TOTP secret not configured"

**Cause:** The gateway cannot find the TOTP secret.

**Solution:**
1. Check `MFA_TOTP_SECRET` environment variable is set
2. Verify Docker secret `/run/secrets/mfa_totp_secret` exists
3. Ensure secret value is a valid base32-encoded string
4. Restart the gateway after configuration changes

### Error: "Failed to generate TOTP code"

**Cause:** The TOTP secret is invalid.

**Solution:**
1. Verify secret is base32-encoded (A-Z, 2-7, no spaces)
2. Check secret matches your AWS SSO MFA configuration
3. Test secret with a TOTP app (Google Authenticator, Authy, etc.)

### Error: Connection Refused

**Cause:** The gateway is not running or not accessible.

**Solution:**
1. Check container status: `docker ps`
2. Check gateway health: `curl http://localhost:8080/health`
3. Verify port mapping in docker-compose.yml
4. Check firewall rules

### Error: Rate Limited (429)

**Cause:** Too many requests in a short time.

**Solution:**
1. Wait a few seconds before retrying
2. Don't poll the endpoint continuously
3. Cache codes for their validity period (30 seconds)
4. Use the helper scripts which handle rate limiting

### Codes Don't Work

**Cause:** Time synchronization issues.

**Solution:**
1. Check system time is accurate: `date`
2. Sync time with NTP: `ntpdate -s time.nist.gov` (Linux)
3. Verify container time matches host time
4. Check timezone settings

## Integration Examples

### Python

```python
import requests
import json

def get_totp_code():
    response = requests.get('http://localhost:8080/totp')
    if response.status_code == 200:
        data = response.json()
        return data['code']
    return None

code = get_totp_code()
print(f"TOTP Code: {code}")
```

### Node.js

```javascript
const fetch = require('node-fetch');

async function getTOTPCode() {
    const response = await fetch('http://localhost:8080/totp');
    const data = await response.json();
    return data.code;
}

getTOTPCode().then(code => {
    console.log(`TOTP Code: ${code}`);
});
```

### Go

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
)

type TOTPResponse struct {
    Code      string `json:"code"`
    ExpiresIn int    `json:"expires_in"`
    Timestamp string `json:"timestamp"`
}

func getTOTPCode() (string, error) {
    resp, err := http.Get("http://localhost:8080/totp")
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    var data TOTPResponse
    if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
        return "", err
    }
    
    return data.Code, nil
}

func main() {
    code, err := getTOTPCode()
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    fmt.Printf("TOTP Code: %s\n", code)
}
```

## Related Documentation

- [Headless Authentication](HEADLESS_AUTHENTICATION.md) - Automated SSO authentication
- [Docker Deployment](DOCKER_DEPLOYMENT.md) - Container deployment guide
- [Security](../SECURITY.md) - Security best practices
- [API Reference](../api/README.md) - Complete API documentation
