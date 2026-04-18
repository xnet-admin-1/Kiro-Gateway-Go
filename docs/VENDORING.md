# Vendoring Guide

## Overview

Kiro Gateway uses Go modules vendoring to ensure:
- **Reproducible builds** - Same dependencies every time
- **Offline builds** - No internet required after initial vendor
- **Security** - Review all dependency code
- **Stability** - Protected from upstream changes

## What is Vendoring?

Vendoring copies all external dependencies into a `vendor/` directory in your project. This means:
- All dependencies are stored locally
- Builds don't require internet access
- Dependency versions are locked
- You can audit all third-party code

## Current Vendored Dependencies

The project vendors the following dependencies:

### Core Dependencies
- `github.com/aws/aws-sdk-go-v2` - AWS SDK for Go v2
- `github.com/aws/aws-sdk-go-v2/config` - AWS configuration
- `github.com/aws/aws-sdk-go-v2/service/ssooidc` - AWS SSO OIDC
- `github.com/aws/aws-sdk-go-v2/service/sts` - AWS STS
- `github.com/golang-jwt/jwt/v5` - JWT handling
- `github.com/joho/godotenv` - .env file loading
- `github.com/mattn/go-sqlite3` - SQLite database (CGO)
- `github.com/pkoukk/tiktoken-go` - Token counting
- `github.com/zalando/go-keyring` - Secure credential storage
- `golang.org/x/time` - Rate limiting

### Transitive Dependencies
All indirect dependencies are also vendored (see `vendor/modules.txt` for complete list).

## Directory Structure

```
kiro-gateway-go/
├── vendor/                    # All vendored dependencies
│   ├── github.com/           # GitHub dependencies
│   │   ├── aws/              # AWS SDK
│   │   ├── joho/             # godotenv
│   │   ├── mattn/            # go-sqlite3
│   │   └── ...
│   ├── golang.org/           # Go standard extensions
│   └── modules.txt           # Vendor manifest
├── go.mod                    # Module definition
├── go.sum                    # Dependency checksums
└── ...
```

## Building with Vendored Dependencies

### Using PowerShell Script (Windows)
```powershell
# Build with vendor
./vendor.ps1 build

# Test with vendor
./vendor.ps1 test

# Verify vendor
./vendor.ps1 verify

# Update vendor
./vendor.ps1 update
```

### Using Makefile (Linux/Mac)
```bash
# Build with vendor
make build-vendor

# Test with vendor
make test-vendor

# Verify vendor
make vendor-verify

# Update vendor
make vendor-update
```

### Using Go Directly
```bash
# Standard Build (Uses Vendor Automatically)
go build -o kiro-gateway.exe ./cmd/kiro-gateway

# Explicit Vendor Build
go build -mod=vendor -o kiro-gateway.exe ./cmd/kiro-gateway

# Verify Vendor is Used
go build -v -mod=vendor -o kiro-gateway.exe ./cmd/kiro-gateway
```

## Updating Vendored Dependencies

### 1. Update go.mod
```bash
# Update all dependencies to latest
go get -u ./...

# Update specific dependency
go get -u github.com/aws/aws-sdk-go-v2@latest

# Update to specific version
go get github.com/aws/aws-sdk-go-v2@v1.24.0
```

### 2. Tidy Dependencies
```bash
go mod tidy
```

### 3. Re-vendor
```bash
go mod vendor
```

### 4. Verify Build
```bash
go build -mod=vendor -o kiro-gateway.exe ./cmd/kiro-gateway
```

### 5. Test
```bash
go test -mod=vendor ./...
```

## Adding New Dependencies

### 1. Add Import to Code
```go
import "github.com/new/dependency"
```

### 2. Tidy and Vendor
```bash
go mod tidy
go mod vendor
```

### 3. Verify
```bash
go build -mod=vendor -o kiro-gateway.exe ./cmd/kiro-gateway
```

## Removing Dependencies

### 1. Remove Import from Code
Remove all imports of the dependency.

### 2. Tidy and Vendor
```bash
go mod tidy
go mod vendor
```

### 3. Verify
```bash
go build -mod=vendor -o kiro-gateway.exe ./cmd/kiro-gateway
```

## Verifying Vendored Dependencies

### Check Vendor Manifest
```bash
cat vendor/modules.txt
```

### List All Dependencies
```bash
go list -m all
```

### Check for Outdated Dependencies
```bash
go list -u -m all
```

### Verify Checksums
```bash
go mod verify
```

## Security Auditing

### 1. Review Vendor Directory
All dependency code is in `vendor/` - you can review it:
```bash
# List all vendored packages
ls -R vendor/

# Search for specific code
grep -r "suspicious_pattern" vendor/
```

### 2. Check for Known Vulnerabilities
```bash
# Install govulncheck
go install golang.org/x/vuln/cmd/govulncheck@latest

# Scan for vulnerabilities
govulncheck ./...
```

### 3. Audit Dependencies
```bash
# List direct dependencies
go list -m -json all | jq -r 'select(.Main != true) | select(.Indirect != true) | .Path'

# Check dependency licenses
go-licenses report ./... --template=licenses.tpl
```

