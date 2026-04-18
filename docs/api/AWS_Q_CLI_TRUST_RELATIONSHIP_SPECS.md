# AWS Q Developer CLI Authentication Handshake - Complete Specification

## Executive Summary

The Kiro Gateway is currently returning **500 Internal Server Error** because it's missing the complete authentication handshake that Q Developer CLI uses. This document specifies the exact handshake process needed to establish proper API access.

## Current Problem

The gateway successfully:
- ✅ Authenticates with AWS (Bearer Token or SigV4)
- ✅ Signs requests correctly
- ✅ Reaches the API endpoints

But fails because:
- ❌ Missing conversation state structure
- ❌ Missing required request fields
- ❌ Missing proper message formatting
- ❌ Missing tool specifications in requests

## Complete Authentication Handshake Flow

### 1. Initial Setup (Already Implemented ✅)

**Bearer Token Mode (Builder ID / Identity Center):**
```
1. Device Registration → Get ClientID + ClientSecret
2. Device Authorization → Get DeviceCode + UserCode
3. User authorizes in browser
4. Poll for token → Get AccessToken + RefreshToken
5. Store token in SQLite/Keychain
```

**SigV4 Mode (AWS Credentials):**
```
1. Load credentials from chain (env, profile, IMDS, etc.)
2. Sign requests with AWS Signature Version 4
```

### 2. API Request Structure (MISSING - NEEDS IMPLEMENTATION)

#### CodeWhisperer Endpoint (`generateAssistantResponse`)

**Endpoint:** `https://codewhisperer.{region}.amazonaws.com`

**Headers:**
```
POST / HTTP/1.1
Host: codewhisperer.{region}.amazonaws.com
Content-Type: application/x-amz-json-1.1
X-Amz-Target: AmazonCodeWhispererStreamingService.GenerateAssistantResponse
x-amzn-codewhisperer-optout: false
Authorization: Bearer {token}  (or AWS4-HMAC-SHA256 signature)
```

**Request Body:**
```json
{
  "conversationState": {
    "conversationId": "conv_1234567890",  // UUID or null for first message
    "currentMessage": {
      "userInputMessage": {
        "content": "User's question here",
        "userInputMessageContext": {
          "editorState": {
            "document": {
              "relativeFilePath": "path/to/file.go",
              "programmingLanguage": {
                "languageName": "go"
              },
              "text": "file content if relevant",
              "cursorState": {
                "position": {
                  "line": 10,
                  "character": 5
                }
              }
            }
          }
        },
        "userIntent": null
      }
    },
    "chatTriggerType": "MANUAL",
    "history": [
      {
        "userInputMessage": {
          "content": "Previous user message",
          "userInputMessageContext": null,
          "userIntent": null
        }
      },
      {
        "assistantResponseMessage": {
          "content": "Previous assistant response",
          "messageId": "msg_abc123"
        }
      }
    ]
  },
  "profileArn": "arn:aws:iam::123456789012:role/MyRole"  // Only for Identity Center mode
}
```

#### Q Developer Endpoint (`sendMessage`)

**Endpoint:** `https://q.{region}.amazonaws.com`

**Headers:**
```
POST / HTTP/1.1
Host: q.{region}.amazonaws.com
Content-Type: application/x-amz-json-1.1
X-Amz-Target: QDeveloperService.SendMessage
x-amzn-codewhisperer-optout: false
Authorization: AWS4-HMAC-SHA256 Credential=...  (SigV4 only)
```

**Request Body:** (Same structure as CodeWhisperer)

### 3. Conversation State Management (MISSING)

**Key Fields:**

1. **conversationId**: 
   - `null` or omitted for first message
   - UUID string for subsequent messages
   - Must be consistent throughout conversation
   - Format: `conv_{timestamp}` or UUID

2. **currentMessage**:
   - Always contains the user's current input
   - Can include context (file content, cursor position, etc.)
   - Can include images (multimodal)
   - Can include tool results

3. **history**:
   - Array of previous user/assistant message pairs
   - Limited to last N messages (typically 10-20 pairs)
   - Must alternate: user → assistant → user → assistant
   - Each assistant message includes `messageId` (utterance_id)

4. **chatTriggerType**:
   - `MANUAL`: User-initiated chat
   - `INLINE_CHAT`: Inline code suggestions
   - `DIAGNOSTIC`: Error/warning diagnostics

### 4. Tool Specifications (MISSING)

When tools are available, they must be included in the request:

```json
{
  "conversationState": { ... },
  "profileArn": "...",
  "customizationArn": null,
  "supplementalContexts": [],
  "tools": [
    {
      "toolSpecification": {
        "name": "read_file",
        "description": "Read the contents of a file",
        "inputSchema": {
          "json": "{\"type\":\"object\",\"properties\":{\"path\":{\"type\":\"string\",\"description\":\"File path\"}},\"required\":[\"path\"]}"
        }
      }
    },
    {
      "toolSpecification": {
        "name": "execute_command",
        "description": "Execute a shell command",
        "inputSchema": {
          "json": "{\"type\":\"object\",\"properties\":{\"command\":{\"type\":\"string\"}},\"required\":[\"command\"]}"
        }
      }
    }
  ]
}
```

