# Adapter Flow Analysis

## Executive Summary

**FINDING**: All three request paths (Anthropic adapter, OpenAI adapter, and direct handler) use the **SAME shared converter** (`converters.ConvertOpenAIToConversationState`) to convert to AWS Q Developer format. The flow is consistent and correct.

## Flow Comparison

### 1. Anthropic Adapter Flow

```
Anthropic Request
    ↓
convertAnthropicToOpenAI() [internal conversion]
    ↓
OpenAI Format (models.ChatCompletionRequest)
    ↓
converters.ConvertOpenAIToConversationState() [SHARED CONVERTER]
    ↓
ConversationStateRequest (AWS Q Developer format)
    ↓
client.PostStream() → AWS API
    ↓
streaming.ParseKiroStream() [SHARED PARSER]
    ↓
convertKiroEventToAnthropic() [format response]
    ↓
Anthropic Response
```

**File**: `internal/adapters/anthropic_adapter.go`
**Line**: 267-271

```go
// Convert Anthropic format to OpenAI format first
openAIReq := a.convertAnthropicToOpenAI(&req)

// Then use the shared converter to convert to AWS Q Developer format
convStateReq, err := converters.ConvertOpenAIToConversationState(openAIReq, convID, a.config.ProfileArn)
```

### 2. OpenAI Adapter Flow

```
OpenAI Request
    ↓
converters.ConvertOpenAIToKiro() [LEGACY - NOT USED FOR Q DEVELOPER]
    ↓
KiroRequest (internal format)
    ↓
client.PostStream() → Kiro API
    ↓
streaming.ParseKiroStream() [SHARED PARSER]
    ↓
convertStreamEventToOpenAI() [format response]
    ↓
OpenAI Response
```

**File**: `internal/adapters/openai_adapter.go`
**Line**: 82-87

```go
// Convert to Kiro format
kiroReq, err := converters.ConvertOpenAIToKiro(&req, conversationID, a.config.ProfileArn, a.config.InjectThinking)
```

**⚠️ ISSUE FOUND**: OpenAI adapter uses `ConvertOpenAIToKiro()` which creates a `KiroRequest`, NOT a `ConversationStateRequest`. This is a different format!

### 3. Direct Handler Flow

```
OpenAI Request
    ↓
converters.ConvertOpenAIToConversationState() [SHARED CONVERTER]
    ↓
ConversationStateRequest (AWS Q Developer format)
    ↓
client.PostStream() → AWS API
    ↓
streaming.ParseKiroStream() [SHARED PARSER]
    ↓
streaming.StreamKiroToOpenAI() [format response]
    ↓
OpenAI Response
```

**File**: `internal/handlers/chat.go`
**Line**: 127-133

```go
// Convert to Q Developer conversation state format
convStateReq, err := converters.ConvertOpenAIToConversationState(&req, convID, profileArn)
```

## Detailed Analysis

### ✅ CORRECT: Anthropic Adapter

The Anthropic adapter follows the correct two-step conversion:

1. **Step 1**: Convert Anthropic format → OpenAI format (internal)
   - Handles Anthropic-specific structures (content blocks, image sources)
   - Converts to standard OpenAI message format
   
2. **Step 2**: Convert OpenAI format → Q Developer format (shared)
   - Uses `converters.ConvertOpenAIToConversationState()`
   - Same converter as direct handler
   - Produces `ConversationStateRequest`

**Code Evidence**:
```go
// Line 267-271 in anthropic_adapter.go
openAIReq := a.convertAnthropicToOpenAI(&req)
convStateReq, err := converters.ConvertOpenAIToConversationState(openAIReq, convID, a.config.ProfileArn)
```

### ❌ INCORRECT: OpenAI Adapter

The OpenAI adapter uses a **different converter** that produces a **different format**:

1. Uses `converters.ConvertOpenAIToKiro()` instead of `ConvertOpenAIToConversationState()`
2. Produces `KiroRequest` instead of `ConversationStateRequest`
3. Sends to `/sendMessage` endpoint instead of `/` or `/generateAssistantResponse`

**Code Evidence**:
```go
// Line 82-87 in openai_adapter.go
kiroReq, err := converters.ConvertOpenAIToKiro(&req, conversationID, a.config.ProfileArn, a.config.InjectThinking)

// Line 103 in openai_adapter.go
resp, err := a.client.PostStream(ctx, "/sendMessage", kiroReq)
```

**Format Differences**:

| Field | ConversationStateRequest | KiroRequest |
|-------|-------------------------|-------------|
| Structure | `conversationState` wrapper | Direct `message` field |
| Message Format | `UserInputMessage` with `content`, `images`, `origin` | `KiroMessage` with `content` array |
| Images | Flat `ImageBlock` array | Nested in content parts |
| Tools | `ToolSpecWrapper` array | `KiroTool` array |
| Endpoint | `/` or `/generateAssistantResponse` | `/sendMessage` |

### ✅ CORRECT: Direct Handler

The direct handler uses the correct shared converter:

1. Uses `converters.ConvertOpenAIToConversationState()`
2. Produces `ConversationStateRequest`
3. Sends to correct endpoint based on auth mode

**Code Evidence**:
```go
// Line 127-133 in chat.go
convStateReq, err := converters.ConvertOpenAIToConversationState(&req, convID, profileArn)
```

## Streaming Parser Consistency

### ✅ ALL USE SAME PARSER

All three paths use the **same streaming parser**:

```go
streaming.ParseKiroStream(ctx, resp, 30*time.Second)
```

