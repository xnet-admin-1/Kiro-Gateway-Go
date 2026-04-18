# MCP Go Client Exploration for kiro-gateway-go

**Date**: January 25, 2026  
**Purpose**: Explore implementing a Go-based MCP client to enable Q Developer tool calling

---

## Executive Summary

After analyzing the MCP specification, official Go SDK, and Q CLI implementation, we have a clear path to implement MCP support in kiro-gateway-go. The official `modelcontextprotocol/go-sdk` provides all the primitives we need, and we can follow Q CLI's architecture as a reference.

**Key Finding**: We can implement MCP client support in **2-3 days** using the official Go SDK.

---

## MCP Protocol Overview

### What is MCP?

Model Context Protocol (MCP) is an open protocol that enables LLM applications to integrate with external tools and data sources using JSON-RPC 2.0.

**Key Components:**
- **Hosts**: LLM applications (our kiro-gateway)
- **Clients**: Connectors within the host (our MCP client)
- **Servers**: Services providing tools (MCP servers)

### Protocol Flow

```
1. Client connects to MCP server (stdio or HTTP)
2. Client calls initialize() - handshake
3. Client calls list_tools() - discover available tools
4. User sends message to Q Developer
5. Q Developer responds with tool_uses
6. Client calls call_tool() on MCP server
7. Client sends tool result back to Q Developer
8. Q Developer provides final response
```

### JSON-RPC Methods

**Client → Server:**
- `initialize` - Handshake and capability negotiation
- `tools/list` - Discover available tools (paginated)
- `tools/call` - Invoke a tool
- `prompts/list` - List available prompts
- `prompts/get` - Get a specific prompt

**Server → Client (Notifications):**
- `notifications/tools/list_changed` - Tool list updated
- `notifications/prompts/list_changed` - Prompt list updated
- `notifications/message/logging` - Server log messages

---

## Official Go SDK Analysis

### Package Structure

```
github.com/modelcontextprotocol/go-sdk/
├── mcp/              # Core MCP types and client/server
├── jsonschema/       # JSON Schema for tool definitions
└── examples/         # Example implementations
```

### Key Types

```go
// Client for connecting to MCP servers
type Client struct {
    implementation *Implementation
    features       *ClientFeatures
}

// Server session after connection
type Session struct {
    // Methods for interacting with server
}

// Tool definition
type Tool struct {
    Name        string
    Description string
    InputSchema *jsonschema.Schema
}

// Tool call parameters
type CallToolParams struct {
    Name      string
    Arguments map[string]any
}

// Tool call result
type CallToolResult struct {
    Content []Content
    IsError bool
}
```

### Transport Types

**1. Stdio Transport** (for external processes):
```go
transport := &mcp.CommandTransport{
    Command: exec.Command("uvx", "mcp-server-aws"),
}
```

**2. HTTP Transport** (for remote servers):
```go
// Not directly provided, but can be implemented
// using standard HTTP client
```

### Client Usage Pattern

```go
// 1. Create client
client := mcp.NewClient(&mcp.Implementation{
    Name:    "kiro-gateway",
    Version: "v1.0.0",
}, nil)

// 2. Connect to server
transport := &mcp.CommandTransport{
    Command: exec.Command("uvx", "mcp-server-aws"),
}
session, err := client.Connect(ctx, transport, nil)
if err != nil {
    return err
}
defer session.Close()

// 3. List tools
tools, err := session.ListTools(ctx, nil)
if err != nil {
    return err
}

// 4. Call tool
result, err := session.CallTool(ctx, &mcp.CallToolParams{
    Name:      "@aws/describe-instance",
    Arguments: map[string]any{"instanceId": "i-1234567890abcdef0"},
})
if err != nil {
    return err
}
```

---

## Q CLI Implementation Analysis

### Architecture

Q CLI uses the `rmcp` Rust crate (Rust MCP implementation) with the following structure:

```
mcp_client/
├── client.rs       # MCP client with auth retry logic
├── messenger.rs    # Communication with chat loop
├── oauth_util.rs   # OAuth token management
└── mod.rs          # Module exports
```

### Key Features

**1. Transport Support:**
- Stdio (external processes)
- HTTP (with OAuth support)

**2. Authentication:**
- OAuth token refresh on expiry
- SigV4 signing for AWS MCP servers
- Environment variable substitution in headers

**3. Tool Discovery:**
- Paginated tool fetching
- Automatic refresh on `tools/list_changed` notification
- Lazy initialization (spawn task, resolve when needed)

