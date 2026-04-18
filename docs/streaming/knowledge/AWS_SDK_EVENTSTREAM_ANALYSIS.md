# AWS SDK Event Stream Implementation Analysis

## Source Files Analyzed

From `D:\repo2\aws-sdk-go-v2`:
- `aws/protocol/eventstream/decode.go` - Core decoder
- `service/bedrockruntime/eventstream.go` - Bedrock Runtime implementation

## Key Findings

### 1. Incremental Message Decoding

AWS SDK decodes messages **one at a time** in a loop, not all at once:

```go
func (r *responseStreamReader) readEventStream() {
    defer r.Close()
    defer close(r.stream)

    for {
        r.payloadBuf = r.payloadBuf[0:0]
        decodedMessage, err := r.decoder.Decode(r.eventStream, r.payloadBuf)
        if err != nil {
            if err == io.EOF {
                return
            }
            select {
            case <-r.done:
                return
            default:
                r.err.SetError(err)
                return
            }
        }

        event, err := r.deserializeEventMessage(&decodedMessage)
        if err != nil {
            r.err.SetError(err)
            return
        }

        select {
        case r.stream <- event:
        case <-r.done:
            return
        }
    }
}
```

**Key Points**:
- Decodes ONE message at a time with `decoder.Decode()`
- Immediately sends event to channel
- Checks `<-r.done` for cancellation
- Reuses `payloadBuf` for efficiency

### 2. Context Awareness via Done Channel

The reader respects cancellation through a `done` channel:

```go
type responseStreamReader struct {
    stream      chan types.ResponseStream
    decoder     *eventstream.Decoder
    eventStream io.ReadCloser
    err         *smithysync.OnceErr
    payloadBuf  []byte
    done        chan struct{}  // ← Cancellation signal
    closeOnce   sync.Once
}

// Check for cancellation
select {
case r.stream <- event:
case <-r.done:
    return
}
```

### 3. Decoder Interface

The decoder reads one message at a time:

```go
func (d *Decoder) Decode(reader io.Reader, payloadBuf []byte) (m Message, err error) {
    m, err = decodeMessage(reader, payloadBuf)
    return m, err
}

func decodeMessage(reader io.Reader, payloadBuf []byte) (m Message, err error) {
    crc := crc32.New(crc32IEEETable)
    hashReader := io.TeeReader(reader, crc)

    prelude, err := decodePrelude(hashReader, crc)
    if err != nil {
        return Message{}, err
    }

    if prelude.HeadersLen > 0 {
        lr := io.LimitReader(hashReader, int64(prelude.HeadersLen))
        m.Headers, err = decodeHeaders(lr)
        if err != nil {
            return Message{}, err
        }
    }

    if payloadLen := prelude.PayloadLen(); payloadLen > 0 {
        buf, err := decodePayload(payloadBuf, io.LimitReader(hashReader, int64(payloadLen)))
        if err != nil {
            return Message{}, err
        }
        m.Payload = buf
    }

    msgCRC := crc.Sum32()
    if err := validateCRC(reader, msgCRC); err != nil {
        return Message{}, err
    }

    return m, nil
}
```

**Key Points**:
- Uses `io.LimitReader` to read exact amounts
- Validates CRC checksums
- Returns single message
- Reuses `payloadBuf` to avoid allocations

### 4. Payload Buffer Reuse

AWS SDK reuses a single buffer for all payloads:

```go
w := &responseStreamReader{
    stream:      make(chan types.ResponseStream),
    decoder:     decoder,
    eventStream: readCloser,
    err:         smithysync.NewOnceErr(),
    done:        make(chan struct{}),
    payloadBuf:  make([]byte, 10*1024),  // ← 10KB buffer
}

// In loop:
r.payloadBuf = r.payloadBuf[0:0]  // Reset to zero length, keep capacity
decodedMessage, err := r.decoder.Decode(r.eventStream, r.payloadBuf)
```

## Comparison with Our Implementation

### Our Current Implementation (WRONG)

