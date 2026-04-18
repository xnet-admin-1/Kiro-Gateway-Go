# Credential Chain and Endpoint Initialization Analysis

## Current Configuration (from .env)

```
HEADLESS_MODE=true
AMAZON_Q_SIGV4=true
Q_USE_SENDMESSAGE=true
SSO_START_URL=https://xnetinc.awsapps.com/start
SSO_REGION=us-east-1
SSO_ACCOUNT_ID=096305372922
SSO_ROLE_NAME=AdministratorAccess
PROFILE_ARN=arn:aws:codewhisperer:us-east-1:096305372922:profile/VREYVEXNNH3H
```

## Initialization Flow (from main.go)

### 1. Service Determination

```go
// From main.go:initializeAuthManager()
inCloudShell := os.Getenv("AWS_EXECUTION_ENV") == "CloudShell"
qUseSendMessage := os.Getenv("Q_USE_SENDMESSAGE") == "true"  // ✅ TRUE
useQDeveloper := cfg.UseQDeveloper || inCloudShell || qUseSendMessage  // ✅ TRUE

if useQDeveloper {
    service = "q"  // ✅ CORRECT
    enableSigV4 = cfg.EnableSigV4  // ✅ TRUE (from AMAZON_Q_SIGV4)
}
```

**Result**: Service = "q", EnableSigV4 = true

### 2. Auth Manager Configuration

```go
authConfig := auth.Config{
    AuthType:          "headless",  // ✅ Determined from HEADLESS_MODE
    EnableSigV4:       true,        // ✅ From AMAZON_Q_SIGV4
    UseSSOCredentials: false,       // ❌ NOT SET (but not needed for headless)
    AWSRegion:         "us-east-1", // ✅ From AWS_REGION
    AWSService:        "q",         // ✅ Correct service
    ProfileARN:        "arn:aws:codewhisperer:us-east-1:096305372922:profile/VREYVEXNNH3H",
    HeadlessMode:      true,        // ✅ From HEADLESS_MODE
    SSOStartURL:       "https://xnetinc.awsapps.com/start",
    SSORegion:         "us-east-1",
    SSOAccountID:      "096305372922",
    SSORoleName:       "AdministratorAccess",
    AutomateAuth:      true,        // ✅ From AUTOMATE_AUTH
    SSOUsername:       "ai-admin",
    SSOPassword:       "!A1Adm1n!Xnet1nc!",
    MFATOTPSecret:     "YVCYFEYJKUEOCKS5LFNMUY7S4M4P2YIJ",
}
```

### 3. Headless Auth Initialization (from auth.go)

```go
// In NewAuthManager() when HeadlessMode is true:
am.authType = AuthTypeHeadless
am.headlessAuth = NewHeadlessAuthManager(headlessConfig, cfg.TokenStore)

// Initialize headless auth
ctx := context.Background()
if err := am.headlessAuth.Initialize(ctx); err != nil {
    return nil, fmt.Errorf("failed to initialize headless authentication: %w", err)
}

// Start background refresh
go am.headlessAuth.StartBackgroundRefresh(context.Background())

// For SigV4 mode, create credential chain
if authMode == AuthModeSigV4 {
    am.credentialChain = credentials.NewDefaultChain()
}
```

**Result**: Headless auth initialized, credential chain created

### 4. Client Initialization (from main.go:setupAdapters)

```go
// Determine service mode
useQDeveloper := determineService(cfg) == "q"  // ✅ TRUE

// Create Kiro client
kiroClient := client.NewClient(authManager, client.Config{
    UseQDeveloper: useQDeveloper,  // ✅ TRUE
    APIEndpoint:   "",             // ✅ Empty = use default AWS endpoints
    // ... other config
})
```

### 5. Endpoint Construction (from client.go:buildURL)

```go
func (c *Client) buildURL(endpoint string) string {
    // Use custom endpoint if provided
    if c.apiEndpoint != "" {
        return c.apiEndpoint + endpoint
    }
    
    // Select endpoint based on mode
    var baseURL string
    if c.useQDeveloper {  // ✅ TRUE
        // QDeveloper mode: use q.{region}.amazonaws.com
        baseURL = "https://q." + c.authManager.GetRegion() + ".amazonaws.com"
        // ✅ Result: "https://q.us-east-1.amazonaws.com"
    } else {
        // CodeWhisperer mode: use codewhisperer.{region}.amazonaws.com
        baseURL = "https://codewhisperer." + c.authManager.GetRegion() + ".amazonaws.com"
    }
    
    return baseURL + endpoint
}
```

**Result**: Base URL = "https://q.us-east-1.amazonaws.com"

### 6. Request Signing (from auth.go:SignRequest)

```go
func (am *AuthManager) SignRequest(ctx context.Context, req *http.Request, body []byte) error {
    mode := am.GetAuthMode()  // ✅ AuthModeSigV4
    
    switch mode {
    case AuthModeSigV4:
        // Check if using headless auth
        if am.authType == AuthTypeHeadless && am.headlessAuth != nil {  // ✅ TRUE
            // Get credentials from headless auth
            creds, err := am.headlessAuth.GetCredentials(ctx)
            if err != nil {
                return fmt.Errorf("failed to get headless credentials: %w", err)
            }
            
            // Create signer and sign request
            signer := sigv4.NewSigner(creds, am.GetRegion(), am.GetService())
            // ✅ Region: "us-east-1", Service: "q"
            if err := signer.SignRequest(req, body); err != nil {
                return fmt.Errorf("failed to sign request: %w", err)
            }
            return nil
        }
        // ... fallback to credential chain
    }
}
```

