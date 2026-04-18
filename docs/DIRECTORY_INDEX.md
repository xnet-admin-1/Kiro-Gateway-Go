# Kiro Gateway - Directory Index

Complete index of all files and directories in this package.

Generated: 2026-01-22 (Updated after cleanup)

## Directory Structure

```
├── .env
├── .env.example
├── .env.local
├── .env.test
├── .env.xnet-admin
├── .gitignore
├── API_KEY_MANAGEMENT.md
├── API_KEY_docs/guides/docs/guides/docs/guides/docs/guides/QUICK_START.md
├── AUTHENTICATION.md
├── BETA_FEATURES_GUIDE.md
├── build_simple.ps1
├── build.config
├── build.ps1
├── build.sh
├── CONCURRENCY_ARCHITECTURE.md
├── CONCURRENCY_COMPLETE_GUIDE.md
├── CREDENTIAL_STORAGE_METHODS.md
├── DIRECTORY_INDEX.md
├── docker-compose.yml
├── Dockerfile
├── go.mod
├── go.sum
├── kiro-gateway-test.exe
├── kiro-gateway-vendored.exe
├── kiro-gateway.exe
├── Makefile
├── PACKAGE_MANIFEST.md
├── PROJECT_STATUS.md
├── QUICK_REFERENCE_BETA_FEATURES.md
├── QUICK_REFERENCE.md
├── QUICKSTART.md
├── README.md
├── RELEASE_NOTES.md
├── SECURITY.md
├── STANDALONE_README.md
├── test_api_keys.ps1
├── test_beta_features.ps1
├── test_both_modes.ps1
├── test_codewhisperer_mode.ps1
├── test_codewhisperer_quick.ps1
├── test_concurrency.ps1
├── test_gateway_health.ps1
├── test_handshake.ps1
├── test_qdeveloper_mode.ps1
├── test_qdeveloper_quick.ps1
├── test_quick.ps1
├── test_simple.ps1
├── test_streaming.ps1
├── test_visual.ps1
├── test_xnet_admin.ps1
├── VALIDATION_SYSTEM.md
├── vendor.ps1
├── VENDORING.md
├── verify_package.ps1
├── .github/
│   └── workflows/
│       └── ci.yml
├── cmd/
│   ├── kiro-gateway/
│   │   ├── main_test.go
│   │   └── main.go
│   ├── manual-test-report/
│   │   └── main.go
│   └── performance-analyzer/
│       └── main.go
├── dev/
│   └── null
├── dist/
│   └── (build artifacts)
├── examples/
│   ├── README.md
│   ├── bearer_token/
│   │   ├── main.go
│   │   └── README.md
│   ├── credential_chain/
│   │   ├── main.go
│   │   └── README.md
│   ├── oidc_device_code/
│   │   ├── main.go
│   │   └── README.md
│   └── sigv4/
│       ├── main.go
│       └── README.md
├── internal/
│   ├── apikeys/
│   │   ├── manager.go
│   │   └── storage.go
│   ├── async/
│   │   └── job_manager.go
│   ├── auth/
│   │   ├── auth_test.go
│   │   ├── auth.go
│   │   ├── bearer_test.go
│   │   ├── bearer.go
│   │   ├── cli_db.go
│   │   ├── comprehensive_bearer_test.go
│   │   ├── desktop.go
│   │   ├── integration_test.go
│   │   ├── manager_comprehensive_test.go
│   │   ├── oidc.go
│   │   ├── credentials/
│   │   ├── oidc/
│   │   └── sigv4/
│   ├── client/
│   │   ├── client_test.go
│   │   ├── client.go
│   │   ├── interceptors_test.go
│   │   ├── interceptors.go
│   │   ├── retry_test.go
│   │   ├── retry.go
│   │   ├── stalledstream_test.go
│   │   └── stalledstream.go
│   ├── concurrency/
│   │   ├── circuit_breaker.go
│   │   ├── connection_pool.go
│   │   ├── job.go
│   │   ├── load_shedder.go
│   │   ├── priority_queue.go
│   │   └── worker_pool.go
│   ├── config/
│   │   ├── config_test.go
│   │   ├── config.go
│   │   └── features.go
│   ├── converters/
│   │   ├── comprehensive_test.go
│   │   ├── conversation.go
│   │   ├── openai_test.go
│   │   └── openai.go
│   ├── errors/
│   │   ├── classifier_test.go
│   │   ├── classifier.go
│   │   ├── test_output.txt
│   │   ├── types_test.go
│   │   └── types.go
│   ├── handlers/
│   │   ├── apikeys.go
│   │   ├── async.go
│   │   ├── chat_test.go
│   │   ├── chat.go
│   │   ├── health.go
│   │   ├── metrics.go
│   │   ├── middleware.go
│   │   ├── models.go
│   │   └── routes.go
│   ├── hotpath/
│   │   ├── analyzer_test.go
│   │   └── analyzer.go
│   ├── models/
│   │   ├── conversation.go
│   │   ├── kiro.go
│   │   └── openai.go
│   ├── optimization/
│   │   ├── optimizer_test.go
│   │   └── optimizer.go
│   ├── profiling/
│   │   ├── profiler_test.go
│   │   └── profiler.go
│   ├── storage/
│   │   ├── comprehensive_store_test.go
│   │   ├── comprehensive_test.go
│   │   ├── encryption_test.go
│   │   ├── encryption.go
│   │   ├── integration_test.go
│   │   ├── keychain_test.go
│   │   ├── keychain.go
│   │   ├── mock_nocgo.go
│   │   ├── sqlite_cgo.go
│   │   ├── sqlite_test.go
│   │   ├── sqlite.go
│   │   ├── store_test.go
│   │   └── store.go
│   ├── streaming/
│   │   ├── eventstream.go
│   │   ├── parser.go
│   │   └── streaming.go
│   └── validation/
│       ├── limits.go
│       ├── ratelimiter.go
│       ├── validator_test.go
│       └── validator.go
├── pkg/
│   └── tokenizer/
│       ├── comprehensive_test.go
│       ├── tokenizer_test.go
│       └── tokenizer.go
├── tests/
│   ├── backward_compatibility_test.go
│   ├── integration/
│   │   ├── api_requests_test.go
│   │   ├── auth_flows_test.go
│   │   ├── migration_test.go
│   │   ├── mock_auth.go
│   │   └── system_integration_test.go
│   ├── manual/
│   │   ├── manual_test_suite.go
│   │   └── manual_test.go
│   ├── performance/
│   │   ├── basic_performance_test.go
│   │   ├── README.md
│   │   ├── sigv4_performance_test.go
│   │   └── task43_2_comprehensive_test.go
│   └── security/
│       ├── auth_security_test.go
│       ├── credential_protection_test.go
│       ├── security_scanner.go
│       ├── storage_security_test.go
│       └── task_44_2_security_test.go
└── vendor/
    ├── modules.txt
    ├── al.essio.dev/
    │   └── pkg/
    ├── github.com/
    │   ├── aws/
    │   ├── danieljoos/
    │   ├── dlclark/
    │   ├── godbus/
    │   ├── golang-jwt/
    │   ├── google/
    │   ├── joho/
    │   ├── mattn/
    │   ├── pkoukk/
    │   └── zalando/
    └── golang.org/
        └── x/
```

