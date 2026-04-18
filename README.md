# Kiro Gateway Go

An OpenAI-compatible API gateway for [Amazon Q Developer](https://aws.amazon.com/q/developer/). Exposes Q Developer's capabilities through standard `/v1/chat/completions` and `/v1/models` endpoints, so any OpenAI-compatible client can use it.

## Features

- **OpenAI-compatible API** — `/v1/chat/completions`, `/v1/models`, streaming support
- **Anthropic adapter** — also serves `/v1/messages` for Claude-format clients
- **Multiple auth modes** — AWS SSO (headless device flow), SigV4, bearer token
- **Auto-refreshing credentials** — SSO tokens refresh automatically in the background
- **API key management** — issue and revoke keys for multi-user access
- **Admin dashboard** — web UI for monitoring, auth status, and key management
- **Conversation context** — maintains conversation history per session
- **Vision/multimodal** — forwards image attachments to Q Developer
- **Response caching** — configurable LRU cache for repeated queries
- **Concurrency controls** — connection pooling, circuit breaker, load shedding, rate limiting
- **TOTP/MFA support** — built-in TOTP generator for automated MFA flows
- **Docker-ready** — multi-stage Alpine build, ~15MB image

## Quick Start

### Docker (recommended)

```bash
docker build -t kiro-gateway .
docker run -d --name kiro-gateway -p 8080:8080 \
  -v ~/.kiro:/root/.kiro \
  kiro-gateway
```

On first start, the gateway will print a device authorization URL. Open it in your browser to authenticate with AWS IAM Identity Center.

### From source

```bash
go build -o kiro-gateway ./cmd/kiro-gateway/
./kiro-gateway
```

### Test it

```bash
# Health check
curl http://localhost:8080/health

# List models
curl http://localhost:8080/v1/models \
  -H "Authorization: Bearer YOUR_API_KEY"

# Chat
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "auto",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

## Configuration

Configuration is via environment variables or a `.env` file:

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Server port |
| `AWS_REGION` | `us-east-1` | AWS region for Q Developer |
| `AMAZON_Q_SIGV4` | `false` | Use SigV4 auth (headless SSO) instead of bearer token |
| `Q_USE_SENDMESSAGE` | `false` | Use Q Developer SendMessage API (enables multimodal) |
| `SSO_START_URL` | | AWS IAM Identity Center start URL |
| `SSO_ACCOUNT_ID` | | AWS account ID for SSO |
| `SSO_ROLE_NAME` | | SSO role name |
| `ADMIN_API_KEY` | *(auto-generated)* | Admin API key (printed on first start) |
| `CACHE_ENABLED` | `true` | Enable response caching |
| `CACHE_MAX_SIZE` | `1000` | Max cached responses |
| `GOMAXPROCS` | *(auto)* | Number of OS threads |

## Authentication

### Headless SSO (recommended for servers)

Set `AMAZON_Q_SIGV4=true` with your SSO details:

```bash
AMAZON_Q_SIGV4=true
SSO_START_URL=https://your-org.awsapps.com/start
SSO_ACCOUNT_ID=123456789012
SSO_ROLE_NAME=YourRoleName
AWS_REGION=us-east-1
```

The gateway performs OIDC device authorization on startup — visit the printed URL to approve. Credentials auto-refresh.

### Bearer Token (desktop)

Without `AMAZON_Q_SIGV4`, the gateway uses CodeWhisperer bearer token auth. Authenticate through the admin dashboard at `http://localhost:8080/admin/`.

## Project Structure

```
cmd/
  kiro-gateway/     # Main server
  totp-generator/   # TOTP code generator CLI
  totp-manager/     # TOTP secret management CLI
internal/
  adapters/         # OpenAI & Anthropic API adapters
  apikeys/          # API key management & storage
  auth/             # SSO, SigV4, bearer, OIDC, credential chain
  cache/            # Response caching (LRU)
  client/           # HTTP client with retries
  concurrency/      # Connection pool, circuit breaker, load shedder
  config/           # Configuration loading
  conversation/     # Conversation context management
  handlers/         # HTTP route handlers
  streaming/        # SSE streaming support
  validation/       # Request validation
  web/              # Admin dashboard (embedded HTML)
pkg/
  tokenizer/        # Token counting utilities
```

## API Endpoints

| Endpoint | Method | Description |
|---|---|---|
| `/v1/chat/completions` | POST | OpenAI-compatible chat |
| `/v1/models` | GET | List available models |
| `/v1/messages` | POST | Anthropic-compatible chat |
| `/health` | GET | Health check |
| `/admin/` | GET | Admin dashboard |
| `/api/chat` | POST | Direct Q Developer API |
| `/api/keys` | GET/POST/DELETE | API key management |
| `/api/totp` | POST | TOTP code generation |

## License

MIT
