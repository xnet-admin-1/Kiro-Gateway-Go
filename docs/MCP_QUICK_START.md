# MCP Quick Start Guide

Get started with Model Context Protocol (MCP) in kiro-gateway in 5 minutes.

---

## What is MCP?

MCP (Model Context Protocol) enables Q Developer to call external tools like AWS APIs, GitHub operations, file system access, and more.

**Without MCP**: Q Developer can only chat  
**With MCP**: Q Developer can take actions

---

## Prerequisites

1. **kiro-gateway** installed and running
2. **Python** with `uv` installed (for MCP servers)
3. **AWS credentials** configured (for AWS MCP server)

### Install uv (Python package manager)

```bash
# macOS/Linux
curl -LsSf https://astral.sh/uv/install.sh | sh

# Windows
powershell -c "irm https://astral.sh/uv/install.ps1 | iex"
```

---

## Quick Start

### Step 1: Configure MCP Server

Create or edit `config.json`:

```json
{
  "mcp": {
    "enabled": true,
    "servers": [
      {
        "name": "aws",
        "type": "stdio",
        "command": "uvx",
        "args": ["mcp-server-aws@latest"],
        "env": {
          "AWS_REGION": "us-east-1"
        }
      }
    ]
  }
}
```

### Step 2: Start Gateway

```bash
./kiro-gateway --config config.json
```

You should see:
```
MCP: Connected to server aws with 15 tools
  - describe-instance: Get EC2 instance details
  - list-buckets: List S3 buckets
  ...
```

### Step 3: Test Tool Calling

Send a request to Q Developer:

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-sonnet-4-5",
    "messages": [{
      "role": "user",
      "content": "Describe EC2 instance i-1234567890abcdef0"
    }]
  }'