### 5. Tool Use Flow (MISSING)

**Step 1: Model requests tool use**
```json
{
  "assistantResponseMessage": {
    "content": "I'll read that file for you.",
    "messageId": "msg_123",
    "toolUses": [
      {
        "toolUseId": "tool_abc",
        "name": "read_file",
        "input": "{\"path\":\"main.go\"}"
      }
    ]
  }
}
```

**Step 2: Execute tool and send results**
```json
{
  "conversationState": {
    "conversationId": "conv_1234567890",
    "currentMessage": {
      "userInputMessage": {
        "content": "",  // Can be empty when only tool results
        "toolResults": [
          {
            "toolUseId": "tool_abc",
            "content": [
              {
                "text": "package main\n\nfunc main() { ... }"
              }
            ],
            "status": "success"
          }
        ]
      }
    },
    "history": [ ... ]
  }
}
```

**Step 3: Model responds with tool results**

### 6. Streaming Response Handling (Already Implemented ✅)

The gateway already handles streaming responses correctly using AWS Event Stream protocol.

## Implementation Checklist

### Phase 1: Basic Conversation State (HIGH PRIORITY)

- [ ] Add `ConversationState` struct to `internal/models/`
- [ ] Implement conversation ID generation (UUID)
- [ ] Add conversation history management
- [ ] Update request converter to include conversation state
- [ ] Add `chatTriggerType` field (default: "MANUAL")

### Phase 2: Message Context (MEDIUM PRIORITY)

- [ ] Add `UserInputMessageContext` struct
- [ ] Implement editor state tracking (optional)
- [ ] Add file context support
- [ ] Add cursor position tracking

### Phase 3: Tool Support (MEDIUM PRIORITY)

- [ ] Add `Tool` and `ToolSpecification` structs
- [ ] Implement tool schema conversion
- [ ] Add tool use request handling
- [ ] Add tool result response handling
- [ ] Implement tool use flow state machine

### Phase 4: Profile ARN (LOW PRIORITY)

- [ ] Add profile ARN detection for Identity Center mode
- [ ] Only include profileArn when using Identity Center auth
- [ ] Omit profileArn for Builder ID mode

### Phase 5: Advanced Features (OPTIONAL)

- [ ] Multimodal support (images)
- [ ] Supplemental contexts
- [ ] Customization ARN support
- [ ] Diagnostic trigger support

## Minimal Working Implementation

To get a **200 OK** response, implement this minimal request:

```json
{
  "conversationState": {
    "conversationId": null,
    "currentMessage": {
      "userInputMessage": {
        "content": "Hello, can you help me?",
        "userInputMessageContext": null,
        "userIntent": null
      }
    },
    "chatTriggerType": "MANUAL",
    "history": null
  }
}
```

**For Identity Center mode, add:**
```json
{
  "conversationState": { ... },
  "profileArn": "arn:aws:iam::096305372922:role/AdministratorAccess"
}
```

## Code Changes Required

### 1. Update `internal/models/kiro.go`

Add complete conversation state structures matching Q CLI format.

### 2. Update `internal/converters/openai.go`

Convert OpenAI format to Q Developer conversation state format.

### 3. Update `internal/handlers/chat.go`

- Generate conversation IDs
- Manage conversation history
- Include conversation state in requests

### 4. Update `cmd/kiro-gateway/main.go`

- Add conversation state storage (in-memory or Redis)
- Add profile ARN configuration

## Testing Strategy

### Test 1: Minimal Request
```bash
curl -X POST http://localhost:8090/v1/chat/completions \
  -H "Authorization: Bearer test-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "anthropic.claude-3-5-sonnet-20241022-v2:0",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": false
  }'
```

**Expected:** 200 OK with assistant response

### Test 2: Conversation History
```bash
# Send first message, capture conversation_id
# Send second message with same conversation_id
# Verify history is maintained
```

### Test 3: Tool Use
```bash
# Send message that triggers tool use
# Verify tool use request format
# Send tool results
# Verify final response
```

## References

- Q CLI Source: `aws/kiro-q/amazon-q-developer-cli/crates/chat-cli/src/api_client/`
- Conversation State: `conversation_state.rs`
- Streaming Client: `clients/streaming_client.rs`
- Models: `model/`

## Next Steps

1. **Implement Phase 1** (Basic Conversation State) - This will fix the 500 errors
2. **Test with xnet-admin profile** - Verify 200 OK responses
3. **Implement Phase 2** (Message Context) - Add file context support
4. **Implement Phase 3** (Tool Support) - Enable tool use capabilities

## Success Criteria

✅ Gateway returns 200 OK instead of 500 Internal Server Error
✅ Assistant responses are received and streamed correctly
✅ Conversation history is maintained across messages
✅ Profile ARN is included for Identity Center mode
✅ Tool use flow works end-to-end (future)
