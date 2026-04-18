# Security Best Practices

This document outlines security best practices for kiro-gateway-go, covering credential management, network security, error handling, and secure storage.

## Credential Management

### DO ✅

- **Store tokens in OS keychain** - All access tokens and refresh tokens are stored encrypted in the operating system's secure credential store
- **Use environment variables for configuration** - Non-sensitive configuration can use environment variables
- **Implement automatic token rotation** - Tokens are automatically refreshed before expiration with a 1-minute safety margin
- **Clear expired tokens regularly** - Expired tokens that cannot be refreshed are automatically deleted
- **Validate token format** - All tokens are validated before use to ensure proper format
- **Use secure credential providers** - Follow AWS credential chain: Environment → Profile → Web Identity → ECS → IMDS

### DON'T ❌

- **Commit credentials to version control** - Never store credentials in code or configuration files
- **Share refresh tokens** - Refresh tokens are single-use and should never be shared between processes
- **Store tokens in plain text logs** - All logging sanitizes credential values
- **Use expired or invalid tokens** - Token expiration is checked with safety margin before each use
- **Log credential values** - Credentials are redacted from all log output
- **Expose tokens in error messages** - Error messages are sanitized to remove sensitive data
- **Store credentials permanently in environment** - Use temporary credentials when possible

## Network Security

### TLS Configuration

```go
// Minimum TLS configuration
tlsConfig := &tls.Config{
    MinVersion: tls.VersionTLS12,
    CipherSuites: []uint16{
        tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
        tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
    },
}
```

### DO ✅

- **Use HTTPS for all API calls** - All communication with AWS services uses TLS encryption
- **Validate SSL certificates** - Certificate validation is enabled and cannot be disabled
- **Implement request timeouts** - All requests have appropriate timeouts to prevent hanging
- **Require TLS 1.2+** - Minimum TLS version is enforced
- **Verify certificate chains** - Full certificate chain validation is performed
- **Use connection pooling** - HTTP connections are reused for performance and security

### DON'T ❌

- **Disable SSL verification** - Certificate validation is always enabled
- **Use unencrypted connections** - All connections must use HTTPS
- **Expose API keys in URLs** - Credentials are only sent in headers or request bodies
- **Allow insecure ciphers** - Only secure cipher suites are permitted
- **Trust self-signed certificates** - Only certificates from trusted CAs are accepted

## Error Handling

### Safe Error Logging

```go
func (l *SafeLogger) LogError(err error) {
    msg := err.Error()
    
    // Sanitize potential credentials
    patterns := []string{
        `access_key[^a-zA-Z0-9]*[a-zA-Z0-9]+`,
        `secret[^a-zA-Z0-9]*[a-zA-Z0-9]+`,
        `token[^a-zA-Z0-9]*[a-zA-Z0-9]+`,
        `Bearer\s+[a-zA-Z0-9\-._~+/]+=*`,
        `AWS4-HMAC-SHA256.*`,
    }
    
    for _, pattern := range patterns {
        re := regexp.MustCompile(pattern)
        msg = re.ReplaceAllString(msg, "[REDACTED]")
    }
    
    l.logger.Printf("Error: %s", msg)
}
```

### DO ✅

- **Sanitize error messages** - Remove all sensitive data from error messages
- **Log errors without sensitive data** - Use structured logging with sanitized fields
- **Include request IDs** - Include AWS request IDs for debugging without exposing credentials
- **Classify errors appropriately** - Use error classification to handle different error types
- **Implement retry logic** - Use exponential backoff for retryable errors
- **Handle timeouts gracefully** - Provide meaningful error messages for timeout scenarios

### DON'T ❌

- **Expose credentials in error messages** - Never include tokens, keys, or signatures in errors
- **Log full request/response bodies** - Request/response bodies may contain sensitive data
- **Return stack traces to clients** - Internal stack traces may expose sensitive information
- **Retry indefinitely** - Implement maximum retry limits to prevent abuse
- **Expose internal error details** - Provide user-friendly error messages without internal details

## Secure Storage

### OS Keychain Integration

The application uses the operating system's secure credential store:

- **Windows**: Windows Credential Manager
- **macOS**: Keychain Services
- **Linux**: Secret Service (libsecret)

### Encrypted SQLite Fallback

When keychain is unavailable, encrypted SQLite storage is used:

```go
type Encryption struct {
    key []byte // AES-256 key derived from machine-specific data
}

func (e *Encryption) Encrypt(data []byte) ([]byte, error) {
    block, err := aes.NewCipher(e.key)
    if err != nil {
        return nil, err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }
    
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }
    
    ciphertext := gcm.Seal(nonce, nonce, data, nil)
    return ciphertext, nil
}
```

### Storage Security Features

- **AES-256-GCM encryption** for SQLite storage
- **Machine-specific key derivation** prevents credential portability
- **Secure deletion** of expired credentials
- **Access control** through OS-level permissions
- **Audit logging** of storage operations (without credential values)

