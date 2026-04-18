# Q Developer Pro with Identity Center Setup Guide

## Overview

Q Developer Pro with Identity Center requires **bearer token authentication** with a **Profile ARN**. This is different from CloudShell which uses SigV4 authentication.

## Authentication Requirements

| Component | Value |
|-----------|-------|
| **Endpoint** | `codewhisperer.us-east-1.amazonaws.com` |
| **Operation** | `/generateAssistantResponse` |
| **Auth Method** | Bearer Token (NOT SigV4) |
| **Profile ARN** | Required (from Identity Center) |
| **Token Source** | Identity Center session via Q CLI |

## Setup Steps

### 1. Login to Q Developer Pro

```bash
# Login with Q Developer Pro license
q login --license pro
```

This will:
- Open your browser to authenticate with Identity Center
- Store bearer token in `~/.amazon-q/data.sqlite3`
- Create an Identity Center session

### 2. Get Your Profile ARN

```bash
# Display your profile information
q profile
```

This will show output like:
```
Profile ARN: arn:aws:q:us-east-1:096305372922:profile/abc123def456
```

Copy the Profile ARN value.

### 3. Configure Environment Variables

Edit your `.env` file:

```bash
# Identity Center Configuration for Q Developer Pro
PORT=8090
PROXY_API_KEY=your-api-key

# Use bearer token authentication (NOT SigV4)
AMAZON_Q_SIGV4=false

# AWS Configuration
AWS_REGION=us-east-1

# Use CodeWhisperer endpoint (NOT Q endpoint)
Q_USE_SENDMESSAGE=false

# Profile ARN from Identity Center - REQUIRED
PROFILE_ARN=arn:aws:q:us-east-1:096305372922:profile/YOUR_PROFILE_ID

# Authentication type
AUTH_TYPE=desktop

# Enable debug logging
LOG_LEVEL=debug
DEBUG=true
```

**Important**: Replace `YOUR_PROFILE_ID` with the actual Profile ARN from step 2.

### 4. Build and Run Gateway

```bash
# Build the gateway
go build -o kiro-gateway.exe ./cmd/kiro-gateway

# Run the gateway
./kiro-gateway.exe
```

### 5. Verify Configuration

Check the logs for:
```
Using CodeWhisperer mode: codewhisperer.{region}.amazonaws.com with bearer token authentication
Initialized auth manager - Type: desktop, Mode: Bearer Token, Service: codewhisperer
```

### 6. Test the Gateway

```powershell
# Test with a simple request
.\test_codewhisperer_quick.ps1
```

## How It Works

### Bearer Token Flow

1. **Token Loading**:
   - Gateway reads bearer token from Q CLI database (`~/.amazon-q/data.sqlite3`)
   - Token is cached in memory with expiration time
   - Profile ARN is loaded from environment variable

2. **Request Authentication**:
   - Bearer token added to `Authorization: Bearer <token>` header
   - Profile ARN included in request body
   - Request sent to CodeWhisperer endpoint

3. **Token Refresh**:
   - Gateway checks token expiration before each request
   - Automatically refreshes token if expired
   - Uses Q CLI's refresh mechanism

### Request Format

```json
{
  "conversationState": {
    "currentMessage": {
      "userInputMessage": {
        "content": "Hello"
      }
    },
    "chatTriggerType": "MANUAL"
  },
  "profileArn": "arn:aws:q:us-east-1:096305372922:profile/abc123def456"
}
```

## Troubleshooting

### Error: "The bearer token included in the request is invalid"

**Cause**: Token expired or Profile ARN missing

**Solution**:
1. Re-login: `q login --license pro`
2. Verify Profile ARN is set in `.env`
3. Restart gateway

### Error: "Profile ARN required"

**Cause**: `PROFILE_ARN` not set in environment

**Solution**:
1. Run `q profile` to get your Profile ARN
2. Add to `.env`: `PROFILE_ARN=arn:aws:q:...`
3. Restart gateway

### Error: "Failed to connect to Kiro API"

**Cause**: Wrong endpoint or authentication mode

**Solution**:
1. Verify `AMAZON_Q_SIGV4=false` (bearer token mode)
2. Verify `Q_USE_SENDMESSAGE=false` (CodeWhisperer endpoint)
3. Check network connectivity

### Token Not Found

**Cause**: Q CLI not logged in or database not accessible

**Solution**:
1. Run `q login --license pro`
2. Verify database exists: `~/.amazon-q/data.sqlite3`
3. Check file permissions

## Configuration Comparison

### ❌ WRONG (SigV4 mode)
```bash
AMAZON_Q_SIGV4=true          # Wrong for Identity Center
Q_USE_SENDMESSAGE=true       # Wrong endpoint
AWS_PROFILE=xnet-admin       # Not needed for bearer token
```

### ✅ CORRECT (Bearer Token mode)
```bash
AMAZON_Q_SIGV4=false         # Bearer token authentication
Q_USE_SENDMESSAGE=false      # CodeWhisperer endpoint
PROFILE_ARN=arn:aws:q:...    # Required for Identity Center
AUTH_TYPE=desktop            # Use Q CLI database
```

## Key Differences from CloudShell

| Feature | Identity Center | CloudShell |
|---------|----------------|------------|
| **Auth Method** | Bearer Token | SigV4 |
| **Endpoint** | codewhisperer | q |
| **Profile ARN** | Required | Not required |
| **Token Source** | Q CLI database | AWS credentials |
| **Multimodal** | No | Yes |

## Code Changes Made

### 1. Added Profile ARN to Auth Config
- `internal/auth/auth.go`: Added `ProfileARN` field to `Config` struct
- Profile ARN now passed from environment to auth manager

### 2. Updated Auth Manager Initialization
- `cmd/kiro-gateway/main.go`: Pass `cfg.ProfileARN` to auth config
- Profile ARN set during auth manager creation

### 3. Fixed Profile ARN Loading
- `internal/auth/desktop.go`: Don't overwrite Profile ARN from environment
- Environment variable takes precedence over database value

### 4. Fixed Profile ARN Usage
- `internal/handlers/chat.go`: Use Profile ARN for bearer token mode
- Previously only used for SigV4 mode (incorrect)

## Next Steps

1. **Get Profile ARN**: Run `q profile` command
2. **Update .env**: Add `PROFILE_ARN=...` to `.env` file
3. **Rebuild**: Run `go build -o kiro-gateway.exe ./cmd/kiro-gateway`
4. **Test**: Run `.\test_codewhisperer_quick.ps1`

## References

- [QUICK_REFERENCE.md](QUICK_REFERENCE.md) - Configuration reference
- [AUTHENTICATION.md](AUTHENTICATION.md) - Authentication methods
- [README.md](README.md) - General documentation
