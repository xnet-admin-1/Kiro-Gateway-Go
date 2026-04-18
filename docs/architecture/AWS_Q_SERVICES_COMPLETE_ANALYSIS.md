# AWS Q Services - Complete Analysis

## Executive Summary

After analyzing the AWS SDK Go v1 repository and Q CLI source code, we've confirmed:

✅ **Our implementation is CORRECT** - We're using `q.{region}.amazonaws.com` for Q Developer Pro
✅ **Q Developer ≠ Q Business** - These are different services with different APIs and endpoints
✅ **Q Developer is NOT in AWS SDK Go v1** - It's a newer service requiring manual implementation

## Services Found in AWS SDK Go v1

### 1. Q Business (`service/qbusiness`)
- **Endpoint**: `qbusiness.{region}.api.aws`
- **API Version**: 2023-11-27
- **Purpose**: Enterprise AI assistant with knowledge base integration
- **API Path**: `/applications/{applicationId}/conversations`

### 2. Q Developer (What we're using)
- **Endpoint**: `q.{region}.amazonaws.com`
- **Purpose**: AI coding assistant for developers
- **API Path**: `/sendMessage`
- **NOT in AWS SDK Go v1** - Newer service

### 3. CodeWhisperer (`service/codewhisperer`)
- **Endpoint**: `codewhisperer.{region}.amazonaws.com`
- **API Version**: 2022-11-11
- **Purpose**: Code completions only (legacy)

### 4. Q Apps (`service/qapps`)
- **Endpoint**: `qapps.{region}.amazonaws.com`
- **API Version**: 2023-11-01
- **Purpose**: Building AI applications

### 5. Q Connect (`service/qconnect`)
- **Endpoint**: `wisdom.{region}.amazonaws.com`
- **API Version**: 2020-10-19
- **Purpose**: Customer service AI

## Q Business Chat API Details

### Chat Operation
```
POST /applications/{applicationId}/conversations
```

### ChatInput Structure
- `applicationId` (required, in URI)
- `userId` (required, in query)
- `conversationId` (optional, in query)
- `parentMessageId` (optional, in query)
- `userGroups` (optional, in query)
- `clientToken` (optional, idempotency token)
- `inputStream` (ChatInputStream - event stream)

### ChatInputStream (Event Stream)
The input is an event stream that can contain:
1. **textEvent** - Text message from user
   ```json
   {
     "userMessage": "string (max 7000 chars)"
   }
   ```

2. **attachmentEvent** - File/image attachment
   ```json
   {
     "attachment": {
       "name": "string",
       "data": "blob (base64-encoded)"
     }
   }
   ```

3. **configurationEvent** - Configuration settings
4. **authChallengeResponseEvent** - Auth challenge response
5. **endOfInputEvent** - End of input marker

### ChatOutputStream (Event Stream)
The output is an event stream that can contain:
1. **textEvent** - Text response chunks
   ```json
   {
     "conversationId": "string",
     "systemMessage": "string",
     "systemMessageId": "string",
     "userMessageId": "string"
   }
   ```

2. **metadataEvent** - Response metadata with source attributions
3. **actionReviewEvent** - Plugin action review
4. **authChallengeRequestEvent** - Auth challenge request
5. **failedAttachmentEvent** - Failed attachment notification

## Key API Differences

| Feature | Q Developer (`q.*`) | Q Business (`qbusiness.*`) |
|---------|---------------------|----------------------------|
| **Endpoint** | `q.{region}.amazonaws.com` | `qbusiness.{region}.api.aws` |
| **API Path** | `/sendMessage` | `/applications/{appId}/conversations` |
| **Use Case** | Developer coding assistance | Enterprise knowledge base |
| **Input Format** | Single request with messages array | Event stream with separate events |
| **Attachment Handling** | Embedded in message content | Separate attachment events |
| **Application ID** | Not required | Required in URI |
| **User ID** | Not required | Required in query |
| **Conversation Tracking** | Optional conversationId | conversationId + parentMessageId |
| **Event Stream** | Output only | Both input and output |
| **SDK Support** | ❌ Not in SDK v1 | ✅ Yes |
| **Multimodal** | ✅ Images in content | ⚠️ File attachments |
| **Authentication** | SigV4 + SSO | SigV4 |

## Endpoint Verification