**4. Error Handling:**
- Retry logic for auth failures
- Graceful degradation on server errors
- Stderr logging from child processes

### Configuration Format

```rust
pub struct CustomToolConfig {
    pub r#type: TransportType,           // Stdio or Http
    pub url: String,                      // HTTP endpoint
    pub headers: HashMap<String, String>, // HTTP headers
    pub oauth_scopes: Vec<String>,        // OAuth scopes
    pub command: String,                  // Stdio command
    pub args: Vec<String>,                // Stdio args
    pub env: Option<HashMap<String, String>>, // Environment vars
    pub timeout: u64,                     // Request timeout
}
```

### Tool Namespacing

Q CLI namespaces tools by server:
```
@server_name/tool_name
```

Example:
```
@aws/describe-instance
@github/create-issue
@filesystem/read-file
```

---

## Implementation Plan for kiro-gateway-go

### Phase 1: Core MCP Client (Day 1)

**Goal**: Basic MCP client with stdio transport

**Tasks:**
1. Add Go SDK dependency
2. Create `internal/mcp/` package structure
3. Implement client wrapper
4. Add stdio transport support
5. Implement tool discovery

**Files to Create:**
```
internal/mcp/
├── client.go       # MCP client wrapper
├── config.go       # Configuration types
├── transport.go    # Transport implementations
├── manager.go      # Multi-server management
└── types.go        # MCP-specific types
```

**Code Structure:**

```go
// internal/mcp/client.go
package mcp

import (
    "context"
    "os/exec"
    
    sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type Client struct {
    serverName string
    config     *ServerConfig
    session    *sdk.Session
    tools      []*sdk.Tool
}

func NewClient(serverName string, config *ServerConfig) *Client {
    return &Client{
        serverName: serverName,
        config:     config,
    }
}

func (c *Client) Connect(ctx context.Context) error {
    // Create SDK client
    client := sdk.NewClient(&sdk.Implementation{
        Name:    "kiro-gateway",
        Version: "v1.0.0",
    }, nil)
    
    // Create transport based on config
    var transport sdk.Transport
    switch c.config.Type {
    case TransportStdio:
        transport = &sdk.CommandTransport{
            Command: exec.Command(c.config.Command, c.config.Args...),
        }
    case TransportHTTP:
        // TODO: Implement HTTP transport
        return fmt.Errorf("HTTP transport not yet implemented")
    }
    
    // Connect to server
    session, err := client.Connect(ctx, transport, nil)
    if err != nil {
        return fmt.Errorf("failed to connect: %w", err)
    }
    
    c.session = session
    return nil
}

func (c *Client) ListTools(ctx context.Context) ([]*sdk.Tool, error) {
    if c.session == nil {
        return nil, fmt.Errorf("not connected")
    }
    
    // List tools with pagination
    var allTools []*sdk.Tool
    var cursor *string
    
    for {
        result, err := c.session.ListTools(ctx, &sdk.ListToolsParams{
            Cursor: cursor,
        })
        if err != nil {
            return nil, err
        }
        
        allTools = append(allTools, result.Tools...)
        
        if result.NextCursor == nil {
            break
        }
        cursor = result.NextCursor
    }
    
    c.tools = allTools
    return allTools, nil
}

func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (*sdk.CallToolResult, error) {
    if c.session == nil {
        return nil, fmt.Errorf("not connected")
    }
    
    return c.session.CallTool(ctx, &sdk.CallToolParams{
        Name:      name,
        Arguments: args,
    })
}

func (c *Client) Close() error {
    if c.session != nil {
        return c.session.Close()
    }
    return nil
}
```

```go
// internal/mcp/config.go
package mcp

type TransportType string

const (
    TransportStdio TransportType = "stdio"
    TransportHTTP  TransportType = "http"
)

type ServerConfig struct {
    Name    string                 `json:"name"`
    Type    TransportType          `json:"type"`
    Command string                 `json:"command,omitempty"`
    Args    []string               `json:"args,omitempty"`
    Env     map[string]string      `json:"env,omitempty"`
    URL     string                 `json:"url,omitempty"`
    Headers map[string]string      `json:"headers,omitempty"`
    Timeout int                    `json:"timeout,omitempty"`
}

type Config struct {
    Servers []ServerConfig `json:"servers"`
}
```

