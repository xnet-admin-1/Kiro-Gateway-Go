# MCP Server Reconnection Implementation

## Overview

This document describes the implementation of automatic reconnection with exponential backoff for MCP (Model Context Protocol) servers in kiro-gateway-go.

## Requirements

**Requirement 1.6**: When an MCP server connection is lost, the gateway shall attempt reconnection with exponential backoff.

## Implementation

### Exponential Backoff Algorithm

The backoff algorithm implements the following sequence:
- **Failure 1**: 1 second
- **Failure 2**: 2 seconds
- **Failure 3**: 4 seconds
- **Failure 4**: 8 seconds
- **Failure 5**: 16 seconds
- **Failure 6**: 32 seconds
- **Failure 7+**: 60 seconds (capped)

The formula is: `backoff = 2^(failureCount-1) seconds`, capped at 60 seconds maximum.

### Connection States

Each MCP client tracks its connection state:

```go
type ConnectionState int

const (
    StateDisconnected    // Not connected
    StateConnecting      // Connection attempt in progress
    StateConnected       // Connected and ready
    StateReconnecting    // Waiting to reconnect after failure
)
```

### Client-Level Tracking

Each `mcpClient` maintains:
- **state**: Current connection state
- **failureCount**: Number of consecutive connection failures
- **lastFailureTime**: Timestamp of the last failure
- **nextRetryTime**: When the next reconnection attempt should occur

### Manager-Level Reconnection Loop

The `manager` runs a background goroutine that:
1. Checks all clients every 5 seconds
2. Identifies clients in `StateReconnecting` that are ready to retry
3. Attempts reconnection for those clients
4. Rediscovers tools after successful reconnection

### Graceful Failure Handling

When a server fails to connect initially:
1. The client is added to the manager in `StateReconnecting`
2. The failure is logged with the retry schedule
3. Other servers continue to initialize normally
4. The background loop will attempt reconnection automatically

### API Changes

#### Client Interface

New methods added to the `Client` interface:

```go
// Reconnect attempts to reconnect if enough time has passed
Reconnect(ctx context.Context) error

// GetConnectionState returns the current connection state
GetConnectionState() ConnectionState

// GetFailureCount returns the number of consecutive connection failures
GetFailureCount() int

// GetNextRetryTime returns when the next reconnection attempt should occur
GetNextRetryTime() time.Time

// ShouldRetry checks if enough time has passed to attempt reconnection
ShouldRetry() bool
```

#### Manager Interface

New methods added to the `Manager` interface:

```go
// GetServer returns a specific MCP client by name
GetServer(serverName string) (Client, bool)

// ListServers returns the names of all connected servers
ListServers() []string
```

### Tool Execution with Disconnected Servers

When `CallTool` is invoked on a disconnected server:
1. The manager checks the connection state
2. Returns an error if the server is not in `StateConnected`
3. The error message indicates the current state

This prevents tool execution attempts on disconnected servers while they're waiting to reconnect.

## Testing

### Unit Tests

**`reconnect_test.go`**:
- `TestCalculateBackoff`: Verifies the exponential backoff algorithm
- `TestBackoffSequence`: Verifies the complete backoff sequence
- `TestConnectionStateTracking`: Verifies state transitions
- `TestFailureCountIncrement`: Verifies failure counting and retry scheduling
- `TestShouldRetry`: Verifies retry timing logic
- `TestFailureCountReset`: Verifies failure count resets on success

**`manager_reconnect_test.go`**:
- `TestManagerReconnectionLoop`: Verifies the background loop starts and stops
- `TestManagerHandlesInitialConnectionFailure`: Verifies graceful handling of initial failures
- `TestManagerCallToolWithDisconnectedServer`: Verifies tool calls fail appropriately
- `TestManagerGetAllToolsWithDisconnectedServer`: Verifies tool discovery with disconnected servers
- `TestClientReconnectBeforeRetryTime`: Verifies early reconnection attempts are rejected

All tests pass successfully.

## Usage Example

### Configuration

```bash
MCP_ENABLED=true
MCP_SERVER_AWS_COMMAND=npx
MCP_SERVER_AWS_ARGS=-y,@modelcontextprotocol/server-aws
```

### Behavior

1. **Initial Connection Failure**:
   ```
   MCP: Server aws connection failed (attempt 1), will retry in 1s
   MCP: Initial connection to server aws failed: exec: "npx": executable file not found
   MCP: Server aws will attempt reconnection with exponential backoff
   ```

2. **Automatic Reconnection**:
   ```
   MCP: Attempting to reconnect to server aws (attempt 2)
   MCP: Server aws connection failed (attempt 2), will retry in 2s
   ```

3. **Successful Reconnection**:
   ```
   MCP: Successfully reconnected to server aws
   MCP: Rediscovered 15 tools for server aws
   ```

4. **Tool Call on Disconnected Server**:
   ```
   Error: server aws is not connected (state: StateReconnecting)
   ```

## Benefits

1. **Resilience**: Servers automatically recover from transient failures
2. **No Manual Intervention**: Reconnection happens automatically in the background
3. **Resource Friendly**: Exponential backoff prevents overwhelming failed servers
4. **Graceful Degradation**: Other servers continue working while one is reconnecting
5. **Transparent**: Clear logging shows reconnection attempts and status

## Future Enhancements

Potential improvements for future iterations:

1. **Circuit Breaker**: Stop attempting reconnection after N consecutive failures
2. **Configurable Backoff**: Allow customization of backoff parameters
3. **Health Checks**: Periodic health checks for connected servers
4. **Metrics**: Expose reconnection metrics for monitoring
5. **Notifications**: Alert when servers are down for extended periods

## Files Modified

- `internal/mcp/client.go`: Added connection state tracking and reconnection logic
- `internal/mcp/manager.go`: Added background reconnection loop
- `internal/mcp/types.go`: Added `ConnectionState` type and updated interfaces
- `internal/mcp/reconnect_test.go`: Unit tests for reconnection logic
- `internal/mcp/manager_reconnect_test.go`: Integration tests for manager reconnection

## Compliance

This implementation satisfies:
- **Requirement 1.6**: Automatic reconnection with exponential backoff
- **Design Property**: Server isolation (failures don't affect other servers)
- **Design Property**: Graceful degradation (system continues with available servers)
