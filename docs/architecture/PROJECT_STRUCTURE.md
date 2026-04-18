# Project Structure

Complete overview of the Kiro Gateway project organization.

## 📁 Directory Layout

```
kiro-gateway-go/
├── README.md                    # Main project documentation
├── docs/architecture/docs/architecture/PROJECT_STRUCTURE.md         # This file
├── go.mod                       # Go module definition
├── go.sum                       # Go dependency checksums
├── Dockerfile                   # Docker build configuration
├── docker-compose.yml           # Docker Compose setup
├── Makefile                     # Build automation
│
├── .env                         # Environment configuration (gitignored)
├── .env.example                 # Example environment file
├── .env.*                       # Environment variants
├── .gitignore                   # Git ignore rules
│
├── cmd/                         # Application entry points
│   ├── kiro-gateway/           # Main gateway application
│   │   ├── main.go             # Application entry point
│   │   └── main_test.go        # Entry point tests
│   ├── manual-test-report/     # Manual testing tool
│   └── performance-analyzer/   # Performance analysis tool
│
├── internal/                    # Private application code
│   ├── apikeys/                # API key management
│   │   ├── manager.go          # Key lifecycle management
│   │   └── storage.go          # Key storage backend
│   │
│   ├── async/                  # Async job processing
│   │   └── job_manager.go      # Job queue management
│   │
│   ├── auth/                   # Authentication system
│   │   ├── auth.go             # Auth manager
│   │   ├── bearer.go           # Bearer token auth
│   │   ├── cli_db.go           # CLI database auth
│   │   ├── desktop.go          # Desktop auth
│   │   ├── oidc.go             # OIDC auth
│   │   ├── credentials/        # AWS credential providers
│   │   │   ├── cache.go        # Credential caching
│   │   │   ├── chain.go        # Credential chain
│   │   │   ├── credentials.go  # Base credentials
│   │   │   ├── ecs.go          # ECS credentials
│   │   │   ├── env.go          # Environment credentials
│   │   │   ├── imds.go         # EC2 IMDS credentials
│   │   │   ├── profile.go      # AWS profile credentials
│   │   │   ├── provider.go     # Provider interface
│   │   │   └── webidentity.go  # Web identity credentials
│   │   ├── oidc/               # OIDC implementation
│   │   │   ├── authorization.go
│   │   │   ├── client.go
│   │   │   ├── polling.go
│   │   │   └── registration.go
│   │   └── sigv4/              # AWS SigV4 signing
│   │       ├── canonical.go    # Canonical request
│   │       ├── signature.go    # Signature generation
│   │       ├── signer.go       # Request signer
│   │       ├── signingkey.go   # Signing key derivation
│   │       └── stringtosign.go # String to sign
│   │
│   ├── client/                 # HTTP client
│   │   ├── client.go           # HTTP client wrapper
│   │   ├── interceptors.go     # Request/response interceptors
│   │   ├── retry.go            # Retry logic
│   │   └── stalledstream.go    # Stalled stream detection
│   │
│   ├── concurrency/            # Concurrency system
│   │   ├── circuit_breaker.go  # Circuit breaker pattern
│   │   ├── connection_pool.go  # Connection pool management
│   │   ├── job.go              # Job definition
│   │   ├── load_shedder.go     # Load shedding
│   │   ├── priority_queue.go   # Priority queue
│   │   └── worker_pool.go      # Worker pool
│   │
│   ├── config/                 # Configuration
│   │   ├── config.go           # Config loading
│   │   └── features.go         # Feature flags
│   │
│   ├── converters/             # Format converters
│   │   ├── conversation.go     # Conversation format
│   │   └── openai.go           # OpenAI format conversion
│   │
│   ├── errors/                 # Error handling
│   │   ├── classifier.go       # Error classification
│   │   └── types.go            # Error types
│   │
│   ├── handlers/               # HTTP handlers
│   │   ├── apikeys.go          # API key endpoints
│   │   ├── async.go            # Async job endpoints
│   │   ├── chat.go             # Chat completion endpoint
│   │   ├── health.go           # Health check endpoint
│   │   ├── metrics.go          # Metrics endpoint
│   │   ├── middleware.go       # HTTP middleware
│   │   ├── models.go           # Models endpoint
│   │   └── routes.go           # Route definitions
│   │
│   ├── hotpath/                # Performance optimization
│   │   └── analyzer.go         # Hot path analysis
│   │
│   ├── models/                 # Data models
│   │   ├── conversation.go     # Conversation models
│   │   ├── kiro.go             # Kiro-specific models
│   │   └── openai.go           # OpenAI models
│   │
│   ├── optimization/           # Performance optimization
│   │   └── optimizer.go        # Request optimizer
│   │
│   ├── profiling/              # Performance profiling
│   │   └── profiler.go         # Profiler implementation
│   │
│   ├── storage/                # Data storage
│   │   ├── encryption.go       # Encryption utilities
│   │   ├── keychain.go         # OS keychain integration
│   │   ├── sqlite.go           # SQLite storage
│   │   └── store.go            # Storage interface
│   │
│   ├── streaming/              # Streaming support
│   │   ├── eventstream.go      # AWS event stream parser
│   │   ├── parser.go           # Stream parser
│   │   └── streaming.go        # Streaming logic
│   │
│   └── validation/             # Request validation
│       ├── limits.go           # Rate limits
│       ├── ratelimiter.go      # Rate limiter
│       └── validator.go        # Request validator
│
├── pkg/                        # Public packages
│   └── tokenizer/              # Token counting
│       └── tokenizer.go        # Tokenizer implementation
│
├── tests/                      # Test suites
│   ├── integration/            # Integration tests
│   │   ├── api_requests_test.go
│   │   ├── auth_flows_test.go
│   │   ├── migration_test.go
│   │   └── system_integration_test.go
│   ├── manual/                 # Manual test tools
│   │   ├── manual_test.go
│   │   └── manual_test_suite.go
│   ├── performance/            # Performance tests
│   │   ├── basic_performance_test.go
│   │   ├── sigv4_performance_test.go
│   │   └── task43_2_comprehensive_test.go
│   └── security/               # Security tests
│       ├── auth_security_test.go
│       ├── credential_protection_test.go
│       ├── security_scanner.go
│       └── storage_security_test.go
│
├── scripts/                    # Build and test scripts
│   ├── README.md               # Scripts documentation
│   ├── build.ps1               # Full build (Windows)
│   ├── build.sh                # Full build (Linux/Mac)
│   ├── build_simple.ps1        # Quick build
│   ├── vendor.ps1              # Vendor dependencies
│   ├── verify_package.ps1      # Package verification
│   ├── test_*.ps1              # Test scripts
│   ├── get_profile_arn.ps1     # Utility scripts
│   └── set_aws_creds.ps1
│
├── docs/                       # Documentation
│   ├── README.md               # Documentation index
│   ├── QUICKSTART.md           # Quick start guide
│   ├── AUTHENTICATION.md       # Authentication guide
│   ├── API_KEY_MANAGEMENT.md   # API key guide
│   ├── SECURITY.md             # Security guide
│   ├── CONCURRENCY_ARCHITECTURE.md
│   ├── VALIDATION_SYSTEM.md
│   ├── PROJECT_STATUS.md       # Project status
│   ├── RELEASE_NOTES.md        # Release notes
│   └── archive/                # Historical docs
│       └── README.md
│
├── examples/                   # Code examples
│   ├── README.md               # Examples index
│   ├── bearer_token/           # Bearer token example
│   ├── credential_chain/       # Credential chain example
│   ├── oidc_device_code/       # OIDC example
│   └── sigv4/                  # SigV4 example
│
├── dist/                       # Build output (gitignored)
│   ├── kiro-gateway-linux-amd64
│   ├── kiro-gateway-linux-arm64
│   ├── kiro-YOUR_API_KEY_HERE
│   ├── kiro-YOUR_API_KEY_HERE
│   └── kiro-YOUR_API_KEY_HERE.exe
│
├── vendor/                     # Vendored dependencies (optional)
│
└── .kiro/                      # Runtime data (gitignored)
    ├── admin-key.txt           # Admin API key
    ├── api-keys.db             # API keys database
    └── credentials.db          # Encrypted credentials
```

