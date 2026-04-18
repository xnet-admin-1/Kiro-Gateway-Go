# Kiro Gateway - Quick Start

## Status: ✅ READY

The gateway is built, configured, and tested. All systems operational.

## Start Gateway

```powershell
.\dist\kiro-YOUR_API_KEY_HERE.exe
```

**Port**: 8090  
**Mode**: Headless/Auto (no manual steps)  
**Authentication**: Automatic SSO with MFA

## Test Endpoints

### Health Check
```powershell
Invoke-RestMethod -Uri "http://localhost:8090/health"
```

### Chat Completion
```powershell
$headers = @{
    "Authorization" = "Bearer kiro-YOUR_API_KEY_HERE"
    "Content-Type" = "application/json"
}

$body = @{
    model = "claude-sonnet-4-5"
    messages = @(@{
        role = "user"
        content = "What is Amazon S3?"
    })
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8090/v1/chat/completions" `
    -Method POST -Headers $headers -Body $body
```

### List Models
```powershell
$headers = @{
    "Authorization" = "Bearer kiro-YOUR_API_KEY_HERE"
}

Invoke-RestMethod -Uri "http://localhost:8090/v1/models" `
    -Method GET -Headers $headers
```

## Run Full Test Suite

```powershell
.\scripts\test_mcp_live.ps1
```

## API Keys

**Admin Key**: `kiro-YOUR_API_KEY_HERE`  
**Location**: `.kiro/admin-key.txt`

### Create New API Key
```powershell
.\scripts\create_test_key.ps1
```

### List API Keys
```powershell
.\scripts\list_keys.ps1
```

## Supported Features

- ✅ Text completions
- ✅ Vision/multimodal (images)
- ✅ Streaming responses
- ✅ OpenAI API compatibility
- ✅ Anthropic API compatibility
- ✅ Dynamic model listing
- ✅ API key management
- ✅ Automatic authentication

## Available Models

- claude-sonnet-4-20250514
- claude-opus-4-20251124
- anthropic.claude-sonnet-4-20250514-v1:0
- ...and 20 more

## Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/health` | GET | Health check |
| `/v1/chat/completions` | POST | OpenAI chat |
| `/v1/messages` | POST | Anthropic chat |
| `/v1/models` | GET | List models |
| `/v1/api-keys` | POST/GET | Manage keys |

## Configuration

**File**: `.env`  
**Key Settings**:
- `HEADLESS_MODE=true` - Automated auth
- `AUTOMATE_AUTH=true` - No manual steps
- `AMAZON_Q_SIGV4=true` - Q Developer mode
- `PORT=8090` - Server port

## Documentation

- `.archive/status-reports/BUILD_AND_TEST_COMPLETE.md` - Full build/test report
- `.archive/test-results/MCP_LIVE_TEST_RESULTS.md` - Detailed test results
- `README.md` - Project overview
- `docs/` - Complete documentation

## Support

All systems tested and operational. Ready for use.
