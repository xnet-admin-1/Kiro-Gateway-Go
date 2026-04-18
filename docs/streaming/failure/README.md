# Streaming Implementation Issues

This directory documents known issues and recommended fixes for event stream parsing.

## Contents

### EVENT_STREAM_FIX_RECOMMENDATION.md
Identifies timeout issue with current event stream parser:
- **Problem**: Parser reads entire stream synchronously without respecting context timeouts
- **Root Cause**: `ParseEventStreamResponse` uses `io.ReadFull` which blocks indefinitely
- **Solution**: Implement incremental event-based parsing like AWS SDK

**Recommended Fixes**:
1. **Quick fix**: Increase timeout for multimodal requests (30 min implementation)
2. **Proper fix**: Implement incremental event parsing (2-3 hours implementation)

### FINAL_IMPLEMENTATION_PLAN.md
Complete implementation plan with three phases:
- **Phase 1**: Quick fix with timeout increase (30 minutes)
- **Phase 2**: Proper fix with incremental parsing (2-3 hours)
- **Phase 3**: Long-term AWS SDK integration (1-2 days, optional)

Includes detailed code examples, testing plan, and deployment strategy.

## The Issue

### Current Implementation Problem
```go
// ❌ WRONG: Reads entire stream before processing
func parseEventStreamBinary(ctx context.Context, resp *http.Response) {
    content, err := ParseEventStreamResponse(resp.Body)  // Blocks
    eventChan <- StreamEvent{Type: "content", Content: content}
}
```

### AWS SDK Pattern (Correct)
```go
// ✅ CORRECT: Decodes one message at a time
for {
    decodedMessage, err := r.decoder.Decode(r.eventStream, r.payloadBuf)
    if err == io.EOF {
        return
    }
    
    event, err := r.deserializeEventMessage(&decodedMessage)
    
    select {
    case r.stream <- event:  // Send immediately
    case <-r.done:           // Check cancellation
        return
    }
}
```

## Impact

- **Text-only requests**: Work fine (fast responses)
- **Multimodal requests**: May timeout (slower responses due to image processing)
- **Large images**: Higher chance of timeout
- **Context cancellation**: Not properly respected

## Recommended Action

**Immediate**: Implement Phase 1 (timeout increase)
- Low risk, minimal code changes
- Gets multimodal working immediately
- Buys time for proper fix

**Short-term**: Implement Phase 2 (incremental parsing)
- Proper long-term solution
- Based on AWS SDK patterns
- Production-ready

## Files to Modify

### Phase 1 (Quick Fix)
1. `internal/config/config.go` - Add `MultimodalFirstTokenTimeout`
2. `internal/handlers/chat.go` - Detect images and adjust timeout
3. `config/examples/config/examples/.env.qdeveloper-test` - Add timeout configuration

### Phase 2 (Proper Fix)
1. `internal/streaming/parser.go` - Refactor to incremental parsing
2. `internal/streaming/eventstream.go` - Add context-aware wrapper

## Success Criteria

- ✅ Text-only requests work (< 30s)
- ✅ Multimodal requests complete (< 90s)
- ✅ Context cancellation works
- ✅ No memory leaks
- ✅ Graceful error handling

## Status

- **Parser implementation**: ✅ Complete (see `docs/streaming/success/`)
- **Timeout issue**: ⚠️ Identified, fix recommended
- **Implementation plan**: ✅ Ready to execute