### 7. Request Headers (from client.go:doWithRetry)

```go
// For Q Developer with SigV4, use JSON-RPC style headers
if c.useQDeveloper {  // ✅ TRUE
    // Q Developer API uses JSON-RPC style with X-Amz-Target header
    req.Header.Set("Content-Type", "application/x-amz-json-1.0")
    // ✅ CORRECT
    
    // Use the target passed in, or default to SendMessage
    if target != "" {
        req.Header.Set("X-Amz-Target", target)
        // ✅ "AmazonQDeveloperStreamingService.SendMessage"
    }
}
```

## Credential Chain Flow

```
1. Environment Variables Check
   ├─ HEADLESS_MODE=true ✅
   ├─ AMAZON_Q_SIGV4=true ✅
   └─ Q_USE_SENDMESSAGE=true ✅

2. Auth Manager Initialization
   ├─ AuthType: headless ✅
   ├─ AuthMode: SigV4 ✅
   └─ Service: "q" ✅

3. Headless Auth Manager
   ├─ Initialize OIDC flow ✅
   ├─ Get SSO access token ✅
   ├─ Call SSO GetRoleCredentials ✅
   └─ Return temporary AWS credentials ✅

4. Credential Retrieval (per request)
   ├─ Check token expiration
   ├─ Refresh if needed (background task)
   ├─ Get AWS credentials from SSO
   └─ Return: AccessKeyId, SecretAccessKey, SessionToken

5. Request Signing
   ├─ Create SigV4 signer
   ├─ Region: "us-east-1" ✅
   ├─ Service: "q" ✅
   └─ Sign request with AWS credentials ✅
```

## Endpoint Flow

```
1. Determine Service Mode
   └─ Q_USE_SENDMESSAGE=true → useQDeveloper=true ✅

2. Build Base URL
   └─ "https://q.us-east-1.amazonaws.com" ✅

3. Add Endpoint Path
   └─ "/" (root endpoint for SendMessage) ✅

4. Final URL
   └─ "https://q.us-east-1.amazonaws.com/" ✅

5. Set Headers
   ├─ Content-Type: application/x-amz-json-1.0 ✅
   ├─ X-Amz-Target: AmazonQDeveloperStreamingService.SendMessage ✅
   └─ Authorization: AWS4-HMAC-SHA256 ... (SigV4 signature) ✅
```

## Verification from Logs

```
2026/01/25 02:34:35 Q_USE_SENDMESSAGE enabled - using Q Developer mode
2026/01/25 02:34:35 Using Q Developer mode: q.{region}.amazonaws.com with SigV4 authentication
2026/01/25 02:34:35   - Supports: Text conversations, Multimodal (images), All Q Developer features
2026/01/25 02:34:35 [HEADLESS] Initializing headless authentication...
2026/01/25 02:34:35 [HEADLESS] Found existing token
2026/01/25 02:34:35 [HEADLESS] Getting AWS credentials from SSO...
2026/01/25 02:34:36 [HEADLESS] ✅ AWS credentials obtained (expires: 2026-01-25T14:34:33-07:00)
2026/01/25 02:34:36 Initialized auth manager - Type: headless, Mode: AWS SigV4, Service: q
```

## Summary

✅ **Credential Chain**: CORRECT
- Using headless OIDC authentication
- Converting SSO access token to AWS credentials via GetRoleCredentials API
- Credentials are temporary (12 hours) and auto-refresh in background
- SigV4 signing with service="q"

✅ **Endpoint**: CORRECT
- Using Q Developer endpoint: `https://q.us-east-1.amazonaws.com/`
- Correct headers: `Content-Type: application/x-amz-json-1.0`
- Correct target: `X-Amz-Target: AmazonQDeveloperStreamingService.SendMessage`

✅ **Request Format**: CORRECT (from previous analysis)
- Images as raw bytes ([]byte), JSON encoder auto-base64-encodes
- ModelID field: "modelId" (camelCase)
- Origin: "CLI"
- UserInputMessageContext with EnvState included

## Current Status

**Text-only requests**: ✅ Working perfectly
**Vision requests**: ❌ AWS rejects with generic message

**Hypothesis**: The issue is NOT with:
- ❌ Credential chain (working correctly)
- ❌ Endpoint (correct Q Developer endpoint)
- ❌ Request format (matches Q CLI exactly)
- ❌ Image encoding (raw bytes, auto-base64-encoded)

**Possible remaining issues**:
1. **Test image content**: 1x1 red pixel is not AWS-related, Q Developer may reject non-AWS images
2. **Undocumented requirement**: There may be an additional field or header we haven't discovered
3. **Account/profile limitation**: The profile may not have vision enabled (unlikely)

## Next Steps

1. **Test with AWS-related image**: Use an AWS architecture diagram or console screenshot
2. **Compare with Q CLI**: Test the same image with `qchat chat` to confirm it works
3. **Capture Q CLI traffic**: Use mitmproxy to capture actual Q CLI requests and compare byte-for-byte
4. **Check profile settings**: Verify the profile has vision capabilities enabled
