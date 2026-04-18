package conversation

import (
	"sync"
	"time"

	"github.com/yourusername/kiro-gateway-go/internal/auth"
	"github.com/yourusername/kiro-gateway-go/internal/models"
)

// Context maintains state for multi-turn conversations with AWS Q Developer.
// This tracks conversation IDs, auth context, and request correlation across
// multiple requests in the same conversation thread.
//
// Note: This is NOT related to MCP tool execution. Tool execution is now
// handled client-side for security. This only tracks conversation state.
type Context struct {
	// ConversationID is the Q Developer conversation ID for follow-up requests
	// This is captured from the MessageMetadataEvent in the streaming response
	ConversationID string

	// UserMessage is the original user message content
	// Preserved for context in follow-up requests
	UserMessage string

	// AuthContext contains authentication information needed for follow-up requests
	AuthContext AuthContext

	// ToolUses tracks tool use requests detected in responses (for logging only)
	// Clients handle actual tool execution
	ToolUses []models.ToolUse

	// ToolResults tracks tool results sent by clients (for logging only)
	ToolResults []models.ToolResult

	// OriginalRequest stores the original ConversationStateRequest
	// Used to preserve context (model ID, etc.) in follow-up requests
	OriginalRequest *models.ConversationStateRequest

	// RequestID is the unique identifier for this request
	// Used for logging and tracing
	RequestID string

	// CreatedAt is the timestamp when this context was created
	CreatedAt time.Time

	// mu protects concurrent access to the context
	mu sync.RWMutex
}

// AuthContext contains authentication information for making follow-up requests.
// This encapsulates all auth-related data needed to authenticate with Q Developer API.
type AuthContext struct {
	// AuthMode indicates the authentication method (Bearer token or SigV4)
	AuthMode auth.AuthMode

	// ProfileArn is the Identity Center profile ARN (for SigV4 mode)
	ProfileArn string

	// Region is the AWS region for the Q Developer API
	Region string

	// BearerToken is the authentication token (for Bearer token mode)
	// Note: This should be handled securely and not logged
	BearerToken string
}

// NewContext creates a new conversation context.
func NewContext(requestID string, userMessage string, authCtx AuthContext, originalReq *models.ConversationStateRequest) *Context {
	return &Context{
		ConversationID:  "", // Will be set when captured from response
		UserMessage:     userMessage,
		AuthContext:     authCtx,
		ToolUses:        make([]models.ToolUse, 0),
		ToolResults:     make([]models.ToolResult, 0),
		OriginalRequest: originalReq,
		RequestID:       requestID,
		CreatedAt:       time.Now(),
	}
}

// SetConversationID sets the conversation ID (thread-safe).
// This is called when the conversation ID is captured from the streaming response.
func (c *Context) SetConversationID(conversationID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ConversationID = conversationID
}

// GetConversationID gets the conversation ID (thread-safe).
func (c *Context) GetConversationID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ConversationID
}

// AddToolUse adds a tool use to the context (thread-safe).
// Tool uses are accumulated as they are detected in the streaming response.
// Note: This is for logging only - clients handle actual tool execution.
func (c *Context) AddToolUse(toolUse models.ToolUse) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ToolUses = append(c.ToolUses, toolUse)
}

// GetToolUses returns all accumulated tool uses (thread-safe).
func (c *Context) GetToolUses() []models.ToolUse {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// Return a copy to prevent external modification
	toolUses := make([]models.ToolUse, len(c.ToolUses))
	copy(toolUses, c.ToolUses)
	return toolUses
}

// AddToolResult adds a tool result to the context (thread-safe).
// Tool results are accumulated as they are received from clients.
// Note: This is for logging only - clients execute tools and send results.
func (c *Context) AddToolResult(toolResult models.ToolResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ToolResults = append(c.ToolResults, toolResult)
}

// GetToolResults returns all accumulated tool results (thread-safe).
func (c *Context) GetToolResults() []models.ToolResult {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// Return a copy to prevent external modification
	toolResults := make([]models.ToolResult, len(c.ToolResults))
	copy(toolResults, c.ToolResults)
	return toolResults
}

// HasToolUses returns true if there are any tool uses detected (thread-safe).
func (c *Context) HasToolUses() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.ToolUses) > 0
}

// HasToolResults returns true if there are any tool results received (thread-safe).
func (c *Context) HasToolResults() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.ToolResults) > 0
}

// Clear clears all accumulated tool uses and results (thread-safe).
// This should be called after a conversation completes.
func (c *Context) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ToolUses = make([]models.ToolUse, 0)
	c.ToolResults = make([]models.ToolResult, 0)
}

// Age returns the age of this context since creation.
// This can be used for cleanup of stale contexts.
func (c *Context) Age() time.Duration {
	return time.Since(c.CreatedAt)
}

// Clone creates a deep copy of the conversation context (thread-safe).
// This is useful when you need to pass the context to a goroutine.
func (c *Context) Clone() *Context {
	c.mu.RLock()
	defer c.mu.RUnlock()

	clone := &Context{
		ConversationID:  c.ConversationID,
		UserMessage:     c.UserMessage,
		AuthContext:     c.AuthContext, // AuthContext is a value type, so this is a copy
		OriginalRequest: c.OriginalRequest, // Pointer is shared, but request is immutable
		RequestID:       c.RequestID,
		CreatedAt:       c.CreatedAt,
	}

	// Deep copy slices
	clone.ToolUses = make([]models.ToolUse, len(c.ToolUses))
	copy(clone.ToolUses, c.ToolUses)

	clone.ToolResults = make([]models.ToolResult, len(c.ToolResults))
	copy(clone.ToolResults, c.ToolResults)

	return clone
}

// Manager manages multiple conversation contexts.
// This is useful for tracking contexts across multiple concurrent requests.
type Manager struct {
	contexts map[string]*Context
	mu       sync.RWMutex
}

// NewManager creates a new conversation context manager.
func NewManager() *Manager {
	return &Manager{
		contexts: make(map[string]*Context),
	}
}

// Add adds a conversation context to the manager.
func (m *Manager) Add(ctx *Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.contexts[ctx.RequestID] = ctx
}

// Get retrieves a conversation context by request ID.
func (m *Manager) Get(requestID string) (*Context, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ctx, exists := m.contexts[requestID]
	return ctx, exists
}

// Remove removes a conversation context from the manager.
func (m *Manager) Remove(requestID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.contexts, requestID)
}

// CleanupStale removes contexts older than the specified duration.
// This should be called periodically to prevent memory leaks.
func (m *Manager) CleanupStale(maxAge time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	removed := 0
	for requestID, ctx := range m.contexts {
		if ctx.Age() > maxAge {
			delete(m.contexts, requestID)
			removed++
		}
	}

	return removed
}

// Count returns the number of active contexts.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.contexts)
}