```

Q Developer will:
1. Recognize it needs to call a tool
2. Return `tool_uses` in response
3. Gateway calls `@aws/describe-instance`
4. Gateway sends result back to Q Developer
5. Q Developer provides final answer

---

## Available MCP Servers

### AWS Server

```json
{
  "name": "aws",
  "type": "stdio",
  "command": "uvx",
  "args": ["mcp-server-aws@latest"],
  "env": {
    "AWS_REGION": "us-east-1"
  }
}
```

**Tools**: EC2, S3, Lambda, RDS, and more

### GitHub Server

```json
{
  "name": "github",
  "type": "stdio",
  "command": "uvx",
  "args": ["mcp-server-github"],
  "env": {
    "GITHUB_TOKEN": "${GITHUB_TOKEN}"
  }
}
```

**Tools**: Create issues, PRs, search code, etc.

### Filesystem Server

```json
{
  "name": "filesystem",
  "type": "stdio",
  "command": "uvx",
  "args": ["mcp-server-filesystem", "/path/to/directory"],
  "env": {}
}
```

**Tools**: Read files, list directories, search files

### Multiple Servers

```json
{
  "mcp": {
    "enabled": true,
    "servers": [
      {
        "name": "aws",
        "type": "stdio",
        "command": "uvx",
        "args": ["mcp-server-aws@latest"]
      },
      {
        "name": "github",
        "type": "stdio",
        "command": "uvx",
        "args": ["mcp-server-github"]
      },
      {
        "name": "filesystem",
        "type": "stdio",
        "command": "uvx",
        "args": ["mcp-server-filesystem", "/tmp"]
      }
    ]
  }
}
```

---

## Tool Naming

Tools are namespaced by server:

```
@server_name/tool_name
```

Examples:
- `@aws/describe-instance`
- `@github/create-issue`
- `@filesystem/read-file`

---

## Environment Variables

Use `${VAR_NAME}` syntax in configuration:

```json
{
  "env": {
    "GITHUB_TOKEN": "${GITHUB_TOKEN}",
    "AWS_REGION": "${AWS_REGION}"
  }
}
```

Gateway will substitute from environment:

```bash
export GITHUB_TOKEN="ghp_xxxxxxxxxxxx"
export AWS_REGION="us-west-2"
./kiro-gateway --config config.json
```

---

## Troubleshooting

### Server Won't Connect

**Error**: `failed to connect to MCP server`

**Solutions**:
1. Check `uvx` is installed: `uvx --version`
2. Test server manually: `uvx mcp-server-aws@latest`
3. Check environment variables are set
4. Review gateway logs for details

### Tool Call Fails

**Error**: `failed to call tool`

**Solutions**:
1. Verify tool exists: Check startup logs
2. Check tool parameters match schema
3. Increase timeout in config
4. Check server logs (stderr)

### No Tools Discovered

**Error**: `Connected to server aws with 0 tools`

**Solutions**:
1. Server may not support tools capability
2. Check server is running correctly
3. Try different server version
4. Review server documentation

---

## Testing

### List Available Tools

```bash
curl http://localhost:8080/v1/mcp/tools
```

Response:
```json
{
  "servers": {
    "aws": [
      {
        "name": "describe-instance",
        "description": "Get EC2 instance details",
        "inputSchema": {...}
      }
    ]
  }
}
```

### Health Check

```bash
curl http://localhost:8080/v1/mcp/health
```

Response:
```json
{
  "servers": {
    "aws": {
      "status": "connected",
      "tools": 15
    }
  }
}
```

---

## Examples

### Example 1: AWS EC2

**Request**:
```json
{
  "model": "claude-sonnet-4-5",
  "messages": [{
    "role": "user",
    "content": "What's the status of EC2 instance i-1234567890abcdef0?"
  }]
}
```

**Q Developer Response** (includes tool_uses):
```json
{
  "tool_uses": [{
    "id": "tool_123",
    "name": "@aws/describe-instance",
    "input": {
      "instanceId": "i-1234567890abcdef0"
    }
  }]
}
```

**Gateway Actions**:
1. Parse tool name: `aws` + `describe-instance`
2. Call MCP server: `manager.CallTool("aws", "describe-instance", {...})`
3. Get result from AWS
4. Send result back to Q Developer

**Final Response**:
```
The EC2 instance i-1234567890abcdef0 is currently running.
Instance type: t2.micro
Region: us-east-1
...
```

### Example 2: GitHub

**Request**:
```json
{
  "model": "claude-sonnet-4-5",
  "messages": [{
    "role": "user",
    "content": "Create an issue in myorg/myrepo titled 'Bug: Login fails'"
  }]
}
```

**Tool Call**: `@github/create-issue`

**Result**: Issue created with link

### Example 3: Filesystem

**Request**:
```json
{
  "model": "claude-sonnet-4-5",
  "messages": [{
    "role": "user",
    "content": "What's in the README.md file?"
  }]
}
```

**Tool Call**: `@filesystem/read-file`

**Result**: File contents returned

---

## Security Best Practices

### 1. Whitelist Servers

Only allow trusted MCP servers:

```json
{
  "mcp": {
    "allowedServers": [
      "mcp-server-aws",
      "mcp-server-github",
      "mcp-server-filesystem"
    ]
  }
}
```

### 2. Limit Filesystem Access

Restrict filesystem server to specific directories:

```json
{
  "name": "filesystem",
  "args": ["mcp-server-filesystem", "/safe/directory"]
}
```

### 3. Use Environment Variables

Never hardcode secrets:

```json
{
  "env": {
    "GITHUB_TOKEN": "${GITHUB_TOKEN}",  // ✅ Good
    "API_KEY": "hardcoded-secret"        // ❌ Bad
  }
}
```

### 4. Rate Limiting

Configure rate limits:

```json
{
  "mcp": {
    "rateLimits": {
      "maxCallsPerMinute": 60,
      "maxCallsPerHour": 1000
    }
  }
}
```

---

## Performance Tips

### 1. Connection Pooling

Reuse connections to MCP servers (automatic).

### 2. Tool Result Caching

Cache frequently used tool results:

```json
{
  "mcp": {
    "cache": {
      "enabled": true,
      "ttl": 300
    }
  }
}
```

### 3. Parallel Tool Calls

Gateway automatically parallelizes independent tool calls.

### 4. Timeouts

Set appropriate timeouts:

```json
{
  "mcp": {
    "timeout": 30
  }
}
```

---

## Next Steps

1. **Read Full Documentation**: `docs/MCP_SETUP.md`
2. **Explore MCP Servers**: https://github.com/modelcontextprotocol/servers
3. **Build Custom Server**: `docs/BUILDING_MCP_SERVERS.md`
4. **Join Community**: https://modelcontextprotocol.io/community

---

## Getting Help

### Documentation
- MCP Specification: https://modelcontextprotocol.io/specification
- Official Go SDK: https://github.com/modelcontextprotocol/go-sdk
- kiro-gateway MCP: `internal/mcp/README.md`

### Support
- GitHub Issues: https://github.com/yourusername/kiro-gateway-go/issues
- Discord: https://discord.gg/kiro-gateway
- Email: support@kiro-gateway.com

---

## Summary

**5-Minute Setup**:
1. Install `uv`
2. Add MCP config
3. Start gateway
4. Test tool calling

**That's it!** Q Developer can now interact with external systems through MCP.

Happy tool calling! 🚀