## File Categories

### Core Executables
- `kiro-gateway.exe` - Main production binary
- `kiro-gateway-test.exe` - Test build
- `kiro-gateway-vendored.exe` - Vendored build

### Source Code
- Go source files: ~80 files
- Go test files: ~40 files
- Total lines: ~15,000+

### Documentation (14 files)
- `README.md` - Main project documentation
- `STANDALONE_README.md` - Standalone package guide
- `QUICKSTART.md` - Quick start guide
- `API_KEY_MANAGEMENT.md` - API key system docs
- `API_KEY_docs/guides/docs/guides/docs/guides/docs/guides/QUICK_START.md` - API key quick guide
- `AUTHENTICATION.md` - Authentication guide
- `BETA_FEATURES_GUIDE.md` - Beta features documentation
- `CONCURRENCY_ARCHITECTURE.md` - Concurrency design
- `CONCURRENCY_COMPLETE_GUIDE.md` - Concurrency guide
- `CREDENTIAL_STORAGE_METHODS.md` - Credential storage docs
- `VALIDATION_SYSTEM.md` - Validation system docs
- `VENDORING.md` - Vendoring guide
- `SECURITY.md` - Security documentation
- `RELEASE_NOTES.md` - Release notes

### Configuration Files
- `.env*` - Environment configurations (5 files)
- `build.config` - Build configuration
- `docker-compose.yml` - Docker compose config
- `Dockerfile` - Docker build file
- `go.mod` / `go.sum` - Go module files
- `Makefile` - Build automation

### Build Scripts
- `build.sh` - Linux/Mac build script
- `build.ps1` - Windows build script
- `build_simple.ps1` - Simple Windows build
- `vendor.ps1` - Vendor management script
- `verify_package.ps1` - Package verification

### Test Scripts (15 files)
- `test_quick.ps1` - Quick test suite
- `test_simple.ps1` - Simple test
- `test_api_keys.ps1` - API key tests
- `test_beta_features.ps1` - Beta feature tests
- `test_both_modes.ps1` - Dual mode tests
- `test_codewhisperer_mode.ps1` - CodeWhisperer tests
- `test_codewhisperer_quick.ps1` - Quick CW tests
- `test_concurrency.ps1` - Concurrency tests
- `test_gateway_health.ps1` - Health check tests
- `test_handshake.ps1` - Handshake tests
- `test_qdeveloper_mode.ps1` - Q Developer tests
- `test_qdeveloper_quick.ps1` - Quick QD tests
- `test_streaming.ps1` - Streaming tests
- `test_visual.ps1` - Visual/multimodal tests
- `test_xnet_admin.ps1` - Admin tests

### Dependencies
- Vendored packages: 50+
- Vendor directory size: ~13 MB
- All dependencies included for offline builds

## Package Status

✅ **Clean and Production-Ready**
- All old files archived
- All dependencies vendored
- Complete documentation
- Comprehensive test suite
- Ready for standalone deployment