## 📦 Package Organization

### cmd/
Application entry points. Each subdirectory is a separate executable.

**Main Application:**
- `kiro-gateway/` - The main gateway server

**Tools:**
- `manual-test-report/` - Manual testing and reporting
- `performance-analyzer/` - Performance analysis

### internal/
Private application code. Cannot be imported by external projects.

**Core Systems:**
- `auth/` - Authentication and authorization
- `handlers/` - HTTP request handlers
- `client/` - HTTP client for AWS APIs
- `streaming/` - Response streaming

**Supporting Systems:**
- `apikeys/` - API key management
- `concurrency/` - Concurrency primitives
- `storage/` - Data persistence
- `validation/` - Request validation

**Utilities:**
- `config/` - Configuration management
- `converters/` - Format conversion
- `errors/` - Error handling
- `models/` - Data models

### pkg/
Public packages that can be imported by external projects.

- `tokenizer/` - Token counting utilities

### tests/
Test suites organized by type.

- `integration/` - Integration tests
- `manual/` - Manual testing tools
- `performance/` - Performance benchmarks
- `security/` - Security tests

### scripts/
Build, test, and utility scripts.

- Build scripts (build.ps1, build.sh)
- Test scripts (test_*.ps1)
- Utility scripts (get_profile_arn.ps1, etc.)