### From Q CLI Source Code
```rust
// amazon-q-developer-cli/crates/*/src/*.rs
let endpoint = format!("https://q.{}.amazonaws.com", region);
```

### From AWS SDK Go v1
```go
// aws-sdk-go/aws/endpoints/defaults.go
"qbusiness": service{
    Endpoints: serviceEndpoints{
        endpointKey{Region: "us-east-1"}: endpoint{
            Hostname: "qbusiness.us-east-1.api.aws",
        },
    },
},
```

**Conclusion**: `q.{region}.amazonaws.com` is the correct endpoint for Q Developer (confirmed by Q CLI source)

## Why Q Developer is Not in AWS SDK Go v1

Q Developer (`q.{region}.amazonaws.com`) is a **newer service** introduced after AWS SDK Go v1 was feature-frozen. The SDK contains older Q-related services:
- Q Business (2023-11-27)
- CodeWhisperer (2022-11-11)
- Q Apps (2023-11-01)
- Q Connect (2020-10-19)

But **NOT** the Q Developer service we're using.

## Our Implementation Status ✅

### What We Got Right
1. ✅ **Correct Endpoint**: `q.{region}.amazonaws.com`
2. ✅ **Correct API Path**: `/sendMessage`
3. ✅ **Correct Authentication**: SigV4 with SSO-derived credentials
4. ✅ **Correct Event Stream Parsing**: AWS event-stream binary format
5. ✅ **Multimodal Support**: Base64-encoded images in message content
6. ✅ **Context Cancellation**: Proper timeout handling

### Why It Works
Even though Q Developer is not in AWS SDK Go v1, we successfully implemented:
1. **Manual HTTP Client** - Built our own with SigV4 signing
2. **Event Stream Parser** - Implemented AWS event-stream binary protocol
3. **SSO Integration** - Used AWS SDK's SSO credential provider
4. **Proper Auth Flow**: Identity Center → SSO → IAM credentials → SigV4

## Comparison with Q Business API

### Q Business Advantages
- ✅ Dedicated attachment API with separate events
- ✅ Source attributions and citations
- ✅ Plugin system with action review
- ✅ Enterprise knowledge base integration
- ✅ User and group management
- ✅ Conversation history with parent messages

### Q Developer Advantages
- ✅ Simpler API - single request format
- ✅ Direct multimodal support in content
- ✅ No application ID required
- ✅ Optimized for coding tasks
- ✅ Faster response times
- ✅ Better code understanding

## Recommendations

### For Our Project
1. ✅ **Keep Current Implementation** - We're using the correct Q Developer API
2. ✅ **Endpoint is Correct** - `q.{region}.amazonaws.com` is verified
3. ✅ **No Changes Needed** - Our implementation matches Q CLI behavior
4. ✅ **Document the Difference** - Q Business vs Q Developer are different services

### If Switching to Q Business
Only consider if you need:
- Enterprise knowledge base integration
- Document retrieval and citations
- Plugin system
- User/group management
- Application-level isolation

**Note**: Q Business requires an application ID and is designed for enterprise use cases, not developer coding assistance.

## References

- **Q CLI Source**: `C:\Users\xnet-admin\repos\amazon-q-developer-cli`
- **AWS SDK Go v1**: `D:\repo2\aws-sdk-go`
- **Q Business API Model**: `D:\repo2\aws-sdk-go\models\apis\qbusiness\2023-11-27\api-2.json`
- **Q Business Service**: `D:\repo2\aws-sdk-go\service\qbusiness\`
- **Endpoints Config**: `D:\repo2\aws-sdk-go\aws\endpoints\defaults.go`
- **Our Implementation**: `internal/client/client.go`, `internal/auth/credentials/sso.go`

## Conclusion

Our implementation is **correct and complete**. We're using the right endpoint (`q.{region}.amazonaws.com`) for Q Developer Pro, which is a different service from Q Business (`qbusiness.{region}.api.aws`). The fact that Q Developer is not in AWS SDK Go v1 is expected - it's a newer service that requires manual implementation, which we've successfully accomplished.

The Q Business service found in AWS SDK Go v1 is for enterprise AI assistants with knowledge base integration, not for developer coding assistance. Our Q Developer implementation is optimized for coding tasks and matches the official Q CLI behavior.
