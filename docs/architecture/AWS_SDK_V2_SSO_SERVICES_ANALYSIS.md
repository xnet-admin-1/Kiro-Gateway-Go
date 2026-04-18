# AWS SDK Go v2 - SSO Services Analysis

## Overview

AWS SDK Go v2 contains three distinct SSO-related services, each serving a different purpose in the IAM Identity Center (formerly AWS SSO) authentication flow.

## Three SSO Services

### 1. **SSO** (`service/sso`)
**Purpose**: Portal API for accessing AWS resources after authentication

**Description**: 
- Web service for assigning user access to IAM Identity Center resources
- Provides access to AWS account applications and roles
- Used to get federated access to applications
- The "runtime" service for authenticated users

**Key Operations**:
- `GetRoleCredentials` - Returns STS short-term credentials for a role
  - Input: AccessToken, AccountId, RoleName
  - Output: RoleCredentials (AccessKeyId, SecretAccessKey, SessionToken, Expiration)
- `ListAccounts` - Lists AWS accounts available to the user
- `ListAccountRoles` - Lists roles available in an account
- `Logout` - Invalidates the user session

**Use Case**: Converting OIDC access tokens into AWS credentials (SigV4)

**Endpoint**: `portal.sso.{region}.amazonaws.com`

---

### 2. **SSO OIDC** (`service/ssooidc`)
**Purpose**: OpenID Connect authentication and token management

**Description**:
- Enables clients (CLI, native apps) to register with IAM Identity Center
- Fetches user access tokens after authentication
- Implements OAuth 2.0 Device Authorization Grant (RFC 8628)
- Supports token refresh without re-authentication

**Key Operations**:
- `RegisterClient` - Registers a client application
  - Input: ClientName, ClientType, Scopes
  - Output: ClientId, ClientSecret, ClientIdIssuedAt, ClientSecretExpiresAt

- `StartDeviceAuthorization` - Initiates device authorization flow
  - Input: ClientId, ClientSecret, StartUrl
  - Output: DeviceCode, UserCode, VerificationUri, VerificationUriComplete, ExpiresIn, Interval

- `CreateToken` - Creates access and refresh tokens
  - Input: ClientId, ClientSecret, GrantType, DeviceCode/Code/RefreshToken
  - Output: AccessToken, RefreshToken, IdToken, ExpiresIn, TokenType
  - Supports grant types: `authorization_code`, `urn:ietf:params:oauth:grant-type:device_code`, `refresh_token`

- `CreateTokenWithIAM` - Creates tokens using IAM authentication

**Use Case**: Initial authentication and token acquisition/refresh

**Endpoint**: `oidc.{region}.amazonaws.com`

**Token Refresh Support**:
- CLI V1: 1.27.10+
- CLI V2: 2.9.0+
- Configurable session durations

---

### 3. **SSO Admin** (`service/ssoadmin`)
**Purpose**: Administrative API for managing IAM Identity Center

**Description**:
- Manages IAM Identity Center configuration
- Connects workforce users to AWS managed applications
- Synchronizes users and groups from identity providers
- Manages permission sets and account assignments

**Key Operations** (Administrative):
- **Permission Sets**:
  - `CreatePermissionSet`, `UpdatePermissionSet`, `DeletePermissionSet`
  - `DescribePermissionSet`, `ListPermissionSets`
  - `PutInlinePolicyToPermissionSet`, `GetInlinePolicyForPermissionSet`
  - `AttachManagedPolicyToPermissionSet`, `DetachManagedPolicyFromPermissionSet`

- **Account Assignments**:
  - `CreateAccountAssignment`, `DeleteAccountAssignment`
  - `ListAccountAssignments`, `ListAccountAssignmentsForPrincipal`
  - `DescribeAccountAssignmentCreationStatus`

- **Applications**:
  - `CreateApplication`, `UpdateApplication`, `DeleteApplication`
  - `ListApplications`, `DescribeApplication`
  - `CreateApplicationAssignment`, `DeleteApplicationAssignment`

- **Instances**:
  - `CreateInstance`, `UpdateInstance`, `DeleteInstance`
  - `ListInstances`, `DescribeInstance`

- **Trusted Token Issuers**:
  - `CreateTrustedTokenIssuer`, `UpdateTrustedTokenIssuer`, `DeleteTrustedTokenIssuer`

**Use Case**: Administrative management of IAM Identity Center (not for end-user authentication)

