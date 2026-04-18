# API Adapters Guide

Complete guide for using Kiro Gateway's OpenAI and Anthropic API adapters.

---

## Quick Start

### 1. Start the Gateway

```bash
.\kiro-gateway.exe
```

### 2. Set Your API Key

The gateway uses `PROXY_API_KEY` from your `.env` file for authentication.

```bash
# In .env
PROXY_API_KEY=my-super-secret-password-123
```

### 3. Make Your First Request

**OpenAI Format:**
```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer my-super-secret-password-123" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-sonnet-4-5",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

**Anthropic Format:**
```bash
curl http://localhost:8080/v1/messages \
  -H "x-api-key: my-super-secret-password-123" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-sonnet-4-5",
    "max_tokens": 1024,
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

---

## OpenAI API Compatibility

### Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/models` | GET | List available models |
| `/v1/chat/completions` | POST | Chat completions (streaming & non-streaming) |

### Authentication

Use the `Authorization` header with Bearer token:

```
Authorization: Bearer <your-api-key>
```

### Request Format

```json
{
  "model": "claude-sonnet-4-5",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Hello!"}
  ],
  "stream": false,
  "temperature": 0.7,
  "max_tokens": 1024
}
```

### Response Format (Non-Streaming)

```json
{
  "id": "chatcmpl-123",
  "object": "chat.completion",
  "created": 1677652288,
  "model": "claude-sonnet-4-5",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! How can I help you today?"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 20,
    "total_tokens": 30
  }
}
```

### Response Format (Streaming)

Server-Sent Events (SSE) format:

```
data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"claude-sonnet-4-5","choices":[{"index":0,"delta":{"content":"Hello"}}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"claude-sonnet-4-5","choices":[{"index":0,"delta":{"content":"!"}}]}

data: [DONE]
```

### Supported Models

- `claude-sonnet-4-5` (latest)
- `claude-3-5-sonnet-20241022`
- `claude-3-5-sonnet-20240620`
- `claude-3-opus-20240229`
- `claude-3-sonnet-20240229`
- `claude-3-haiku-20240307`

---

## Anthropic API Compatibility

### Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/messages` | POST | Messages API (streaming & non-streaming) |

### Authentication

Use the `x-api-key` header:

```
x-api-key: <your-api-key>
anthropic-version: 2023-06-01
```

Or use Bearer token (also supported):

```
Authorization: Bearer <your-api-key>
anthropic-version: 2023-06-01
```

### Request Format

```json
{
  "model": "claude-sonnet-4-5",
  "max_tokens": 1024,
  "system": "You are a helpful assistant.",
  "messages": [
    {"role": "user", "content": "Hello!"}
  ],
  "stream": false,
  "temperature": 0.7
}
```

### Response Format (Non-Streaming)

```json
{
  "id": "msg_123",
  "type": "message",
  "role": "assistant",
  "content": [
    {
      "type": "text",
      "text": "Hello! How can I help you today?"
    }
  ],
  "model": "claude-sonnet-4-5",
  "stop_reason": "end_turn",
  "usage": {
    "input_tokens": 10,
    "output_tokens": 20
  }
}
```

### Response Format (Streaming)

Server-Sent Events (SSE) format:

```
event: message_start
data: {"type":"message_start","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-sonnet-4-5"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"!"}}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"}}

event: message_stop
data: {"type":"message_stop"}
```

---

## SDK Integration

### Python - OpenAI SDK

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="my-super-secret-password-123"
)

# Non-streaming
response = client.chat.completions.create(
    model="claude-sonnet-4-5",
    messages=[
        {"role": "system", "content": "You are a helpful assistant."},
        {"role": "user", "content": "What is 2+2?"}
    ]
)
print(response.choices[0].message.content)

# Streaming
stream = client.chat.completions.create(
    model="claude-sonnet-4-5",
    messages=[{"role": "user", "content": "Count to 5"}],
    stream=True
)

for chunk in stream:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="")
```

### Python - Anthropic SDK

```python
import anthropic

client = anthropic.Anthropic(
    api_key="my-super-secret-password-123",
    base_url="http://localhost:8080"
)

# Non-streaming
response = client.messages.create(
    model="claude-sonnet-4-5",
    max_tokens=1024,
    messages=[
        {"role": "user", "content": "What is 2+2?"}
    ]
)
print(response.content[0].text)

# Streaming
with client.messages.stream(
    model="claude-sonnet-4-5",
    max_tokens=1024,
    messages=[{"role": "user", "content": "Count to 5"}]
) as stream:
    for text in stream.text_stream:
        print(text, end="", flush=True)
```

### JavaScript/TypeScript - OpenAI SDK

```typescript
import OpenAI from 'openai';

const client = new OpenAI({
  baseURL: 'http://localhost:8080/v1',
  apiKey: 'my-super-secret-password-123',
});

// Non-streaming
const response = await client.chat.completions.create({
  model: 'claude-sonnet-4-5',
  messages: [
    { role: 'user', content: 'What is 2+2?' }
  ],
});
console.log(response.choices[0].message.content);

// Streaming
const stream = await client.chat.completions.create({
  model: 'claude-sonnet-4-5',
  messages: [{ role: 'user', content: 'Count to 5' }],
  stream: true,
});

for await (const chunk of stream) {
  process.stdout.write(chunk.choices[0]?.delta?.content || '');
}
```

### LangChain

```python
from langchain_openai import ChatOpenAI

