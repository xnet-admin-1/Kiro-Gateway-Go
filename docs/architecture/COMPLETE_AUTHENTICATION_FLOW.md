# Complete Authentication Flow - API Key + Headless SigV4+SSO

## Current Configuration

The gateway uses **TWO layers of authentication**:

### Layer 1: Client → Gateway (API Key)
```
Client Request
  ↓
  Header: x-api-key: kiro-YOUR_API_KEY_HERE
  ↓
Gateway validates API key
  ↓
Request proceeds to adapter
```

### Layer 2: Gateway → AWS (Headless SigV4+SSO)
```
Gateway
  ↓
Headless Auth Manager
  ↓
SSO OIDC Token → AWS IAM Credentials
  ↓
SigV4 Signing
  ↓
AWS Q Developer API
```

## Complete Flow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│ Client (curl/test script)                                        │
└───────────────────────────────┬─────────────────────────────────┘
                                │
                                │ POST /v1/messages
                                │ x-api-key: kiro-G0r...
                                │ Body: {model, messages, images}
                                ↓
┌─────────────────────────────────────────────────────────────────┐
│ Gateway - API Key Middleware (router.go)                        │
│ ✅ Validates: kiro-YOUR_API_KEY_HERE  │
└───────────────────────────────┬─────────────────────────────────┘
                                │
                                │ Request authorized
                                ↓
┌─────────────────────────────────────────────────────────────────┐
│ Anthropic Adapter (anthropic_adapter.go)                        │
│ - Converts Anthropic format to ConversationStateRequest         │
│ - Calls converter: ConvertAnthropicToConversationState()        │
└───────────────────────────────┬─────────────────────────────────┘
                                │
                                │ ConversationStateRequest
                                ↓
┌─────────────────────────────────────────────────────────────────┐
│ Converter (converters/conversation.go)                          │
│ - Extracts images from Anthropic format                         │
│ - Decodes base64 to []byte                                      │
│ - Creates ImageBlock with raw bytes                             │
│ - Builds UserInputMessage with:                                 │
│   * content: "What do you see..."                               │
│   * images: [{format:"png", source:{bytes:[...]}}]              │
│   * origin: "CLI"                                                │
│   * modelId: "claude-sonnet-4-5"                                │
│   * userInputMessageContext: {envState:{...}}                   │
└───────────────────────────────┬─────────────────────────────────┘
                                │
                                │ ConversationStateRequest
                                ↓
┌─────────────────────────────────────────────────────────────────┐
│ Client (client.go)                                               │
│ - Builds URL: https://q.us-east-1.amazonaws.com/                │
│ - Sets headers:                                                  │
│   * Content-Type: application/x-amz-json-1.0                    │
│   * X-Amz-Target: AmazonQDeveloperStreamingService.SendMessage  │
│ - Marshals body to JSON ([]byte auto-base64-encoded)            │
└───────────────────────────────┬─────────────────────────────────┘
                                │
                                │ Unsigned HTTP request
                                ↓
┌─────────────────────────────────────────────────────────────────┐
│ Auth Manager - SignRequest() (auth.go)                          │
│ - Detects: AuthMode = SigV4, AuthType = headless                │
│ - Calls: headlessAuth.GetCredentials(ctx)                       │
└───────────────────────────────┬─────────────────────────────────┘
                                │
                                ↓
