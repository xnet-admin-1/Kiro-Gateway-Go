# Q CLI Source Code Analysis - Complete

## Date: January 24, 2026

## Summary

Analyzed the Amazon Q CLI source code to understand:
1. ✅ Response parsing (event stream handling)
2. ✅ Model capabilities (vision/multimodal support)
3. ✅ Image handling implementation

## Key Findings

### 1. Event Stream Response Parsing

**Location**: `crates/amzn-qdeveloper-streaming-client/src/event_stream_serde.rs`

The Q CLI uses AWS Smithy's event stream unmarshaller to parse responses. Key event types:

```rust
match response_headers.smithy_type.as_str() {
    "messageMetadataEvent" => MessageMetadataEvent,
    "assistantResponseEvent" => AssistantResponseEvent,  // Main content
    "reasoningContentEvent" => ReasoningContentEvent,    // Extended thinking
    "codeEvent" => CodeEvent,                            // Code blocks
    "toolUseEvent" => ToolUseEvent,                      // Tool calls
    "toolResultEvent" => ToolResultEvent,                // Tool results
    "metadataEvent" => MetadataEvent,                    // Metadata
    "meteringEvent" => MeteringEvent,                    // Usage/billing
    "citationEvent" => CitationEvent,                    // Citations
    "invalidStateEvent" => InvalidStateEvent,            // Errors
    // ... more event types
}
```

**Important**: The response is **ALWAYS** an event stream (`application/vnd.amazon.eventstream`), even for non-streaming requests. The Q CLI parses this format for all responses.

### 2. Image/Vision Support

**Location**: `crates/chat-cli/src/api_client/model.rs`

#### Supported Image Formats

```rust
pub enum ImageFormat {
    Gif,      // ✅ Supported
    Jpeg,     // ✅ Supported
    Png,      // ✅ Supported
    Webp,     // ✅ Supported
}
```

**Note**: Our test uses PNG, which is supported ✅

#### Image Structure

```rust
pub struct ImageBlock {
    pub format: ImageFormat,
    pub source: ImageSource,
}

pub enum ImageSource {
    Bytes(Vec<u8>),  // Raw image bytes
    Unknown,
}
```

#### User Input Message with Images

```rust
pub struct UserInputMessage {
    pub content: String,
    pub user_input_message_context: Option<UserInputMessageContext>,
    pub user_intent: Option<UserIntent>,
    pub images: Option<Vec<ImageBlock>>,  // ← Images here!
    pub model_id: Option<String>,         // ← Model ID here!
}
```

**Key Insight**: The Q CLI includes `model_id` in the `UserInputMessage`, not at the top level of the request!

### 3. Model ID Location

From the Q CLI source, the model ID is specified in the `UserInputMessage`:

```rust
UserInputMessage {
    content: "What's in this image?",
    images: Some(vec![ImageBlock { ... }]),
    model_id: Some("model id".to_string()),  // ← HERE!
    // ...
}
```

This is different from OpenAI format where model is at the request level.

### 4. Response Stream Events

**Location**: `crates/chat-cli/src/api_client/model.rs` (lines 600-700)

The Q CLI converts event stream events to a unified format:

```rust
pub enum ChatResponseStream {
    AssistantResponseEvent { content: String },  // Main text response
    CodeEvent { content: String },               // Code blocks
    ToolUseEvent { ... },                        // Tool calls
    MessageMetadataEvent { ... },                // Conversation metadata
    InvalidStateEvent { ... },                   // Errors
    // ... more event types
}
```

**For vision responses**, the content comes through `AssistantResponseEvent` just like text responses.

### 5. Both CodeWhisperer and Q Developer Support

The Q CLI supports BOTH endpoints with the same code:

```rust
// Converts to CodeWhisperer format
impl From<UserInputMessage> for amzn_codewhisperer_streaming_client::types::UserInputMessage { ... }

// Converts to Q Developer format
impl From<UserInputMessage> for amzn_qdeveloper_streaming_client::types::UserInputMessage { ... }
```

**Both endpoints support images** - the image handling code is identical for both.

## Comparison: Our Gateway vs Q CLI

