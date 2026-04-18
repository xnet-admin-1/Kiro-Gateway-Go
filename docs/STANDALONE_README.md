# Kiro Gateway - Standalone Package

This is a complete, self-contained package of Kiro Gateway that can be moved to any location.

## 📦 Package Contents

This directory contains everything needed to build, run, and deploy Kiro Gateway:

### Source Code
- `cmd/` - Application entry points
- `internal/` - Internal packages (auth, handlers, concurrency, etc.)
- `pkg/` - Public packages
- `examples/` - Example implementations
- `tests/` - Test suites

### Dependencies
- `vendor/` - All vendored dependencies (~50MB)
- `go.mod` - Module definition
- `go.sum` - Dependency checksums

### Documentation
- `README.md` - Main gateway documentation
- `QUICKSTART.md` - Quick setup guide
- `QUICK_REFERENCE.md` - Command reference
- `AUTHENTICATION.md` - Authentication modes
- `API_KEY_MANAGEMENT.md` - API key management
- `CONCURRENCY_ARCHITECTURE.md` - Concurrency system
- `BETA_FEATURES_GUIDE.md` - Beta features
- `VALIDATION_SYSTEM.md` - Request validation
- `VENDORING.md` - Dependency management
- `SECURITY.md` - Security best practices
- And more...

### Configuration
- `.env.example` - Example configuration
- `.gitignore` - Git ignore rules
- `Makefile` - Build automation (Linux/Mac)
- `vendor.ps1` - Vendor management (Windows)

### Build Scripts
- `build.ps1` - Windows build script
- `build.sh` - Linux/Mac build script
- `test_*.ps1` - Various test scripts

## 🚀 Quick Start

### 1. Move to Desired Location
```bash
# Move this entire directory anywhere
mv kiro-gateway-go ~/projects/kiro-gateway
cd ~/projects/kiro-gateway
```

### 2. Configure
```bash
# Copy example config
cp .env.example .env

# Edit configuration
# Set AWS credentials, region, etc.
```

### 3. Build
```bash
# Windows
go build -o kiro-gateway.exe ./cmd/kiro-gateway

# Linux/Mac
go build -o kiro-gateway ./cmd/kiro-gateway

# Or use vendor script (Windows)
./vendor.ps1 build

# Or use Makefile (Linux/Mac)
make build
```

### 4. Run
```bash
# Windows
./kiro-gateway.exe

# Linux/Mac
./kiro-gateway
```

## 📋 Requirements

### Minimum Requirements
- Go 1.21 or later
- CGO enabled (for SQLite support)
- ~100MB disk space (with vendor)

### Optional Requirements
- Make (for Makefile support on Linux/Mac)
- Docker (for containerized deployment)
- Git (for version control)

## 🔧 Configuration

### Environment Variables
Create a `.env` file with:

```bash
# Server
PORT=8080

# AWS Configuration
AWS_REGION=us-east-1
AWS_PROFILE=your-profile

# Authentication
PROXY_API_KEY=your-api-key
ADMIN_API_KEY=your-admin-key

# API Mode
Q_USE_SENDMESSAGE=true
AMAZON_Q_SIGV4=true

# Beta Features
ENABLE_EXTENDED_CONTEXT=true
ENABLE_EXTENDED_THINKING=true

# API Key Storage
API_KEY_STORAGE_DIR=.kiro/api-keys
```

## 🏗️ Building

### Standard Build
```bash
go build -o kiro-gateway.exe ./cmd/kiro-gateway
```

### Vendored Build (Offline)
```bash
# Windows
./vendor.ps1 build

# Linux/Mac
make build-vendor

# Or directly
go build -mod=vendor -o kiro-gateway.exe ./cmd/kiro-gateway
```

### Multi-Platform Build
```bash
# Linux/Mac
make build-all

# Windows
go build -o kiro-gateway-windows.exe ./cmd/kiro-gateway
GOOS=linux go build -o kiro-gateway-linux ./cmd/kiro-gateway
GOOS=darwin go build -o kiro-gateway-macos ./cmd/kiro-gateway
```

## 🧪 Testing

### Run All Tests
```bash
go test ./...
```

### Run Specific Tests
```bash
# API key tests
./test_api_keys.ps1

# Concurrency tests
./test_concurrency.ps1

# Gateway health
./test_gateway_health.ps1
```

