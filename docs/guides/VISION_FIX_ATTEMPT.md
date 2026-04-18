# Vision Fix Attempt - Base64 String vs Bytes

## Date
January 25, 2026

## Issue Reported
User reports that the Python version of kiro-gateway works with vision/multimodal requests, but the Go version does not.

## Investigation

### Python Implementation
**File**: `kiro-gateway-py/kiro/converters_core.py`

The Python version keeps the base64 data as a **string**:
```python
kiro_images.append({
    "format": format_str,
    "source": {
        "bytes": data  # <-- String (base64-encoded)
    }
})
```

### Go Implementation (Original)
**File**: `internal/models/conversation.go`

The Go version was using `[]byte`:
```go
type ImageSource struct {
	Bytes []byte `json:"bytes,omitempty"`
}
```

This caused the converter to:
1. Decode base64 string to `[]byte`
2. JSON encoder re-encodes `[]byte` back to base64

## Fix Applied

### Changed Model Definition
```go
type ImageSource struct {
	Bytes string `json:"bytes,omitempty"`  // Changed from []byte to string
}
```

### Changed Converter
```go
// Keep base64 string as-is (don't decode)
base64Data := parts[1]

return &models.ImageBlock{
	Format: format,
	Source: models.ImageSource{
		Bytes: base64Data,  // Now a string, not []byte
	},
}
```

## Test Results

After the fix, the request still shows the same structure in logs:
```json
{
  "format": "png",
  "source": {
    "bytes": "iVBORw0KGgoAAAANSUhEUgAAAA..."
  }
}
```

Response: Still getting "I'm not able to see any architecture diagram"

## Next Steps

Need to verify:
1. **Test Python version directly** - Confirm it actually works with vision
2. **Compare actual JSON payloads** - Byte-by-byte comparison of Python vs Go output
3. **Check for other differences** - Origin field, modelId, or other fields

## Hypothesis

The issue may not be the base64 encoding at all. It could be:
- Different `origin` field value ("AI_EDITOR" vs "CLI")
- Different `modelId` field value
- Different request structure in history vs current message
- Python version may also not be working (need to verify user's claim)

