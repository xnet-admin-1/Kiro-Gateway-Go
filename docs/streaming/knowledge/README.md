# Streaming Knowledge Base

This directory contains analysis and research documents about AWS event stream format and implementation patterns.

## Contents

### Q_CLI_ANALYSIS_COMPLETE.md
Complete analysis of Amazon Q CLI source code covering:
- Event stream response parsing implementation
- Supported image formats (PNG, JPEG, GIF, WEBP)
- Model ID location in requests
- Response stream event types
- Comparison between Q CLI and our gateway implementation

**Key Finding**: All Q Developer responses use event stream format, even for non-streaming requests.

### AWS_SDK_EVENTSTREAM_ANALYSIS.md
Deep dive into AWS SDK Go v2 event stream implementation:
- Incremental message decoding pattern
- Context-aware cancellation via done channel
- Decoder interface and message structure
- Payload buffer reuse for efficiency
- Comparison with our implementation

**Key Finding**: AWS SDK decodes messages one at a time in a loop, not all at once.

### test_streaming.ps1
PowerShell test script for validating event stream parsing with Q Developer API.

## Key Insights

### Event Stream Format
AWS event stream uses binary format with:
- Prelude (8 bytes): message length + headers length
- Prelude CRC (4 bytes)
- Headers (variable): typed key-value pairs
- Payload (variable): JSON event data
- Message CRC (4 bytes)

### Event Types
- `assistantResponseEvent` - Main text response content
- `messageMetadataEvent` - Conversation metadata
- `codeEvent` - Code blocks
- `toolUseEvent` - Tool calls
- `meteringEvent` - Usage/billing
- `invalidStateEvent` - Errors

### Best Practices from AWS SDK
1. **Decode incrementally**: One message at a time
2. **Check context**: Use `select` with `<-ctx.Done()` for cancellation
3. **Emit immediately**: Send events as soon as decoded
4. **Reuse buffers**: Single payload buffer, reset between messages
5. **Handle EOF properly**: EOF means normal stream end

## Usage

These documents provide the foundation for understanding how event stream parsing should work. Reference them when:
- Debugging event stream parsing issues
- Implementing new event types
- Optimizing parser performance
- Understanding AWS SDK patterns