| Aspect | Q CLI | Our Gateway | Status |
|--------|-------|-------------|--------|
| **Endpoint Path** | `/` | `/` | ✅ Match |
| **X-Amz-Target** | `AmazonQDeveloperStreamingService.SendMessage` | `AmazonQDeveloperStreamingService.SendMessage` | ✅ Match |
| **Content-Type** | `application/x-amz-json-1.0` | `application/x-amz-json-1.0` | ✅ Match |
| **Response Format** | Event stream parser | Event stream parser | ⚠️ Needs fix |
| **Image Support** | `images` array in `UserInputMessage` | `images` array in request | ✅ Implemented |
| **Model ID** | In `UserInputMessage` | At request level | ⚠️ Different |
| **Image Formats** | PNG, JPEG, GIF, WEBP | PNG, JPEG, GIF, WEBP | ✅ Match |

## Issues Identified

### Issue 1: Event Stream Parsing ⚠️

**Problem**: Our gateway doesn't properly parse the `application/vnd.amazon.eventstream` format.

**Q CLI Solution**: Uses AWS Smithy's event stream unmarshaller to parse binary event stream format.

**Our Current Implementation**: Tries to parse as JSON, which fails.

**Fix Needed**: Implement event stream parser similar to Q CLI's approach.

### Issue 2: Model ID Location (Minor)

**Q CLI**: Model ID is in `UserInputMessage.model_id`
**Our Gateway**: Model ID is at request level

This might not be critical, but worth noting for compatibility.

## Vision Test Analysis

### Why Our Vision Test Returns Empty

1. ✅ Model supports vision (`claude-3-5-sonnet-20241022-v2`)
2. ✅ Image is included in request (PNG format)
3. ✅ Q Developer endpoint is used
4. ✅ SigV4 authentication works
5. ✅ API accepts request (200 OK)
6. ❌ **Response parsing fails** - We don't parse event stream format

**Root Cause**: AWS returns `application/vnd.amazon.eventstream` format, but our parser expects JSON or SSE format.

### Event Stream Format

The response is a binary format with:
- Message headers (`:message-type`, `:smithy-type`)
- Message payload (JSON for each event)
- Multiple events in sequence

Example event structure:
```
:message-type = "event"
:smithy-type = "assistantResponseEvent"
:content-type = "application/json"

{"content": "I see a red pixel in this image."}
```

## Recommendations

### 1. Implement Event Stream Parser (High Priority)

Create a parser for `application/vnd.amazon.eventstream` format:

```go
// internal/streaming/eventstream.go
type EventStreamParser struct {
    // Parse binary event stream format
}

func (p *EventStreamParser) Parse(reader io.Reader) (<-chan Event, error) {
    // Parse message headers
    // Parse message payload
    // Emit events
}
```

### 2. Test with Q CLI Format (Medium Priority)

Verify our request format matches Q CLI exactly:
- Image bytes encoding
- Message structure
- Model ID location

### 3. Add Event Stream Tests (Medium Priority)

Create tests for event stream parsing:
- Single event
- Multiple events
- Error events
- Tool use events

## Conclusion

✅ **Our implementation is correct for endpoint and authentication**
✅ **Vision/multimodal is supported by the model**
✅ **Image format (PNG) is supported**
❌ **Response parsing needs to handle event stream format**

The Q CLI source code confirms that:
1. All responses use event stream format (not just streaming requests)
2. Vision responses come through `AssistantResponseEvent` like text
3. Both CodeWhisperer and Q Developer endpoints support images
4. Our endpoint configuration is correct

**Next Step**: Implement event stream parser to handle the binary response format.

## Files Analyzed

### Q CLI Source Files
- `crates/amzn-qdeveloper-streaming-client/src/event_stream_serde.rs` - Event stream parsing
- `crates/amzn-qdeveloper-streaming-client/src/operation/send_message.rs` - Request format
- `crates/amzn-qdeveloper-streaming-client/src/types/_image_block.rs` - Image structure
- `crates/amzn-qdeveloper-streaming-client/src/types/_image_format.rs` - Supported formats
- `crates/chat-cli/src/api_client/model.rs` - Model types and conversions

### Our Gateway Files
- `internal/client/client.go` - HTTP client (correct)
- `internal/handlers/chat.go` - Request handling (correct)
- `internal/streaming/streaming.go` - Response parsing (needs event stream support)
- `internal/validation/limits.go` - Model capabilities (correct)