```go
// internal/mcp/manager.go
package mcp

import (
    "context"
    "fmt"
    "sync"
)

type Manager struct {
    clients map[string]*Client
    mu      sync.RWMutex
}

func NewManager() *Manager {
    return &Manager{
        clients: make(map[string]*Client),
    }
}

func (m *Manager) AddServer(ctx context.Context, config *ServerConfig) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    if _, exists := m.clients[config.Name]; exists {
        return fmt.Errorf("server %s already exists", config.Name)
    }
    
    client := NewClient(config.Name, config)
    if err := client.Connect(ctx); err != nil {
        return fmt.Errorf("failed to connect to %s: %w", config.Name, err)
    }
    
    // Discover tools
    if _, err := client.ListTools(ctx); err != nil {
        client.Close()
        return fmt.Errorf("failed to list tools for %s: %w", config.Name, err)
    }
    
    m.clients[config.Name] = client
    return nil
}

func (m *Manager) GetAllTools() map[string][]*Tool {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    tools := make(map[string][]*Tool)
    for serverName, client := range m.clients {
        tools[serverName] = client.tools
    }
    return tools
}

func (m *Manager) CallTool(ctx context.Context, serverName, toolName string, args map[string]any) (*CallToolResult, error) {
    m.mu.RLock()
    client, exists := m.clients[serverName]
    m.mu.RUnlock()
    
    if !exists {
        return nil, fmt.Errorf("server %s not found", serverName)
    }
    
    return client.CallTool(ctx, toolName, args)
}

func (m *Manager) Close() error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    for _, client := range m.clients {
        client.Close()
    }
    m.clients = make(map[string]*Client)
    return nil
}
```

### Phase 2: Q Developer Integration (Day 2)

**Goal**: Integrate MCP with Q Developer chat flow

**Tasks:**
1. Parse `tool_uses` from Q Developer responses
2. Map tool calls to MCP servers
3. Execute tools via MCP client
4. Format tool results for Q Developer
5. Handle tool errors gracefully

**Integration Points:**

```go
// internal/handlers/chat.go

func (h *ChatHandler) handleToolUses(ctx context.Context, toolUses []ToolUse, conversationID string) error {
    for _, toolUse := range toolUses {
        // Parse tool name: @server/tool
        serverName, toolName, err := parseToolName(toolUse.Name)
        if err != nil {
            return err
        }
        
        // Call tool via MCP
        result, err := h.mcpManager.CallTool(ctx, serverName, toolName, toolUse.Input)
        if err != nil {
            // Send error result back to Q Developer
            return h.sendToolError(ctx, conversationID, toolUse.ID, err)
        }
        
        // Send tool result back to Q Developer
        return h.sendToolResult(ctx, conversationID, toolUse.ID, result)
    }
    return nil
}

func parseToolName(fullName string) (server, tool string, err error) {
    // Parse @server/tool format
    if !strings.HasPrefix(fullName, "@") {
        return "", "", fmt.Errorf("invalid tool name format: %s", fullName)
    }
    
    parts := strings.SplitN(fullName[1:], "/", 2)
    if len(parts) != 2 {
        return "", "", fmt.Errorf("invalid tool name format: %s", fullName)
    }
    
    return parts[0], parts[1], nil
}
```

### Phase 3: Configuration & Testing (Day 3)

**Goal**: Configuration system and end-to-end testing

**Tasks:**
1. Add MCP configuration to gateway config
2. Implement configuration loading
3. Add health checks for MCP servers
4. Write integration tests
5. Document MCP setup

**Configuration Format:**

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
      },
      {
        "name": "github",
        "type": "stdio",
        "command": "uvx",
        "args": ["mcp-server-github"],
        "env": {
          "GITHUB_TOKEN": "${GITHUB_TOKEN}"
        }
      }
    ]
  }
}
```

---

## Advantages of Official Go SDK

### 1. Production-Ready

- Maintained by MCP team + Google
- Follows MCP spec exactly
- Regular updates with spec changes
- Well-tested and documented

### 2. Complete Feature Set

- ✅ Client and server implementations
- ✅ Stdio and custom transports
- ✅ JSON Schema support
- ✅ Pagination handling
- ✅ Error handling
- ✅ Context support

### 3. Type Safety

```go
// Strongly typed tool definitions
type Tool struct {
    Name        string
    Description string
    InputSchema *jsonschema.Schema
}

