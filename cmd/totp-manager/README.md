# Kiro Gateway TOTP Manager

A desktop GUI application for managing TOTP codes and API keys for the Kiro Gateway.

## Features

- **TOTP Code Retrieval**: Get current TOTP codes from the deployed gateway container
- **Auto-Refresh**: Automatically refresh TOTP codes every 5 seconds
- **Copy to Clipboard**: One-click copy of TOTP codes
- **API Key Management**: Create, view, and delete secondary API keys
- **Secure Storage**: Configuration stored in user config directory with restricted permissions

## Prerequisites

- Kiro Gateway deployed and running (container or standalone)
- Admin API key from the gateway
- Go 1.23 or later (for building from source)

## Installation

### From Source

```powershell
# Navigate to the totp-manager directory
cd cmd/totp-manager

# Download dependencies
go mod download

# Build the application
go build -o totp-manager.exe .

# Run the application
.\totp-manager.exe
```

### Pre-built Binary

Download the latest release from the releases page and run the executable.

## Configuration

### Initial Setup

1. Launch the application
2. Go to the "Configuration" tab
3. Enter your gateway URL (default: `http://localhost:8080`)
4. Enter your admin API key
5. Click "Save Configuration"

### Getting the Admin API Key

The admin API key is generated when the gateway first starts. You can retrieve it from:

**From container logs:**
```powershell
docker logs kiro-gateway-go-kiro-gateway-1 | Select-String "ADMIN API KEY"
```

**From container filesystem:**
```powershell
docker exec kiro-gateway-go-kiro-gateway-1 cat /app/.kiro/api-keys/*.json
```

**From local filesystem (if running standalone):**
```powershell
Get-Content .kiro\api-keys\*.json
```

## Usage

### Retrieving TOTP Codes

1. Go to the "TOTP" tab
2. Click "Get TOTP Code"
3. The current code will be displayed with expiration time
4. Click "Copy to Clipboard" to copy the code
5. Enable "Auto-refresh" to automatically update the code every 5 seconds

### Managing API Keys

1. Go to the "API Keys" tab
2. Click "Create New API Key"
3. Fill in the form:
   - **Name**: Descriptive name for the key (e.g., "totp-access")
   - **User ID**: User identifier (e.g., "totp-user")
   - **Permissions**: Select read, write, and/or admin permissions
4. Click "Create"
5. Copy the generated key (it won't be shown again!)
6. The key is automatically saved to your configuration

### Deleting API Keys

1. Go to the "API Keys" tab
2. Click "Delete" next to the key you want to remove
3. Confirm the deletion

Note: This only removes the key from the local configuration. The key remains valid on the gateway until explicitly revoked.

## Configuration File

The application stores its configuration in:

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

The configuration file contains:
- Gateway URL
- Admin API key (encrypted in memory, stored as plain text in file)
- List of created API keys with metadata

**Security Note**: The configuration file is created with restricted permissions (0600) to prevent unauthorized access.

## API Endpoints Used

### GET /totp

Retrieves the current TOTP code.

**Request:**
```
GET /totp
Authorization: Bearer YOUR_API_KEY
```

**Response:**
```json
{
  "code": "123456",
  "expires_in": 25,
  "timestamp": "2026-01-31T12:00:00Z"
}
```

### POST /v1/api-keys

Creates a new API key.

**Request:**
```
POST /v1/api-keys
Authorization: Bearer ADMIN_API_KEY
Content-Type: application/json

{
  "name": "totp-access",
  "user_id": "totp-user",
  "permissions": ["read"]
}
```

**Response:**
```json
{
  "key": "generated-api-key",
  "name": "totp-access",
  "user_id": "totp-user",
  "permissions": ["read"],
  "created_at": "2026-01-31T12:00:00Z"
}
```

## Troubleshooting

### "Gateway URL not configured"

Go to the Configuration tab and enter your gateway URL.

### "Authentication failed"

Your API key is invalid or expired. Check the admin API key in the Configuration tab.

### "Failed to get TOTP: connection refused"

The gateway is not running or not accessible. Check:
- Gateway is running: `docker ps`
- Gateway is healthy: `curl http://localhost:8080/health`
- Firewall rules allow connections

### "TOTP secret not configured"

The gateway doesn't have the TOTP secret configured. Set the `MFA_TOTP_SECRET` environment variable in your gateway configuration.

## Security Considerations

1. **Protect the Admin Key**: The admin API key has full access to the gateway. Store it securely.
2. **Limit Permissions**: Create API keys with minimal required permissions (e.g., read-only for TOTP access).
3. **Rotate Keys**: Periodically rotate API keys and update the configuration.
4. **Secure the Config File**: The configuration file contains sensitive data. Keep it protected.
5. **Network Security**: Only expose the gateway on localhost or use HTTPS for remote access.

## Building for Different Platforms

### Windows

```powershell
go build -o totp-manager.exe .
```

### Linux

```bash
go build -o totp-manager .
```

### macOS

```bash
go build -o totp-manager .
```

### Cross-Compilation

```powershell
# Windows from Linux/Mac
$env:GOOS="windows"; $env:GOARCH="amd64"; go build -o totp-manager.exe .

# Linux from Windows/Mac
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o totp-manager .

# macOS from Windows/Linux
$env:GOOS="darwin"; $env:GOARCH="amd64"; go build -o totp-manager .
```

## Related Documentation

- [TOTP Code Retrieval Guide](../../docs/guides/TOTP_CODE_RETRIEVAL.md)
- [TOTP MFA Setup](../../docs/TOTP_MFA_SETUP.md)
- [API Key Management](../../docs/API_KEY_MANAGEMENT.md)
- [Docker Deployment](../../docs/DOCKER_DEPLOYMENT.md)

## License

Same as the main Kiro Gateway project.