**Endpoint**: `sso.{region}.amazonaws.com`

---

## Authentication Flow Comparison

### Our Current Implementation (v1 SDK Pattern)
```
1. Load SSO token from cache (~/.aws/sso/cache/)
2. Use SSO token to call GetRoleCredentials
3. Get temporary AWS credentials (AccessKey, SecretKey, SessionToken)
4. Use credentials for SigV4 signing
5. Call Q Developer API with SigV4
```

### Standard OIDC Flow (v2 SDK Pattern)
```
1. RegisterClient (ssooidc) - Get ClientId/ClientSecret
2. StartDeviceAuthorization (ssooidc) - Get DeviceCode/UserCode
3. User visits VerificationUri and enters UserCode
4. Poll CreateToken (ssooidc) with DeviceCode until authorized
5. Receive AccessToken and RefreshToken
6. Use AccessToken to call GetRoleCredentials (sso)
7. Get temporary AWS credentials
8. Use credentials for SigV4 signing
```

### Token Refresh Flow (v2 SDK)
```
1. Check if AccessToken is expired
2. If expired, call CreateToken (ssooidc) with RefreshToken
3. Receive new AccessToken and RefreshToken
4. Use new AccessToken to call GetRoleCredentials (sso)
5. Get fresh AWS credentials
```

---

## Service Comparison Table

| Feature | SSO | SSO OIDC | SSO Admin |
|---------|-----|----------|-----------|
| **Purpose** | Get AWS credentials | Authenticate users | Manage IAM Identity Center |
| **User Type** | End users | End users | Administrators |
| **Authentication** | Requires access token | Provides access token | Requires admin credentials |
| **Main Use** | Runtime credential access | Initial auth + token refresh | Configuration management |
| **Endpoint** | `portal.sso.{region}.amazonaws.com` | `oidc.{region}.amazonaws.com` | `sso.{region}.amazonaws.com` |
| **Key Operation** | GetRoleCredentials | CreateToken | CreatePermissionSet |
| **In Our Project** | ✅ Used | ❌ Not used (we load cached tokens) | ❌ Not needed |

---

## API Namespaces

IAM Identity Center uses multiple API namespaces:
- **sso** - Portal API (runtime access)
- **sso-oauth** / **ssooidc** - OIDC authentication
- **identitystore** - User and group management
- **ssoadmin** - Administrative configuration

---

## Token Types

### 1. SSO Access Token (OIDC)
- **Source**: SSO OIDC service (`CreateToken`)
- **Purpose**: Authenticate to SSO Portal API
- **Used For**: Calling `GetRoleCredentials`
- **Storage**: `~/.aws/sso/cache/`
- **Format**: JWT (JSON Web Token)
- **Lifetime**: Configurable (default: 8 hours)
- **Refresh**: Using RefreshToken with `CreateToken`

### 2. AWS Credentials (STS)
- **Source**: SSO service (`GetRoleCredentials`)
- **Purpose**: Sign AWS API requests (SigV4)
- **Used For**: Calling AWS services (S3, EC2, Q Developer, etc.)
- **Components**: AccessKeyId, SecretAccessKey, SessionToken
- **Lifetime**: Typically 1 hour
- **Refresh**: Call `GetRoleCredentials` again with valid SSO token

### 3. Refresh Token
- **Source**: SSO OIDC service (`CreateToken`)
- **Purpose**: Get new access tokens without re-authentication
- **Used For**: Calling `CreateToken` with `grant_type=refresh_token`
- **Storage**: `~/.aws/sso/cache/`
- **Lifetime**: Longer than access token (configurable)

---

## Our Implementation vs SDK v2

### What We're Doing (v1 Pattern)
```go
// 1. Load cached SSO token
token := loadSSOToken("~/.aws/sso/cache/kiro-auth-token.json")

// 2. Call SSO GetRoleCredentials
ssoClient := sso.New(session)
creds, err := ssoClient.GetRoleCredentials(&sso.GetRoleCredentialsInput{
    AccessToken: token,
    AccountId:   accountID,
    RoleName:    roleName,
})

// 3. Use credentials for SigV4
sigv4Signer := sigv4.NewSigner(creds)
signedRequest := sigv4Signer.Sign(request)
```