// Type-safe tool calls
result, err := session.CallTool(ctx, &CallToolParams{
    Name:      "tool_name",
    Arguments: map[string]any{"key": "value"},
})
```

### 4. Easy Integration

```go
// Just 3 steps:
client := mcp.NewClient(impl, nil)
session, _ := client.Connect(ctx, transport, nil)
result, _ := session.CallTool(ctx, params)
```

---

## Comparison with Alternatives

### Option 1: Official Go SDK (Recommended)

**Pros:**
- Official implementation
- Complete feature set
- Well-maintained
- Type-safe
- 2-3 days implementation

**Cons:**
- None significant

### Option 2: Third-Party Libraries

**Examples:**
- `mark3labs/mcp-go`
- `yincongcyincong/mcp-client-go`
- `Convict3d/mcp-go`

**Pros:**
- May have additional features
- Different API styles

**Cons:**
- Not official
- Less maintained
- May lag behind spec
- Smaller community

### Option 3: Build from Scratch

**Pros:**
- Full control
- Custom optimizations

**Cons:**
- 10-15 days development
- Need to implement JSON-RPC
- Need to track MCP spec changes
- More maintenance burden

---

## Security Considerations

### 1. Process Execution

**Risk**: Stdio transport spawns external processes

**Mitigation:**
- Whitelist allowed MCP servers
- Validate command paths
- Use absolute paths
- Restrict environment variables

```go
var allowedServers = map[string]bool{
    "mcp-server-aws":    true,
    "mcp-server-github": true,
    "mcp-server-filesystem": true,
}

func validateServerCommand(command string) error {
    base := filepath.Base(command)
    if !allowedServers[base] {
        return fmt.Errorf("server not allowed: %s", base)
    }
    return nil
}
```

### 2. Tool Execution

**Risk**: Tools can access filesystem, network, etc.

**Mitigation:**
- User consent for tool execution
- Rate limiting
- Audit logging
- Sandboxing (future)

```go
type ToolExecutionPolicy struct {
    RequireConsent bool
    MaxCallsPerMin int
    AllowedTools   []string
}

func (p *ToolExecutionPolicy) CanExecute(toolName string) bool {
    if len(p.AllowedTools) == 0 {
        return true // No restrictions
    }
    for _, allowed := range p.AllowedTools {
        if toolName == allowed {
            return true
        }
    }
    return false
}
```

### 3. Data Privacy

**Risk**: Tools may access sensitive data

**Mitigation:**
- Clear documentation of tool capabilities
- User control over tool access
- Data minimization
- Encryption in transit

---

## Testing Strategy

### Unit Tests

```go
func TestMCPClient_Connect(t *testing.T) {
    config := &ServerConfig{
        Name:    "test-server",
        Type:    TransportStdio,
        Command: "echo-server",
    }
    
    client := NewClient("test", config)
    err := client.Connect(context.Background())
    assert.NoError(t, err)
    defer client.Close()
}

func TestMCPClient_ListTools(t *testing.T) {
    // Test tool discovery
}

