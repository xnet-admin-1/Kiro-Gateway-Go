# Scripts Directory

This directory contains all build, test, and utility scripts for the Kiro Gateway project.

## Build Scripts

### build.ps1 (Windows) / build.sh (Linux/Mac)
Full build script with testing and validation.

```bash
# Windows
.\scripts\build.ps1

# Linux/Mac
./scripts/build.sh
```

**What it does:**
- Runs all tests
- Builds for all platforms (Linux, macOS, Windows)
- Creates binaries in `dist/` directory
- Generates checksums
- Creates release package

### build_simple.ps1
Quick build without full test suite.

```powershell
.\scripts\build_simple.ps1
```

**What it does:**
- Builds for all platforms
- Validates binaries
- Faster than full build

### vendor.ps1
Vendor dependencies for offline builds.

```powershell
.\scripts\vendor.ps1
```

**What it does:**
- Downloads all Go dependencies
- Creates `vendor/` directory
- Enables air-gapped builds

### verify_package.ps1
Verify build package integrity.

```powershell
.\scripts\verify_package.ps1
```

## Test Scripts

### test_gateway_health.ps1
Comprehensive health check.

```powershell
.\scripts\test_gateway_health.ps1
```

**Tests:**
- Gateway is running
- Authentication works
- AWS profile configured
- Correct API mode

### test_api_keys.ps1
API key management tests.

```powershell
.\scripts\test_api_keys.ps1
```

**Tests:**
- Admin key exists
- Key creation
- Key listing
- Key revocation
- Key deletion

### test_visual.ps1
Vision/multimodal functionality tests.

```powershell
.\scripts\test_visual.ps1
```

**Tests:**
- Single image processing
- Multiple images
- External image URLs
- Image format support

### test_streaming.ps1
Streaming response tests.

```powershell
.\scripts\test_streaming.ps1
```

**Tests:**
- SSE streaming
- Event parsing
- Token counting
- Stream interruption

### test_both_modes.ps1
Test both CodeWhisperer and Q Developer modes.

```powershell
.\scripts\test_both_modes.ps1
```

**Tests:**
- Bearer token mode
- SigV4 mode
- Mode switching
- Configuration validation

### test_codewhisperer_mode.ps1
CodeWhisperer-specific tests.

```powershell
.\scripts\test_codewhisperer_mode.ps1
```

### test_qdeveloper_mode.ps1
Q Developer-specific tests.

```powershell
.\scripts\test_qdeveloper_mode.ps1
```

### test_quick.ps1
Quick smoke test.

```powershell
.\scripts\test_quick.ps1
```

### test_simple.ps1
Simple request/response test.

```powershell
.\scripts\test_simple.ps1
```

### test_models_endpoint.ps1
Test /v1/models endpoint.

```powershell
.\scripts\test_models_endpoint.ps1
```

### test_concurrency.ps1
Concurrency and performance tests.

```powershell
.\scripts\test_concurrency.ps1
```

### test_beta_features.ps1
Beta feature tests.

```powershell
.\scripts\test_beta_features.ps1
```

### test_handshake.ps1
Connection handshake tests.

```powershell
.\scripts\test_handshake.ps1
```

## Utility Scripts

### get_profile_arn.ps1
Retrieve Q Developer profile ARN.

```powershell
.\scripts\get_profile_arn.ps1
```

**What it does:**
- Runs `qchat profile` command
- Extracts profile ARN
- Displays for configuration

### set_aws_creds.ps1
Set AWS credentials for testing.

```powershell
.\scripts\set_aws_creds.ps1
```

**What it does:**
- Sets AWS environment variables
- Configures test credentials
- Validates configuration

## Usage Patterns

### Development Workflow

```powershell
# 1. Make code changes
# 2. Quick build
.\scripts\build_simple.ps1

# 3. Run health check
.\scripts\test_gateway_health.ps1

# 4. Run specific tests
.\scripts\test_visual.ps1

# 5. Full build before commit
.\scripts\build.ps1
```

### Testing Workflow

```powershell
# 1. Start gateway
.\kiro-gateway.exe

# 2. Run all tests
.\scripts\test_gateway_health.ps1
.\scripts\test_api_keys.ps1
.\scripts\test_visual.ps1
.\scripts\test_streaming.ps1
.\scripts\test_both_modes.ps1

# 3. Review results
```

### Release Workflow

```powershell
# 1. Update version
# 2. Run full test suite
go test ./...

# 3. Build all platforms
.\scripts\build.ps1

# 4. Verify package
.\scripts\verify_package.ps1

# 5. Create release
# Package dist/ directory
```

## Script Requirements

### Windows
- PowerShell 5.1+
- Go 1.21+
- curl (for API tests)

### Linux/Mac
- Bash
- Go 1.21+
- curl

## Environment Variables

Scripts may use these environment variables:

- `PORT` - Gateway port (default: 8080)
- `AWS_REGION` - AWS region
- `AWS_PROFILE` - AWS profile name
- `ADMIN_API_KEY` - Admin API key for tests
- `PROXY_API_KEY` - Test API key

## Troubleshooting

### Script Won't Run (Windows)

```powershell
# Set execution policy
Set-ExecutionPolicy RemoteSigned -Scope CurrentUser
```

### Script Won't Run (Linux/Mac)

```bash
# Make executable
chmod +x scripts/*.sh
```

### Tests Fail

1. Check gateway is running
2. Verify port configuration
3. Check API keys are valid
4. Ensure AWS credentials configured

### Build Fails

1. Check Go version: `go version`
2. Download dependencies: `go mod download`
3. Check disk space
4. Review error messages

## Contributing

When adding new scripts:

1. Add to appropriate category (build/test/utility)
2. Document in this README
3. Include usage examples
4. Add error handling
5. Test on all platforms

## See Also

- [Main README](../README.md)
- [Agents & Automation Guide](../docs/AGENTS_AND_AUTOMATION.md)
- [Quick Start Guide](../docs/QUICKSTART.md)
