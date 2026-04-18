# AWS Event Stream Documentation

This directory contains documentation about AWS event stream format implementation and troubleshooting.

## Directory Structure

### knowledge/
Analysis and research documents about AWS event stream format:
- **Q_CLI_ANALYSIS_COMPLETE.md** - Amazon Q CLI source code analysis
- **AWS_SDK_EVENTSTREAM_ANALYSIS.md** - AWS SDK Go v2 implementation patterns
- **test_streaming.ps1** - Test script for event stream parsing
- **README.md** - Knowledge base index

### success/
Successful implementation documentation:
- **EVENT_STREAM_IMPLEMENTATION_SUCCESS.md** - Complete parser implementation details
- **README.md** - Success documentation index

### failure/
Known issues and recommended fixes:
- **EVENT_STREAM_FIX_RECOMMENDATION.md** - Timeout issue analysis and solutions
- **FINAL_IMPLEMENTATION_PLAN.md** - Complete implementation plan with phases
- **README.md** - Known issues and fixes index

## Overview

Amazon Q Developer API returns responses in `application/vnd.amazon.eventstream` binary format. This format is used for all responses, including non-streaming requests.

## Event Stream Format

AWS event stream uses a binary format with the following structure:

```
[prelude][prelude_crc][headers][payload][message_crc]
```

### Components

1. **Prelude** (8 bytes)
   - 4 bytes: Total message length (big-endian uint32)
   - 4 bytes: Headers length (big-endian uint32)

2. **Prelude CRC** (4 bytes)
   - CRC32 checksum of prelude

3. **Headers** (variable length)
   - Key-value pairs with type information
   - Common headers: `:message-type`, `:event-type`, `:content-type`

4. **Payload** (variable length)
   - JSON data for the event

5. **Message CRC** (4 bytes)
   - CRC32 checksum of entire message (excluding this CRC)

## Event Types

The gateway handles these event types:

- `assistantResponseEvent` - Main text response content
- `messageMetadataEvent` - Conversation metadata and final message indicator
- `codeEvent` - Code blocks
- `toolUseEvent` - Tool calls
- `meteringEvent` - Usage/billing information
- `invalidStateEvent` - Error events

## Implementation Status

### ✅ Completed

- Event stream binary parser (`internal/streaming/eventstream.go`)
- Message parsing with CRC verification
- Header parsing for all types
- Payload extraction and JSON parsing
- Event type handling
- Integration with Q Developer API

### ⚠️ Known Issues

- Timeout issues with slow streams (multimodal requests)
- No incremental event emission
- Context cancellation not fully respected

See `failure/` directory for detailed analysis and recommended fixes.

## Quick Reference

### For Understanding Event Streams
→ See `knowledge/Q_CLI_ANALYSIS_COMPLETE.md` for Q CLI implementation  
→ See `knowledge/AWS_SDK_EVENTSTREAM_ANALYSIS.md` for AWS SDK patterns

### For Implementation Success
→ See `success/EVENT_STREAM_IMPLEMENTATION_SUCCESS.md` for what works

### For Fixing Issues
→ See `failure/EVENT_STREAM_FIX_RECOMMENDATION.md` for timeout fixes  
→ See `failure/FINAL_IMPLEMENTATION_PLAN.md` for complete implementation plan

## Troubleshooting

### Empty Responses

If you get empty responses:
1. Check that `Content-Type: application/x-amz-json-1.0` is set
2. Verify `X-Amz-Target: AmazonQDeveloperStreamingService.SendMessage` header
3. Ensure event stream parser is being used (not JSON parser)

### Timeout Issues

If requests timeout:
1. Check if request includes images (multimodal)
2. Consider increasing timeout for multimodal requests (see `failure/EVENT_STREAM_FIX_RECOMMENDATION.md`)
3. Implement incremental event parsing (see `failure/FINAL_IMPLEMENTATION_PLAN.md`)

### Generic Responses

If Q Developer gives generic "I can't answer that" responses:
- This is expected for non-AWS questions
- For AWS questions, check request format
- Verify images are properly formatted (for vision requests)

## Best Practices from AWS SDK

1. **Decode incrementally**: One message at a time, not entire stream
2. **Check context**: Use `select` with `<-ctx.Done()` for cancellation
3. **Emit immediately**: Send events as soon as decoded
4. **Reuse buffers**: Single payload buffer, reset between messages
5. **Handle EOF properly**: EOF means normal stream end, not an error

## References

- Amazon Q CLI source code
- AWS SDK Go v2 event stream implementation
- AWS event stream specification
