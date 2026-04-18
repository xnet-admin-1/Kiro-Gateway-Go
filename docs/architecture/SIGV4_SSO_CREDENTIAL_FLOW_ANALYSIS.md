# SigV4+SSO Credential Flow Analysis

## Overview

This document traces the complete SigV4+SSO credential flow from initialization through request signing.

## Configuration (from .env)

```bash
HEADLESS_MODE=true              # Enable headless SSO OIDC
AMAZON_Q_SIGV4=true             # Enable SigV4 signing
Q_USE_SENDMESSAGE=true          # Use Q Developer endpoint
SSO_START_URL=https://xnetinc.awsapps.com/start
SSO_REGION=us-east-1
SSO_ACCOUNT_ID=096305372922
SSO_ROLE_NAME=AdministratorAccess
PROFILE_ARN=arn:aws:codewhisperer:us-east-1:096305372922:profile/VREYVEXNNH3H
```

## Complete Flow

### Phase 1: Initialization (main.go → auth.go)

```go
// 1. Determine service and auth mode
service = "q"                    // ✅ From Q_USE_SENDMESSAGE=true
enableSigV4 = true               // ✅ From AMAZON_Q_SIGV4=true

// 2. Create auth config
authConfig := auth.Config{
    AuthType:      "headless",   // ✅ From HEADLESS_MODE=true
    EnableSigV4:   true,          // ✅ SigV4 signing enabled
    AWSRegion:     "us-east-1",   // ✅ Region for SSO and signing
    AWSService:    "q",           // ✅ Service name for SigV4
    HeadlessMode:  true,
    SSOStartURL:   "https://xnetinc.awsapps.com/start",
    SSORegion:     "us-east-1",
    SSOAccountID:  "096305372922",
    SSORoleName:   "AdministratorAccess",
    AutomateAuth:  true,
    // ... automation credentials
}

// 3. Create auth manager
authManager := auth.NewAuthManager(authConfig)
```

### Phase 2: Headless Auth Initialization (auth.go → headless_auth.go)

```go
// In NewAuthManager() when HeadlessMode=true:

// 1. Override auth type
am.authType = AuthTypeHeadless

// 2. Create headless auth manager
am.headlessAuth = NewHeadlessAuthManager(headlessConfig, tokenStore)

// 3. Initialize headless auth
ctx := context.Background()
am.headlessAuth.Initialize(ctx)
    ↓
    // In Initialize():
    // a. Try to load existing token
    token, err := m.tokenStore.LoadToken()
    
    // b. If token exists and valid:
    if !token.IsExpired() {
        // Use existing token
        return m.refreshAWSCredentials(ctx, token)
    }
    
    // c. If token expired or missing:
    return m.performDeviceFlow(ctx)

// 4. For SigV4 mode, create credential chain
if authMode == AuthModeSigV4 {
    am.credentialChain = credentials.NewDefaultChain()
}

// 5. Start background refresh
go am.headlessAuth.StartBackgroundRefresh(context.Background())
```

### Phase 3: SSO OIDC Device Flow (headless_auth.go)

```go
// In performDeviceFlow():

// 1. Register OIDC client with AWS IAM Identity Center
registration := m.oidcClient.RegisterClient(ctx)
// Returns: ClientID, ClientSecret, ExpiresAt

// 2. Start device authorization
authResp := m.oidcClient.StartDeviceAuthorization(ctx)
// Returns: DeviceCode, UserCode, VerificationUri, Interval, ExpiresIn

// 3. User authorization (automated or manual)
if m.shouldAutomateAuth() {
    // Browser automation: navigate, login, MFA, approve
    m.automateAuthorization(ctx, authResp)
} else {
    // Display instructions to user
    m.displayAuthInstructions(authResp)
}

// 4. Poll for token
tokenResp := m.oidcClient.PollForToken(ctx, authResp.DeviceCode, authResp.Interval)
// Returns: AccessToken, RefreshToken, TokenType, ExpiresIn

// 5. Save token
token := &StoredToken{
    AccessToken:  tokenResp.AccessToken,     // ✅ SSO access token (bearer)
    RefreshToken: tokenResp.RefreshToken,    // ✅ For token refresh
    ExpiresAt:    time.Now().Add(tokenResp.ExpiresIn),
    ClientID:     registration.ClientID,
    ClientSecret: registration.ClientSecret,
    Region:       "us-east-1",
    StartURL:     "https://xnetinc.awsapps.com/start",
    AccountID:    "096305372922",
    RoleName:     "AdministratorAccess",
}
m.tokenStore.SaveToken(token)

// 6. Get AWS credentials from SSO
return m.refreshAWSCredentials(ctx, token)
```

### Phase 4: SSO to AWS Credentials Conversion (headless_auth.go → credentials/sso.go)

