# MCP Troubleshooting Guide

Comprehensive guide for diagnosing and resolving MCP (Model Context Protocol) tool calling issues in kiro-gateway-go.

---

## Table of Contents

1. [Quick Diagnostics](#quick-diagnostics)
2. [Common Issues and Solutions](#common-issues-and-solutions)
3. [Log Analysis Guide](#log-analysis-guide)
4. [Connectivity Testing](#connectivity-testing)
5. [Debug Mode Usage](#debug-mode-usage)
6. [Performance Troubleshooting](#performance-troubleshooting)
7. [Error Reference](#error-reference)
8. [Advanced Debugging](#advanced-debugging)

---

## Quick Diagnostics

### 5-Minute Health Check

Run these commands to quickly identify common issues:

```bash
# 1. Check if MCP is enabled
curl http://localhost:8080/health | jq '.mcp.enabled'

# 2. List configured servers
curl http://localhost:8080/health | jq '.mcp.servers'

# 3. List available tools
curl http://localhost:8080/mcp/tools | jq 'length'

# 4. Check gateway logs for errors
tail -100 /var/log/kiro-gateway.log | grep -i error

# 5. Verify MCP server processes are running
ps aux | grep -E "npx|uvx|mcp-server"
```

### Quick Fix Checklist

Before diving into detailed troubleshooting, verify:

- [ ] `MCP_ENABLED=true` is set
- [ ] Server commands exist in PATH (`which npx`, `which uvx`)
- [ ] Required dependencies installed (Node.js, Python, etc.)
- [ ] Credentials configured (AWS, GitHub tokens, etc.)
- [ ] No port conflicts (gateway can bind to configured port)
- [ ] Sufficient disk space and memory
- [ ] Gateway has been restarted after configuration changes

---

## Common Issues and Solutions

### Issue 1: MCP Not Enabled

**Symptoms**:
- Tools endpoint returns empty array
- No MCP servers in health check
- Logs show "MCP disabled"

**Root Cause**: `MCP_ENABLED` environment variable not set to `true`.

**Solution**:
```bash
# Set the environment variable
export MCP_ENABLED=true

# Restart the gateway
./kiro-gateway

# Verify MCP is enabled
curl http://localhost:8080/health | jq '.mcp.enabled'
# Expected: true
```

**Prevention**: Add `MCP_ENABLED=true` to your `.env` file or deployment configuration.

---

### Issue 2: Server Command Not Found

**Symptoms**:
- Error: "executable file not found in $PATH"
- Error: "command not found: npx"
- Server status shows "failed"

**Root Cause**: The command specified in `MCP_SERVER_{NAME}_COMMAND` is not installed or not in PATH.

**Diagnosis**:
```bash
# Check if command exists
which npx
which uvx
which python

# Check PATH
echo $PATH

# Try running command manually
npx --version
uvx --version
```

**Solution**:
```bash
# Option 1: Install missing command
npm install -g npx  # For Node.js
pip install uv      # For Python

# Option 2: Use full path
export MCP_SERVER_AWS_COMMAND=/usr/local/bin/npx

# Option 3: Add to PATH
export PATH=$PATH:/usr/local/bin

# Restart gateway
./kiro-gateway
```

**Prevention**: Document prerequisites in deployment guide and verify in CI/CD.

---

### Issue 3: Server Fails to Start

**Symptoms**:
- Error: "failed to start MCP server"
- Error: "connection timeout"
- Server process exits immediately

**Root Cause**: Server process crashes on startup due to missing dependencies, invalid arguments, or configuration errors.

**Diagnosis**:
```bash
# Enable debug logging
export LOG_LEVEL=debug
./kiro-gateway

# Test server command manually
npx -y @modelcontextprotocol/server-aws

# Check for error output
npx -y @modelcontextprotocol/server-aws 2>&1 | head -20

# Verify server package exists
npm info @modelcontextprotocol/server-aws
```

**Common Causes and Solutions**:

**Cause 1: Invalid Package Name**
```bash
# ❌ Wrong package name
MCP_SERVER_AWS_ARGS=-y,@modelcontextprotocol/aws-server

# ✅ Correct package name
MCP_SERVER_AWS_ARGS=-y,@modelcontextprotocol/server-aws
```

**Cause 2: Missing Dependencies**
```bash
# Install Node.js dependencies
npm install -g @modelcontextprotocol/server-aws

# Install Python dependencies
pip install mcp-server-github
```

**Cause 3: Permission Issues**
```bash
# Check file permissions
ls -la $(which npx)

# Fix permissions if needed
chmod +x /usr/local/bin/npx
```

**Prevention**: Test server commands manually before adding to configuration.

---

### Issue 4: No Tools Discovered

**Symptoms**:
- `/mcp/tools` endpoint returns empty array
- Server connects but no tools available
- Logs show "discovered 0 tools"

**Root Cause**: Server doesn't implement tool listing, returns empty list, or tool discovery fails.

**Diagnosis**:
```bash
# Enable debug logging
export LOG_LEVEL=debug
./kiro-gateway

# Check logs for tool discovery
tail -f /var/log/kiro-gateway.log | grep -i "tool"

# Test server manually and check for tools
npx -y @modelcontextprotocol/server-aws 2>&1 | grep -i tool
```

**Solution**:
```bash
# Option 1: Update to latest server version
MCP_SERVER_AWS_ARGS=-y,@modelcontextprotocol/server-aws@latest

# Option 2: Verify server supports tools
npm info @modelcontextprotocol/server-aws

# Option 3: Check server documentation
# Some servers require specific configuration to expose tools
```

**Prevention**: Verify server version and capabilities before deployment.

---

### Issue 5: Authentication Failures

**Symptoms**:
- Tools execute but return "Unauthorized" or "Access Denied"
- Error: "Invalid credentials"
- Error: "Authentication failed"

**Root Cause**: Missing, invalid, or expired credentials for the service the MCP server accesses.

**Diagnosis by Server Type**:

**AWS Server**:
```bash
# Check AWS credentials
aws sts get-caller-identity

# Check AWS configuration
aws configure list

# Verify region
echo $AWS_REGION

# Test AWS access
aws s3 ls
```

**GitHub Server**:
```bash
# Check GitHub token
echo $GITHUB_TOKEN

# Verify token is valid
curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/user

# Check token scopes
curl -H "Authorization: token $GITHUB_TOKEN" -I https://api.github.com/user | grep x-oauth-scopes
```

**Solution**:

**For AWS**:
```bash
# Option 1: Configure AWS CLI
aws configure

# Option 2: Use environment variables
export AWS_ACCESS_KEY_ID=your_key
export AWS_SECRET_ACCESS_KEY=your_secret
export AWS_REGION=us-east-1

# Option 3: Pass to MCP server
export MCP_SERVER_AWS_ENV=AWS_REGION=us-east-1,AWS_PROFILE=myprofile

# Restart gateway
./kiro-gateway
```

**For GitHub**:
```bash
# Create new token at https://github.com/settings/tokens
# Required scopes: repo, read:org, read:user

# Set token
export GITHUB_TOKEN=ghp_your_token_here

# Pass to MCP server
export MCP_SERVER_GITHUB_ENV=GITHUB_TOKEN=$GITHUB_TOKEN

# Restart gateway
./kiro-gateway
```

**Prevention**: Use secrets management (AWS Secrets Manager, HashiCorp Vault) and rotate credentials regularly.

---

### Issue 6: Tool Execution Timeouts

**Symptoms**:
- Error: "tool execution timed out after 30 seconds"
- Tools work sometimes but timeout other times
- Long-running operations fail

**Root Cause**: Tool operation takes longer than the 30-second timeout.

**Diagnosis**:
```bash
# Enable debug logging to see execution time
export LOG_LEVEL=debug
./kiro-gateway

# Check logs for timing
tail -f /var/log/kiro-gateway.log | grep "execution completed"

# Test tool manually to measure time
time npx -y @modelcontextprotocol/server-aws
```

**Solution**:

**Short-term**: Currently, timeout is hardcoded to 30 seconds. Optimize the operation:
```bash
# For AWS: Use more specific queries
# Instead of: list all S3 buckets
# Use: list buckets with prefix

# For GitHub: Use pagination
# Instead of: get all issues
# Use: get first 10 issues
```

**Long-term**: Future enhancement will add configurable timeout:
```bash
# Future configuration (not yet implemented)
export MCP_TOOL_TIMEOUT=60s
```

**Workaround**: Break long operations into smaller chunks or use async patterns.

**Prevention**: Design tools to complete within 30 seconds or provide progress updates.

---

### Issue 7: Tool Result Too Large

**Symptoms**:
- Error: "tool result exceeds maximum size"
- Large responses truncated
- Memory usage spikes

**Root Cause**: Tool result exceeds 1 MB limit.

**Diagnosis**:
```bash
# Check result size in logs
tail -f /var/log/kiro-gateway.log | grep "result size"

# Test tool manually and check output size
npx -y @modelcontextprotocol/server-aws | wc -c
```

**Solution**:
```bash
# Option 1: Use pagination or filtering
# Request smaller chunks of data

# Option 2: Summarize results
# Return only essential information

# Option 3: Use streaming (future enhancement)
# Stream large results incrementally
```

**Prevention**: Design tools to return concise, focused results.

---

### Issue 8: Multiple Tool Execution Failures

**Symptoms**:
- First tool succeeds, subsequent tools fail
- Error: "tool execution in progress"
- Tools execute out of order

**Root Cause**: Tools are executed sequentially, and a failure in one tool affects subsequent tools.

**Diagnosis**:
```bash
# Enable debug logging
export LOG_LEVEL=debug
./kiro-gateway

# Check execution order in logs
tail -f /var/log/kiro-gateway.log | grep "tool execution"
```

**Solution**:
```bash
# Ensure each tool execution is independent
# Check for shared state or resources

# Verify tool cleanup after execution
# Check logs for resource leaks
```

**Prevention**: Design tools to be stateless and independent.

---

### Issue 9: Server Crashes or Hangs

**Symptoms**:
- Server process exits unexpectedly
- Server stops responding
- Gateway logs show "server disconnected"

**Root Cause**: Server bug, resource exhaustion, or unhandled error.

**Diagnosis**:
```bash
# Check server process status
ps aux | grep mcp-server

# Monitor server resource usage
top -p $(pgrep -f mcp-server)

# Check server logs (if available)
# Server-specific logging varies by implementation

# Enable debug mode
export DEBUG=true
export LOG_LEVEL=debug
./kiro-gateway
```

**Solution**:
```bash
# Option 1: Update server to latest version
MCP_SERVER_AWS_ARGS=-y,@modelcontextprotocol/server-aws@latest

# Option 2: Increase resource limits
ulimit -n 4096  # Increase file descriptors
ulimit -v unlimited  # Remove memory limit

# Option 3: Restart server automatically
# Gateway implements automatic reconnection with backoff

# Option 4: Report bug to server maintainers
# Include logs and reproduction steps
```

**Prevention**: Monitor server health and implement alerting for crashes.

---

### Issue 10: Conversation Context Lost

**Symptoms**:
- Tool results not sent back to Q Developer
- Q Developer doesn't see tool execution results
- Conversation breaks after tool use

**Root Cause**: Conversation context not preserved across tool execution.

**Diagnosis**:
```bash
# Enable debug logging
export LOG_LEVEL=debug
./kiro-gateway

# Check for context preservation in logs
tail -f /var/log/kiro-gateway.log | grep "conversation"

# Verify conversation ID is maintained
tail -f /var/log/kiro-gateway.log | grep "conversationId"
```

**Solution**:
```bash
# Ensure gateway is up to date
# Context preservation is a core feature

# Check for errors in follow-up request
tail -f /var/log/kiro-gateway.log | grep "follow-up"

# Verify Q Developer API is accessible
curl -X POST https://q.us-east-1.amazonaws.com/
```

**Prevention**: Test tool calling end-to-end in development environment.

---

## Log Analysis Guide

### Understanding Log Levels

The gateway uses structured logging with four levels:

```bash
# ERROR: Critical failures requiring immediate attention
ERROR: MCP server 'aws' failed to start: command not found

# WARN: Issues that don't prevent operation but need attention
WARN: MCP server 'github' connection lost, attempting reconnection

# INFO: Normal operational messages
INFO: MCP server 'aws' connected, discovered 15 tools

# DEBUG: Detailed diagnostic information
DEBUG: MCP: Sending tools/list request to server 'aws'
```

### Log Format

Logs follow this structure:
```
[TIMESTAMP] [LEVEL] [COMPONENT] Message [CONTEXT]
```

Example:
```
[2024-01-15T10:30:45Z] [INFO] [MCP] Server 'aws' initialized successfully [server=aws, tools=15, duration=1.2s]
```

### Key Log Patterns

#### Server Initialization

**Successful Initialization**:
```
INFO: Loading MCP configuration from environment
INFO: Found 3 MCP servers: aws, github, filesystem
INFO: Starting MCP server 'aws'
INFO: MCP server 'aws' connected
INFO: Discovered 15 tools from server 'aws'
INFO: MCP server 'aws' initialized successfully
```

**Failed Initialization**:
```
INFO: Loading MCP configuration from environment
INFO: Found 1 MCP server: aws
INFO: Starting MCP server 'aws'
ERROR: Failed to start MCP server 'aws': exec: "npx": executable file not found in $PATH
WARN: MCP server 'aws' marked as unavailable
```

#### Tool Discovery

**Successful Discovery**:
```
DEBUG: MCP: Sending tools/list request to server 'aws'
DEBUG: MCP: Received tools/list response [count=15]
DEBUG: MCP: Namespacing tools with prefix '@aws/'
INFO: Discovered 15 tools from server 'aws': @aws/describe-instance, @aws/list-buckets, ...
```

**Failed Discovery**:
```
DEBUG: MCP: Sending tools/list request to server 'aws'
ERROR: MCP: tools/list request failed: timeout after 5s
WARN: No tools discovered from server 'aws'
```

#### Tool Execution

**Successful Execution**:
```
INFO: Tool execution started [tool=@aws/describe-instance, id=tool_abc123]
DEBUG: MCP: Sending tools/call request [tool=describe-instance, args={"instanceId":"i-123"}]
DEBUG: MCP: Received tools/call response [size=1024 bytes, duration=1.2s]
INFO: Tool execution completed [tool=@aws/describe-instance, id=tool_abc123, duration=1.2s, status=success]
```

**Failed Execution**:
```
INFO: Tool execution started [tool=@aws/describe-instance, id=tool_abc123]
DEBUG: MCP: Sending tools/call request [tool=describe-instance, args={"instanceId":"i-123"}]
ERROR: MCP: tools/call request failed: instance not found
INFO: Tool execution completed [tool=@aws/describe-instance, id=tool_abc123, duration=0.5s, status=error]
```

**Timeout**:
```
INFO: Tool execution started [tool=@aws/slow-operation, id=tool_xyz789]
DEBUG: MCP: Sending tools/call request [tool=slow-operation, args={}]
WARN: Tool execution timeout [tool=@aws/slow-operation, id=tool_xyz789, timeout=30s]
INFO: Tool execution completed [tool=@aws/slow-operation, id=tool_xyz789, duration=30.0s, status=timeout]
```

#### Tool Result Delivery

**Successful Delivery**:
```
INFO: Sending tool results to Q Developer [count=1, conversationId=conv_123]
DEBUG: Building follow-up request [toolResults=1, conversationId=conv_123]
DEBUG: Sending follow-up request to Q Developer API
INFO: Tool results delivered successfully [count=1, duration=0.5s]
```

**Failed Delivery**:
```
INFO: Sending tool results to Q Developer [count=1, conversationId=conv_123]
DEBUG: Building follow-up request [toolResults=1, conversationId=conv_123]
DEBUG: Sending follow-up request to Q Developer API
ERROR: Failed to deliver tool results: network timeout
ERROR: Tool result delivery failed [count=1, error=timeout]
```

#### Server Reconnection

**Successful Reconnection**:
```
WARN: MCP server 'aws' connection lost
INFO: Attempting reconnection to server 'aws' [attempt=1, backoff=1s]
INFO: MCP server 'aws' reconnected successfully
INFO: Rediscovering tools from server 'aws'
INFO: Discovered 15 tools from server 'aws'
```

**Failed Reconnection**:
```
WARN: MCP server 'aws' connection lost
INFO: Attempting reconnection to server 'aws' [attempt=1, backoff=1s]
ERROR: Reconnection failed: connection refused
INFO: Attempting reconnection to server 'aws' [attempt=2, backoff=2s]
ERROR: Reconnection failed: connection refused
INFO: Attempting reconnection to server 'aws' [attempt=3, backoff=4s]
ERROR: Reconnection failed: connection refused
WARN: MCP server 'aws' marked as unavailable after 5 failed attempts
```

### Analyzing Logs for Common Issues

#### Issue: Server Won't Start

**Look for**:
```bash
grep -i "failed to start" /var/log/kiro-gateway.log
grep -i "command not found" /var/log/kiro-gateway.log
grep -i "executable file not found" /var/log/kiro-gateway.log
```

**Indicates**: Command not in PATH or not installed.

#### Issue: Tools Not Discovered

**Look for**:
```bash
grep -i "discovered 0 tools" /var/log/kiro-gateway.log
grep -i "tools/list request failed" /var/log/kiro-gateway.log
grep -i "no tools discovered" /var/log/kiro-gateway.log
```

**Indicates**: Server doesn't support tools or tool discovery failed.

#### Issue: Tool Execution Failures

**Look for**:
```bash
grep -i "tool execution.*failed" /var/log/kiro-gateway.log
grep -i "tool execution.*error" /var/log/kiro-gateway.log
grep -i "tool execution.*timeout" /var/log/kiro-gateway.log
```

**Indicates**: Tool operation failed, check error message for details.

#### Issue: Authentication Problems

**Look for**:
```bash
grep -i "unauthorized" /var/log/kiro-gateway.log
grep -i "access denied" /var/log/kiro-gateway.log
grep -i "authentication failed" /var/log/kiro-gateway.log
grep -i "invalid credentials" /var/log/kiro-gateway.log
```

**Indicates**: Missing or invalid credentials.

#### Issue: Performance Problems

**Look for**:
```bash
grep -i "duration=" /var/log/kiro-gateway.log | awk -F'duration=' '{print $2}' | sort -n
grep -i "timeout" /var/log/kiro-gateway.log
grep -i "slow" /var/log/kiro-gateway.log
```

**Indicates**: Operations taking too long, may need optimization.

### Log Aggregation and Analysis

#### Using grep for Pattern Matching

```bash
# Find all errors
grep ERROR /var/log/kiro-gateway.log

# Find errors for specific server
grep "server='aws'" /var/log/kiro-gateway.log | grep ERROR

# Find tool execution times
grep "Tool execution completed" /var/log/kiro-gateway.log | grep -oP "duration=\K[0-9.]+s"

# Count errors by type
grep ERROR /var/log/kiro-gateway.log | awk '{print $NF}' | sort | uniq -c
```

#### Using jq for Structured Logs

If logs are in JSON format:

```bash
# Parse JSON logs
cat /var/log/kiro-gateway.log | jq -r 'select(.level=="ERROR")'

# Extract specific fields
cat /var/log/kiro-gateway.log | jq -r 'select(.component=="MCP") | {time, level, message}'

# Aggregate by error type
cat /var/log/kiro-gateway.log | jq -r 'select(.level=="ERROR") | .error_type' | sort | uniq -c
```

#### Using awk for Statistics

```bash
# Calculate average tool execution time
grep "Tool execution completed" /var/log/kiro-gateway.log | \
  grep -oP "duration=\K[0-9.]+" | \
  awk '{sum+=$1; count++} END {print "Average:", sum/count, "seconds"}'

# Count tool executions by status
grep "Tool execution completed" /var/log/kiro-gateway.log | \
  grep -oP "status=\K\w+" | \
  sort | uniq -c
```

### Real-Time Log Monitoring

```bash
# Follow logs in real-time
tail -f /var/log/kiro-gateway.log

# Follow only errors
tail -f /var/log/kiro-gateway.log | grep ERROR

# Follow with highlighting
tail -f /var/log/kiro-gateway.log | grep --color=always -E "ERROR|WARN|$"

# Follow multiple patterns
tail -f /var/log/kiro-gateway.log | grep -E "ERROR|tool execution|server.*failed"
```

---

## Connectivity Testing

### Testing MCP Server Connectivity

#### Step 1: Test Command Availability

```bash
# Test if command exists
which npx
which uvx
which python

# Test command execution
npx --version
uvx --version
python --version

# Expected output: Version number
# If "command not found", install the command
```

#### Step 2: Test Server Package

```bash
# Test NPM-based server
npx -y @modelcontextprotocol/server-aws

# Test Python-based server
uvx mcp-server-github@latest

# Expected: Server starts and waits for input
# Press Ctrl+C to exit

# If error, check package name and version
npm info @modelcontextprotocol/server-aws
pip search mcp-server-github
```

#### Step 3: Test Server Communication

```bash
# Create test script to send MCP protocol messages
cat > test-mcp.sh << 'EOF'
#!/bin/bash
# Start server in background
npx -y @modelcontextprotocol/server-aws &
SERVER_PID=$!

# Wait for server to start
sleep 2

# Send initialize request
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | nc localhost 3000

# Clean up
kill $SERVER_PID
EOF

chmod +x test-mcp.sh
./test-mcp.sh

# Expected: JSON response with server capabilities
```

#### Step 4: Test with Gateway

```bash
# Start gateway with debug logging
export LOG_LEVEL=debug
export MCP_ENABLED=true
export MCP_SERVER_AWS_COMMAND=npx
export MCP_SERVER_AWS_ARGS=-y,@modelcontextprotocol/server-aws
./kiro-gateway

# In another terminal, check server status
curl http://localhost:8080/health | jq '.mcp'

# Expected output:
# {
#   "enabled": true,
#   "servers": [
#     {
#       "name": "aws",
#       "status": "connected",
#       "tools": 15
#     }
#   ]
# }
```

### Testing Tool Execution

#### Step 1: List Available Tools

```bash
# Get all tools
curl http://localhost:8080/mcp/tools | jq

# Get tools from specific server
curl http://localhost:8080/mcp/tools | jq '.[] | select(.name | startswith("@aws/"))'

# Count tools per server
curl http://localhost:8080/mcp/tools | jq 'group_by(.serverName) | map({server: .[0].serverName, count: length})'
```

#### Step 2: Test Tool Execution via Chat

```bash
# Send chat request with tool use
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "message": "List my S3 buckets",
    "conversationId": null
  }'

# Expected: Q Developer responds with tool use, gateway executes tool, returns results
```

#### Step 3: Monitor Tool Execution

```bash
# In one terminal, tail logs
tail -f /var/log/kiro-gateway.log | grep -E "tool execution|@aws"

# In another terminal, send chat request
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "Describe EC2 instance i-123456"}'

# Watch logs for:
# - Tool use detection
# - Tool execution start
# - Tool execution completion
# - Tool result delivery
```

### Network Connectivity Tests

#### Test Gateway Accessibility

```bash
# Test health endpoint
curl http://localhost:8080/health

# Test with timeout
curl --max-time 5 http://localhost:8080/health

# Test from remote host
curl http://gateway-host:8080/health
```

#### Test Q Developer API Connectivity

```bash
# Test Q Developer endpoint
curl -X POST https://q.us-east-1.amazonaws.com/ \
  -H "Content-Type: application/x-amz-json-1.0" \
  -H "X-Amz-Target: AmazonQDeveloperStreamingService.SendMessage"

# Expected: 403 (authentication required) or 400 (missing body)
# Not expected: Connection timeout or refused

# Test with authentication
curl -X POST https://q.us-east-1.amazonaws.com/ \
  -H "Content-Type: application/x-amz-json-1.0" \
  -H "X-Amz-Target: AmazonQDeveloperStreamingService.SendMessage" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{}'

# Expected: 400 (invalid request body)
# Indicates API is accessible
```

#### Test External Service Connectivity

```bash
# Test AWS API (for AWS MCP server)
aws sts get-caller-identity

# Test GitHub API (for GitHub MCP server)
curl https://api.github.com/user \
  -H "Authorization: token $GITHUB_TOKEN"

# Test NPM registry (for NPM-based servers)
curl https://registry.npmjs.org/@modelcontextprotocol/server-aws

# Test PyPI (for Python-based servers)
curl https://pypi.org/pypi/mcp-server-github/json
```

### Automated Connectivity Test Script

```bash
#!/bin/bash
# mcp-connectivity-test.sh - Comprehensive connectivity test

echo "=== MCP Connectivity Test ==="
echo

# Test 1: Command availability
echo "Test 1: Command Availability"
for cmd in npx uvx python node; do
  if command -v $cmd &> /dev/null; then
    echo "  ✓ $cmd: $(command -v $cmd)"
  else
    echo "  ✗ $cmd: not found"
  fi
done
echo

# Test 2: Gateway health
echo "Test 2: Gateway Health"
if curl -s --max-time 5 http://localhost:8080/health > /dev/null; then
  echo "  ✓ Gateway is accessible"
  MCP_ENABLED=$(curl -s http://localhost:8080/health | jq -r '.mcp.enabled')
  echo "  MCP Enabled: $MCP_ENABLED"
else
  echo "  ✗ Gateway is not accessible"
fi
echo

# Test 3: MCP servers
echo "Test 3: MCP Servers"
SERVERS=$(curl -s http://localhost:8080/health | jq -r '.mcp.servers[]? | "\(.name): \(.status)"')
if [ -n "$SERVERS" ]; then
  echo "$SERVERS" | while read line; do
    echo "  $line"
  done
else
  echo "  No servers configured"
fi
echo

# Test 4: Available tools
echo "Test 4: Available Tools"
TOOL_COUNT=$(curl -s http://localhost:8080/mcp/tools | jq 'length')
echo "  Total tools: $TOOL_COUNT"
echo

# Test 5: Q Developer API
echo "Test 5: Q Developer API Connectivity"
if curl -s --max-time 5 https://q.us-east-1.amazonaws.com/ > /dev/null 2>&1; then
  echo "  ✓ Q Developer API is accessible"
else
  echo "  ✗ Q Developer API is not accessible"
fi
echo

# Test 6: External services
echo "Test 6: External Service Connectivity"
if aws sts get-caller-identity &> /dev/null; then
  echo "  ✓ AWS API is accessible"
else
  echo "  ✗ AWS API is not accessible"
fi

if curl -s --max-time 5 https://api.github.com > /dev/null; then
  echo "  ✓ GitHub API is accessible"
else
  echo "  ✗ GitHub API is not accessible"
fi
echo

echo "=== Test Complete ==="
```

Save and run:
```bash
chmod +x mcp-connectivity-test.sh
./mcp-connectivity-test.sh
```

---

## Debug Mode Usage

### Enabling Debug Mode

Debug mode provides detailed logging of MCP operations, including full protocol messages.

```bash
# Method 1: Environment variable
export LOG_LEVEL=debug
./kiro-gateway

# Method 2: Command line flag (if supported)
./kiro-gateway --log-level=debug

# Method 3: Configuration file
# Add to config.yaml:
# log_level: debug
```

### Debug Mode Output

With debug mode enabled, you'll see:

**Server Initialization**:
```
DEBUG: MCP: Loading configuration from environment
DEBUG: MCP: Found server 'aws' with command 'npx'
DEBUG: MCP: Starting server process [command=npx, args=[-y, @modelcontextprotocol/server-aws]]
DEBUG: MCP: Server process started [pid=12345]
DEBUG: MCP: Sending initialize request
DEBUG: MCP: Initialize request: {"jsonrpc":"2.0","id":1,"method":"initialize",...}
DEBUG: MCP: Initialize response: {"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05",...}}
DEBUG: MCP: Server 'aws' initialized successfully
```

**Tool Discovery**:
```
DEBUG: MCP: Sending tools/list request to server 'aws'
DEBUG: MCP: tools/list request: {"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
DEBUG: MCP: tools/list response: {"jsonrpc":"2.0","id":2,"result":{"tools":[...]}}
DEBUG: MCP: Received 15 tools from server 'aws'
DEBUG: MCP: Namespacing tool 'describe-instance' as '@aws/describe-instance'
DEBUG: MCP: Tool registry updated [total=15]
```

**Tool Execution**:
```
DEBUG: MCP: Tool use detected [tool=@aws/describe-instance, id=tool_abc123]
DEBUG: MCP: Routing tool to server 'aws'
DEBUG: MCP: Sending tools/call request
DEBUG: MCP: tools/call request: {"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"describe-instance","arguments":{"instanceId":"i-123"}}}
DEBUG: MCP: tools/call response: {"jsonrpc":"2.0","id":3,"result":{"content":[{"type":"text","text":"..."}]}}
DEBUG: MCP: Tool execution completed [duration=1.2s, size=1024 bytes]
```

**Tool Result Delivery**:
```
DEBUG: MCP: Building follow-up request with tool results
DEBUG: MCP: Follow-up request: {"conversationState":{"currentMessage":{"userInputMessage":{"content":"...","toolResults":[...]}}}}
DEBUG: MCP: Sending follow-up request to Q Developer
DEBUG: MCP: Follow-up response received [status=200]
DEBUG: MCP: Tool results delivered successfully
```

### Filtering Debug Output

Debug mode can be verbose. Filter for specific information:

```bash
# Only MCP-related debug messages
./kiro-gateway 2>&1 | grep "DEBUG: MCP"

# Only tool execution
./kiro-gateway 2>&1 | grep "tool execution"

# Only errors and warnings
./kiro-gateway 2>&1 | grep -E "ERROR|WARN"

# Save to file for analysis
./kiro-gateway 2>&1 | tee debug.log
```

### Debug Mode Performance Impact

Debug mode has minimal performance impact:
- **CPU**: <5% increase
- **Memory**: ~10MB for log buffering
- **Latency**: <10ms per request

**Recommendation**: Safe to use in production for troubleshooting, but disable when not needed.

### Advanced Debug Techniques

#### Trace Individual Requests

```bash
# Add request ID to logs
export DEBUG_REQUEST_ID=true

# Logs will include request ID
DEBUG: [req-abc123] MCP: Tool execution started
DEBUG: [req-abc123] MCP: Tool execution completed
```

#### Protocol Message Inspection

```bash
# Enable full protocol message logging
export DEBUG_MCP_PROTOCOL=true

# Logs will include full JSON-RPC messages
DEBUG: MCP: >>> {"jsonrpc":"2.0","id":1,"method":"initialize",...}
DEBUG: MCP: <<< {"jsonrpc":"2.0","id":1,"result":{...}}
```

#### Performance Profiling

```bash
# Enable timing information
export DEBUG_TIMING=true

# Logs will include detailed timing
DEBUG: MCP: Server initialization [total=1.2s, connect=0.5s, discover=0.7s]
DEBUG: MCP: Tool execution [total=1.5s, call=1.2s, format=0.3s]
```

---

## Performance Troubleshooting

### Identifying Performance Issues

#### Symptom: High Latency

**Diagnosis**:
```bash
# Enable timing logs
export LOG_LEVEL=debug
./kiro-gateway

# Monitor execution times
tail -f /var/log/kiro-gateway.log | grep "duration="

# Calculate statistics
grep "duration=" /var/log/kiro-gateway.log | \
  grep -oP "duration=\K[0-9.]+" | \
  awk '{sum+=$1; count++; if($1>max) max=$1} END {print "Avg:", sum/count, "Max:", max}'
```

**Common Causes**:
1. **Slow tool execution**: Tool takes >5s to complete
2. **Network latency**: High latency to Q Developer API or external services
3. **Resource contention**: CPU or memory exhaustion

**Solutions**:
```bash
# Option 1: Optimize tool operations
# Use more specific queries, pagination, caching

# Option 2: Increase resources
# Add more CPU, memory, or network bandwidth

# Option 3: Use async patterns
# Execute tools asynchronously where possible
```

#### Symptom: High Memory Usage

**Diagnosis**:
```bash
# Monitor memory usage
ps aux | grep kiro-gateway | awk '{print $6}'

# Monitor over time
while true; do
  ps aux | grep kiro-gateway | awk '{print $6}'
  sleep 5
done

# Check for memory leaks
# Memory should stabilize after initial spike
```

**Common Causes**:
1. **Large tool results**: Results >1MB not being cleaned up
2. **Conversation context accumulation**: Old contexts not being cleaned up
3. **Connection leaks**: MCP server connections not being closed

**Solutions**:
```bash
# Option 1: Reduce result size
# Implement pagination or filtering in tools

# Option 2: Increase cleanup frequency
export CONTEXT_CLEANUP_INTERVAL=1m

# Option 3: Restart gateway periodically
# Implement health checks and automatic restart
```

#### Symptom: High CPU Usage

**Diagnosis**:
```bash
# Monitor CPU usage
top -p $(pgrep kiro-gateway)

# Profile CPU usage
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30

# Identify hot paths
go tool pprof -top http://localhost:8080/debug/pprof/profile?seconds=30
```

**Common Causes**:
1. **JSON parsing**: Large payloads being parsed repeatedly
2. **Regex matching**: Complex patterns in log filtering
3. **Concurrent tool execution**: Too many tools executing simultaneously

**Solutions**:
```bash
# Option 1: Optimize JSON parsing
# Use streaming parsers for large payloads

# Option 2: Limit concurrency
export MCP_MAX_CONCURRENT_TOOLS=5

# Option 3: Add caching
# Cache tool discovery results, parsed configurations
```

### Performance Benchmarking

#### Baseline Performance Test

```bash
# Test without MCP
export MCP_ENABLED=false
./kiro-gateway &
GATEWAY_PID=$!

# Send 100 requests
for i in {1..100}; do
  curl -X POST http://localhost:8080/chat \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"message": "Hello"}' \
    -w "%{time_total}\n" -o /dev/null -s
done | awk '{sum+=$1; count++} END {print "Average:", sum/count, "seconds"}'

kill $GATEWAY_PID
```

#### MCP Performance Test

```bash
# Test with MCP enabled
export MCP_ENABLED=true
export MCP_SERVER_AWS_COMMAND=npx
export MCP_SERVER_AWS_ARGS=-y,@modelcontextprotocol/server-aws
./kiro-gateway &
GATEWAY_PID=$!

# Wait for initialization
sleep 5

# Send 100 requests (no tool use)
for i in {1..100}; do
  curl -X POST http://localhost:8080/chat \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"message": "Hello"}' \
    -w "%{time_total}\n" -o /dev/null -s
done | awk '{sum+=$1; count++} END {print "Average:", sum/count, "seconds"}'

kill $GATEWAY_PID

# Expected: <100ms overhead compared to baseline
```

#### Tool Execution Performance Test

```bash
# Test tool execution latency
export MCP_ENABLED=true
export MCP_SERVER_AWS_COMMAND=npx
export MCP_SERVER_AWS_ARGS=-y,@modelcontextprotocol/server-aws
export LOG_LEVEL=debug
./kiro-gateway &
GATEWAY_PID=$!

# Wait for initialization
sleep 5

# Send request with tool use
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "List my S3 buckets"}' \
  -w "Total time: %{time_total}s\n"

# Check logs for breakdown
grep "Tool execution completed" /var/log/kiro-gateway.log | tail -1

kill $GATEWAY_PID
```

### Performance Optimization Tips

1. **Enable Caching**:
   - Cache tool discovery results (refreshed on reconnection)
   - Cache parsed configurations
   - Cache authentication tokens

2. **Optimize Tool Operations**:
   - Use pagination for large result sets
   - Implement filtering at the source
   - Return only essential data

3. **Resource Limits**:
   - Set maximum tool result size (1MB default)
   - Limit concurrent tool executions
   - Set appropriate timeouts

4. **Connection Pooling**:
   - Reuse MCP server connections
   - Implement connection health checks
   - Close idle connections

5. **Monitoring and Alerting**:
   - Monitor latency metrics
   - Alert on high error rates
   - Track resource usage trends

---

## Error Reference

### Configuration Errors

#### ERROR: MCP_ENABLED not set

**Full Message**: `WARN: MCP servers configured but MCP_ENABLED is not set to true. MCP functionality will be disabled.`

**Cause**: Server configurations exist but MCP is not enabled.

**Solution**:
```bash
export MCP_ENABLED=true
```

#### ERROR: Duplicate server name

**Full Message**: `ERROR: Duplicate MCP server name 'aws'. Server names must be unique.`

**Cause**: Multiple servers configured with the same name (case-insensitive).

**Solution**:
```bash
# Rename one of the servers
export MCP_SERVER_AWS_COMMAND=npx
export MCP_SERVER_AWS2_COMMAND=uvx  # Different name
```

#### ERROR: Server has no command

**Full Message**: `ERROR: MCP server 'aws' has no command specified. Set MCP_SERVER_AWS_COMMAND.`

**Cause**: Server configured with args but no command.

**Solution**:
```bash
export MCP_SERVER_AWS_COMMAND=npx
export MCP_SERVER_AWS_ARGS=-y,@modelcontextprotocol/server-aws
```

#### ERROR: Invalid environment variable format

**Full Message**: `WARN: Invalid environment variable format in MCP_SERVER_AWS_ENV: 'AWS_REGION' (expected KEY=VALUE)`

**Cause**: Environment variable not in KEY=VALUE format.

**Solution**:
```bash
# ❌ Wrong
export MCP_SERVER_AWS_ENV=AWS_REGION

# ✅ Correct
export MCP_SERVER_AWS_ENV=AWS_REGION=us-east-1
```

### Connection Errors

#### ERROR: Command not found

**Full Message**: `ERROR: MCP server 'aws' command 'npx' not found in PATH`

**Cause**: Command executable not installed or not in PATH.

**Solution**:
```bash
# Install command
npm install -g npx

# Or use full path
export MCP_SERVER_AWS_COMMAND=/usr/local/bin/npx
```

#### ERROR: Failed to start server

**Full Message**: `ERROR: Failed to start MCP server 'aws': exit status 1`

**Cause**: Server process failed to start (invalid args, missing dependencies, etc.).

**Solution**:
```bash
# Test command manually
npx -y @modelcontextprotocol/server-aws

# Check for errors
npx -y @modelcontextprotocol/server-aws 2>&1 | head -20

# Verify package exists
npm info @modelcontextprotocol/server-aws
```

#### ERROR: Connection timeout

**Full Message**: `ERROR: MCP server 'aws' connection timeout after 30s`

**Cause**: Server started but didn't respond to initialize request.

**Solution**:
```bash
# Check if server is hanging
ps aux | grep mcp-server

# Kill and restart
pkill -f mcp-server
./kiro-gateway
```

#### ERROR: Server disconnected

**Full Message**: `WARN: MCP server 'aws' connection lost`

**Cause**: Server process crashed or was killed.

**Solution**:
```bash
# Gateway will attempt automatic reconnection
# Check logs for reconnection status
tail -f /var/log/kiro-gateway.log | grep reconnect

# If reconnection fails, check server logs
# Server-specific logging varies by implementation
```

### Tool Discovery Errors

#### ERROR: Tool discovery failed

**Full Message**: `ERROR: MCP: tools/list request failed: timeout after 5s`

**Cause**: Server didn't respond to tools/list request.

**Solution**:
```bash
# Enable debug logging
export LOG_LEVEL=debug
./kiro-gateway

# Check if server supports tool listing
# Some servers may not implement this feature

# Try updating server to latest version
export MCP_SERVER_AWS_ARGS=-y,@modelcontextprotocol/server-aws@latest
```

#### WARN: No tools discovered

**Full Message**: `WARN: No tools discovered from server 'aws'`

**Cause**: Server returned empty tool list.

**Solution**:
```bash
# Verify server is configured correctly
# Some servers require specific configuration to expose tools

# Check server documentation
npm info @modelcontextprotocol/server-aws

# Test server manually
npx -y @modelcontextprotocol/server-aws
```

#### ERROR: Invalid tool schema

**Full Message**: `WARN: Invalid tool schema for tool 'describe-instance', skipping`

**Cause**: Tool schema doesn't match expected format.

**Solution**:
```bash
# Update server to latest version
export MCP_SERVER_AWS_ARGS=-y,@modelcontextprotocol/server-aws@latest

# Report issue to server maintainers if persists
```

### Tool Execution Errors

#### ERROR: Tool not found

**Full Message**: `ERROR: Tool '@aws/unknown-tool' not found in registry`

**Cause**: Q Developer requested a tool that doesn't exist.

**Solution**:
```bash
# List available tools
curl http://localhost:8080/mcp/tools | jq '.[].name'

# Verify tool name is correct
# Tool names are case-sensitive and must include namespace
```

#### ERROR: Tool execution timeout

**Full Message**: `WARN: Tool execution timeout [tool=@aws/slow-operation, timeout=30s]`

**Cause**: Tool operation took longer than 30 seconds.

**Solution**:
```bash
# Currently timeout is hardcoded to 30s
# Optimize tool operation to complete faster

# Future: Configurable timeout
# export MCP_TOOL_TIMEOUT=60s
```

#### ERROR: Tool execution failed

**Full Message**: `ERROR: MCP: tools/call request failed: instance not found`

**Cause**: Tool operation failed (invalid arguments, resource not found, etc.).

**Solution**:
```bash
# Check error message for details
# Error is returned from the MCP server

# Verify arguments are correct
# Check tool documentation for required parameters

# Test tool manually
npx -y @modelcontextprotocol/server-aws
# Then send tools/call request manually
```

#### ERROR: Invalid tool arguments

**Full Message**: `ERROR: Invalid arguments for tool '@aws/describe-instance': missing required field 'instanceId'`

**Cause**: Tool called with missing or invalid arguments.

**Solution**:
```bash
# Check tool schema for required arguments
curl http://localhost:8080/mcp/tools | jq '.[] | select(.name=="@aws/describe-instance") | .inputSchema'

# Verify Q Developer is sending correct arguments
# Enable debug logging to see actual arguments
export LOG_LEVEL=debug
```

### Tool Result Errors

#### ERROR: Tool result too large

**Full Message**: `ERROR: Tool result exceeds maximum size of 1MB`

**Cause**: Tool returned result larger than 1MB limit.

**Solution**:
```bash
# Implement pagination in tool
# Return smaller chunks of data

# Or implement filtering
# Return only essential information
```

#### ERROR: Tool result delivery failed

**Full Message**: `ERROR: Failed to deliver tool results: network timeout`

**Cause**: Follow-up request to Q Developer failed.

**Solution**:
```bash
# Check Q Developer API connectivity
curl -X POST https://q.us-east-1.amazonaws.com/

# Check network connectivity
ping q.us-east-1.amazonaws.com

# Verify authentication is valid
# Token may have expired
```

#### ERROR: Invalid tool result format

**Full Message**: `ERROR: Invalid tool result format: missing 'toolUseId' field`

**Cause**: Tool result doesn't match expected format.

**Solution**:
```bash
# This is a gateway bug, not a configuration issue
# Report to gateway maintainers with logs
```

### Authentication Errors

#### ERROR: AWS credentials not found

**Full Message**: `ERROR: Tool execution failed: Unable to locate credentials`

**Cause**: AWS credentials not configured for AWS MCP server.

**Solution**:
```bash
# Configure AWS credentials
aws configure

# Or use environment variables
export AWS_ACCESS_KEY_ID=your_key
export AWS_SECRET_ACCESS_KEY=your_secret
export AWS_REGION=us-east-1

# Pass to MCP server
export MCP_SERVER_AWS_ENV=AWS_REGION=us-east-1
```

#### ERROR: GitHub token invalid

**Full Message**: `ERROR: Tool execution failed: Bad credentials`

**Cause**: GitHub token is invalid or expired.

**Solution**:
```bash
# Create new token at https://github.com/settings/tokens
# Required scopes: repo, read:org, read:user

# Set token
export GITHUB_TOKEN=ghp_your_new_token

# Pass to MCP server
export MCP_SERVER_GITHUB_ENV=GITHUB_TOKEN=$GITHUB_TOKEN
```

#### ERROR: Insufficient permissions

**Full Message**: `ERROR: Tool execution failed: Access denied`

**Cause**: Credentials don't have required permissions.

**Solution**:
```bash
# For AWS: Check IAM permissions
aws iam get-user
aws iam list-attached-user-policies --user-name your-user

# For GitHub: Check token scopes
curl -H "Authorization: token $GITHUB_TOKEN" -I https://api.github.com/user | grep x-oauth-scopes

# Grant required permissions
# AWS: Attach appropriate IAM policies
# GitHub: Create new token with required scopes
```

---

## Advanced Debugging

### Debugging with Packet Capture

Capture network traffic to debug communication issues:

```bash
# Capture traffic to Q Developer API
sudo tcpdump -i any -w qdev-traffic.pcap host q.us-east-1.amazonaws.com

# Analyze captured traffic
tcpdump -r qdev-traffic.pcap -A | less

# Or use Wireshark for GUI analysis
wireshark qdev-traffic.pcap
```

### Debugging MCP Protocol

#### Manual MCP Server Testing

Test MCP server manually to isolate issues:

```bash
# Start server
npx -y @modelcontextprotocol/server-aws &
SERVER_PID=$!

# Send initialize request
cat > init.json << 'EOF'
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {},
    "clientInfo": {
      "name": "test",
      "version": "1.0"
    }
  }
}
EOF

# Send via stdin
cat init.json | npx -y @modelcontextprotocol/server-aws

# Send tools/list request
cat > list.json << 'EOF'
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list",
  "params": {}
}
EOF

cat list.json | npx -y @modelcontextprotocol/server-aws

# Clean up
kill $SERVER_PID
```

#### Intercepting MCP Communication

Use a proxy to intercept MCP protocol messages:

```bash
# Create proxy script
cat > mcp-proxy.py << 'EOF'
#!/usr/bin/env python3
import sys
import json

# Read from stdin, write to stdout
# Log all messages to stderr
for line in sys.stdin:
    try:
        msg = json.loads(line)
        print(f">>> {json.dumps(msg, indent=2)}", file=sys.stderr)
        print(line)
        sys.stdout.flush()
    except:
        pass
EOF

chmod +x mcp-proxy.py

# Use proxy in server command
export MCP_SERVER_AWS_COMMAND=./mcp-proxy.py
export MCP_SERVER_AWS_ARGS=npx,-y,@modelcontextprotocol/server-aws
```

### Debugging with Go Profiling

Profile the gateway to identify performance bottlenecks:

```bash
# Enable pprof endpoint (if not already enabled)
# Add to main.go:
# import _ "net/http/pprof"
# go func() { http.ListenAndServe("localhost:6060", nil) }()

# CPU profiling
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Memory profiling
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine profiling
go tool pprof http://localhost:6060/debug/pprof/goroutine

# Generate flame graph
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/profile?seconds=30
```

### Debugging with Distributed Tracing

Add distributed tracing to track requests across components:

```bash
# Using OpenTelemetry (example)
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
export OTEL_SERVICE_NAME=kiro-gateway
export OTEL_TRACES_SAMPLER=always_on

# Start Jaeger for trace visualization
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 4318:4318 \
  jaegertracing/all-in-one:latest

# View traces at http://localhost:16686
```

### Debugging Race Conditions

Detect race conditions in concurrent code:

```bash
# Build with race detector
go build -race -o kiro-gateway-race ./cmd/kiro-gateway

# Run with race detector
./kiro-gateway-race

# Race detector will panic if race is detected
# Example output:
# WARNING: DATA RACE
# Write at 0x00c000100000 by goroutine 7:
#   main.(*MCPManager).ExecuteTool()
# Previous read at 0x00c000100000 by goroutine 6:
#   main.(*MCPManager).GetTools()
```

### Debugging Memory Leaks

Identify memory leaks using heap profiling:

```bash
# Take initial heap snapshot
curl http://localhost:6060/debug/pprof/heap > heap-before.prof

# Run workload (send many requests)
for i in {1..1000}; do
  curl -X POST http://localhost:8080/chat \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"message": "Hello"}'
done

# Take final heap snapshot
curl http://localhost:6060/debug/pprof/heap > heap-after.prof

# Compare snapshots
go tool pprof -base heap-before.prof heap-after.prof

# Look for growing allocations
# (pprof) top
# (pprof) list <function-name>
```

### Debugging Deadlocks

Detect deadlocks using goroutine profiling:

```bash
# If gateway appears hung, check goroutines
curl http://localhost:6060/debug/pprof/goroutine?debug=2 > goroutines.txt

# Look for blocked goroutines
grep -A 10 "goroutine.*blocked" goroutines.txt

# Common deadlock patterns:
# - Multiple goroutines waiting on same mutex
# - Circular wait on channels
# - Goroutine waiting on itself
```

### Debugging with strace/dtrace

Trace system calls to debug low-level issues:

```bash
# Linux: Use strace
strace -f -e trace=network,process ./kiro-gateway 2>&1 | tee strace.log

# macOS: Use dtrace
sudo dtrace -n 'syscall:::entry /execname == "kiro-gateway"/ { @[probefunc] = count(); }'

# Look for:
# - Failed system calls (return -1)
# - Excessive system calls (performance issue)
# - Unexpected system calls (bug)
```

### Creating Minimal Reproduction

When reporting bugs, create minimal reproduction:

```bash
# 1. Minimal configuration
cat > minimal.env << 'EOF'
MCP_ENABLED=true
MCP_SERVER_TEST_COMMAND=npx
MCP_SERVER_TEST_ARGS=-y,@modelcontextprotocol/server-aws
LOG_LEVEL=debug
EOF

# 2. Minimal test case
cat > test-case.sh << 'EOF'
#!/bin/bash
source minimal.env
./kiro-gateway &
GATEWAY_PID=$!
sleep 5

# Send request that triggers bug
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "List S3 buckets"}'

kill $GATEWAY_PID
EOF

chmod +x test-case.sh
./test-case.sh

# 3. Collect logs
tail -100 /var/log/kiro-gateway.log > bug-report-logs.txt

# 4. Report with:
# - minimal.env
# - test-case.sh
# - bug-report-logs.txt
# - Expected vs actual behavior
```

### Emergency Debugging Checklist

When production is down and you need to debug quickly:

1. **Check if gateway is running**:
   ```bash
   ps aux | grep kiro-gateway
   curl http://localhost:8080/health
   ```

2. **Check recent errors**:
   ```bash
   tail -100 /var/log/kiro-gateway.log | grep ERROR
   ```

3. **Check MCP server status**:
   ```bash
   curl http://localhost:8080/health | jq '.mcp.servers'
   ps aux | grep mcp-server
   ```

4. **Check resource usage**:
   ```bash
   top -p $(pgrep kiro-gateway)
   df -h
   free -m
   ```

5. **Check network connectivity**:
   ```bash
   curl https://q.us-east-1.amazonaws.com/
   ping q.us-east-1.amazonaws.com
   ```

6. **Restart with debug logging**:
   ```bash
   pkill kiro-gateway
   export LOG_LEVEL=debug
   ./kiro-gateway &
   tail -f /var/log/kiro-gateway.log
   ```

7. **Test basic functionality**:
   ```bash
   curl http://localhost:8080/health
   curl http://localhost:8080/mcp/tools
   ```

8. **Collect diagnostic information**:
   ```bash
   # Save for later analysis
   curl http://localhost:8080/health > health.json
   curl http://localhost:8080/mcp/tools > tools.json
   tail -1000 /var/log/kiro-gateway.log > recent-logs.txt
   ps aux > processes.txt
   env > environment.txt
   ```

---

## Getting Help

### Before Asking for Help

Collect this information:

1. **Configuration**:
   ```bash
   env | grep MCP > mcp-config.txt
   ```

2. **Logs**:
   ```bash
   tail -500 /var/log/kiro-gateway.log > logs.txt
   ```

3. **System Information**:
   ```bash
   uname -a > system-info.txt
   go version >> system-info.txt
   node --version >> system-info.txt
   python --version >> system-info.txt
   ```

4. **Gateway Version**:
   ```bash
   ./kiro-gateway --version > version.txt
   ```

5. **Health Status**:
   ```bash
   curl http://localhost:8080/health > health.json
   ```

6. **Reproduction Steps**:
   - What you did
   - What you expected
   - What actually happened

### Where to Get Help

1. **Documentation**:
   - [MCP Configuration Guide](MCP_CONFIGURATION_GUIDE.md)
   - [API Reference](API_REFERENCE.md)
   - [Architecture Overview](ARCHITECTURE.md)

2. **Community**:
   - GitHub Issues: Report bugs and feature requests
   - Discussions: Ask questions and share solutions
   - Stack Overflow: Tag with `kiro-gateway` and `mcp`

3. **Support**:
   - Enterprise Support: Contact your support representative
   - Professional Services: For implementation assistance

### Reporting Bugs

When reporting bugs, include:

1. **Title**: Brief description of the issue
2. **Environment**: OS, Go version, Node.js version
3. **Configuration**: Relevant environment variables (redact secrets)
4. **Steps to Reproduce**: Minimal steps to trigger the bug
5. **Expected Behavior**: What should happen
6. **Actual Behavior**: What actually happens
7. **Logs**: Relevant log excerpts (with debug logging enabled)
8. **Workarounds**: Any temporary solutions you've found

**Example Bug Report**:
```markdown
## Bug: MCP server fails to start with npx

**Environment**:
- OS: Ubuntu 22.04
- Go: 1.21.0
- Node.js: 18.17.0
- Gateway: v1.0.0

**Configuration**:
```bash
MCP_ENABLED=true
MCP_SERVER_AWS_COMMAND=npx
MCP_SERVER_AWS_ARGS=-y,@modelcontextprotocol/server-aws
```

**Steps to Reproduce**:
1. Set environment variables as above
2. Start gateway: `./kiro-gateway`
3. Check logs

**Expected**: Server starts successfully
**Actual**: Error "command not found: npx"

**Logs**:
```
ERROR: Failed to start MCP server 'aws': exec: "npx": executable file not found in $PATH
```

**Workaround**: Using full path works:
```bash
MCP_SERVER_AWS_COMMAND=/usr/local/bin/npx
```
```

---

## Appendix

### Useful Commands Reference

```bash
# Health check
curl http://localhost:8080/health | jq

# List tools
curl http://localhost:8080/mcp/tools | jq

# Check logs
tail -f /var/log/kiro-gateway.log

# Enable debug mode
export LOG_LEVEL=debug

# Test server command
npx -y @modelcontextprotocol/server-aws

# Check processes
ps aux | grep -E "kiro-gateway|mcp-server"

# Monitor resources
top -p $(pgrep kiro-gateway)

# Test connectivity
curl https://q.us-east-1.amazonaws.com/

# Restart gateway
pkill kiro-gateway && ./kiro-gateway &
```

### Environment Variables Quick Reference

```bash
# Core
MCP_ENABLED=true                    # Enable MCP
LOG_LEVEL=debug                     # Logging level

# Server configuration
MCP_SERVER_{NAME}_COMMAND=npx       # Server command
MCP_SERVER_{NAME}_ARGS=-y,package   # Server arguments
MCP_SERVER_{NAME}_ENV=KEY=VALUE     # Server environment

# Performance
CONTEXT_CLEANUP_INTERVAL=5m         # Context cleanup frequency
MCP_TOOL_TIMEOUT=30s                # Tool execution timeout (future)
MCP_MAX_CONCURRENT_TOOLS=5          # Concurrent tool limit (future)
```

### Common Log Patterns

```bash
# Server initialization
grep "MCP server.*initialized" /var/log/kiro-gateway.log

# Tool discovery
grep "Discovered.*tools" /var/log/kiro-gateway.log

# Tool execution
grep "Tool execution" /var/log/kiro-gateway.log

# Errors
grep ERROR /var/log/kiro-gateway.log

# Performance
grep "duration=" /var/log/kiro-gateway.log
```

---

## Next Steps

- **Configuration**: See [MCP_CONFIGURATION_GUIDE.md](MCP_CONFIGURATION_GUIDE.md)
- **API Reference**: See [API_REFERENCE.md](API_REFERENCE.md)
- **Architecture**: See [ARCHITECTURE.md](ARCHITECTURE.md)
- **Development**: See [DEVELOPMENT_GUIDE.md](DEVELOPMENT_GUIDE.md)

---

## Version History

- **v1.0.0** (2024-01): Initial MCP troubleshooting guide
  - Common issues and solutions
  - Log analysis guide
  - Connectivity testing procedures
  - Debug mode usage
  - Performance troubleshooting
  - Error reference
  - Advanced debugging techniques

