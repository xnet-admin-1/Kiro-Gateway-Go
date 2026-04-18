# Authentication Guide

This document provides comprehensive information about authentication methods supported by Kiro Gateway.

## Table of Contents

1. [Overview](#overview)
2. [Authentication Methods](#authentication-methods)
3. [Credential Providers](#credential-providers)
4. [Token Management](#token-management)
5. [Configuration Examples](#configuration-examples)
6. [Security Best Practices](#security-best-practices)

## Overview

Kiro Gateway supports multiple authentication methods to connect to Amazon Q Developer API:

- **Kiro Desktop**: Uses Kiro Desktop's SQLite database
- **AWS SSO/OIDC**: Uses AWS Builder ID or IAM Identity Center
- **AWS SigV4**: Uses IAM credentials with Signature Version 4
- **CLI Database**: Uses Amazon Q CLI's SQLite database

## Authentication Methods

### 1. Kiro Desktop Authentication

**Best for:** Users with Kiro Desktop installed

Kiro Desktop authentication uses the SQLite database maintained by Kiro Desktop application. This method provides automatic token refresh and secure credential storage.

**Configuration:**
```env
AUTH_TYPE=desktop
KIRO_DB_PATH=~/.kiro/kiro.db
KIRO_PROFILE_ARN=arn:aws:codewhisperer:us-east-1:123456789012:profile/abc123
```

**How it works:**
1. Gateway reads refresh token and profile ARN from Kiro Desktop database
2. Checks if access token is expired (10-minute threshold)
3. Refreshes token if needed using refresh token
4. Adds Bearer token to API requests

**Token Refresh:**
- Automatic refresh 10 minutes before expiration
- Refresh endpoint: `https://prod.{region}.auth.desktop.kiro.dev/refreshToken`
- Requires: refreshToken, profileArn, region

**Database Schema:**
```sql
-- Kiro Desktop database structure
CREATE TABLE auth_kv (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- Keys used:
-- 'kiro:oidc:token' - Contains refresh token
-- 'kiro:profile:arn' - Contains profile ARN
```

### 2. AWS SSO/OIDC Authentication

**Best for:** Users with AWS Builder ID or IAM Identity Center

OIDC authentication uses the OAuth2 device code flow for browser-less authentication. Supports both AWS Builder ID (free tier) and corporate IAM Identity Center.

**Configuration:**
```env
AUTH_TYPE=oidc
OIDC_CLIENT_ID=your-client-id
OIDC_CLIENT_SECRET=your-client-secret
OIDC_TOKEN_URL=https://oidc.us-east-1.amazonaws.com/token
OIDC_REFRESH_TOKEN=your-initial-refresh-token
OIDC_START_URL=https://my-sso-portal.awsapps.com/start
OIDC_REGION=us-east-1
```

**How it works:**
1. **Device Registration** (cached):
   - Register OIDC client with AWS SSO-OIDC
   - Receive client ID and client secret
   - Cache registration to avoid repeated registrations

2. **Device Authorization**:
   - Start device authorization flow
   - Display user code and verification URL
   - User visits URL and enters code

3. **Token Polling**:
   - Poll for access token with exponential backoff
   - Handle authorization_pending (continue polling)
   - Handle slow_down (increase interval)
   - Receive access token and refresh token

4. **Token Refresh**:
   - Check expiration with 1-minute safety margin
   - Refresh using refresh token
   - Store new tokens securely

**Token Storage:**
- Tokens stored in OS keychain (encrypted at rest)
- Fallback to encrypted SQLite if keychain unavailable
- Automatic cleanup of expired tokens

**OIDC Endpoints:**
```
# Device Registration
POST https://oidc.{region}.amazonaws.com/client/register

# Device Authorization
POST https://oidc.{region}.amazonaws.com/device_authorization

# Token Creation
POST https://oidc.{region}.amazonaws.com/token
```

### 3. AWS SigV4 Authentication

**Best for:** Production deployments with IAM roles

SigV4 authentication uses AWS IAM credentials instead of bearer tokens. This provides enhanced security through AWS's standard credential chain.

**Configuration:**
```env
AMAZON_Q_SIGV4=true
AWS_REGION=us-east-1
AWS_PROFILE=default  # Optional
```

**How it works:**
1. **Credential Resolution**:
   - Try credential providers in order (see Credential Providers section)
   - Cache credentials until expiration
   - Automatic refresh when expired

2. **Request Signing**:
   - Calculate canonical request
   - Create string to sign
   - Derive signing key
   - Calculate signature
   - Add Authorization header

3. **Request Headers**:
   ```
   Authorization: AWS4-HMAC-SHA256 Credential=..., SignedHeaders=..., Signature=...
   X-Amz-Date: 20260122T120000Z
   X-Amz-Security-Token: ... (if using temporary credentials)
   ```

**Signature Calculation:**
```
1. Canonical Request:
   HTTP_METHOD + "\n" +
   CANONICAL_URI + "\n" +
   CANONICAL_QUERY_STRING + "\n" +
   CANONICAL_HEADERS + "\n" +
   SIGNED_HEADERS + "\n" +
   HASHED_PAYLOAD

2. String to Sign:
   "AWS4-HMAC-SHA256" + "\n" +
   TIMESTAMP + "\n" +
   CREDENTIAL_SCOPE + "\n" +
   HASHED_CANONICAL_REQUEST

3. Signing Key:
   kDate = HMAC("AWS4" + SECRET_KEY, DATE)
   kRegion = HMAC(kDate, REGION)
   kService = HMAC(kRegion, SERVICE)
   kSigning = HMAC(kService, "aws4_request")

4. Signature:
   HMAC(kSigning, STRING_TO_SIGN)
```

### 4. CLI Database Authentication

**Best for:** Users with Amazon Q CLI installed

CLI Database authentication reads credentials from Amazon Q CLI's SQLite database.

**Configuration:**
```env
AUTH_TYPE=cli_db
CLI_DB_PATH=~/.aws/amazonq/cache.db
```

**How it works:**
1. Read credentials from CLI database
2. Check token expiration
3. Refresh if needed (using CLI's refresh mechanism)
4. Add Bearer token to API requests

**Database Schema:**
```sql
-- Amazon Q CLI database structure
CREATE TABLE auth_kv (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- Keys used:
-- 'kirocli:oidc:token' or 'codewhisperer:oidc:token'
```

## Credential Providers

When `AMAZON_Q_SIGV4=true`, credentials are resolved using the AWS credential provider chain.

### Provider Chain Order

1. **Environment Variables**
2. **AWS Profile Files**
3. **Web Identity Token**
4. **ECS Container Credentials**
5. **EC2 Instance Metadata (IMDS)**

### 1. Environment Provider

Reads credentials from environment variables.

**Environment Variables:**
```bash
export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
export AWS_SESSION_TOKEN=token  # Optional, for temporary credentials
```

**Characteristics:**
- No expiration by default (static credentials)
- Highest priority in credential chain
- Suitable for development and testing

### 2. Profile Provider

Reads credentials from AWS profile files.

**Profile Files:**
```ini
# ~/.aws/credentials
[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

[production]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

# ~/.aws/config
[profile production]
region = us-west-2
output = json
```

**Usage:**
```bash
# Use default profile
./kiro-gateway

# Use specific profile
export AWS_PROFILE=production
./kiro-gateway
```

**Characteristics:**
- Supports multiple profiles
- Can include assume role configuration
- Suitable for multi-account access

### 3. Web Identity Token Provider

Uses OIDC tokens for authentication (Kubernetes IRSA).

**Environment Variables:**
```bash
export AWS_WEB_IDENTITY_TOKEN_FILE=/var/run/secrets/eks.amazonaws.com/serviceaccount/token
export AWS_ROLE_ARN=arn:aws:iam::123456789012:role/my-role
export AWS_ROLE_SESSION_NAME=my-session  # Optional
```

**How it works:**
1. Read token from file
2. Call STS AssumeRoleWithWebIdentity
3. Receive temporary credentials
4. Cache until expiration

**Characteristics:**
- Automatic credential refresh
- Suitable for Kubernetes deployments
- Supports IAM Roles for Service Accounts (IRSA)

### 4. ECS Provider

Reads credentials from ECS container metadata.

**Environment Variables:**
```bash
# Automatically set by ECS
export AWS_CONTAINER_CREDENTIALS_RELATIVE_URI=/v2/credentials/...
```

**How it works:**
1. Read URI from environment
2. Call ECS metadata endpoint (169.254.170.2)
3. Parse JSON response
4. Cache credentials until expiration

**Characteristics:**
- Automatic in ECS environments
- Credentials rotate automatically
- No configuration needed

### 5. IMDS Provider

Reads credentials from EC2 instance metadata.

**How it works:**
1. Get IMDSv2 token (if available)
2. Call metadata endpoint (169.254.169.254)
3. Parse JSON response
4. Cache credentials until expiration

**Characteristics:**
- Automatic on EC2 instances
- Supports IMDSv2 (token-based)
- Credentials rotate automatically
- Lowest priority in chain

## Token Management

### Token Lifecycle

1. **Initial Authentication**:
   - Load credentials from configured source
   - Obtain access token
   - Store tokens securely
   - Set up automatic refresh

2. **Token Validation**:
   - Check expiration before each request
   - Use safety margin (1-10 minutes depending on method)
   - Refresh if expired or near expiration

3. **Token Refresh**:
   - Use refresh token to obtain new access token
   - Update stored credentials
   - Handle refresh failures gracefully

4. **Token Expiration**:
   - Delete expired tokens that cannot be refreshed
   - Return authentication error
   - Require re-authentication

### Token Storage

**OS Keychain (Primary):**
- Windows: Credential Manager
- macOS: Keychain
- Linux: Secret Service (libsecret)

**Encrypted SQLite (Fallback):**
- AES-256-GCM encryption
- Key derived from machine-specific data
- Automatic migration from keychain

**Security Features:**
- Tokens encrypted at rest
- Secure deletion of expired tokens
- No tokens in logs or error messages
- Thread-safe access

### Token Refresh Strategy

**Bearer Token (Kiro Desktop, OIDC):**
```go
// Check expiration with safety margin
if time.Until(tokenExpiration) < 10*time.Minute {
    // Refresh token
    newToken := refreshToken(refreshToken)
    // Update stored token
    storeToken(newToken)
}
```

**SigV4 (AWS Credentials):**
```go
// Check expiration
if credentials.IsExpired() {
    // Try to refresh from provider
    newCredentials := provider.Retrieve()
    // Update cached credentials
    cacheCredentials(newCredentials)
}
```

## Configuration Examples

### Example 1: Development with Environment Variables

```bash
# .env
PORT=8080
PROXY_API_KEY=dev-secret-key
AUTH_TYPE=desktop
KIRO_DB_PATH=~/.kiro/kiro.db
AWS_REGION=us-east-1
DEBUG=true
```

### Example 2: Production with SigV4 and IAM Role

```bash
# .env
PORT=8080
PROXY_API_KEY=prod-secret-key
AMAZON_Q_SIGV4=true
AWS_REGION=us-east-1
MAX_RETRIES=5
MAX_BACKOFF=60s
CONNECT_TIMEOUT=15s
READ_TIMEOUT=60s
OPERATION_TIMEOUT=120s
LOG_LEVEL=info
```

### Example 3: Kubernetes with IRSA

```yaml
# kubernetes-deployment.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kiro-gateway
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/kiro-gateway-role
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kiro-gateway
spec:
  template:
    spec:
      serviceAccountName: kiro-gateway
      containers:
      - name: kiro-gateway
        image: kiro-gateway:latest
        env:
        - name: AMAZON_Q_SIGV4
          value: "true"
        - name: AWS_REGION
          value: "us-east-1"
        - name: PROXY_API_KEY
          valueFrom:
            secretKeyRef:
              name: kiro-gateway-secret
              key: api-key
```

### Example 4: Multi-Account with Profile ARN

```bash
# .env
PORT=8080
PROXY_API_KEY=secret-key
AUTH_TYPE=desktop
KIRO_DB_PATH=~/.kiro/kiro.db
KIRO_PROFILE_ARN=arn:aws:codewhisperer:us-east-1:123456789012:profile/abc123
AWS_REGION=us-east-1
```

### Example 5: OIDC with IAM Identity Center

```bash
# .env
PORT=8080
PROXY_API_KEY=secret-key
AUTH_TYPE=oidc
OIDC_CLIENT_ID=client-id-from-registration
OIDC_CLIENT_SECRET=client-secret-from-registration
OIDC_TOKEN_URL=https://oidc.us-east-1.amazonaws.com/token
OIDC_REFRESH_TOKEN=initial-refresh-token
OIDC_START_URL=https://my-company.awsapps.com/start
OIDC_REGION=us-east-1
```

## Security Best Practices

### Credential Management

**DO:**
- ✅ Use environment variables for sensitive data
- ✅ Store tokens in OS keychain
- ✅ Implement automatic token rotation
- ✅ Use IAM roles when possible (SigV4)
- ✅ Check expiration with safety margin
- ✅ Clear expired tokens regularly

**DON'T:**
- ❌ Commit credentials to version control
- ❌ Share refresh tokens
- ❌ Store tokens in plain text
- ❌ Log credential values
- ❌ Expose tokens in error messages

### Network Security

**DO:**
- ✅ Use HTTPS for all API calls
- ✅ Validate SSL certificates
- ✅ Implement request timeouts
- ✅ Use connection pooling
- ✅ Require TLS 1.2+

**DON'T:**
- ❌ Disable SSL verification
- ❌ Use unencrypted connections
- ❌ Expose API keys in URLs
- ❌ Allow insecure ciphers

### Token Refresh

**DO:**
- ✅ Refresh before expiration (safety margin)
- ✅ Handle refresh failures gracefully
- ✅ Implement retry logic with backoff
- ✅ Use thread-safe token access

**DON'T:**
- ❌ Refresh on every request
- ❌ Ignore refresh errors
- ❌ Use expired tokens
- ❌ Allow concurrent refresh attempts

### Logging

**DO:**
- ✅ Log authentication events (success/failure)
- ✅ Include request IDs for debugging
- ✅ Sanitize error messages
- ✅ Use structured logging

**DON'T:**
- ❌ Log credentials or tokens
- ❌ Log full request/response bodies
- ❌ Expose internal error details
- ❌ Log sensitive headers

## Troubleshooting

See [README.md](README.md#troubleshooting) for detailed troubleshooting guide.

## Additional Resources

- [AWS Signature Version 4](https://docs.aws.amazon.com/general/latest/gr/signature-version-4.html)
- [AWS Credential Provider Chain](https://docs.aws.amazon.com/sdkref/latest/guide/standardized-credentials.html)
- [AWS SSO OIDC](https://docs.aws.amazon.com/singlesignon/latest/OIDCAPIReference/Welcome.html)
- [IAM Roles for Service Accounts](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html)