llm = ChatOpenAI(
    base_url="http://localhost:8080/v1",
    api_key="my-super-secret-password-123",
    model="claude-sonnet-4-5"
)

response = llm.invoke("What is 2+2?")
print(response.content)
```

---

## Advanced Features

### System Prompts

**OpenAI Format:**
```json
{
  "model": "claude-sonnet-4-5",
  "messages": [
    {"role": "system", "content": "You are a helpful math tutor."},
    {"role": "user", "content": "What is 2+2?"}
  ]
}
```

**Anthropic Format:**
```json
{
  "model": "claude-sonnet-4-5",
  "max_tokens": 1024,
  "system": "You are a helpful math tutor.",
  "messages": [
    {"role": "user", "content": "What is 2+2?"}
  ]
}
```

### Temperature Control

Both APIs support temperature control:

```json
{
  "model": "claude-sonnet-4-5",
  "temperature": 0.7,
  "messages": [...]
}
```

- `0.0` = More deterministic
- `1.0` = More creative

### Max Tokens

**OpenAI Format:**
```json
{
  "model": "claude-sonnet-4-5",
  "max_tokens": 1024,
  "messages": [...]
}
```

**Anthropic Format:**
```json
{
  "model": "claude-sonnet-4-5",
  "max_tokens": 1024,
  "messages": [...]
}
```

---

## Tool Integration

### Cursor IDE

1. Open Cursor Settings
2. Go to "Models" → "OpenAI API"
3. Set Base URL: `http://localhost:8080/v1`
4. Set API Key: Your `PROXY_API_KEY`
5. Select Model: `claude-sonnet-4-5`

### Continue.dev

In `.continue/config.json`:

```json
{
  "models": [
    {
      "title": "Claude via Kiro",
      "provider": "openai",
      "model": "claude-sonnet-4-5",
      "apiBase": "http://localhost:8080/v1",
      "apiKey": "my-super-secret-password-123"
    }
  ]
}
```

### Cline (VSCode Extension)

1. Open Cline settings
2. Select "OpenAI Compatible"
3. Base URL: `http://localhost:8080/v1`
4. API Key: Your `PROXY_API_KEY`
5. Model: `claude-sonnet-4-5`

---

## Testing

### Run Test Suite

```powershell
.\scripts\test_adapters.ps1
```

This tests:
- ✅ OpenAI models endpoint
- ✅ OpenAI non-streaming chat
- ✅ OpenAI streaming chat
- ✅ Anthropic non-streaming messages
- ✅ Anthropic messages with system prompt
- ✅ Anthropic streaming messages

### Manual Testing

**Test OpenAI endpoint:**
```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer test-key" \
  -H "Content-Type: application/json" \
  -d '{"model":"claude-sonnet-4-5","messages":[{"role":"user","content":"Say hello"}]}'
```

**Test Anthropic endpoint:**
```bash
curl http://localhost:8080/v1/messages \
  -H "x-api-key: test-key" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{"model":"claude-sonnet-4-5","max_tokens":1024,"messages":[{"role":"user","content":"Say hello"}]}'
```

---

## Troubleshooting

### Common Issues

#### 401 Unauthorized
- Check that `PROXY_API_KEY` is set in `.env`
- Verify the API key in your request matches
- Ensure you're using the correct header (`Authorization` or `x-api-key`)

#### 404 Not Found
- Verify the endpoint path is correct
- OpenAI: `/v1/chat/completions`
- Anthropic: `/v1/messages`

#### 500 Internal Server Error
- Check gateway logs: `logs/gateway.log`
- Verify AWS credentials are configured
- Check that `PROFILE_ARN` is set correctly

#### Streaming Not Working
- Ensure `stream: true` in request body
- Check that your client supports SSE
- Verify no proxy is buffering responses

### Debug Mode

Enable debug logging:

```bash
# In .env
LOG_LEVEL=debug
```

Then check logs:
```bash
tail -f logs/gateway.log
```

---

## Performance Tips

1. **Use Streaming**: Lower latency for long responses
2. **Connection Pooling**: Gateway reuses connections to AWS
3. **Appropriate Timeouts**: Set reasonable timeouts in your client
4. **Batch Requests**: Use async/parallel requests when possible

---

## Security Best Practices

1. **Secure API Keys**: Never commit API keys to version control
2. **Use HTTPS**: In production, always use HTTPS
3. **Rate Limiting**: Consider adding rate limits per API key
4. **Network Security**: Restrict gateway access to trusted networks
5. **Audit Logging**: Enable request logging for security audits

---

## Configuration Reference

### Environment Variables

```bash
# Required
PROXY_API_KEY=your-secret-key
AWS_REGION=us-east-1
PROFILE_ARN=your-profile-arn

# Optional
PORT=8080
MAX_CONNECTIONS=100
ENABLE_EXTENDED_CONTEXT=true
ENABLE_EXTENDED_THINKING=true
```

### Model Aliases

You can use short names that map to full model IDs:
- `claude-sonnet-4-5` → Latest Sonnet 4.5
- `claude-opus` → Latest Opus
- `claude-haiku` → Latest Haiku

---

## Support

For issues or questions:
1. Check the logs: `logs/gateway.log`
2. Review this guide
3. Check the main README.md
4. Review .archive/status-reports/.archive/status-reports/ADAPTER_INTEGRATION_COMPLETE.md

---

## What's Next?

Future enhancements planned:
- Tool/function calling support
- Vision/multimodal requests
- Extended thinking content
- Usage analytics
- Rate limiting per API key

---

**Happy coding! 🚀**