func TestMCPClient_CallTool(t *testing.T) {
    // Test tool execution
}
```

### Integration Tests

```go
func TestMCPIntegration_EndToEnd(t *testing.T) {
    // 1. Start gateway with MCP enabled
    // 2. Connect to test MCP server
    // 3. Send Q Developer request with tool use
    // 4. Verify tool is called
    // 5. Verify result is returned
}
```

### Manual Testing

1. **AWS MCP Server**
   ```bash
   # Start gateway with AWS MCP server
   ./kiro-gateway --config config-with-mcp.json
   
   # Send request to Q Developer
   curl -X POST http://localhost:8080/v1/chat/completions \
     -H "Content-Type: application/json" \
     -d '{
       "model": "claude-sonnet-4-5",
       "messages": [{"role": "user", "content": "Describe EC2 instance i-1234567890abcdef0"}]
     }'
   ```

2. **GitHub MCP Server**
   ```bash
   # Test GitHub tool calling
   ```

3. **Filesystem MCP Server**
   ```bash
   # Test file operations
   ```

---

## Documentation Plan

### User Documentation

**1. MCP Setup Guide** (`docs/MCP_SETUP.md`)
- What is MCP
- How to configure MCP servers
- Available MCP servers
- Troubleshooting

**2. MCP Server Configuration** (`docs/MCP_CONFIGURATION.md`)
- Configuration format
- Environment variables
- Security settings
- Examples

**3. Tool Calling Guide** (`docs/TOOL_CALLING.md`)
- How tool calling works
- Q Developer integration
- Tool namespacing
- Error handling

### Developer Documentation

**1. MCP Architecture** (`docs/MCP_ARCHITECTURE.md`)
- System design
- Component interaction
- Data flow
- Extension points

**2. Adding MCP Servers** (`docs/ADDING_MCP_SERVERS.md`)
- How to add new servers
- Configuration options
- Testing
- Best practices

---

## Migration Path

### For Existing Users

**No Breaking Changes:**
- MCP is opt-in via configuration
- Existing functionality unchanged
- OpenAI/Anthropic tool calling still works

**Enabling MCP:**
```json
{
  "mcp": {
    "enabled": true,
    "servers": [...]
  }
}
```

### For Q Developer Users

**Before MCP:**
- Tool calling not available
- Direct responses only

**After MCP:**
- Full tool calling support
- Access to MCP ecosystem
- Enhanced capabilities

---

## Timeline & Milestones

### Day 1: Core Implementation
- ✅ Add Go SDK dependency
- ✅ Create MCP package structure
- ✅ Implement client wrapper
- ✅ Add stdio transport
- ✅ Implement tool discovery
- ✅ Write unit tests

### Day 2: Integration
- ✅ Parse Q Developer tool_uses
- ✅ Map tools to MCP servers
- ✅ Execute tools via MCP
- ✅ Format tool results
- ✅ Handle errors
- ✅ Write integration tests

### Day 3: Polish & Documentation
- ✅ Configuration system
- ✅ Health checks
- ✅ End-to-end testing
- ✅ User documentation
- ✅ Developer documentation
- ✅ Examples

---

## Success Criteria

### Functional Requirements

- ✅ Connect to MCP servers (stdio)
- ✅ Discover tools from servers
- ✅ Execute tools on demand
- ✅ Return results to Q Developer
- ✅ Handle errors gracefully
- ✅ Support multiple servers
- ✅ Tool namespacing (@server/tool)

### Non-Functional Requirements

- ✅ < 100ms overhead per tool call
- ✅ Graceful degradation on server failure
- ✅ Clear error messages
- ✅ Comprehensive logging
- ✅ Memory efficient (< 50MB per server)
- ✅ Thread-safe operations

### Quality Requirements

- ✅ 80%+ test coverage
- ✅ No race conditions
- ✅ Clear documentation
- ✅ Example configurations
- ✅ Troubleshooting guide

---

## Risks & Mitigation

### Risk 1: MCP Server Stability

**Risk**: External MCP servers may crash or hang

**Mitigation:**
- Timeout on all operations
- Health checks
- Automatic restart
- Fallback to direct responses

```go
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()

result, err := client.CallTool(ctx, params)
if err != nil {
    // Log error and continue without tool
    log.Error("Tool call failed", "error", err)
    return nil // Graceful degradation
}
```

### Risk 2: Performance Impact

**Risk**: Tool calls add latency

**Mitigation:**
- Async tool execution
- Caching tool results
- Parallel tool calls
- Streaming responses

### Risk 3: Security Vulnerabilities

**Risk**: Malicious MCP servers

**Mitigation:**
- Server whitelisting
- Sandboxing (future)
- Audit logging
- User consent

---

## Future Enhancements

### Phase 4: HTTP Transport (Week 2)

- Implement HTTP transport
- Add OAuth support
- SigV4 signing for AWS
- Connection pooling

### Phase 5: Advanced Features (Week 3)

- Tool result caching
- Parallel tool execution
- Tool chaining
- Custom tool definitions

### Phase 6: Monitoring & Observability (Week 4)

- Prometheus metrics
- Tool call tracing
- Performance dashboards
- Alert system

---

## Conclusion

Implementing MCP support in kiro-gateway-go is **feasible and straightforward** using the official Go SDK. The 2-3 day timeline is realistic, and the implementation will enable full Q Developer tool calling capabilities.

**Key Advantages:**
- Official SDK (production-ready)
- Clean architecture
- Type-safe implementation
- Easy maintenance
- Extensible design

**Next Steps:**
1. Review this exploration with team
2. Get approval for implementation
3. Start Phase 1 (Core Implementation)
4. Iterate based on testing feedback

**Recommendation**: Proceed with implementation using official Go SDK.

---

## References

- MCP Specification: https://modelcontextprotocol.io/specification/2024-11-05
- Official Go SDK: https://github.com/modelcontextprotocol/go-sdk
- Q CLI Source: `C:\Users\xnet-admin\Repos\amazon-q-developer-cli`
- Tool Calling Analysis: `docs/architecture/TOOL_CALLING_COMPLETE_ANALYSIS.md`
- MCP Servers: https://github.com/modelcontextprotocol/servers

