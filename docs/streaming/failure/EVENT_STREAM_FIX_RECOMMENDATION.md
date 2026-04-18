# Event Stream Timeout Fix Recommendation

## Problem

The current event stream parser (`internal/streaming/parser.go`) times out when processing multimodal requests because it reads the entire stream synchronously without respecting context timeouts.

## Root Cause

```go
func parseEventStreamBinary(ctx context.Context, resp *http.Response) (<-chan StreamEvent, error) {
	eventChan := make(chan StreamEvent, 10)
	
	go func() {
		defer close(eventChan)
		defer resp.Body.Close()
		
		// ❌ This blocks until entire stream is read
		content, err := ParseEventStreamResponse(resp.Body)
		// ...
	}()
	
	return eventChan, nil
}
```

The `ParseEventStreamResponse` function uses `io.ReadFull` which blocks indefinitely, ignoring the context.

## Solution: Incremental Event-Based Parsing

Based on AWS SDK Go v2's implementation, we should parse and emit events incrementally as they arrive, not wait for the entire stream.

### Reference Implementation

AWS SDK Go v2 handles streaming like this:

```go
output, err := client.InvokeModelWithResponseStream(ctx, &bedrockruntime.InvokeModelWithResponseStreamInput{
	Body:        body,
	ModelId:     aws.String(modelId),
	ContentType: aws.String("application/json"),
})

// Process events as they arrive
for event := range output.GetStream().Events() {
	switch v := event.(type) {
	case *types.ResponseStreamMemberChunk:
		var resp Response
		err := json.NewDecoder(bytes.NewReader(v.Value.Bytes)).Decode(&resp)
		if err != nil {
			return resp, err
		}
		// Process chunk immediately
		handler(ctx, []byte(resp.Completion))
	}
}
```

### Recommended Fix for Our Code

**File**: `internal/streaming/parser.go`

```go
func parseEventStreamBinary(ctx context.Context, resp *http.Response) (<-chan StreamEvent, error) {
	eventChan := make(chan StreamEvent, 10)
	
	go func() {
		defer close(eventChan)
		defer resp.Body.Close()
		
		// Parse messages incrementally
		for {
			select {
			case <-ctx.Done():
				eventChan <- StreamEvent{Type: "error", Error: ctx.Err()}
				return
			default:
				// Parse one message at a time
				msg, err := parseMessageWithContext(ctx, resp.Body)
				if err == io.EOF {
					// Stream ended normally
					eventChan <- StreamEvent{Type: "done"}
					return
				}
				if err != nil {
					eventChan <- StreamEvent{Type: "error", Error: err}
					return
				}
				
				// Process and emit event immediately
				event := processMessage(msg)
				if event != nil {
					eventChan <- *event
				}
			}
		}
	}()
	
	return eventChan, nil
}

// parseMessageWithContext reads one event stream message with context awareness
func parseMessageWithContext(ctx context.Context, reader io.Reader) (*EventStreamMessage, error) {
	// Create a context-aware reader
	ctxReader := &contextReader{
		ctx:    ctx,
		reader: reader,
	}
	
	// Parse one message
	return parseMessage(ctxReader)
}

// contextReader wraps an io.Reader to respect context cancellation
type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (n int, err error) {
	// Check context before reading
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
	}
	
	// Use a channel to make the read cancellable
	type result struct {
		n   int
		err error
	}
	
	ch := make(chan result, 1)
	go func() {
		n, err := r.reader.Read(p)
		ch <- result{n, err}
	}()
	
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	case res := <-ch:
		return res.n, res.err
	}
}

// processMessage converts EventStreamMessage to StreamEvent
func processMessage(msg *EventStreamMessage) *StreamEvent {
	// Check message type header
	messageType, ok := msg.Headers[":message-type"].(string)
	if !ok || messageType != "event" {
		return nil
	}
	
	// Check event type
	eventType, ok := msg.Headers[":event-type"].(string)
	if !ok {
		return nil
	}
	
	// Parse payload as JSON
	var payload map[string]interface{}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return nil
	}
	
	// Extract content based on event type
	switch eventType {
	case "assistantResponseEvent":
		if contentStr, ok := payload["content"].(string); ok {
			return &StreamEvent{
				Type:    "content",
				Content: contentStr,
			}
		}
	case "messageMetadataEvent":
		// Handle metadata
		return nil
	default:
		return nil
	}
	
	return nil
}
```

## Alternative: Increase Timeout for Multimodal

As a simpler short-term fix, detect multimodal requests and use a longer timeout:

**File**: `internal/handlers/chat.go`

```go
// Determine timeout based on request type
timeout := h.config.FirstTokenTimeout
if hasImages {
	// Multimodal requests take longer - use 90 second timeout
	timeout = 90 * time.Second
	log.Printf("[%s] Using extended timeout for multimodal request: %v", requestID, timeout)
}

// Create context with timeout
ctx, cancel := context.WithTimeout(r.Context(), timeout)
defer cancel()

// Make request with timeout context
resp, err := httpClient.PostStream(ctx, apiEndpoint, convStateReq)
```

**File**: `internal/config/config.go`

```go
type Config struct {
	// ...existing fields...
	
	// Timeout Configuration
	FirstTokenTimeout          time.Duration
	MultimodalFirstTokenTimeout time.Duration // New field
}

func LoadConfig() (*Config, error) {
	// ...existing code...
	
	config := &Config{
		// ...existing fields...
		FirstTokenTimeout:          getDurationEnv("FIRST_TOKEN_TIMEOUT", 30*time.Second),
		MultimodalFirstTokenTimeout: getDurationEnv("MULTIMODAL_FIRST_TOKEN_TIMEOUT", 90*time.Second),
	}
	
	return config, nil
}
```

## Recommended Approach

1. **Immediate (Quick Fix)**: Implement the timeout increase for multimodal requests
   - Simple to implement
   - Solves the immediate problem
   - Can be done in < 30 minutes

2. **Short-term (Proper Fix)**: Implement incremental event parsing
   - More robust solution
   - Handles slow streams properly
   - Respects context cancellation
   - Estimated: 2-3 hours

3. **Long-term (Optimization)**: Consider using AWS SDK Go v2 directly
   - Use official AWS SDK for Bedrock Runtime
   - Handles all edge cases
   - Better maintained
   - Estimated: 1-2 days for full migration

## Testing

After implementing the fix, test with:

1. **Text-only requests**: Should work as before (30s timeout)
2. **Multimodal requests**: Should complete without timeout (90s timeout)
3. **Large images**: Test with actual architecture diagrams
4. **Context cancellation**: Verify that cancelling the request works properly
5. **Slow streams**: Simulate slow responses to verify incremental parsing

## Files to Modify

1. `internal/streaming/parser.go` - Implement incremental parsing
2. `internal/streaming/eventstream.go` - Add context-aware message parsing
3. `internal/handlers/chat.go` - Detect multimodal and adjust timeout
4. `internal/config/config.go` - Add multimodal timeout configuration
5. `config/examples/config/examples/.env.qdeveloper-test` - Add `MULTIMODAL_FIRST_TOKEN_TIMEOUT=90s`

## Conclusion

The core image format issue is **FIXED**. The timeout issue can be resolved with either:
- **Quick fix**: Increase timeout for multimodal requests (recommended for immediate deployment)
- **Proper fix**: Implement incremental event-based parsing (recommended for production)

Both approaches are valid and can be implemented independently.
