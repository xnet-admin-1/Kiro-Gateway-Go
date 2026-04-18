# Final Implementation Plan - Multimodal Vision Support

## Status Summary

### ✅ COMPLETED: Image Format Fix
- Fixed `ImageSource` structure to match Q CLI implementation
- Images now correctly formatted in API requests
- API accepts requests (200 OK)
- SSO credentials and SigV4 signing working

### ⚠️ IDENTIFIED: Event Stream Timeout Issue
- Current implementation reads entire stream at once
- No context cancellation support
- Causes timeouts on slow/multimodal requests

### 📚 ANALYZED: AWS SDK Implementation
- Reviewed actual AWS SDK Go v2 source code from `D:\repo2\aws-sdk-go-v2`
- Found correct incremental decoding pattern
- Identified key differences from our implementation

## Root Cause Analysis

### Our Current Implementation
```go
// ❌ WRONG: Reads entire stream before processing
func parseEventStreamBinary(ctx context.Context, resp *http.Response) {
    content, err := ParseEventStreamResponse(resp.Body)  // Blocks until entire stream read
    eventChan <- StreamEvent{Type: "content", Content: content}
}
```

### AWS SDK Implementation
```go
// ✅ CORRECT: Decodes one message at a time
func (r *responseStreamReader) readEventStream() {
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
}
```

## Implementation Plan

### Phase 1: Quick Fix (30 minutes) - RECOMMENDED FOR IMMEDIATE DEPLOYMENT

**Goal**: Get multimodal working with minimal code changes

**Changes**:
1. Detect multimodal requests (images present)
2. Use longer timeout (90 seconds vs 30 seconds)
3. Add configuration option

**Files to modify**:
- `internal/config/config.go` - Add `MultimodalFirstTokenTimeout`
- `internal/handlers/chat.go` - Detect images and adjust timeout
- `config/examples/config/examples/.env.qdeveloper-test` - Add `MULTIMODAL_FIRST_TOKEN_TIMEOUT=90s`

**Code**:
```go
// internal/handlers/chat.go
timeout := h.config.FirstTokenTimeout
if hasImages {
    timeout = h.config.MultimodalFirstTokenTimeout
    log.Printf("[%s] Using extended timeout for multimodal: %v", requestID, timeout)
}

ctx, cancel := context.WithTimeout(r.Context(), timeout)
defer cancel()
```

### Phase 2: Proper Fix (2-3 hours) - RECOMMENDED FOR PRODUCTION

**Goal**: Implement AWS SDK-style incremental event stream parsing

**Changes**:
1. Refactor `parseEventStreamBinary` to decode incrementally
2. Add context cancellation checks
3. Emit events immediately as decoded
4. Reuse payload buffers

**Files to modify**:
- `internal/streaming/parser.go` - Refactor to incremental parsing
- `internal/streaming/eventstream.go` - Keep existing decoder, add wrapper

**Implementation**:

```go
// File: internal/streaming/parser.go

func parseEventStreamBinary(ctx context.Context, resp *http.Response) (<-chan StreamEvent, error) {
    eventChan := make(chan StreamEvent, 10)
    
    go func() {
        defer close(eventChan)
        defer resp.Body.Close()
        
        payloadBuf := make([]byte, 10*1024)
        
        for {
            // Reset buffer
            payloadBuf = payloadBuf[0:0]
            
            // Decode ONE message
            msg, err := parseMessage(resp.Body)
            if err != nil {
                if err == io.EOF {
                    eventChan <- StreamEvent{Type: "done"}
                    return
                }
                select {
                case <-ctx.Done():
                    eventChan <- StreamEvent{Type: "error", Error: ctx.Err()}
                    return
                default:
                    eventChan <- StreamEvent{Type: "error", Error: err}
                    return
                }
            }
            
            // Process and emit immediately
            event := processEventStreamMessage(msg)
            if event != nil {
                select {
                case eventChan <- *event:
                case <-ctx.Done():
                    eventChan <- StreamEvent{Type: "error", Error: ctx.Err()}
                    return
                }
            }
        }
    }()
    
    return eventChan, nil
}

func processEventStreamMessage(msg *EventStreamMessage) *StreamEvent {
    messageType, ok := msg.Headers[":message-type"].(string)
    if !ok || messageType != "event" {
        return nil
    }
    
    eventType, ok := msg.Headers[":event-type"].(string)
    if !ok {
        return nil
    }
    
    var payload map[string]interface{}
    if err := json.Unmarshal(msg.Payload, &payload); err != nil {
        return nil
    }
    
    switch eventType {
    case "assistantResponseEvent":
        if content, ok := payload["content"].(string); ok {
            return &StreamEvent{
                Type:    "content",
                Content: content,
            }
        }
    case "messageMetadataEvent":
        return nil
    }
    
    return nil
}
```