```go
// In refreshAWSCredentials():

// 1. Create SSO provider
ssoProvider := credentials.NewSSOProvider(credentials.SSOProviderConfig{
    Region:    "us-east-1",
    AccountID: "096305372922",
    RoleName:  "AdministratorAccess",
    AccessTokenProvider: func() (string, error) {
        return token.AccessToken, nil  // ✅ SSO access token
    },
})

// 2. Retrieve AWS credentials
creds := ssoProvider.Retrieve(ctx)
    ↓
    // In SSOProvider.Retrieve():
    
    // a. Get SSO access token
    accessToken := p.accessTokenProvider()  // ✅ Bearer token from OIDC
    
    // b. Call AWS SSO GetRoleCredentials API
    input := &sso.GetRoleCredentialsInput{
        AccessToken: aws.String(accessToken),  // ✅ SSO bearer token
        AccountId:   aws.String("096305372922"),
        RoleName:    aws.String("AdministratorAccess"),
    }
    
    output := p.ssoClient.GetRoleCredentials(ctx, input)
    
    // c. Extract temporary IAM credentials
    creds := &Credentials{
        AccessKeyID:     output.RoleCredentials.AccessKeyId,     // ✅ AWS_ACCESS_KEY_ID
        SecretAccessKey: output.RoleCredentials.SecretAccessKey, // ✅ AWS_SECRET_ACCESS_KEY
        SessionToken:    output.RoleCredentials.SessionToken,    // ✅ AWS_SESSION_TOKEN
        Expires:         time.UnixMilli(output.RoleCredentials.Expiration),
        Source:          "SSOProvider",
        CanExpire:       true,
    }
    
    return creds

// 3. Store credentials in memory
m.credentials = creds
m.credExpiry = creds.Expires

// Result: Temporary IAM credentials valid for ~12 hours
```

### Phase 5: Request Signing (auth.go → sigv4/signer.go)

```go
// When a request is made:

// 1. Get credentials (with auto-refresh)
creds := am.headlessAuth.GetCredentials(ctx)
    ↓
    // In GetCredentials():
    
    // a. Check if credentials need refresh (within 5 min of expiry)
    if time.Until(m.credExpiry) < 5*time.Minute {
        // Refresh access token if needed
        if token.NeedsRefresh() {
            m.refreshAccessToken(ctx, token)  // OIDC token refresh
        }
        // Get fresh AWS credentials
        m.refreshAWSCredentials(ctx, token)   // SSO GetRoleCredentials
    }
    
    // b. Return current credentials
    return m.credentials

// 2. Create SigV4 signer
signer := sigv4.NewSigner(creds, "us-east-1", "q")

// 3. Sign request
signer.SignRequest(req, body)
    ↓
    // In SignRequest():
    
    // a. Add timestamp header
    req.Header.Set("X-Amz-Date", timestamp.Format("20060102T150405Z"))
    
    // b. Add security token (for temporary credentials)
    req.Header.Set("X-Amz-Security-Token", creds.SessionToken)  // ✅ CRITICAL
    
    // c. Build canonical request
    canonicalRequest := buildCanonicalRequest(req, body)
    
    // d. Build string to sign
    stringToSign := buildStringToSign(timestamp, "us-east-1", "q", canonicalRequest)
    
    // e. Derive signing key
    signingKey := deriveSigningKey(creds.SecretAccessKey, "us-east-1", "q", timestamp)
    
    // f. Calculate signature
    signature := calculateSignature(signingKey, stringToSign)
    
    // g. Build authorization header
    authHeader := fmt.Sprintf(
        "AWS4-HMAC-SHA256 Credential=%s/%s/%s/%s/aws4_request, SignedHeaders=%s, Signature=%s",
        creds.AccessKeyID,
        timestamp.Format("20060102"),
        "us-east-1",
        "q",
        signedHeaders,
        signature,
    )
    
    // h. Add authorization header
    req.Header.Set("Authorization", authHeader)
```

### Phase 6: Request Execution (client.go)

```go
// In Client.doWithRetry():

// 1. Build URL
url := "https://q.us-east-1.amazonaws.com/"  // ✅ Q Developer endpoint

// 2. Create request
req := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))

// 3. Set headers (BEFORE signing)
req.Header.Set("Content-Type", "application/x-amz-json-1.0")
req.Header.Set("X-Amz-Target", "AmazonQDeveloperStreamingService.SendMessage")

// 4. Sign request (adds Authorization, X-Amz-Date, X-Amz-Security-Token)
authManager.SignRequest(ctx, req, body)

// 5. Execute request
resp := httpClient.Do(req)
```

## Credential Types at Each Stage