## CI/CD Integration

### GitHub Actions Example
```yaml
name: Build with Vendor

on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Verify vendor is up to date
        run: |
          go mod tidy
          go mod vendor
          git diff --exit-code vendor/
      
      - name: Build with vendor
        run: go build -mod=vendor -o kiro-gateway ./cmd/kiro-gateway
      
      - name: Test with vendor
        run: go test -mod=vendor ./...
```

## Troubleshooting

### "vendor directory out of sync"
```bash
go mod tidy
go mod vendor
```

### "cannot find package in vendor"
```bash
# Re-vendor everything
rm -rf vendor/
go mod vendor
```

### "checksum mismatch"
```bash
# Clear module cache and re-download
go clean -modcache
go mod download
go mod vendor
```

### "build fails with vendor but works without"
```bash
# Check for missing dependencies
go mod tidy
go mod vendor

# Verify go.mod and go.sum are correct
go mod verify
```

## Best Practices

### 1. Always Vendor After Dependency Changes
```bash
go get -u ./...
go mod tidy
go mod vendor
```

### 2. Commit Vendor Directory
The `vendor/` directory should be committed to version control:
```bash
git add vendor/
git commit -m "Update vendored dependencies"
```

### 3. Keep Dependencies Updated
Regularly update dependencies for security patches:
```bash
# Monthly or quarterly
go get -u ./...
go mod tidy
go mod vendor
```

### 4. Review Dependency Changes
Before updating, review what will change:
```bash
go list -u -m all
```

### 5. Test After Vendoring
Always test after updating vendor:
```bash
go test -mod=vendor ./...
```

## Vendor vs Non-Vendor Builds

### With Vendor (Recommended for Production)
```bash
go build -mod=vendor -o kiro-gateway.exe ./cmd/kiro-gateway
```
- ✅ Uses local vendor/ directory
- ✅ No internet required
- ✅ Reproducible builds
- ✅ Faster builds (no download)

### Without Vendor (Development)
```bash
go build -o kiro-gateway.exe ./cmd/kiro-gateway
```
- ✅ Uses module cache (~/.go/pkg/mod)
- ⚠️ May download dependencies
- ⚠️ Requires internet on first build

## Module Cache vs Vendor

### Module Cache (`~/.go/pkg/mod`)
- Shared across all projects
- Downloaded on demand
- Can be cleared with `go clean -modcache`

### Vendor Directory (`./vendor`)
- Project-specific
- Committed to version control
- Always available offline

## Vendor Size

Current vendor directory size: ~50MB (varies with dependencies)

To check:
```bash
du -sh vendor/
```

## FAQ

### Q: Should I commit vendor/ to git?
**A: Yes!** This ensures reproducible builds and offline capability.

### Q: How often should I update dependencies?
**A: Monthly or quarterly**, or immediately for security patches.

### Q: Can I use vendor with go modules?
**A: Yes!** Vendor works seamlessly with go modules.

### Q: Does vendor slow down builds?
**A: No!** Vendor actually speeds up builds by avoiding downloads.

### Q: Can I mix vendor and non-vendor builds?
**A: Yes!** Use `-mod=vendor` to force vendor, or omit for module cache.

### Q: What if a dependency is missing from vendor?
**A: Run `go mod vendor` to re-vendor all dependencies.**

### Q: How do I audit vendored code?
**A: Review the `vendor/` directory directly - all code is there.**

## Summary

Vendoring provides:
- ✅ Reproducible builds
- ✅ Offline capability
- ✅ Security auditing
- ✅ Dependency stability
- ✅ Faster CI/CD builds
- ✅ No external dependencies at build time

The vendor directory is now part of the project and should be committed to version control.

## Commands Quick Reference

### PowerShell (Windows)
```powershell
# Initial vendor
./vendor.ps1 vendor

# Update dependencies
./vendor.ps1 update

# Build with vendor
./vendor.ps1 build

# Test with vendor
./vendor.ps1 test

# Verify vendor
./vendor.ps1 verify

# Show help
./vendor.ps1 help
```

### Makefile (Linux/Mac)
```bash
# Initial vendor
make vendor

# Update dependencies
make vendor-update

# Build with vendor
make build-vendor

# Test with vendor
make test-vendor

# Verify vendor
make vendor-verify
```

### Go Commands (All Platforms)
```bash
# Initial vendor
go mod tidy
go mod vendor

# Update dependencies
go get -u ./...
go mod tidy
go mod vendor

# Build with vendor
go build -mod=vendor -o kiro-gateway.exe ./cmd/kiro-gateway

# Test with vendor
go test -mod=vendor ./...

# Verify vendor
go mod verify

# Check for updates
go list -u -m all

# Clean and re-vendor
rm -rf vendor/
go mod vendor
```

For more information, see:
- [Go Modules Reference](https://go.dev/ref/mod)
- [Go Modules Wiki](https://github.com/golang/go/wiki/Modules)