### Phase 3: Long-term (1-2 days) - OPTIONAL

**Goal**: Use AWS SDK Go v2 directly for Bedrock Runtime

**Benefits**:
- Official AWS implementation
- Handles all edge cases
- Better maintained
- Automatic updates

**Changes**:
- Replace custom HTTP client with AWS SDK
- Use `bedrockruntime.Client`
- Leverage `InvokeModelWithResponseStream`

## Testing Plan

### Test 1: Text-Only Request
```powershell
# Should work as before with 30s timeout
.\scripts\test_qdeveloper_quick.ps1
```

### Test 2: Multimodal Request
```powershell
# Should complete without timeout
.\scripts\test_vision_fixed.ps1
```

### Test 3: Large Image
```powershell
# Test with actual architecture diagram (not 1x1 pixel)
# Create real test image
$image = [System.Drawing.Bitmap]::new(1024, 768)
# ... draw diagram ...
```

### Test 4: Context Cancellation
```powershell
# Start request, cancel mid-stream
# Verify graceful shutdown
```

## Deployment Strategy

### Option A: Quick Fix First (RECOMMENDED)
1. Deploy Phase 1 (timeout increase) immediately
2. Test in production with real multimodal requests
3. Implement Phase 2 in parallel
4. Deploy Phase 2 after thorough testing

### Option B: Proper Fix Only
1. Implement Phase 2 directly
2. Test thoroughly in staging
3. Deploy to production

## Success Criteria

- ✅ Text-only requests work (< 30s response time)
- ✅ Multimodal requests complete (< 90s response time)
- ✅ Images correctly analyzed by model
- ✅ Context cancellation works properly
- ✅ No memory leaks from buffer allocation
- ✅ Graceful error handling

## Risk Assessment

### Phase 1 (Quick Fix)
- **Risk**: Low
- **Impact**: Minimal code changes
- **Rollback**: Easy (revert config)

### Phase 2 (Proper Fix)
- **Risk**: Medium
- **Impact**: Core streaming logic changes
- **Rollback**: Moderate (revert parser.go)

### Phase 3 (AWS SDK)
- **Risk**: High
- **Impact**: Major refactoring
- **Rollback**: Difficult (significant changes)

## Recommendation

**Implement Phase 1 immediately** (30 minutes):
- Low risk, high reward
- Gets multimodal working today
- Buys time for proper fix

**Implement Phase 2 next week** (2-3 hours):
- Proper long-term solution
- Based on AWS SDK patterns
- Production-ready

**Consider Phase 3 for future** (optional):
- Only if maintaining custom implementation becomes burden
- Evaluate after Phase 2 is stable

## Files Modified Summary

### Phase 1 (Quick Fix)
1. `internal/config/config.go`
2. `internal/handlers/chat.go`
3. `config/examples/config/examples/.env.qdeveloper-test`

### Phase 2 (Proper Fix)
1. `internal/streaming/parser.go`
2. `internal/streaming/eventstream.go`

### Documentation
1. `VISION_MULTIMODAL_FIX_SUMMARY.md` ✅
2. `MULTIMODAL_IMPLEMENTATION_STATUS.md` ✅
3. `EVENT_STREAM_FIX_RECOMMENDATION.md` ✅
4. `AWS_SDK_EVENTSTREAM_ANALYSIS.md` ✅
5. `FINAL_IMPLEMENTATION_PLAN.md` ✅

## Conclusion

The image format issue is **FIXED**. The timeout issue has a clear implementation path with two options:
1. **Quick fix**: Increase timeout for multimodal (30 min)
2. **Proper fix**: Implement AWS SDK-style incremental parsing (2-3 hours)

Both are well-understood and ready to implement.