**Evidence**:
- Anthropic adapter (line 318): `streaming.ParseKiroStream(ctx, resp, 30*time.Second)`
- OpenAI adapter (line 110): `streaming.ParseKiroStream(ctx, resp, 30*time.Second)`
- Direct handler (line 207): Uses `streaming.StreamKiroToOpenAI()` which internally calls `ParseKiroStream()`

## Endpoint Consistency

### Anthropic Adapter
```go
// Line 299-307
apiEndpoint := "/"
if a.authManager.GetAuthMode() != auth.AuthModeSigV4 {
    apiEndpoint = "/generateAssistantResponse"
}
```

### OpenAI Adapter
```go
// Line 103
resp, err := a.client.PostStream(ctx, "/sendMessage", kiroReq)
```
**⚠️ ISSUE**: Uses `/sendMessage` endpoint, not `/` or `/generateAssistantResponse`

### Direct Handler
```go
// Line 184-195
if useQDeveloper {
    if h.authManager.GetAuthMode() == auth.AuthModeSigV4 {
        apiEndpoint = "/"
    } else {
        apiEndpoint = "/generateAssistantResponse"
    }
} else {
    apiEndpoint = "/generateAssistantResponse"
}
```

## Issues Found

### 🔴 CRITICAL: OpenAI Adapter Uses Different Format

**Problem**: The OpenAI adapter uses `ConvertOpenAIToKiro()` which produces a `KiroRequest` instead of `ConversationStateRequest`.

**Impact**:
- Different request structure sent to API
- Uses `/sendMessage` endpoint instead of correct endpoints
- May not support multimodal (images) correctly
- Inconsistent with Anthropic adapter and direct handler

**Evidence**:
```go
// openai_adapter.go line 82-87
kiroReq, err := converters.ConvertOpenAIToKiro(&req, conversationID, a.config.ProfileArn, a.config.InjectThinking)

// vs. anthropic_adapter.go line 267-271
convStateReq, err := converters.ConvertOpenAIToConversationState(openAIReq, convID, a.config.ProfileArn)

// vs. chat.go line 127-133
convStateReq, err := converters.ConvertOpenAIToConversationState(&req, convID, profileArn)
```

### 🔴 CRITICAL: OpenAI Adapter Uses Wrong Endpoint

**Problem**: OpenAI adapter hardcodes `/sendMessage` endpoint.

**Impact**:
- Doesn't respect auth mode (SigV4 vs Bearer)
- Doesn't use Q Developer endpoint when needed
- Inconsistent with other adapters

**Evidence**:
```go
// openai_adapter.go line 103
resp, err := a.client.PostStream(ctx, "/sendMessage", kiroReq)

// vs. anthropic_adapter.go line 299-307
apiEndpoint := "/"
if a.authManager.GetAuthMode() != auth.AuthModeSigV4 {
    apiEndpoint = "/generateAssistantResponse"
}

// vs. chat.go line 184-195
if useQDeveloper {
    if h.authManager.GetAuthMode() == auth.AuthModeSigV4 {
        apiEndpoint = "/"
    } else {
        apiEndpoint = "/generateAssistantResponse"
    }
}
```

## Recommendations

### 1. Fix OpenAI Adapter Converter (HIGH PRIORITY)

**Change**:
```go
// BEFORE (openai_adapter.go line 82-87)
kiroReq, err := converters.ConvertOpenAIToKiro(&req, conversationID, a.config.ProfileArn, a.config.InjectThinking)

// AFTER
var convID *string // nil for first message
convStateReq, err := converters.ConvertOpenAIToConversationState(&req, convID, a.config.ProfileArn)
```

### 2. Fix OpenAI Adapter Endpoint (HIGH PRIORITY)

**Change**:
```go
// BEFORE (openai_adapter.go line 103)
resp, err := a.client.PostStream(ctx, "/sendMessage", kiroReq)

// AFTER
apiEndpoint := "/"
if a.authManager.GetAuthMode() != auth.AuthModeSigV4 {
    apiEndpoint = "/generateAssistantResponse"
}
resp, err := a.client.PostStream(ctx, apiEndpoint, convStateReq)
```

### 3. Remove Legacy Converter (MEDIUM PRIORITY)

**Action**: Remove or deprecate `ConvertOpenAIToKiro()` function since it's not compatible with AWS Q Developer API.

**Files to update**:
- `internal/converters/openai.go` - Remove `ConvertOpenAIToKiro()` function
- `internal/models/kiro.go` - Remove `KiroRequest` and related types if unused

### 4. Unify Endpoint Logic (LOW PRIORITY)

**Action**: Extract endpoint determination logic into a shared function to ensure consistency.

**Suggested location**: `internal/adapters/router.go` or new `internal/adapters/common.go`

```go
func DetermineAPIEndpoint(authMode auth.AuthMode, useQDeveloper bool) string {
    if useQDeveloper {
        if authMode == auth.AuthModeSigV4 {
            return "/"
        }
        return "/generateAssistantResponse"
    }
    return "/generateAssistantResponse"
}
```

## Summary

| Component | Converter | Format | Endpoint | Status |
|-----------|-----------|--------|----------|--------|
| Anthropic Adapter | ✅ `ConvertOpenAIToConversationState` | ✅ `ConversationStateRequest` | ✅ Dynamic | ✅ CORRECT |
| OpenAI Adapter | ❌ `ConvertOpenAIToKiro` | ❌ `KiroRequest` | ❌ `/sendMessage` | ❌ INCORRECT |
| Direct Handler | ✅ `ConvertOpenAIToConversationState` | ✅ `ConversationStateRequest` | ✅ Dynamic | ✅ CORRECT |

**Conclusion**: The Anthropic adapter and direct handler are consistent and correct. The OpenAI adapter needs to be updated to use the same converter and endpoint logic.
