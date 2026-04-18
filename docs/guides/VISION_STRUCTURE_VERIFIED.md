# Vision Request Structure - VERIFIED CORRECT

## Status: Structure is Correct, Still Getting 400

### What We've Verified

1. **Image Structure**: `{"format":"png","source":{"bytes":"base64..."}}` ✅ CORRECT
   - Confirmed from Q CLI serialization code (`shape_image_block.rs`)
   - The structure IS nested, not flat
   - Our implementation matches exactly

2. **Origin Field**: `"CLI"` ✅ CORRECT
   - Q CLI hardcodes this to `Origin::Cli` which serializes to `"CLI"`
   - Confirmed in `model.rs`: `.origin(amzn_qdeveloper_streaming_client::types::Origin::Cli)`

3. **Model ID**: `"claude-sonnet-4-5"` ✅ PRESENT
   - Included in `userInputMessage` as required

4. **Endpoint**: `https://q.us-east-1.amazonaws.com/` ✅ CORRECT

5. **Authentication**: SigV4+SSO ✅ WORKING
   - Text-only requests work perfectly
   - 200 OK responses for non-vision requests

### Current Issue

AWS returns: `400 Bad Request - {"__type":"com.amazon.aws.codewhisperer#ValidationException","message":"Improperly formed request."}`

### What This Means

The structure is correct based on Q CLI source code, but AWS is still rejecting it. Possible causes:

1. **Field Name Case Sensitivity**: Need to verify exact field names match
2. **Missing Required Field**: There may be an undocumented required field for vision
3. **Image Data Issue**: The base64 encoding or image bytes might have an issue
4. **Request Size**: The request might be too large
5. **Model Compatibility**: The model ID might not support vision in this context

### Next Steps

1. **Capture actual Q CLI request** using mitmproxy to see byte-for-byte what it sends
2. **Compare JSON field names** - ensure exact case match (camelCase vs snake_case)
3. **Test with smaller image** - verify it's not a size issue
4. **Check model ID format** - ensure it matches Q CLI exactly
5. **Verify all field serialization** - print actual JSON being sent

### Files Modified

- `internal/models/conversation.go` - Restored nested ImageBlock structure
- `internal/converters/conversation.go` - Updated to create nested structure
- Both now match Q CLI serialization exactly

### Test Command

```powershell
.\scripts/test/test-vision-detailed.ps1
```

### Gateway Logs Show

```
"images":[{"format":"png","source":{"bytes":"iVBORw0KGgoAAAA..."}}]
"origin":"CLI"
"modelId":"claude-sonnet-4-5"
```

Structure looks correct in logs, but AWS still rejects it.

## Critical Insight

**The structure is correct.** We've verified it matches the Q CLI source code exactly. The 400 error must be due to something else - likely a field name case issue, missing field, or data encoding problem.

**WE NEED TO CAPTURE AN ACTUAL Q CLI REQUEST** to see what's different.