### Run with Coverage
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 📦 Deployment

### Local Installation
```bash
# Install to ~/go/bin
go install ./cmd/kiro-gateway

# Run from anywhere
kiro-gateway
```

### Docker Deployment
```bash
# Build image
docker build -t kiro-gateway .

# Run container
docker run -p 8080:8080 --env-file .env kiro-gateway
```

### Production Deployment
See `QUICKSTART.md` for detailed production deployment guide.

## 📚 Documentation

All documentation is included in this directory:

### Getting Started
- `README.md` - Overview and features
- `QUICKSTART.md` - Quick setup guide
- `QUICK_REFERENCE.md` - Command reference

### Features
- `API_KEY_MANAGEMENT.md` - Multi-key management
- `CONCURRENCY_ARCHITECTURE.md` - Scalability
- `BETA_FEATURES_GUIDE.md` - Extended features
- `VALIDATION_SYSTEM.md` - Request validation

### Configuration
- `AUTHENTICATION.md` - Auth modes
- `VENDORING.md` - Dependency management
- `SECURITY.md` - Security practices

### API Reference
- `OPENAI_API_COMPATIBILITY_VERIFICATION.md` - OpenAI compatibility
- `OPENAI_CLIENT_EXAMPLES.md` - Client examples

## 🔐 Security

### Best Practices
- Store credentials securely
- Use API key expiration
- Enable rate limiting
- Monitor usage metrics
- Regular dependency updates

### Security Features
- Secure credential storage (keyring)
- API key encryption
- Request validation
- Rate limiting
- Circuit breaker
- Load shedding

## 🆘 Support

### Common Issues

**Build fails**
```bash
go mod tidy
go mod vendor
go build ./cmd/kiro-gateway
```

**Auth fails**
- Check AWS credentials
- Verify region configuration
- Check profile permissions

**Port in use**
- Change PORT in .env
- Check for other running instances

### Troubleshooting
See `QUICK_REFERENCE.md` for detailed troubleshooting guide.

## 📊 Package Information

### Size
- Source code: ~5MB
- Vendor dependencies: ~50MB
- Documentation: ~1MB
- Total: ~56MB

### Dependencies
- 50+ packages (all vendored)
- No external downloads required
- Offline build capable

### Compatibility
- Windows (x64, arm64)
- Linux (x64, arm64)
- macOS (x64, arm64)

## 🎯 Features

### Core Features
- ✅ OpenAI API compatible
- ✅ AWS Q Developer integration
- ✅ Multimodal support (text + images)
- ✅ Streaming responses
- ✅ Tool calling

### Enterprise Features
- ✅ Multi-key management
- ✅ Concurrency system
- ✅ Rate limiting
- ✅ Request validation
- ✅ Circuit breaker
- ✅ Connection pooling

### Authentication
- ✅ Bearer token
- ✅ AWS SigV4
- ✅ OIDC device code
- ✅ Hybrid mode

## 📝 License

See LICENSE files in this directory.

## 🔄 Updates

### Updating Dependencies
```bash
# Update all
go get -u ./...
go mod tidy
go mod vendor

# Or use scripts
./vendor.ps1 update  # Windows
make vendor-update   # Linux/Mac
```

### Checking for Updates
```bash
go list -u -m all
```

## 🚢 Moving This Package

This directory is completely self-contained and can be moved anywhere:

```bash
# Move to any location
mv kiro-gateway-go /path/to/new/location

# Or copy
cp -r kiro-gateway-go /path/to/new/location

# Or compress for transfer
tar -czf kiro-gateway.tar.gz kiro-gateway-go
# or
zip -r kiro-gateway.zip kiro-gateway-go
```

Everything will continue to work after moving because:
- All dependencies are vendored
- All documentation is included
- All configuration is relative
- No absolute paths are used

## ✅ Verification

After moving, verify the package:

```bash
cd /new/location/kiro-gateway-go

# Check vendor
ls vendor/

# Check build
go build ./cmd/kiro-gateway

# Check tests
go test ./...

# Check docs
ls *.md
```

## 📞 Contact

For issues, questions, or contributions, see the main project repository.

---

**Package Version**: 1.0.0  
**Package Date**: January 22, 2026  
**Status**: ✅ Production Ready
