# Quick Start Guide

Get Kiro Gateway Go running in 5 minutes!

## Prerequisites

- Go 1.21 or later
- Kiro Desktop installed (or AWS SSO credentials)

## Step 1: Clone and Build

```bash
cd kiro-gateway-go
go mod download
make build
```

## Step 2: Configure

Create `.env` file:

```bash
cp .env.example .env
```

Edit `.env` with your settings:

```env
# Required
PORT=8080
PROXY_API_KEY=my-secret-key-123

# For Kiro Desktop (easiest)
AUTH_TYPE=desktop
KIRO_DB_PATH=~/.kiro/kiro.db
```

## Step 3: Run

```bash
./kiro-gateway
```

You should see:
```
Starting Kiro Gateway v1.0.0
Auth Type: desktop
API Host: https://q.us-east-1.amazonaws.com
Authentication initialized successfully
Server listening on port 8080
```

## Step 4: Test

### Test Health Endpoint

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "healthy",
  "timestamp": "2026-01-22T11:00:00Z",
  "version": "1.0.0"
}
```

### Test Models Endpoint

```bash
curl http://localhost:8080/v1/models \
  -H "Authorization: Bearer my-secret-key-123"
```

Expected response:
```json
{
  "object": "list",
  "data": [
    {
      "id": "claude-3-5-sonnet-20241022",
      "object": "model",
      "created": 1737547200,
      "owned_by": "anthropic",
      "description": "Claude model via Kiro API"
    },
    ...
  ]
}
```

### Test Chat Completions (Coming Soon)

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer my-secret-key-123" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

## Step 5: Use with OpenAI SDK

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="my-secret-key-123"
)

# List models
models = client.models.list()
print([m.id for m in models.data])

# Chat (coming soon)
# response = client.chat.completions.create(
#     model="claude-3-5-sonnet-20241022",
#     messages=[{"role": "user", "content": "Hello!"}]
# )
# print(response.choices[0].message.content)
```

## Troubleshooting

### "Failed to load initial token"

**Problem**: Can't read Kiro Desktop database

**Solution**: Check that Kiro Desktop is installed and you're logged in:
```bash
ls ~/.kiro/kiro.db
```

### "Invalid or missing API Key"

**Problem**: Wrong API key in request

**Solution**: Make sure Authorization header matches PROXY_API_KEY in .env:
```bash
curl http://localhost:8080/v1/models \
  -H "Authorization: Bearer my-secret-key-123"
```

### "Connection refused"

**Problem**: Server not running

**Solution**: Start the server:
```bash
./kiro-gateway
```

## Next Steps

- Read [README.md](README.md) for full documentation
- Check [PROJECT_STATUS.md](PROJECT_STATUS.md) for implementation status
- See [GO_IMPLEMENTATION_ANALYSIS.md](../GO_IMPLEMENTATION_ANALYSIS.md) for architecture details

## Development

Run with debug logging:
```bash
DEBUG=true ./kiro-gateway
```

Run tests:
```bash
make test
```

Build for all platforms:
```bash
make build-all
```

## Docker

Build and run with Docker:
```bash
make docker-build
make docker-run
```

Or manually:
```bash
docker build -t kiro-gateway-go .
docker run -p 8080:8080 --env-file .env kiro-gateway-go
```

## Support

- Issues: https://github.com/yourusername/kiro-gateway-go/issues
- Discussions: https://github.com/yourusername/kiro-gateway-go/discussions
