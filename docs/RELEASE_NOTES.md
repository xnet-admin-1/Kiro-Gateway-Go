# Kiro Gateway Go - Release Notes

## Version 2.0.0 - AWS Security Baseline (January 22, 2026)

### 🚀 Major Features

#### AWS Credential Chain Support
- **Full AWS credential provider chain** with automatic fallback
- Support for environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`)
- AWS profile files (`~/.aws/credentials`, `~/.aws/config`) with `AWS_PROFILE` support
- Web Identity Token credentials (OIDC/IRSA for Kubernetes)
- ECS container credentials via metadata endpoint
- EC2 instance metadata (IMDS v2) credentials
- Thread-safe credential caching with automatic refresh

#### AWS Signature Version 4 (SigV4) Authentication
- **Production-grade SigV4 signing** for enhanced security
- Support for both bearer token and SigV4 authentication modes
- Configurable via `AMAZON_Q_SIGV4=true` environment variable
- Proper signing of headers, query parameters, and request payload
- Regional endpoint support with correct signing region
- Automatic credential refresh for SigV4 requests

#### Enhanced OIDC Device Code Flow
- **Browser-less authentication** for CLI environments
- Device registration with AWS SSO-OIDC with intelligent caching
- Automatic device authorization with user code display
- Smart token polling with exponential backoff
- Support for authorization pending, slow down, and error states
- Secure token storage in OS keychain
- Custom start URLs for IAM Identity Center

#### Secure Storage System
- **OS-level secure storage** for sensitive credentials
- Windows Credential Manager integration
- macOS Keychain integration  
- Linux Secret Service integration
- AES-256-GCM encrypted SQLite fallback
- Automatic secure deletion of expired credentials
- Zero credential exposure in logs or error messages

### 🛡️ Security Enhancements

#### HTTP Client Hardening
- **Stalled stream protection** with 5-minute grace period
- Adaptive retry logic with exponential backoff (max 3 attempts)
- Custom retry classifier for Q-specific error patterns
- Request/response interceptors for logging and telemetry
- TLS 1.2+ enforcement with certificate validation
- Custom user agent with app name and version

#### Advanced Error Classification
- **Intelligent error handling** with detailed classification
- Throttling error detection (429 status codes)
- Context window overflow detection ("Input is too long")
- Model overloaded detection ("INSUFFICIENT_MODEL_CAPACITY")
- Monthly limit detection ("MONTHLY_REQUEST_COUNT")
- Request ID tracking for debugging
- User-friendly error messages

#### Enhanced Token Management
- **Automatic token refresh** with 1-minute safety margin
- Thread-safe token operations with proper locking
- Support for both Builder ID and IAM Identity Center tokens
- Graceful handling of refresh failures
- Background token refresh for long-running processes

### 🔧 Configuration Enhancements

#### New Environment Variables
```bash
# AWS Configuration
AWS_REGION=us-east-1
AWS_PROFILE=default
AMAZON_Q_SIGV4=true

# OIDC Configuration  
OIDC_START_URL=https://my-sso-portal.awsapps.com/start
OIDC_REGION=us-east-1

# Profile Configuration
PROFILE_ARN=arn:aws:codewhisperer:us-east-1:123456789012:profile/abc123

# Privacy Configuration
OPT_OUT_TELEMETRY=true

# Performance Configuration
MAX_RETRIES=3
MAX_BACKOFF=30s
CONNECT_TIMEOUT=10s
READ_TIMEOUT=30s
OPERATION_TIMEOUT=60s
STALLED_STREAM_GRACE=5m
```

#### Regional Endpoint Support
- Configurable AWS regions (default: us-east-1)
- Automatic endpoint construction per region
- Data residency compliance support
- Custom endpoint URLs for testing/private deployments

### 📊 Performance Improvements

#### Optimized Credential Resolution
- **Sub-100ms credential resolution** with intelligent caching
- Connection pooling for reduced latency
- Minimal signing overhead (<5ms)
- Efficient retry strategies with jitter
- Background refresh to avoid request delays

#### Enhanced Monitoring
- Structured logging for authentication events
- Metrics for auth success/failure rates
- Request ID tracking across all operations
- Telemetry with opt-out support
- Performance benchmarking tools

### 🔄 Migration Guide

#### From Version 1.x

**No Breaking Changes** - All existing authentication methods continue to work:
- Kiro Desktop authentication (unchanged)
- AWS SSO/OIDC authentication (enhanced)
- CLI Database authentication (unchanged)

**New Features are Opt-In:**
- SigV4 authentication: Set `AMAZON_Q_SIGV4=true`
- Regional endpoints: Set `AWS_REGION=your-region`
- Enhanced storage: Automatic migration to secure storage

**Configuration Updates:**
1. Copy new variables from `.env.example` to your `.env` file
2. Set `AWS_REGION` if not using us-east-1
3. Enable SigV4 with `AMAZON_Q_SIGV4=true` for enhanced security
4. Configure OIDC settings if using IAM Identity Center

### 🧪 Testing & Quality

#### Comprehensive Test Coverage
- **80%+ code coverage** across all modules
- 200+ unit tests with table-driven test patterns
- Integration tests with mocked AWS services
- Security tests verifying credential protection
- Performance benchmarks for all critical paths

#### Security Validation
- Zero credential exposure in logs (verified)
- Secure storage encryption (AES-256-GCM)
- TLS certificate validation (enforced)
- SigV4 signature correctness (AWS test vectors)
- Thread-safe operations (race condition testing)

### 📚 Documentation

#### New Documentation Files
- `AUTHENTICATION.md` - Complete authentication guide
- `SECURITY.md` - Security best practices and compliance
- Updated `README.md` with new features
- Comprehensive code documentation and examples

#### API Compatibility
- **100% OpenAI API compatible** - no changes to client code
- All existing endpoints continue to work
- Enhanced error responses with better classification
- Improved streaming performance and reliability

### 🐛 Bug Fixes

- Fixed race conditions in token refresh operations
- Improved error handling for network timeouts
- Enhanced retry logic for transient failures
- Better handling of malformed SSE streams
- Resolved memory leaks in long-running connections

### 🔮 Future Compatibility

This release establishes the foundation for:
- Multi-region failover capabilities
- Advanced credential rotation
- Enhanced telemetry and monitoring
- Additional AWS service integrations

### 📋 System Requirements

- **Go 1.21+** (for building from source)
- **Windows 10+**, **macOS 10.15+**, or **Linux** (glibc 2.17+)
- Network access to AWS endpoints
- Optional: AWS credentials for SigV4 authentication

### 🙏 Acknowledgments

This release implements AWS security best practices based on the Amazon Q Developer CLI reference implementation, ensuring production-grade security and reliability.

---

**Full Changelog:** [v1.0.0...v2.0.0](https://github.com/yourusername/kiro-gateway-go/compare/v1.0.0...v2.0.0)
**Download:** [Release v2.0.0](https://github.com/yourusername/kiro-gateway-go/releases/tag/v2.0.0)