### docs/
Project documentation.

- User guides
- Configuration guides
- Architecture documentation
- Reference documentation

### examples/
Code examples demonstrating usage.

- Authentication examples
- Integration examples
- Use case examples

## 🔧 Configuration Files

### Go Configuration
- `go.mod` - Go module definition
- `go.sum` - Dependency checksums

### Build Configuration
- `Makefile` - Build automation
- `Dockerfile` - Docker image build
- `docker-compose.yml` - Docker Compose setup
- `scripts/build.config` - Build configuration

### Environment Configuration
- `.env` - Active environment (gitignored)
- `.env.example` - Example configuration
- `.env.*` - Environment variants

### Git Configuration
- `.gitignore` - Git ignore rules
- `.github/` - GitHub workflows (if present)

## 🏗️ Build Artifacts

### Compiled Binaries
Located in `dist/` directory:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

### Runtime Data
Located in `.kiro/` directory:
- Admin API key
- API keys database
- Encrypted credentials
- Logs (if file logging enabled)

## 📊 Code Statistics

### Lines of Code (Approximate)
- Go source: ~15,000 lines
- Tests: ~5,000 lines
- Scripts: ~2,000 lines
- Documentation: ~10,000 lines

### Package Count
- Internal packages: 15
- Public packages: 1
- Test packages: 4

### File Count
- Go files: ~100
- Test files: ~40
- Scripts: ~25
- Documentation: ~30

## 🔍 Key Files

### Entry Points
- `cmd/kiro-gateway/main.go` - Application entry point

### Core Logic
- `internal/handlers/chat.go` - Main chat endpoint
- `internal/auth/auth.go` - Authentication manager
- `internal/streaming/eventstream.go` - Event stream parser
- `internal/client/client.go` - HTTP client

### Configuration
- `.env` - Environment configuration
- `internal/config/config.go` - Config loader

### Documentation
- `README.md` - Main documentation
- `docs/QUICKSTART.md` - Quick start guide
- `docs/AUTHENTICATION.md` - Auth guide

## 🚀 Build Process

### Development Build
```bash
go build -o kiro-gateway ./cmd/kiro-gateway
```

### Production Build
```bash
./scripts/build.ps1  # Windows
./scripts/build.sh   # Linux/Mac
```

### Docker Build
```bash
docker build -t kiro-gateway .
```

## 🧪 Testing

### Run All Tests
```bash
go test ./...
```

### Run Specific Tests
```bash
go test ./internal/auth/...
go test ./tests/integration/...
```

### Run with Coverage
```bash
go test -cover ./...
```

## 📝 Documentation

### Main Documentation
- `README.md` - Project overview
- `docs/README.md` - Documentation index

### User Guides
- `docs/QUICKSTART.md`
- `docs/AUTHENTICATION.md`
- `docs/API_KEY_MANAGEMENT.md`

### Developer Guides
- `docs/CONCURRENCY_ARCHITECTURE.md`
- `docs/PACKAGE_MANIFEST.md`
- `docs/DIRECTORY_INDEX.md`

## 🔐 Security

### Sensitive Files (Gitignored)
- `.env` - Environment variables
- `.kiro/admin-key.txt` - Admin API key
- `.kiro/*.db` - Databases
- `dist/` - Build artifacts

### Secure Storage
- Credentials encrypted at rest
- API keys hashed
- OS keychain integration

## 🤝 Contributing

### Adding New Features
1. Create package in `internal/`
2. Add tests in package
3. Update documentation
4. Add examples if applicable

### Adding Tests
1. Unit tests in package directory
2. Integration tests in `tests/integration/`
3. Performance tests in `tests/performance/`

### Adding Documentation
1. User docs in `docs/`
2. Code examples in `examples/`
3. Update `docs/README.md` index

## 📞 Support

- Issues: GitHub Issues
- Documentation: `docs/` directory
- Examples: `examples/` directory

---

**Last Updated:** January 24, 2026  
**Project Version:** 0.9.0 (Beta)