### What SDK v2 Does (Full OIDC Flow)
```go
// 1. Register client
oidcClient := ssooidc.NewFromConfig(cfg)
registerResp, _ := oidcClient.RegisterClient(ctx, &ssooidc.RegisterClientInput{
    ClientName: aws.String("my-app"),
    ClientType: aws.String("public"),
})

// 2. Start device authorization
authResp, _ := oidcClient.StartDeviceAuthorization(ctx, &ssooidc.StartDeviceAuthorizationInput{
    ClientId:     registerResp.ClientId,
    ClientSecret: registerResp.ClientSecret,
    StartUrl:     aws.String("https://my-sso-portal.awsapps.com/start"),
})

// 3. User authorizes device (manual step)
fmt.Printf("Visit: %s\nEnter code: %s\n", 
    *authResp.VerificationUri, *authResp.UserCode)

// 4. Poll for token
tokenResp, _ := oidcClient.CreateToken(ctx, &ssooidc.CreateTokenInput{
    ClientId:     registerResp.ClientId,
    ClientSecret: registerResp.ClientSecret,
    GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
    DeviceCode:   authResp.DeviceCode,
})

// 5. Get AWS credentials
ssoClient := sso.NewFromConfig(cfg)
credsResp, _ := ssoClient.GetRoleCredentials(ctx, &sso.GetRoleCredentialsInput{
    AccessToken: tokenResp.AccessToken,
    AccountId:   aws.String("123456789012"),
    RoleName:    aws.String("MyRole"),
})
```

---

## Key Differences: v1 vs v2

| Aspect | SDK v1 (Our Implementation) | SDK v2 |
|--------|----------------------------|--------|
| **Token Source** | Pre-cached from AWS CLI | Generated via OIDC flow |
| **Client Registration** | Not needed (uses CLI's client) | Required (`RegisterClient`) |
| **Device Authorization** | Not needed (CLI handles it) | Required (`StartDeviceAuthorization`) |
| **Token Creation** | Not needed (CLI handles it) | Required (`CreateToken`) |
| **Token Refresh** | Manual (reload from cache) | Built-in (`CreateToken` with refresh_token) |
| **Credential Provider** | Custom implementation | Built-in credential providers |
| **Session Management** | Manual | Automatic with refresh |

---

## Recommendations

### For Our Project
1. ✅ **Keep Current Approach** - We're correctly using the SSO service to get credentials
2. ✅ **Token Loading Works** - Loading cached SSO tokens from CLI is valid
3. ⚠️ **Consider Token Refresh** - Could implement refresh using SSO OIDC service
4. ⚠️ **Monitor Token Expiry** - Add better handling for expired tokens

### If Implementing Full OIDC Flow
Only needed if:
- Building a standalone app without AWS CLI
- Need custom session durations
- Want automatic token refresh
- Need to manage client registration

**Steps**:
1. Use `ssooidc.RegisterClient` to get client credentials
2. Use `ssooidc.StartDeviceAuthorization` for device flow
3. Poll `ssooidc.CreateToken` until user authorizes
4. Store AccessToken and RefreshToken
5. Use `sso.GetRoleCredentials` to get AWS credentials
6. Refresh tokens using `ssooidc.CreateToken` with refresh_token grant

---

## References

- **SDK v2 SSO**: `D:\repo2\aws-sdk-go-v2\service\sso`
- **SDK v2 SSO OIDC**: `D:\repo2\aws-sdk-go-v2\service\ssooidc`
- **SDK v2 SSO Admin**: `D:\repo2\aws-sdk-go-v2\service\ssoadmin`
- **Our Implementation**: `internal/auth/credentials/sso.go`
- **IAM Identity Center Docs**: https://docs.aws.amazon.com/singlesignon/
- **OIDC API Reference**: https://docs.aws.amazon.com/singlesignon/latest/OIDCAPIReference/
- **Portal API Reference**: https://docs.aws.amazon.com/singlesignon/latest/PortalAPIReference/

---

## Conclusion

Our implementation correctly uses the **SSO service** (`GetRoleCredentials`) to convert cached SSO tokens into AWS credentials for SigV4 signing. We don't need the **SSO OIDC service** because we rely on the AWS CLI to handle the initial authentication and token caching. The **SSO Admin service** is for administrative tasks and not relevant to our use case.

The three services work together in the complete authentication flow:
1. **SSO OIDC** - Get access tokens (handled by AWS CLI for us)
2. **SSO** - Convert tokens to AWS credentials (what we use)
3. **SSO Admin** - Manage IAM Identity Center (not needed for runtime auth)
