# MCP Integration Analysis for kiro-gateway-go

**Date**: January 25, 2026  
**Purpose**: Analyze current architecture and plan MCP client integration

---

## Current Architecture Overview

### Request Flow

```
HTTP Request (OpenAI format)
    ↓
Handler (routes.go)
    ↓
Auth Middleware (requireAuth)
    ↓
Chat Handler (chat.go)
    ↓
Converter (OpenAI → Q Developer)
    ↓
HTTP Client (client.go)
    ↓
Q Developer API
    ↓
Streaming Parser
    ↓
Converter (Q Developer → OpenAI)
    ↓
HTTP Response (OpenAI format)
```

### Key Components

#### 1. **Handlers** (`internal/handlers/`)
- `routes.go` - Route setup and middleware chain
- `chat.go` - Main chat completion handler
- `models.go` - Model listing
- `health.go` - Health checks
- `middleware.go` - Auth, logging, CORS, rate limiting

#### 2. **Converters** (`internal/converters/`)
- `conversation.go` - OpenAI ↔ Q Developer conversion
- `openai.go` - OpenAI-specific conversions

#### 3. **Models** (`internal/models/`)
- `openai.go` - OpenAI API types
- `conversation.go` - Q Developer API types
- `kiro.go` - Internal types

#### 4. **Client** (`internal/client/`)
- HTTP client for Q Developer API
- Connection pooling
- Retry logic
- Timeout handling

#### 5. **Config** (`internal/config/`)
- Configuration loading
- Environment variables
- Beta features

---

## MCP Integration Points

### 1. Response Parsing (Primary Integration Point)

**Location**: `internal/streaming/streaming.go`

**Current Flow**:
```go
Q Developer Response → Parse Events → Convert to OpenAI → Stream to Client
```

**With MCP**:
```go
Q Developer Response → Parse Events → Detect tool_uses → Execute via MCP → Send results → Continue streaming
```

**Key Insight**: Q Developer returns `tool_uses` in the response stream. We need to:
1. Detect `tool_uses` events
2. Pause streaming
3. Execute tools via MCP
4. Send tool results back to Q Developer
5. Resume streaming

### 2. Request Building (Secondary Integration Point)

**Location**: `internal/converters/conversation.go`

**Current**: Converts OpenAI tools to Q Developer format (but Q Developer ignores them)