## Authentication Security

### AWS Signature Version 4 (SigV4)

When using IAM credentials, all requests are signed with SigV4:

```go
func (s *Signer) SignRequest(req *http.Request, body []byte) error {
    // 1. Create canonical request
    canonicalReq := s.buildCanonicalRequest(req, body)
    
    // 2. Create string to sign with timestamp
    stringToSign := s.buildStringToSign(time.Now().UTC(), canonicalReq)
    
    // 3. Calculate signature with derived key
    signature := s.calculateSignature(stringToSign)
    
    // 4. Add authorization header
    req.Header.Set("Authorization", s.buildAuthHeader(signature))
    
    return nil
}
```

### Token Refresh Security

```go
func (m *AuthManager) refreshToken(ctx context.Context) (*Token, error) {
    // Use double-checked locking to prevent refresh storms
    m.mu.RLock()
    if time.Until(m.tokenExp) > time.Minute {
        token := m.token
        m.mu.RUnlock()
        return &Token{AccessToken: token}, nil
    }
    m.mu.RUnlock()
    
    m.mu.Lock()
    defer m.mu.Unlock()
    
    // Double-check after acquiring write lock
    if time.Until(m.tokenExp) > time.Minute {
        return &Token{AccessToken: m.token}, nil
    }
    
    // Perform refresh
    newToken, err := m.oidcClient.RefreshToken(ctx, m.refreshToken)
    if err != nil {
        // Delete invalid refresh token
        m.tokenStore.Delete("refresh_token")
        return nil, fmt.Errorf("token refresh failed: %w", err)
    }
    
    m.token = newToken.AccessToken
    m.tokenExp = newToken.ExpiresAt
    
    return newToken, nil
}
```

## Security Checklist

### Pre-Deployment Security Verification

- [ ] **Credential Storage**: All credentials stored in OS keychain or encrypted SQLite
- [ ] **Log Sanitization**: No credentials appear in any log output
- [ ] **TLS Configuration**: TLS 1.2+ enforced with secure cipher suites
- [ ] **Certificate Validation**: SSL certificate validation enabled and tested
- [ ] **Token Management**: Token refresh with 1-minute safety margin implemented
- [ ] **Retry Logic**: Exponential backoff with maximum retry limits
- [ ] **Timeout Configuration**: All requests have appropriate timeouts
- [ ] **Error Sanitization**: Error messages sanitized to remove sensitive data
- [ ] **Security Scanning**: Code scanned with gosec, go vet, and staticcheck
- [ ] **Test Coverage**: Security-critical code has ≥90% test coverage
- [ ] **Integration Testing**: End-to-end security flows tested
- [ ] **Performance Testing**: Security overhead measured and acceptable

### Runtime Security Monitoring

- [ ] **Request ID Tracking**: All API requests include tracking IDs
- [ ] **Error Classification**: Errors properly classified and monitored
- [ ] **Telemetry**: Security events sent to monitoring systems
- [ ] **Audit Logging**: Security-relevant events logged for compliance
- [ ] **Credential Rotation**: Automatic credential refresh monitored
- [ ] **Connection Security**: TLS connection security monitored
- [ ] **Rate Limiting**: Request rate limiting to prevent abuse

## Compliance Requirements

### Data Protection

- **Encryption at Rest**: All stored credentials encrypted with AES-256
- **Encryption in Transit**: All network communication uses TLS 1.2+
- **Data Minimization**: Only necessary credential data is stored
- **Secure Deletion**: Expired credentials are securely deleted
- **Access Control**: Credentials protected by OS-level access controls

### Audit and Monitoring

- **Request Tracking**: All API requests include unique request IDs
- **Error Classification**: Detailed error classification for security analysis
- **Telemetry**: Security metrics sent to monitoring systems
- **Structured Logging**: Security events logged in structured format
- **Compliance Reporting**: Security metrics available for compliance reporting

## Incident Response

### Security Incident Procedures

1. **Credential Compromise**:
   - Immediately revoke compromised credentials
   - Clear all stored tokens
   - Force re-authentication
   - Audit access logs

2. **TLS/Network Issues**:
   - Verify certificate validity
   - Check TLS configuration
   - Review network connectivity
   - Validate proxy settings

3. **Authentication Failures**:
   - Check credential provider chain
   - Verify token expiration
   - Review error classification
   - Check service availability

### Security Contact

For security issues or questions:
- Review this documentation
- Check application logs (credentials will be redacted)
- Verify configuration settings
- Test with minimal credentials

## Security Updates

This security documentation should be reviewed and updated:
- When new authentication methods are added
- When credential storage mechanisms change
- When network security requirements change
- After security incidents or audits
- At least quarterly as part of security reviews

---

**Last Updated**: January 22, 2026  
**Version**: 1.0.0  
**Next Review**: April 22, 2026