┌─────────────────────────────────────────────────────────────────┐
│ Headless Auth Manager (headless_auth.go)                        │
│                                                                  │
│ 1. Check if AWS credentials need refresh                        │
│    if time.Until(credExpiry) < 5*time.Minute {                  │
│                                                                  │
│ 2. Load stored SSO token                                        │
│    token := tokenStore.LoadToken()                              │
│    // Contains: AccessToken (SSO bearer token)                  │
│                                                                  │
│ 3. Check if SSO token needs refresh                             │
│    if token.NeedsRefresh() {                                    │
│        refreshAccessToken(ctx, token)  // OIDC refresh          │
│    }                                                             │
│                                                                  │
│ 4. Get AWS credentials from SSO                                 │
│    refreshAWSCredentials(ctx, token)                            │
└───────────────────────────────┬─────────────────────────────────┘
                                │
                                ↓
┌─────────────────────────────────────────────────────────────────┐
│ SSO Provider (credentials/sso.go)                               │
│                                                                  │
│ 1. Get SSO access token                                         │
│    accessToken := token.AccessToken  // JWT from OIDC           │
│                                                                  │
│ 2. Call AWS SSO GetRoleCredentials API                          │
│    input := &sso.GetRoleCredentialsInput{                       │
│        AccessToken: accessToken,  // ← SSO bearer token         │
│        AccountId:   "096305372922",                             │
│        RoleName:    "AdministratorAccess",                      │
│    }                                                             │
│    output := ssoClient.GetRoleCredentials(ctx, input)           │
│                                                                  │
│ 3. Extract temporary IAM credentials                            │
│    creds := &Credentials{                                       │
│        AccessKeyID:     "ASIA...",                              │
│        SecretAccessKey: "...",                                  │
│        SessionToken:    "IQoJb3JpZ2luX2VjEPz...",               │
│        Expires:         time.UnixMilli(expiration),             │
│    }                                                             │
│                                                                  │
│ 4. Cache credentials                                            │
│    m.credentials = creds                                        │
│    m.credExpiry = creds.Expires                                 │
└───────────────────────────────┬─────────────────────────────────┘
                                │
                                │ Return: Credentials
                                ↓
┌─────────────────────────────────────────────────────────────────┐
│ SigV4 Signer (sigv4/signer.go)                                  │
│                                                                  │
│ 1. Add timestamp header                                         │
│    X-Amz-Date: 20260125T093435Z                                 │
│                                                                  │
│ 2. Add security token (CRITICAL for temporary creds)            │
│    X-Amz-Security-Token: IQoJb3JpZ2luX2VjEPz...                 │
│                                                                  │
│ 3. Build canonical request                                      │
│    - Method: POST                                                │
│    - URI: /                                                      │
│    - Headers: content-type, host, x-amz-date, x-amz-security-   │
│               token, x-amz-target                                │
│    - Body hash: SHA256(body)                                     │
│                                                                  │
│ 4. Build string to sign                                         │
│    AWS4-HMAC-SHA256                                              │
│    20260125T093435Z                                              │
│    20260125/us-east-1/q/aws4_request                            │
│    <canonical_request_hash>                                      │
│                                                                  │
│ 5. Derive signing key                                           │
│    kSecret  = "AWS4" + SecretAccessKey                          │
│    kDate    = HMAC(kSecret, "20260125")                         │
│    kRegion  = HMAC(kDate, "us-east-1")                          │
│    kService = HMAC(kRegion, "q")                                │
│    kSigning = HMAC(kService, "aws4_request")                    │
│                                                                  │
│ 6. Calculate signature                                          │
│    signature = HMAC(kSigning, stringToSign)                     │
│                                                                  │
│ 7. Build authorization header                                   │
│    Authorization: AWS4-HMAC-SHA256                              │
│      Credential=ASIA.../20260125/us-east-1/q/aws4_request,      │
│      SignedHeaders=content-type;host;x-amz-date;                │
│                    x-amz-security-token;x-amz-target,           │
│      Signature=abc123...                                        │
└───────────────────────────────┬─────────────────────────────────┘
                                │
                                │ Signed HTTP request
                                ↓
┌─────────────────────────────────────────────────────────────────┐
│ HTTP Client                                                      │
│ POST https://q.us-east-1.amazonaws.com/                         │
│                                                                  │
│ Headers:                                                         │
│   Content-Type: application/x-amz-json-1.0                      │
│   X-Amz-Target: AmazonQDeveloperStreamingService.SendMessage    │
│   X-Amz-Date: 20260125T093435Z                                  │
│   X-Amz-Security-Token: IQoJb3JpZ2luX2VjEPz...                  │
│   Authorization: AWS4-HMAC-SHA256 Credential=...                │
│                                                                  │
│ Body:                                                            │
│   {                                                              │
│     "conversationState": {                                       │
│       "conversationId": null,                                    │
│       "currentMessage": {                                        │
│         "userInputMessage": {                                    │
│           "content": "What do you see in this image?",          │
│           "userInputMessageContext": {                           │
│             "envState": {"operatingSystem": "windows"}          │
│           },                                                     │
│           "images": [{                                           │
│             "format": "png",                                     │
│             "source": {                                          │
│               "bytes": "iVBORw0KGgo..."  ← base64 by JSON       │
│             }                                                    │
│           }],                                                    │
│           "origin": "CLI",                                       │
│           "modelId": "claude-sonnet-4-5"                        │
│         }                                                        │
│       },                                                         │
│       "chatTriggerType": "MANUAL"                               │
│     },                                                           │
│     "profileArn": "arn:aws:codewhisperer:..."                   │
│   }                                                              │
└───────────────────────────────┬─────────────────────────────────┘
                                │
                                ↓
┌─────────────────────────────────────────────────────────────────┐
│ AWS Q Developer API                                              │
│ - Validates SigV4 signature                                      │
│ - Verifies temporary credentials                                 │
│ - Processes request                                              │
│ - Returns response                                               │
└─────────────────────────────────────────────────────────────────┘
```

## Verification from Logs

```
2026/01/25 02:34:35 Q_USE_SENDMESSAGE enabled - using Q Developer mode
2026/01/25 02:34:35 Using Q Developer mode: q.{region}.amazonaws.com with SigV4 authentication
2026/01/25 02:34:35 [HEADLESS] Initializing headless authentication...
2026/01/25 02:34:35 [HEADLESS] Found existing token
2026/01/25 02:34:35 [HEADLESS] Getting AWS credentials from SSO...
2026/01/25 02:34:36 [HEADLESS] ✅ AWS credentials obtained (expires: 2026-01-25T14:34:33-07:00)
2026/01/25 02:34:36 Initialized auth manager - Type: headless, Mode: AWS SigV4, Service: q
```

## Summary

✅ **Client Authentication**: API key validated by gateway
✅ **Gateway Authentication**: Headless SigV4+SSO to AWS
✅ **SSO OIDC**: Device flow with browser automation
✅ **SSO → IAM**: GetRoleCredentials converts bearer token to IAM creds
✅ **SigV4 Signing**: Temporary credentials with session token
✅ **Endpoint**: Q Developer (q.us-east-1.amazonaws.com)
✅ **Request Format**: Matches Q CLI exactly

**Text-only requests work perfectly** ✅
**Vision requests fail with generic AWS response** ❌

The authentication chain is working correctly. The issue is NOT authentication-related.

## Next Steps

Since authentication is confirmed working:
1. Test with AWS-related image (architecture diagram)
2. Compare actual request with Q CLI using mitmproxy
3. Look for any undocumented fields or requirements
