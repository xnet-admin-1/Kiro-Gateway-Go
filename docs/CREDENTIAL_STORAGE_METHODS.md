# Amazon Q Developer - Credential Storage Methods

## Overview

Amazon Q Developer CLI and Kiro Gateway support **three primary methods** for storing and retrieving credentials:

1. **SQLite Database** (encrypted, persistent)
2. **Environment Variables** (temporary, session-based)
3. **AWS Credential Chain** (standard AWS credential providers)

## 1. SQLite Database Storage

### Description
Credentials are stored in an encrypted SQLite database file on the local filesystem. This is the **primary storage method** for OAuth2 bearer tokens (Builder ID and Identity Center).

### Location
```
~/.amazon-q/data.sqlite3
```

### What's Stored
- **OAuth2 Bearer Tokens** (Builder ID and Identity Center)
  - Access token
  - Refresh token
  - Expiration time
  - Region
  - Start URL
  - OAuth flow type
  - Scopes

- **Device Registration** (OAuth2 client credentials)
  - Client ID
  - Client secret
  - Expiration time
  - Region

- **Profile Information**
  - Profile ARN (for Identity Center)
  - Profile name
  - Selected profile

- **Application State**
  - Settings
  - Conversation history
  - Telemetry data

### Database Schema

```sql
-- Auth table for credentials
CREATE TABLE auth_kv (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- State table for application state
CREATE TABLE state (
    key TEXT PRIMARY KEY,
    value BLOB NOT NULL
);

-- Conversations table
CREATE TABLE conversations (
    id TEXT PRIMARY KEY,
    -- conversation data
);
```

### Storage Keys

```rust
// Bearer token storage
const SECRET_KEY: &str = "codewhisperer:odic:token";

// Device registration storage
const SECRET_KEY: &str = "codewhisperer:oidc:device-registration";

// Profile storage
const CODEWHISPERER_PROFILE_KEY: &str = "api.codewhisperer.profile";

// Identity Center configuration
const START_URL_KEY: &str = "auth.idc.start-url";
const IDC_REGION_KEY: &str = "auth.idc.region";
```

### API Methods

```rust
// Get secret from database
pub async fn get_secret(&self, key: &str) -> Result<Option<Secret>, DatabaseError>

// Set secret in database
pub async fn set_secret(&self, key: &str, value: &str) -> Result<(), DatabaseError>

// Delete secret from database
pub async fn delete_secret(&self, key: &str) -> Result<(), DatabaseError>
```

### Usage Example

```rust
// Load bearer token from database
let token = BuilderIdToken::load(&database, Some(&telemetry)).await?;

// Save bearer token to database
token.save(&database).await?;

// Delete bearer token from database
token.delete(&database).await?;
```

### Advantages
- ✅ Persistent across sessions
- ✅ Encrypted storage
- ✅ Automatic token refresh
- ✅ Cross-platform support
- ✅ No manual configuration needed

### Disadvantages
- ⚠️ Requires file system access
- ⚠️ Database file must be protected
- ⚠️ Not suitable for containerized environments

## 2. Environment Variables

### Description
Credentials are provided via environment variables. This is useful for **temporary sessions**, **CI/CD pipelines**, and **containerized environments**.

### Environment Variables

#### AWS Credentials (SigV4)
```bash
# Access key credentials
export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
export AWS_SESSION_TOKEN=AQoDYXdzEJr...  # Optional, for temporary credentials

# Region
export AWS_REGION=us-east-1

# Profile (alternative to access keys)
export AWS_PROFILE=my-profile
```

#### Bearer Token (OAuth2)
```bash
# Bearer token (if manually provided)
export BEARER_TOKEN=eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...

# OIDC configuration
export OIDC_START_URL=https://view.awsapps.com/start
export OIDC_REGION=us-east-1
```

#### Gateway Configuration
```bash
# API endpoint selection
export Q_USE_SENDMESSAGE=true          # Use Q endpoint
export AMAZON_Q_SIGV4=true             # Use SigV4 auth

# Profile ARN (for Identity Center)
export PROFILE_ARN=arn:aws:q:us-east-1:123456789012:profile/abc123

# Gateway settings
export PORT=8090
export PROXY_API_KEY=your-api-key
```


