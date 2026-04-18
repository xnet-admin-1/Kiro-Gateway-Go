# Vision ModelID Fix - January 25, 2026

## Problem Discovered

Vision requests were returning generic responses: "I can't answer that question, but I can answer questions about AWS and AWS services."

## Root Cause

By reading the official AWS Q Developer CLI source code, I discovered the field name was wrong:

**WRONG** (what we had):
```json
{
  "userInputMessage": {
    "model_id": "claude-sonnet-4-5"  // snake_case
  }
}
```

**CORRECT** (from Q CLI source):
```rust
// From: crates/amzn-qdeveloper-streaming-client/src/protocol_serde/shape_user_input_message.rs
if let Some(var_9) = &input.model_id {
    object.key("modelId").string(var_9.as_str());  // camelCase!
}
```

**CORRECT** (what we need):
```json
{
  "userInputMessage": {
    "modelId": "claude-sonnet-4-5"  // camelCase
  }
}
```

## Fix Applied

Changed in `internal/models/conversation.go`:

```go
// BEFORE:
ModelID string `json:"model_id,omitempty"` // WRONG

// AFTER:
ModelID string `json:"modelId,omitempty"` // CORRECT
```

## Verification

### Text-Only Requests
✅ **WORKING** - Returns detailed AWS Lambda explanation

### Vision Requests  
❌ **STILL FAILING** - Returns generic response

## Current Status

- Fixed `modelId` field name (camelCase)
- Text-only requests work perfectly
- Vision requests still return generic responses
- Need to investigate other potential issues

## Next Steps

1. Compare complete JSON structure with Q CLI
2. Check if there are other missing/incorrect fields
3. Verify image encoding is correct
4. Test with actual Q CLI capture to compare byte-for-byte

## Files Modified

- `internal/models/conversation.go` - Fixed ModelID JSON tag

## Testing

```powershell
# Text-only (WORKS)
$body = '{"model":"claude-sonnet-4-5","messages":[{"role":"user","content":"What is AWS Lambda?"}],"stream":false}'
Invoke-RestMethod -Uri 'http://localhost:8090/v1/chat/completions' -Method Post -Headers $headers -Body $body

# Vision (STILL FAILS)
$body = Get-Content 'test-data/payloads/test-vision-aws.json' -Raw
Invoke-RestMethod -Uri 'http://localhost:8090/v1/chat/completions' -Method Post -Headers $headers -Body $body
```

## Conclusion

The `modelId` field name fix was necessary but not sufficient. There must be another issue with the vision request format.