```go
func parseEventStreamBinary(ctx context.Context, resp *http.Response) (<-chan StreamEvent, error) {
    eventChan := make(chan StreamEvent, 10)
    
    go func() {
        defer close(eventChan)
        defer resp.Body.Close()
        
        // ❌ Reads ENTIRE stream at once
        content, err := ParseEventStreamResponse(resp.Body)
        if err != nil {
            eventChan <- StreamEvent{Type: "error", Error: err}
            return
        }
        
        // ❌ Only sends one event after reading everything
        if content != "" {
            eventChan <- StreamEvent{Type: "content", Content: content}
        }
        
        eventChan <- StreamEvent{Type: "done"}
    }()
    
    return eventChan, nil
}
```

**Problems**:
1. Reads entire stream before processing
2. No context cancellation support
3. No incremental event emission
4. Blocks on slow streams

### AWS SDK Implementation (CORRECT)

```go
func (r *responseStreamReader) readEventStream() {
    defer r.Close()
    defer close(r.stream)

    for {
        // ✅ Decode ONE message at a time
        decodedMessage, err := r.decoder.Decode(r.eventStream, r.payloadBuf)
        if err != nil {
            if err == io.EOF {
                return
            }
            // ✅ Check for cancellation
            select {
            case <-r.done:
                return
            default:
                r.err.SetError(err)
                return
            }
        }

        // ✅ Process message immediately
        event, err := r.deserializeEventMessage(&decodedMessage)
        if err != nil {
            r.err.SetError(err)
            return
        }

        // ✅ Send event immediately with cancellation check
        select {
        case r.stream <- event:
        case <-r.done:
            return
        }
    }
}
```

**Advantages**:
1. Incremental processing
2. Context-aware cancellation
3. Immediate event emission
4. Handles slow streams gracefully

## Recommended Implementation for Our Code

Based on AWS SDK patterns, here's how we should fix our code:

```go
// File: internal/streaming/parser.go

func parseEventStreamBinary(ctx context.Context, resp *http.Response) (<-chan StreamEvent, error) {
    eventChan := make(chan StreamEvent, 10)
    
    // Create decoder (reuse our existing eventstream.go code)
    decoder := &EventStreamDecoder{
        payloadBuf: make([]byte, 10*1024),
    }
    
    go func() {
        defer close(eventChan)
        defer resp.Body.Close()
        
        for {
            // Reset payload buffer
            decoder.payloadBuf = decoder.payloadBuf[0:0]
            
            // Decode ONE message at a time
            msg, err := decoder.DecodeMessage(resp.Body)
            if err != nil {
                if err == io.EOF {
                    // Stream ended normally
                    eventChan <- StreamEvent{Type: "done"}
                    return
                }
                // Check for context cancellation
                select {
                case <-ctx.Done():
                    eventChan <- StreamEvent{Type: "error", Error: ctx.Err()}
                    return
                default:
                    eventChan <- StreamEvent{Type: "error", Error: err}
                    return
                }
            }
            
            // Process message immediately
            event := processEventStreamMessage(msg)
            if event != nil {
                // Send event with cancellation check
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

type EventStreamDecoder struct {
    payloadBuf []byte
}

func (d *EventStreamDecoder) DecodeMessage(reader io.Reader) (*EventStreamMessage, error) {
    // Use our existing parseMessage function from eventstream.go
    return parseMessage(reader)
}

func processEventStreamMessage(msg *EventStreamMessage) *StreamEvent {
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

## Key Takeaways

1. **Decode incrementally**: One message at a time, not entire stream
2. **Check context**: Use `select` with `<-ctx.Done()` for cancellation
3. **Emit immediately**: Send events as soon as they're decoded
4. **Reuse buffers**: Use a single payload buffer, reset between messages
5. **Handle EOF properly**: EOF means normal stream end, not an error

## Implementation Priority

1. **Immediate**: Refactor `parseEventStreamBinary` to use incremental decoding
2. **Short-term**: Add context cancellation checks
3. **Medium-term**: Optimize with buffer reuse
4. **Long-term**: Consider using AWS SDK directly

This matches exactly how AWS SDK handles Bedrock Runtime streaming responses.