## Recommended Usage

### Local Development (Builder ID)
**Method:** SQLite Database
```bash
# Login once
q login --license free

# Gateway uses stored credentials
./kiro-gateway
```

### Local Development (Identity Center)
**Method:** SQLite Database + Profile ARN
```bash
# Login once
q login --license pro --identity-provider https://your-org.awsapps.com/start

# Select profile
q profile

# Gateway uses stored credentials
export PROFILE_ARN=arn:aws:q:us-east-1:123456789012:profile/abc123
./kiro-gateway
```

### CloudShell / AWS Environment
**Method:** AWS Credential Chain (SigV4)
```bash
# No login needed - uses IAM credentials
export Q_USE_SENDMESSAGE=true
export AMAZON_Q_SIGV4=true
export AWS_PROFILE=xnet-admin
./kiro-gateway
```

### CI/CD Pipeline
**Method:** Environment Variables
```bash
# Set in CI/CD secrets
export AWS_ACCESS_KEY_ID=$CI_AWS_ACCESS_KEY_ID
export AWS_SECRET_ACCESS_KEY=$CI_AWS_SECRET_ACCESS_KEY
export AWS_REGION=us-east-1
export Q_USE_SENDMESSAGE=true
export AMAZON_Q_SIGV4=true
./kiro-gateway
```

### Docker Container
**Method:** Environment Variables or AWS Credential Chain
```dockerfile
# Option 1: Environment variables
ENV AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
ENV AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
ENV AWS_REGION=us-east-1

# Option 2: Mount credentials file
VOLUME /root/.aws

# Option 3: Use ECS task role (no config needed)
```

### Kubernetes/EKS
**Method:** AWS Credential Chain (IRSA)
```yaml
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
      - name: gateway
        image: kiro-gateway:latest
        env:
        - name: Q_USE_SENDMESSAGE
          value: "true"
        - name: AMAZON_Q_SIGV4
          value: "true"
```

## Security Best Practices

### SQLite Database
- ✅ Store database in user home directory
- ✅ Use file system permissions (chmod 600)
- ✅ Enable encryption at rest
- ✅ Regular token rotation
- ❌ Don't commit database to version control

### Environment Variables
- ✅ Use secrets management (AWS Secrets Manager, HashiCorp Vault)
- ✅ Rotate credentials regularly
- ✅ Use temporary credentials when possible
- ✅ Clear variables after use
- ❌ Don't log environment variables
- ❌ Don't commit to version control

### AWS Credential Chain
- ✅ Use IAM roles instead of access keys
- ✅ Enable MFA for sensitive operations
- ✅ Use least privilege principle
- ✅ Rotate access keys regularly
- ✅ Use temporary credentials (STS)
- ❌ Don't share credentials
- ❌ Don't hardcode credentials

## Troubleshooting

### Database Issues
```bash
# Check database exists
ls -la ~/.amazon-q/data.sqlite3

# Check permissions
chmod 600 ~/.amazon-q/data.sqlite3

# Re-authenticate
q logout
q login --license free
```

### Environment Variable Issues
```bash
# Check variables are set
echo $AWS_ACCESS_KEY_ID
echo $AWS_SECRET_ACCESS_KEY
echo $AWS_REGION

# Test credentials
aws sts get-caller-identity
```

### Credential Chain Issues
```bash
# Check AWS configuration
aws configure list

# Test credential chain
aws sts get-caller-identity --profile xnet-admin

# Enable debug logging
export AWS_SDK_LOG_LEVEL=debug
./kiro-gateway
```

## Summary

Amazon Q Developer supports three credential storage methods:

1. **SQLite Database** - Best for local development with OAuth2 (Builder ID/Identity Center)
2. **Environment Variables** - Best for CI/CD, containers, and temporary sessions
3. **AWS Credential Chain** - Best for AWS environments with SigV4 authentication

Choose the method that best fits your deployment environment and security requirements.

---

**Date:** January 22, 2026
**Status:** ✅ Complete
**Implementation:** Fully supported in Kiro Gateway
