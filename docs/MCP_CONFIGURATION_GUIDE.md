# MCP Configuration Guide

Complete reference for configuring Model Context Protocol (MCP) tool calling in kiro-gateway-go.

---

## Table of Contents

1. [Overview](#overview)
2. [Environment Variables Reference](#environment-variables-reference)
3. [Server Configuration Format](#server-configuration-format)
4. [Example Configurations](#example-configurations)
5. [Configuration Validation Rules](#configuration-validation-rules)
6. [Common Configuration Errors](#common-configuration-errors)
7. [Advanced Configuration](#advanced-configuration)
8. [Troubleshooting](#troubleshooting)

---

## Overview

MCP (Model Context Protocol) integration in kiro-gateway enables Amazon Q Developer to execute external tools through MCP servers. Configuration is done through environment variables, making it easy to deploy across different environments without code changes.

### Key Concepts

- **MCP Server**: An external process that implements the MCP protocol and provides tools
- **Tool**: A callable function exposed by an MCP server (e.g., AWS API operations, file operations)
- **Namespacing**: Tools are prefixed with server name (e.g., `@aws/describe-instance`)
- **stdio Transport**: MCP servers communicate via standard input/output

### Configuration Flow

```
Environment Variables → Gateway Startup → Server Initialization → Tool Discovery → Ready for Tool Calls
```

---

## Environment Variables Reference

### Core MCP Variables

#### `MCP_ENABLED`

**Type**: Boolean  
**Default**: `false`  
**Required**: Yes (to enable MCP)

Enables or disables MCP tool calling functionality.

```bash
# Enable MCP
MCP_ENABLED=true

# Disable MCP (default)
MCP_ENABLED=false
```

**Behavior**:
- When `true`: Gateway initializes all configured MCP servers on startup
- When `false` or unset: Gateway operates in normal mode without MCP (zero overhead)

---

### Server Configuration Variables

MCP servers are configured using a naming pattern: `MCP_SERVER_{NAME}_{PROPERTY}`

#### `MCP_SERVER_{NAME}_COMMAND`

**Type**: String  
**Required**: Yes (for each server)

The executable command to launch the MCP server process.

```bash
# Using npx (Node.js)
MCP_SERVER_AWS_COMMAND=npx

# Using uvx (Python)
MCP_SERVER_GITHUB_COMMAND=uvx

# Using direct binary
MCP_SERVER_CUSTOM_COMMAND=/usr/local/bin/my-mcp-server
```

**Common Commands**:
- `npx`: Node.js package executor (requires Node.js installed)
- `uvx`: Python package executor (requires uv installed)
- `python`: Python interpreter (for local scripts)
- `node`: Node.js runtime (for local scripts)
- Direct binary path: For compiled MCP servers

#### `MCP_SERVER_{NAME}_ARGS`

**Type**: Comma-separated string  
**Required**: No (but usually needed)

Arguments passed to the server command.

```bash
# NPM package with flags
MCP_SERVER_AWS_ARGS=-y,@modelcontextprotocol/server-aws

# Python package
MCP_SERVER_GITHUB_ARGS=mcp-server-github@latest

# Multiple arguments
MCP_SERVER_CUSTOM_ARGS=--config,/path/to/config.json,--verbose
```

**Format**:
- Arguments are comma-separated
- No spaces around commas
- Quotes are preserved if needed

#### `MCP_SERVER_{NAME}_ENV`

**Type**: Comma-separated key=value pairs  
**Required**: No

Environment variables passed to the MCP server process.

```bash
# Single variable
MCP_SERVER_AWS_ENV=AWS_REGION=us-east-1

# Multiple variables
MCP_SERVER_AWS_ENV=AWS_REGION=us-east-1,AWS_PROFILE=default

# With special characters (use quotes in shell)
MCP_SERVER_CUSTOM_ENV="API_KEY=abc123,BASE_URL=https://api.example.com"
```

**Common Use Cases**:
- AWS region configuration
- API keys and tokens
- Base URLs for APIs
- Debug flags

---

### Gateway Configuration Variables

These variables affect MCP behavior at the gateway level.

#### `LOG_LEVEL`

**Type**: String  
**Default**: `info`  
**Values**: `debug`, `info`, `warn`, `error`

Controls logging verbosity for MCP operations.

```bash
# Detailed MCP protocol logging
LOG_LEVEL=debug

# Standard logging
LOG_LEVEL=info
```

**Debug Mode**: When set to `debug`, logs include:
- Full MCP protocol messages
- Tool execution timing
- Detailed error context
- Connection state changes

#### `DEBUG`

**Type**: Boolean  
**Default**: `false`

Enables additional debug output for MCP operations.

```bash
# Enable debug mode
DEBUG=true
```

#### `CONTEXT_CLEANUP_INTERVAL`

**Type**: Duration  
**Default**: `5m`

How often to clean up stale conversation contexts.

```bash
# Clean up every 10 minutes
CONTEXT_CLEANUP_INTERVAL=10m

# Clean up every 30 seconds
CONTEXT_CLEANUP_INTERVAL=30s
```

---

## Server Configuration Format

### Naming Convention

Server names are extracted from environment variable names:

```
MCP_SERVER_{NAME}_COMMAND
           ^^^^
           Server name (used for namespacing)
```

**Rules**:
- Server names must be unique
- Server names are case-insensitive
- Server names should be alphanumeric (no spaces or special characters)
- Server names are used as namespace prefixes (e.g., `@aws/tool-name`)

### Complete Server Configuration

A complete server configuration requires at minimum the `COMMAND` variable:

```bash
# Minimal configuration
MCP_SERVER_AWS_COMMAND=npx

# Recommended configuration
MCP_SERVER_AWS_COMMAND=npx
MCP_SERVER_AWS_ARGS=-y,@modelcontextprotocol/server-aws
MCP_SERVER_AWS_ENV=AWS_REGION=us-east-1

# Full configuration with multiple env vars
MCP_SERVER_GITHUB_COMMAND=uvx
MCP_SERVER_GITHUB_ARGS=mcp-server-github@latest
MCP_SERVER_GITHUB_ENV=GITHUB_TOKEN=ghp_xxx,GITHUB_API_URL=https://api.github.com
```

---

## Example Configurations

### AWS MCP Server

Access AWS services through Q Developer.

```bash
# Enable MCP
MCP_ENABLED=true

# Configure AWS server
MCP_SERVER_AWS_COMMAND=npx
MCP_SERVER_AWS_ARGS=-y,@modelcontextprotocol/server-aws
MCP_SERVER_AWS_ENV=AWS_REGION=us-east-1
```

**Available Tools**:
- `@aws/describe-instance`: Get EC2 instance details
- `@aws/list-buckets`: List S3 buckets
- `@aws/get-parameter`: Get SSM parameter
- And more...

**Prerequisites**:
- Node.js installed
- AWS credentials configured (`~/.aws/credentials` or environment variables)

### GitHub MCP Server

Interact with GitHub repositories, issues, and pull requests.

```bash
# Enable MCP
MCP_ENABLED=true

# Configure GitHub server
MCP_SERVER_GITHUB_COMMAND=uvx
MCP_SERVER_GITHUB_ARGS=mcp-server-github@latest
MCP_SERVER_GITHUB_ENV=GITHUB_TOKEN=ghp_your_token_here
```

**Available Tools**:
- `@github/list-repos`: List repositories
- `@github/get-issue`: Get issue details
- `@github/create-pr`: Create pull request
- And more...

**Prerequisites**:
- Python with `uv` installed
- GitHub personal access token

### Filesystem MCP Server

Read and write files on the local filesystem.

```bash
# Enable MCP
MCP_ENABLED=true

# Configure filesystem server
MCP_SERVER_FILESYSTEM_COMMAND=npx
MCP_SERVER_FILESYSTEM_ARGS=-y,@modelcontextprotocol/server-filesystem,/allowed/path
```

**Available Tools**:
- `@filesystem/read-file`: Read file contents
- `@filesystem/write-file`: Write file contents
- `@filesystem/list-directory`: List directory contents
- And more...

**Prerequisites**:
- Node.js installed
- Appropriate file system permissions

**Security Note**: Only paths specified in the args are accessible.

### Multiple Servers

Configure multiple MCP servers simultaneously.

```bash
# Enable MCP
MCP_ENABLED=true

# AWS Server
MCP_SERVER_AWS_COMMAND=npx
MCP_SERVER_AWS_ARGS=-y,@modelcontextprotocol/server-aws
MCP_SERVER_AWS_ENV=AWS_REGION=us-east-1

# GitHub Server
MCP_SERVER_GITHUB_COMMAND=uvx
MCP_SERVER_GITHUB_ARGS=mcp-server-github@latest
MCP_SERVER_GITHUB_ENV=GITHUB_TOKEN=ghp_xxx

# Filesystem Server
MCP_SERVER_FILESYSTEM_COMMAND=npx
MCP_SERVER_FILESYSTEM_ARGS=-y,@modelcontextprotocol/server-filesystem,/workspace
```

**Behavior**:
- All servers initialize in parallel
- Tools from all servers are available
- Each server operates independently
- Failure of one server doesn't affect others

### Docker Deployment

Configuration for containerized deployments.

```dockerfile
# Dockerfile
FROM golang:1.21-alpine

# Install Node.js for NPM-based MCP servers
RUN apk add --no-cache nodejs npm

# Install Python and uv for Python-based MCP servers
RUN apk add --no-cache python3 py3-pip
RUN pip3 install uv

# Copy and build gateway
COPY . /app
WORKDIR /app
RUN go build -o kiro-gateway ./cmd/kiro-gateway

# Set environment variables
ENV MCP_ENABLED=true
ENV MCP_SERVER_AWS_COMMAND=npx
ENV MCP_SERVER_AWS_ARGS=-y,@modelcontextprotocol/server-aws
ENV MCP_SERVER_AWS_ENV=AWS_REGION=us-east-1

CMD ["./kiro-gateway"]
```

### Kubernetes Deployment

Configuration using ConfigMap and Secrets.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kiro-gateway-config
data:
  MCP_ENABLED: "true"
  MCP_SERVER_AWS_COMMAND: "npx"
  MCP_SERVER_AWS_ARGS: "-y,@modelcontextprotocol/server-aws"
  MCP_SERVER_AWS_ENV: "AWS_REGION=us-east-1"
  LOG_LEVEL: "info"

---
apiVersion: v1
kind: Secret
metadata:
  name: kiro-gateway-secrets
type: Opaque
stringData:
  MCP_SERVER_GITHUB_ENV: "GITHUB_TOKEN=ghp_xxx"

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kiro-gateway
spec:
  template:
    spec:
      containers:
      - name: gateway
        image: kiro-gateway:latest
        envFrom:
        - configMapRef:
            name: kiro-gateway-config
        - secretRef:
            name: kiro-gateway-secrets
```

---

## Configuration Validation Rules

The gateway validates configuration on startup and provides clear error messages for issues.

### Rule 1: MCP_ENABLED Must Be Set

**Validation**: If MCP servers are configured but `MCP_ENABLED` is not set to `true`, a warning is logged.

```bash
# ❌ Invalid: Servers configured but MCP not enabled
MCP_SERVER_AWS_COMMAND=npx
# MCP_ENABLED not set

# ✅ Valid: MCP explicitly enabled
MCP_ENABLED=true
MCP_SERVER_AWS_COMMAND=npx
```

**Error Message**:
```
WARN: MCP servers configured but MCP_ENABLED is not set to true. MCP functionality will be disabled.
```

### Rule 2: Server Names Must Be Unique

**Validation**: Each server must have a unique name.

```bash
# ❌ Invalid: Duplicate server names (case-insensitive)
MCP_SERVER_AWS_COMMAND=npx
MCP_SERVER_aws_COMMAND=uvx  # Duplicate!

# ✅ Valid: Unique server names
MCP_SERVER_AWS_COMMAND=npx
MCP_SERVER_GITHUB_COMMAND=uvx
```

**Error Message**:
```
ERROR: Duplicate MCP server name 'aws'. Server names must be unique.
```

### Rule 3: Command Must Be Executable

**Validation**: The command specified must exist and be executable.

```bash
# ❌ Invalid: Command doesn't exist
MCP_SERVER_AWS_COMMAND=nonexistent-command

# ✅ Valid: Command exists in PATH
MCP_SERVER_AWS_COMMAND=npx
```

**Error Message**:
```
ERROR: MCP server 'aws' command 'nonexistent-command' not found in PATH
```

**Note**: This validation is performed at runtime when the server starts, not at configuration load time.

### Rule 4: Arguments Must Be Properly Formatted

**Validation**: Arguments must be comma-separated without syntax errors.

```bash
# ❌ Invalid: Spaces around commas
MCP_SERVER_AWS_ARGS=-y , @modelcontextprotocol/server-aws

# ✅ Valid: No spaces
MCP_SERVER_AWS_ARGS=-y,@modelcontextprotocol/server-aws
```

**Note**: The gateway splits on commas, so spaces become part of the argument values.

### Rule 5: Environment Variables Must Be Key=Value Pairs

**Validation**: Environment variables must follow the `KEY=VALUE` format.

```bash
# ❌ Invalid: Missing equals sign
MCP_SERVER_AWS_ENV=AWS_REGION

# ❌ Invalid: Multiple equals signs (ambiguous)
MCP_SERVER_AWS_ENV=KEY=VALUE=EXTRA

# ✅ Valid: Proper key=value format
MCP_SERVER_AWS_ENV=AWS_REGION=us-east-1

# ✅ Valid: Multiple variables
MCP_SERVER_AWS_ENV=AWS_REGION=us-east-1,AWS_PROFILE=default
```

**Error Message**:
```
WARN: Invalid environment variable format in MCP_SERVER_AWS_ENV: 'AWS_REGION' (expected KEY=VALUE)
```

### Rule 6: Server Must Have Command

**Validation**: Each server must have at least a `COMMAND` variable.

```bash
# ❌ Invalid: Only args, no command
MCP_SERVER_AWS_ARGS=-y,@modelcontextprotocol/server-aws

# ✅ Valid: Command specified
MCP_SERVER_AWS_COMMAND=npx
MCP_SERVER_AWS_ARGS=-y,@modelcontextprotocol/server-aws
```

**Error Message**:
```
ERROR: MCP server 'aws' has no command specified. Set MCP_SERVER_AWS_COMMAND.
```

---

## Common Configuration Errors

### Error 1: MCP Not Enabled

**Symptom**: Servers configured but tools not available.

**Cause**: `MCP_ENABLED` not set to `true`.

**Solution**:
```bash
# Add this line
MCP_ENABLED=true
```

### Error 2: Command Not Found

**Symptom**: Error message "command not found" or "executable file not found".

**Cause**: The specified command is not in the system PATH.

**Solution**:
```bash
# Option 1: Install the command
npm install -g npx  # For npx
pip install uv      # For uvx

# Option 2: Use full path
MCP_SERVER_AWS_COMMAND=/usr/local/bin/npx

# Option 3: Add to PATH
export PATH=$PATH:/path/to/command
```

### Error 3: Server Fails to Start

**Symptom**: Error message "failed to start MCP server" or "connection timeout".

**Cause**: Server process crashes immediately or fails to initialize.

**Solution**:
```bash
# Test the command manually
npx -y @modelcontextprotocol/server-aws

# Check logs for detailed error
LOG_LEVEL=debug ./kiro-gateway

# Verify prerequisites (Node.js, Python, etc.)
node --version
python --version
```

### Error 4: Tools Not Discovered

**Symptom**: Server connects but no tools are available.

**Cause**: Server doesn't implement tool listing or returns empty list.

**Solution**:
```bash
# Verify server supports tools
npx -y @modelcontextprotocol/server-aws --help

# Check server version
MCP_SERVER_AWS_ARGS=-y,@modelcontextprotocol/server-aws@latest

# Enable debug logging
LOG_LEVEL=debug
```

### Error 5: Authentication Failures

**Symptom**: Tools execute but return authentication errors.

**Cause**: Missing or invalid credentials in environment variables.

**Solution**:
```bash
# For AWS: Ensure credentials are available
aws configure list

# For GitHub: Verify token is valid
curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/user

# Pass credentials to MCP server
MCP_SERVER_AWS_ENV=AWS_REGION=us-east-1,AWS_PROFILE=myprofile
MCP_SERVER_GITHUB_ENV=GITHUB_TOKEN=ghp_xxx
```

### Error 6: Tool Execution Timeouts

**Symptom**: Tool calls timeout after 30 seconds.

**Cause**: Tool operation takes longer than the default timeout.

**Solution**:
```bash
# Currently, timeout is hardcoded to 30 seconds
# Future: Will be configurable via MCP_TOOL_TIMEOUT

# Workaround: Optimize the tool operation or use async patterns
```

### Error 7: Port Conflicts

**Symptom**: Gateway fails to start with "address already in use".

**Cause**: Another process is using the configured port.

**Solution**:
```bash
# Change the gateway port
PORT=8090 ./kiro-gateway

# Or kill the conflicting process
lsof -ti:8080 | xargs kill
```

---

## Advanced Configuration

### Custom MCP Servers

You can configure custom MCP servers that you've built.

```bash
# Local Python script
MCP_SERVER_CUSTOM_COMMAND=python
MCP_SERVER_CUSTOM_ARGS=/path/to/my_mcp_server.py

# Local Node.js script
MCP_SERVER_CUSTOM_COMMAND=node
MCP_SERVER_CUSTOM_ARGS=/path/to/my-mcp-server.js

# Compiled binary
MCP_SERVER_CUSTOM_COMMAND=/usr/local/bin/my-mcp-server
MCP_SERVER_CUSTOM_ARGS=--config,/etc/mcp/config.json
```

### Environment Variable Inheritance

MCP server processes inherit environment variables from the gateway process, plus any specified in `MCP_SERVER_{NAME}_ENV`.

```bash
# Gateway environment
export AWS_REGION=us-west-2
export AWS_PROFILE=production

# MCP server will inherit these
MCP_ENABLED=true
MCP_SERVER_AWS_COMMAND=npx
MCP_SERVER_AWS_ARGS=-y,@modelcontextprotocol/server-aws

# Or override explicitly
MCP_SERVER_AWS_ENV=AWS_REGION=us-east-1  # Overrides inherited value
```

### Dynamic Configuration Reloading

Currently, MCP configuration is loaded once at startup. To change configuration:

1. Update environment variables
2. Restart the gateway

**Future Enhancement**: Hot-reload configuration without restart.

### Resource Limits

MCP servers run as child processes with the following limits:

- **Tool execution timeout**: 30 seconds (hardcoded)
- **Tool result size**: 1 MB maximum
- **Concurrent executions**: Sequential (one at a time per request)
- **Memory**: Inherited from gateway process limits

**Future Enhancement**: Configurable resource limits per server.

### Logging Configuration

Control MCP logging verbosity:

```bash
# Minimal logging (errors only)
LOG_LEVEL=error

# Standard logging (info and above)
LOG_LEVEL=info

# Detailed logging (debug and above)
LOG_LEVEL=debug

# Enable debug mode for MCP protocol details
DEBUG=true
```

**Debug Mode Output**:
```
DEBUG: MCP: Sending initialize request to server 'aws'
DEBUG: MCP: Received initialize response: {"protocolVersion":"2024-11-05"}
DEBUG: MCP: Sending tools/list request to server 'aws'
DEBUG: MCP: Received 15 tools from server 'aws'
DEBUG: MCP: Tool execution started: @aws/describe-instance
DEBUG: MCP: Tool execution completed in 1.2s
```

---

## Troubleshooting

### Diagnostic Commands

#### Check MCP Status

```bash
# View gateway logs
tail -f /var/log/kiro-gateway.log

# Check if MCP is enabled
curl http://localhost:8080/health | jq '.mcp'

# List available tools
curl http://localhost:8080/mcp/tools | jq
```

#### Test Server Connectivity

```bash
# Test command manually
npx -y @modelcontextprotocol/server-aws

# Test with environment variables
AWS_REGION=us-east-1 npx -y @modelcontextprotocol/server-aws

# Check server output
npx -y @modelcontextprotocol/server-aws 2>&1 | head -20
```

#### Verify Prerequisites

```bash
# Check Node.js
node --version
npm --version
npx --version

# Check Python
python --version
pip --version
uv --version

# Check AWS credentials
aws sts get-caller-identity

# Check GitHub token
curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/user
```

### Common Issues and Solutions

#### Issue: "MCP server failed to start"

**Possible Causes**:
1. Command not found in PATH
2. Missing dependencies (Node.js, Python)
3. Invalid arguments
4. Permission issues

**Debug Steps**:
```bash
# Enable debug logging
LOG_LEVEL=debug ./kiro-gateway

# Test command manually
npx -y @modelcontextprotocol/server-aws

# Check command exists
which npx

# Check permissions
ls -la $(which npx)
```

#### Issue: "Tool execution timeout"

**Possible Causes**:
1. Tool operation takes > 30 seconds
2. Network latency
3. Server hanging

**Debug Steps**:
```bash
# Test tool manually
npx -y @modelcontextprotocol/server-aws

# Check network connectivity
ping api.aws.amazon.com

# Monitor server process
ps aux | grep mcp-server
```

#### Issue: "Authentication failed"

**Possible Causes**:
1. Missing credentials
2. Invalid credentials
3. Expired credentials
4. Wrong region

**Debug Steps**:
```bash
# Verify AWS credentials
aws sts get-caller-identity

# Verify GitHub token
curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/user

# Check environment variables
env | grep AWS
env | grep GITHUB

# Test with explicit credentials
MCP_SERVER_AWS_ENV=AWS_REGION=us-east-1,AWS_PROFILE=myprofile
```

#### Issue: "No tools discovered"

**Possible Causes**:
1. Server doesn't support tool listing
2. Server returns empty tool list
3. Tool discovery failed

**Debug Steps**:
```bash
# Enable debug logging
LOG_LEVEL=debug ./kiro-gateway

# Check server output
npx -y @modelcontextprotocol/server-aws 2>&1 | grep -i tool

# Verify server version
npm info @modelcontextprotocol/server-aws version
```

### Getting Help

If you're still experiencing issues:

1. **Check Logs**: Enable `LOG_LEVEL=debug` and review logs
2. **Test Manually**: Run MCP server commands manually to isolate issues
3. **Verify Prerequisites**: Ensure all dependencies are installed
4. **Check Documentation**: Review MCP server documentation
5. **Report Issues**: Open an issue with logs and configuration

---

## Next Steps

- **Quick Start**: See [MCP_docs/guides/docs/guides/docs/guides/docs/guides/QUICK_START.md](MCP_docs/guides/docs/guides/docs/guides/docs/guides/QUICK_START.md) for a 5-minute setup guide
- **API Reference**: See [API_REFERENCE.md](API_REFERENCE.md) for tool calling API details
- **Building Servers**: See [BUILDING_MCP_SERVERS.md](BUILDING_MCP_SERVERS.md) for custom server development
- **Troubleshooting**: See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for detailed debugging guides

---

## Configuration Examples Repository

For more configuration examples, see:
- [examples/mcp/aws.env](../examples/mcp/aws.env)
- [examples/mcp/github.env](../examples/mcp/github.env)
- [examples/mcp/filesystem.env](../examples/mcp/filesystem.env)
- [examples/mcp/multi-server.env](../examples/mcp/multi-server.env)

---

## Version History

- **v1.0.0** (2024-01): Initial MCP configuration guide
  - Environment variable reference
  - Example configurations
  - Validation rules
  - Troubleshooting guide