**With MCP**: 
- Remove tool definitions from request (Q Developer doesn't use them)
- Tools are discovered from MCP servers
- Tool availability is implicit (MCP servers provide them)

### 3. Configuration (Tertiary Integration Point)

**Location**: `internal/config/config.go`

**Add MCP Configuration**:
```go
type Config struct {
    // ... existing fields ...
    
    // MCP Configuration
    MCP MCPConfig `json:"mcp"`
}

type MCPConfig struct {
    Enabled bool           `json:"enabled"`
    Servers []MCPServer    `json:"servers"`
}

type MCPServer struct {
    Name    string            `json:"name"`
    Type    string            `json:"type"` // "stdio" or "http"
    Command string            `json:"command,omitempty"`
    Args    []string          `json:"args,omitempty"`
    Env     map[string]string `json:"env,omitempty"`
}
```

---

## Detailed Integration Plan

### Phase 1: MCP Manager Integration

**Goal**: Add MCP manager to Handler struct

**Changes**:

1. **Update Handler struct** (`internal/handlers/routes.go`):
```go
type Handler struct {
    authManager     *auth.AuthManager
    config          *config.Config
    priorityQueue   *concurrency.PriorityQueue
    loadShedder     *concurrency.LoadShedder
    asyncJobManager *async.AsyncJobManager
    apiKeyManager   *apikeys.PersistentAPIKeyManager
    validator       *validation.RequestValidator
    rateLimiter     *validation.RateLimiter
    quotaTracker    *validation.QuotaTracker
    client          interface { ... }
    
    // NEW: MCP Manager
    mcpManager      mcp.Manager  // Add this
}
```

2. **Update SetupRoutes** (`internal/handlers/routes.go`):
```go
func SetupRoutes(
    mux *http.ServeMux,
    authManager *auth.AuthManager,
    cfg *config.Config,
    priorityQueue *concurrency.PriorityQueue,
    loadShedder *concurrency.LoadShedder,
    asyncJobManager *async.AsyncJobManager,
    apiKeyManager *apikeys.PersistentAPIKeyManager,
    mcpManager mcp.Manager,  // Add this parameter
    client interface { ... },
) {
    h := &Handler{
        // ... existing fields ...
        mcpManager: mcpManager,  // Add this
    }
    
    // ... rest of setup ...
}
```

3. **Update main.go** (gateway entry point):
```go
func main() {
    // ... existing setup ...
    
    // Initialize MCP manager if enabled
    var mcpManager mcp.Manager
    if cfg.MCP.Enabled {
        mcpManager = mcp.NewManager()
        
        // Connect to configured MCP servers
        for _, serverCfg := range cfg.MCP.Servers {
            if err := mcpManager.AddServer(ctx, &serverCfg); err != nil {
                log.Printf("Failed to connect to MCP server %s: %v", serverCfg.Name, err)
                // Continue with other servers
            }
        }
        
        defer mcpManager.Close()
    }
    
    // Setup routes with MCP manager
    handlers.SetupRoutes(mux, authManager, cfg, priorityQueue, loadShedder, 
        asyncJobManager, apiKeyManager, mcpManager, client)
    
    // ... rest of main ...
}
```

### Phase 2: Tool Use Detection

**Goal**: Detect and parse tool_uses from Q Developer responses

**Changes**:

1. **Add tool use detection** (`internal/streaming/streaming.go`):
```go
// ParseToolUses extracts tool uses from Q Developer response
func ParseToolUses(event *models.ChatResponseStream) []models.ToolUse {
    if event.AssistantResponseMessage != nil && 
       len(event.AssistantResponseMessage.ToolUses) > 0 {
        return event.AssistantResponseMessage.ToolUses
    }
    return nil
}
```

2. **Update streaming handler** (`internal/handlers/chat.go`):
```go
func (h *Handler) handleStreamingWithRequestID(
    w http.ResponseWriter,
    ctx context.Context,
    resp *http.Response,
    model string,
    req *models.ChatCompletionRequest,
    reqCtx *RequestContext,
) {
    // ... existing setup ...
    
    // Convert Kiro stream to OpenAI format
    eventChan, err := streaming.StreamKiroToOpenAI(ctx, resp, model, 
        h.config.FirstTokenTimeout, requestMessages, requestTools)
    if err != nil {
        h.writeErrorWithRequestID(w, http.StatusInternalServerError, 
            "Failed to start streaming", err, reqCtx.RequestID)
        return
    }
    
    log.Printf("[%s] Starting streaming response", reqCtx.RequestID)
    
    // Stream events to client
    for chunk := range eventChan {
        // NEW: Check for tool uses
        if h.mcpManager != nil {
            toolUses := streaming.ParseToolUses(chunk)
            if len(toolUses) > 0 {
                // Execute tools via MCP
                if err := h.executeTools(ctx, toolUses, reqCtx); err != nil {
                    log.Printf("[%s] Tool execution failed: %v", reqCtx.RequestID, err)
                    // Continue streaming (graceful degradation)
                }
            }
        }
        
        fmt.Fprint(w, chunk)
        flusher.Flush()
    }
    
    log.Printf("[%s] Streaming completed", reqCtx.RequestID)
}
```

### Phase 3: Tool Execution

**Goal**: Execute tools via MCP and send results back to Q Developer

**Changes**:

1. **Add tool execution method** (`internal/handlers/chat.go`):
```go
// executeTools executes tools via MCP and sends results back to Q Developer
func (h *Handler) executeTools(
    ctx context.Context,
    toolUses []models.ToolUse,
    reqCtx *RequestContext,
) error {
    for _, toolUse := range toolUses {
        log.Printf("[%s] Executing tool: %s", reqCtx.RequestID, toolUse.Name)
        
        // Parse tool name (@server/tool)
        serverName, toolName, err := mcp.ParseToolName(toolUse.Name)
        if err != nil {
            log.Printf("[%s] Invalid tool name %s: %v", reqCtx.RequestID, toolUse.Name, err)
            continue
        }
        
        // Parse tool input (JSON string → map)
        var toolInput map[string]interface{}
        if err := json.Unmarshal([]byte(toolUse.Input), &toolInput); err != nil {
            log.Printf("[%s] Failed to parse tool input: %v", reqCtx.RequestID, err)
            continue
        }
        
        // Execute tool via MCP with timeout
        toolCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
        defer cancel()
        
        result, err := h.mcpManager.CallTool(toolCtx, serverName, toolName, toolInput)
        if err != nil {
            log.Printf("[%s] Tool %s failed: %v", reqCtx.RequestID, toolUse.Name, err)
            
            // Send error result back to Q Developer
            if err := h.sendToolError(ctx, toolUse.ToolUseID, err, reqCtx); err != nil {
                return fmt.Errorf("failed to send tool error: %w", err)
            }
            continue
        }
        
        // Send tool result back to Q Developer
        if err := h.sendToolResult(ctx, toolUse.ToolUseID, result, reqCtx); err != nil {
            return fmt.Errorf("failed to send tool result: %w", err)
        }
        
        log.Printf("[%s] Tool %s executed successfully", reqCtx.RequestID, toolUse.Name)
    }
    
    return nil
}

// sendToolResult sends tool result back to Q Developer
func (h *Handler) sendToolResult(
    ctx context.Context,
    toolUseID string,
    result *sdk.CallToolResult,
    reqCtx *RequestContext,
) error {
    // Convert MCP result to Q Developer format
    toolResult := models.ToolResult{
        ToolUseID: toolUseID,
        Status:    "success",
        Content:   convertMCPContentToQ(result.Content),
    }
    
    // Build follow-up request with tool result
    followUpReq := &models.ConversationStateRequest{
        ConversationState: &models.ConversationState{
            ConversationID: &reqCtx.ConversationID,
            CurrentMessage: models.ChatMessage{
                UserInputMessage: &models.UserInputMessage{
                    Content:     "", // Empty content, just tool result
                    ToolResults: []models.ToolResult{toolResult},
                },
            },
            ChatTriggerType: "MANUAL",
        },
        ProfileArn: h.authManager.GetProfileArn(),
    }
    
    // Send to Q Developer API
    // ... HTTP client call ...
    
    return nil
}

// sendToolError sends tool error back to Q Developer
func (h *Handler) sendToolError(
    ctx context.Context,
    toolUseID string,
    err error,
    reqCtx *RequestContext,
) error {
    toolResult := models.ToolResult{
        ToolUseID: toolUseID,
        Status:    "error",
        Content: []models.ToolResultContent{
            {Text: fmt.Sprintf("Tool execution failed: %v", err)},
        },
    }
    
    // Build follow-up request with error
    // ... similar to sendToolResult ...
    
    return nil
}

// convertMCPContentToQ converts MCP content to Q Developer format
func convertMCPContentToQ(content []sdk.Content) []models.ToolResultContent {
    var qContent []models.ToolResultContent
    
    for _, item := range content {
        switch c := item.(type) {
        case *sdk.TextContent:
            qContent = append(qContent, models.ToolResultContent{
                Text: c.Text,
            })
        case *sdk.ImageContent:
            // Handle image content if needed
        }
    }
    
    return qContent
}
```

### Phase 4: Configuration Loading

**Goal**: Load MCP configuration from environment/file

**Changes**:

1. **Update Config struct** (`internal/config/config.go`):
```go
type Config struct {
    // ... existing fields ...
    
    // MCP Configuration
    MCP MCPConfig
}

type MCPConfig struct {
    Enabled bool
    Servers []MCPServerConfig
}

type MCPServerConfig struct {
    Name    string
    Type    string // "stdio" or "http"
    Command string
    Args    []string
    Env     map[string]string
}

func Load() *Config {
    cfg := &Config{
        // ... existing fields ...
        MCP: loadMCPConfig(),
    }
    return cfg
}

func loadMCPConfig() MCPConfig {
    // Check if MCP is enabled
    enabled := getBoolEnv("MCP_ENABLED", false)
    if !enabled {
        return MCPConfig{Enabled: false}
    }
    
    // Load MCP configuration from file or environment
    // For now, support environment variables
    
    // Example: MCP_SERVER_AWS_COMMAND=uvx
    //          MCP_SERVER_AWS_ARGS=mcp-server-aws@latest
    
    return MCPConfig{
        Enabled: true,
        Servers: []MCPServerConfig{
            // Load from environment or config file
        },
    }
}
```

2. **Add JSON configuration support**:
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

---

## Integration Challenges & Solutions

### Challenge 1: Streaming Interruption

**Problem**: Need to pause streaming to execute tools

**Solution**: 
- Buffer tool use events
- Execute tools asynchronously
- Resume streaming after tool results are sent

### Challenge 2: Conversation State Management

**Problem**: Need to maintain conversation ID across tool calls

**Solution**:
- Store conversation ID in RequestContext
- Include in follow-up requests with tool results

### Challenge 3: Error Handling

**Problem**: Tool execution may fail

**Solution**:
- Graceful degradation (continue without tool)
- Send error result to Q Developer
- Log errors for debugging

### Challenge 4: Timeout Management

**Problem**: Tool execution may be slow

**Solution**:
- Separate timeout for tool calls (30s)
- Don't block main request timeout
- Cancel tool execution if request times out

---

## Testing Strategy

### Unit Tests

1. **MCP Manager Tests**
   - Test server connection
   - Test tool discovery
   - Test tool execution
   - Test error handling

2. **Tool Use Detection Tests**
   - Test parsing tool_uses from responses
   - Test tool name parsing
   - Test tool input parsing

3. **Integration Tests**
   - Test end-to-end tool calling flow
   - Test with mock MCP server
   - Test error scenarios

### Manual Tests

1. **AWS MCP Server**
   ```bash
   # Configure AWS MCP server
   export MCP_ENABLED=true
   export MCP_SERVER_AWS_COMMAND=uvx
   export MCP_SERVER_AWS_ARGS="mcp-server-aws@latest"
   
   # Start gateway
   ./kiro-gateway
   
   # Test tool calling
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

2. **Multiple Servers**
   - Test with AWS + GitHub servers
   - Verify tool namespacing
   - Test concurrent tool calls

---

## File Structure After Integration

```
internal/
├── adapters/
├── apikeys/
├── async/
├── auth/
├── client/
├── concurrency/
├── config/
│   ├── config.go          # Updated with MCP config
│   └── features.go
├── converters/
│   ├── conversation.go    # Updated for tool results
│   └── openai.go
├── errors/
├── handlers/
│   ├── routes.go          # Updated with MCP manager
│   ├── chat.go            # Updated with tool execution
│   ├── models.go
│   └── middleware.go
├── hotpath/
├── mcp/                   # NEW: MCP client package
│   ├── client.go          # MCP client implementation
│   ├── manager.go         # Multi-server management
│   ├── types.go           # MCP types
│   └── README.md          # Documentation
├── models/
│   ├── openai.go
│   ├── conversation.go    # Updated with tool types
│   └── kiro.go
├── optimization/
├── profiling/
├── storage/
├── streaming/
│   ├── streaming.go       # Updated with tool detection
│   └── parser.go
└── validation/
```

---

## Migration Path

### Phase 1: Foundation (Day 1)
- ✅ Create MCP package structure
- ✅ Implement MCP client
- ✅ Implement MCP manager
- ✅ Add configuration support
- ✅ Write unit tests

### Phase 2: Integration (Day 2)
- [ ] Update Handler struct
- [ ] Add tool use detection
- [ ] Implement tool execution
- [ ] Add tool result sending
- [ ] Write integration tests

### Phase 3: Testing & Polish (Day 3)
- [ ] Manual testing with real MCP servers
- [ ] Error handling improvements
- [ ] Documentation
- [ ] Performance optimization

---

## Success Metrics

### Functional
- ✅ Connect to MCP servers
- ✅ Discover tools
- ✅ Execute tools on Q Developer request
- ✅ Send results back to Q Developer
- ✅ Handle errors gracefully

### Performance
- < 100ms overhead for tool execution
- < 50MB memory per MCP server
- No impact on non-tool requests

### Quality
- 80%+ test coverage
- No race conditions
- Clear error messages
- Comprehensive logging

---

## Conclusion

The MCP integration is **straightforward and well-defined**. The current architecture provides clean integration points:

1. **Handler** - Add MCP manager
2. **Streaming** - Detect tool uses
3. **Chat Handler** - Execute tools
4. **Config** - Load MCP configuration

**Key Advantages**:
- Minimal changes to existing code
- Clean separation of concerns
- Graceful degradation on errors
- Easy to test and debug

**Next Step**: Start Phase 1 implementation.

