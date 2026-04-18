# Vision Debugging - Next Steps

## Current Status

- Text-only requests work perfectly ✅
- Vision requests return generic responses ❌
- JSON structure looks correct based on Q CLI source
- Tried both `modelId` (camelCase) and `model_id` (snake_case) - neither works

## What We've Verified

1. Image bytes are `[]byte` type (correct)
2. Base64 decoding works correctly
3. JSON encoding produces valid base64
4. Origin is "AI_EDITOR"
5. Model ID is included
6. Image structure has format and source fields
7. Authentication works (text requests succeed)

## The Only Way Forward

**CAPTURE ACTUAL Q CLI REQUEST** and compare byte-for-byte with our request.

## Steps to Capture

1. Start mitmproxy:
   ```powershell
   mitmdump -s scripts\capture-q-cli-traffic.py --set confdir=~/.mitmproxy
   ```

2. In another terminal, set proxy and run Q CLI:
   ```powershell
   $env:HTTPS_PROXY = "http://localhost:8080"
   $env:HTTP_PROXY = "http://localhost:8080"
   qchat chat -i
   ```

3. When prompted, type: "What color is this pixel?"

4. Drag/drop or paste: `test-data\test_diagram.png`

5. After response, stop mitmproxy (Ctrl+C)

6. Analyze capture:
   ```powershell
   .\scripts\analyze-q-cli-captures.ps1 -ShowFull
   ```

7. Compare the JSON in `q-cli-captures/*.json` with our JSON

## What to Look For

- Field name differences (camelCase vs snake_case)
- Missing or extra fields
- Different image structure
- Different enum serialization for ImageSource
- Any other structural differences

## Current Request (Ours)

```json
{
  "conversationState": {
    "conversationId": null,
    "currentMessage": {
      "userInputMessage": {
        "content": "What color is this 1x1 pixel image?",
        "images": [{
          "format": "png",
          "source": {
            "bytes": "iVBORw0KG..."
          }
        }],
        "origin": "AI_EDITOR",
        "model_id": "claude-sonnet-4-5"
      }
    },
    "chatTriggerType": "MANUAL"
  },
  "profileArn": "arn:aws:codewhisperer:us-east-1:096305372922:profile/VREYVEXNNH3H"
}
```

This MUST be compared with actual Q CLI JSON to find the difference.
