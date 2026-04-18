# Kiro Gateway - Quick Reference

## Quick Start

```bash
# Compile
go build -o kiro-gateway.exe ./cmd/kiro-gateway

# Run
./kiro-gateway.exe
```

## Credential Storage Methods

Kiro Gateway supports **three methods** for storing/retrieving credentials:

### 1. SQLite Database (OAuth2 tokens)
- Location: `~/.amazon-q/data.sqlite3`
- Used for: Builder ID and Identity Center bearer tokens
- Setup: `q login --license [free|pro]`
- Persistent: Yes
- Auto-refresh: Yes

### 2. Environment Variables
- Used for: AWS credentials, temporary sessions
- Setup: `export AWS_ACCESS_KEY_ID=...`
- Persistent: No (session-only)
- Auto-refresh: No

### 3. AWS Credential Chain
- Used for: SigV4 authentication
- Sources: Environment → Profile → Web Identity → ECS → EC2 IMDS
- Setup: `~/.aws/credentials` or IAM role
- Persistent: Yes (profiles)
- Auto-refresh: Yes (temporary credentials)

See **CREDENTIAL_STORAGE_METHODS.md** for detailed information.

## Configuration Modes

### 1. Builder ID (Free) - Default
```bash
export Q_USE_SENDMESSAGE=false
export AMAZON_Q_SIGV4=false
export AWS_REGION=us-east-1
export PORT=8090
export PROXY_API_KEY=your-key
./kiro-gateway.exe
```
- Endpoint: `codewhisperer.us-east-1.amazonaws.com`
- Operation: `/generateAssistantResponse`
- Auth: Bearer token (Builder ID)
- Profile ARN: Not required

### 2. Identity Center (Pro/Enterprise)
```bash
export Q_USE_SENDMESSAGE=false
export AMAZON_Q_SIGV4=false
export AWS_REGION=us-east-1
export PROFILE_ARN=arn:aws:q:us-east-1:123456789012:profile/abc123
export PORT=8090
export PROXY_API_KEY=your-key
./kiro-gateway.exe
```
- Endpoint: `codewhisperer.us-east-1.amazonaws.com`
- Operation: `/generateAssistantResponse`
- Auth: Bearer token (Identity Center)
- Profile ARN: Required

### 3. CloudShell (SigV4)
```bash
export Q_USE_SENDMESSAGE=true
export AMAZON_Q_SIGV4=true
export AWS_REGION=us-east-1
export AWS_PROFILE=xnet-admin
export PORT=8090
export PROXY_API_KEY=your-key
./kiro-gateway.exe
```
- Endpoint: `q.us-east-1.amazonaws.com`
- Operation: `/sendMessage`
- Auth: AWS SigV4 (IAM)
- Profile ARN: Not required

## Environment Variables

| Variable | Values | Description |
|----------|--------|-------------|
| `Q_USE_SENDMESSAGE` | `true`/`false` | Use Q endpoint (true) or CodeWhisperer (false) |
| `AMAZON_Q_SIGV4` | `true`/`false` | Use SigV4 (true) or bearer token (false) |
| `AWS_REGION` | `us-east-1`, etc. | AWS region |
| `AWS_PROFILE` | Profile name | AWS credentials profile (for SigV4) |
| `PROFILE_ARN` | ARN string | Q Developer profile ARN (for Identity Center) |
| `PORT` | Port number | Gateway port (default: 8080) |
| `PROXY_API_KEY` | API key | Proxy authentication key |

## API Endpoints

### Health Check
```bash
curl http://localhost:8090/health
```

### List Models
```bash
curl -H "Authorization: Bearer your-key" \
     http://localhost:8090/v1/models
```

### Chat Completion
```bash
curl -X POST http://localhost:8090/v1/chat/completions \
  -H "Authorization: Bearer your-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": false
  }'
```

## Test Scripts

```bash
# Test CodeWhisperer mode
.\test_codewhisperer_quick.ps1

# Test Q mode
.\test_qdeveloper_quick.ps1

# Test both modes
.\test_both_modes.ps1
```

## Troubleshooting

### Gateway won't start
- Check port is not in use: `netstat -ano | findstr :8090`
- Check AWS credentials: `aws sts get-caller-identity`
- Check environment variables: `echo $env:AWS_REGION`

### Authentication errors
- Builder ID: Run `q login --license free`
- Identity Center: Run `q login --license pro`
- SigV4: Check `AWS_PROFILE` is set

### Profile ARN required
- Identity Center users: Run `q profile`
- Or set manually: `export PROFILE_ARN=arn:aws:q:...`

### API 500 errors
- Gateway is working correctly
- API needs additional configuration
- Check credentials and permissions

## Mode Selection Logic

```
Q_USE_SENDMESSAGE=false → CodeWhisperer endpoint
Q_USE_SENDMESSAGE=true  → Q endpoint

AMAZON_Q_SIGV4=false → Bearer token
AMAZON_Q_SIGV4=true  → SigV4

PROFILE_ARN set → Identity Center mode
PROFILE_ARN not set → Builder ID mode
```

## Quick Decision Tree

```
Are you in CloudShell?
├─ Yes → Use Q endpoint with SigV4
│         Q_USE_SENDMESSAGE=true
│         AMAZON_Q_SIGV4=true
│
└─ No → Are you using Identity Center (Pro)?
    ├─ Yes → Use CodeWhisperer with bearer token + profile ARN
    │         Q_USE_SENDMESSAGE=false
    │         AMAZON_Q_SIGV4=false
    │         PROFILE_ARN=arn:aws:q:...
    │
    └─ No → Use CodeWhisperer with bearer token (Builder ID)
              Q_USE_SENDMESSAGE=false
              AMAZON_Q_SIGV4=false
```

## Logs

Gateway logs show:
- Selected mode (CodeWhisperer or QDeveloper)
- Authentication type (Bearer Token or SigV4)
- Service name (codewhisperer or q)
- API endpoint being used
- Request IDs for debugging

Example:
```
Using CodeWhisperer mode: codewhisperer.{region}.amazonaws.com with bearer token authentication
Initialized auth manager - Type: desktop, Mode: Bearer Token, Service: codewhisperer
Starting Kiro Gateway with AWS Security Baseline on port 8090
API Mode: CodeWhisperer (bearer token), Region: us-east-1
```

## Documentation

- **AUTHENTICATION_MODES_COMPLETE.md** - Authentication modes guide
- **MULTIMODAL_API_CONFIGURATION.md** - API configuration details
- **COMPLETE_IMPLEMENTATION_SUMMARY.md** - Full implementation summary
- **QUICK_REFERENCE.md** - This document

---

**Need Help?**
- Check logs for error messages
- Verify environment variables
- Test with health endpoint first
- Ensure AWS credentials are valid