```
Stage 1: OIDC Device Flow
├─ Input:  User authorization (browser)
└─ Output: SSO Access Token (bearer token)
           Format: JWT
           Lifetime: ~8 hours
           Purpose: Authenticate to AWS SSO

Stage 2: SSO GetRoleCredentials
├─ Input:  SSO Access Token
└─ Output: Temporary IAM Credentials
           - AccessKeyID:     ASIA...
           - SecretAccessKey: (secret)
           - SessionToken:    (long token)
           Lifetime: ~12 hours
           Purpose: Sign AWS API requests

Stage 3: SigV4 Signing
├─ Input:  Temporary IAM Credentials
└─ Output: Signed HTTP Request
           Headers:
           - Authorization: AWS4-HMAC-SHA256 Credential=...
           - X-Amz-Date: 20260125T...
           - X-Amz-Security-Token: (session token)
```

## Key Differences: Bearer Token vs SigV4+SSO

### Bearer Token Mode (NOT USED)
```
Request → Add "Authorization: Bearer <token>" → AWS
         ↑
         SSO Access Token (JWT)
```

### SigV4+SSO Mode (CURRENT)
```
Request → Sign with IAM Credentials → AWS
         ↑                            ↑
         Temporary IAM Creds          |
         ↑                            |
         SSO GetRoleCredentials       |
         ↑                            |
         SSO Access Token (JWT)       |
         ↑                            |
         OIDC Device Flow             |
```

## Verification from Logs

```
[HEADLESS] Initializing headless authentication...
[HEADLESS] Found existing token
[HEADLESS] Getting AWS credentials from SSO...
[DEBUG] Getting access token from provider...
[DEBUG] Got access token from provider (length: 230)
[DEBUG] Calling SSO GetRoleCredentials API (account: 096305372922, role: AdministratorAccess, region: us-east-1)
[DEBUG] Successfully retrieved SSO credentials
[DEBUG] SSO credentials expire at: 2026-01-25T14:34:33-07:00 (in 11h59m56s)
[HEADLESS] ✅ AWS credentials obtained (expires: 2026-01-25T14:34:33-07:00)
Initialized auth manager - Type: headless, Mode: AWS SigV4, Service: q
```

## Request Headers (Final)

```http
POST / HTTP/1.1
Host: q.us-east-1.amazonaws.com
Content-Type: application/x-amz-json-1.0
X-Amz-Target: AmazonQDeveloperStreamingService.SendMessage
X-Amz-Date: 20260125T093435Z
X-Amz-Security-Token: IQoJb3JpZ2luX2VjEPz//////////...  ← Temporary session token
Authorization: AWS4-HMAC-SHA256 Credential=ASIA.../20260125/us-east-1/q/aws4_request, SignedHeaders=content-type;host;x-amz-date;x-amz-security-token;x-amz-target, Signature=abc123...
```

## Background Refresh

```go
// Runs every 1 minute in background:

1. Check if SSO access token needs refresh (within 5 min of expiry)
   ├─ If yes: Call OIDC RefreshToken API
   └─ Update stored token

2. Check if AWS credentials need refresh (within 5 min of expiry)
   ├─ If yes: Call SSO GetRoleCredentials API
   └─ Update cached credentials

3. Sleep 1 minute, repeat
```

## Summary

✅ **SSO OIDC Flow**: Working correctly
- Device authorization with IAM Identity Center
- Browser automation for login/MFA
- Access token obtained and stored

✅ **SSO to IAM Conversion**: Working correctly
- SSO access token → GetRoleCredentials API
- Temporary IAM credentials obtained (AccessKeyID, SecretAccessKey, SessionToken)
- Credentials cached and auto-refreshed

✅ **SigV4 Signing**: Working correctly
- Using temporary IAM credentials
- Service: "q" (Q Developer)
- Region: "us-east-1"
- Session token included in X-Amz-Security-Token header

✅ **Endpoint**: Correct
- URL: https://q.us-east-1.amazonaws.com/
- Headers: Content-Type, X-Amz-Target
- Target: AmazonQDeveloperStreamingService.SendMessage

## Conclusion

The entire SigV4+SSO credential chain is working correctly:
1. OIDC device flow → SSO access token ✅
2. SSO access token → Temporary IAM credentials ✅
3. IAM credentials → SigV4 signed request ✅
4. Request → Q Developer endpoint ✅

**Text-only requests work perfectly**, confirming the credential chain is correct.

**Vision requests fail** with generic AWS responses, but this is NOT due to:
- ❌ Credential chain (working)
- ❌ Endpoint (correct)
- ❌ Request format (matches Q CLI)
- ❌ Authentication (working for text)

**Most likely cause**: Test image content (1x1 red pixel) is not AWS-related, and Q Developer may reject non-AWS images.

**Next step**: Test with an AWS-related image (architecture diagram, console screenshot).
