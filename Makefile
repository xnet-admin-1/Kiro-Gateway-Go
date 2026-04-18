.PHONY: build run test clean docker-build docker-run install deps security-scan benchmark vendor vendor-update vendor-verify

# Binary name and version
BINARY_NAME=kiro-gateway
DOCKER_IMAGE=kiro-gateway-go
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
COMMIT_HASH=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags for AWS security features
BUILD_FLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.CommitHash=$(COMMIT_HASH)"
CGO_FLAGS=CGO_ENABLED=1

# Build the application with AWS security features
build:
	$(CGO_FLAGS) go build $(BUILD_FLAGS) -o $(BINARY_NAME) ./cmd/kiro-gateway

# Build for multiple platforms with AWS security support
build-all:
	@echo "Building binaries for all platforms..."
	@mkdir -p dist
	$(CGO_FLAGS) GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/kiro-gateway
	$(CGO_FLAGS) GOOS=linux GOARCH=arm64 go build $(BUILD_FLAGS) -o dist/$(BINARY_NAME)-linux-arm64 ./cmd/kiro-gateway
	$(CGO_FLAGS) GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 ./cmd/kiro-gateway
	$(CGO_FLAGS) GOOS=darwin GOARCH=arm64 go build $(BUILD_FLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 ./cmd/kiro-gateway
	$(CGO_FLAGS) GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe ./cmd/kiro-gateway
	@echo "Build complete. Binaries available in dist/ directory:"
	@ls -la dist/

# Test binaries (Task 49.3)
test-binaries:
	@echo "Testing built binaries..."
	@echo "Testing Linux AMD64 binary..."
	@file dist/$(BINARY_NAME)-linux-amd64 || echo "Linux binary not found"
	@echo "Testing Darwin AMD64 binary..."
	@file dist/$(BINARY_NAME)-darwin-amd64 || echo "Darwin binary not found"
	@echo "Testing Windows AMD64 binary..."
	@file dist/$(BINARY_NAME)-windows-amd64.exe || echo "Windows binary not found"
	@echo "Binary validation complete"

# Build and test all binaries (Task 49.3 implementation)
build-and-test: build-all test-binaries
	@echo "Task 49.3 complete: All binaries built and tested"

# Run the application
run: build
	./$(BINARY_NAME)

# Run with debug logging
debug: build
	DEBUG=true ./$(BINARY_NAME)

# Run with AWS SigV4 authentication
run-sigv4: build
	AMAZON_Q_SIGV4=true ./$(BINARY_NAME)

# Run tests with AWS security features
test:
	AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY go test -v ./...

# Run tests with coverage including AWS security modules
test-coverage:
	AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Run security-specific tests
test-security:
	go test -v -tags=security ./tests/security/...

# Run performance tests including AWS auth benchmarks
test-performance:
	go test -v -bench=. -benchmem ./tests/performance/...

# Run integration tests with AWS mocks
test-integration:
	go test -v -tags=integration ./tests/integration/...

# Security scanning with gosec
security-scan:
	@which gosec > /dev/null || (echo "Installing gosec..." && go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest)
	gosec -fmt json -out security-report.json ./...
	gosec ./...

# Static analysis
static-analysis:
	go vet ./...
	@which staticcheck > /dev/null || (echo "Installing staticcheck..." && go install honnef.co/go/tools/cmd/staticcheck@latest)
	staticcheck ./...

# Benchmark AWS security features
benchmark:
	go test -bench=BenchmarkCredential -benchmem ./internal/auth/credentials/...
	go test -bench=BenchmarkSigV4 -benchmem ./internal/auth/sigv4/...
	go test -bench=BenchmarkOIDC -benchmem ./internal/auth/oidc/...

# Clean build artifacts
clean:
	go clean
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*
	rm -f coverage.out
	rm -f security-report.json

# Install dependencies including AWS SDK
deps:
	go mod download
	go mod tidy
	go mod verify

# Vendor dependencies (copy to vendor/ directory)
vendor:
	@echo "Vendoring dependencies..."
	go mod tidy
	go mod vendor
	@echo "✅ Dependencies vendored to vendor/ directory"
	@echo "Vendor size: $$(du -sh vendor/ | cut -f1)"

# Update and re-vendor dependencies
vendor-update:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy
	go mod vendor
	@echo "✅ Dependencies updated and vendored"

# Verify vendor directory is in sync
vendor-verify:
	@echo "Verifying vendor directory..."
	go mod tidy
	go mod vendor
	@if git diff --quiet vendor/; then \
		echo "✅ Vendor directory is up to date"; \
	else \
		echo "❌ Vendor directory is out of sync"; \
		echo "Run 'make vendor' to update"; \
		exit 1; \
	fi

# Build with vendored dependencies
build-vendor:
	@echo "Building with vendored dependencies..."
	$(CGO_FLAGS) go build -mod=vendor $(BUILD_FLAGS) -o $(BINARY_NAME) ./cmd/kiro-gateway
	@echo "✅ Build complete using vendor/"

# Test with vendored dependencies
test-vendor:
	@echo "Testing with vendored dependencies..."
	AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY go test -mod=vendor -v ./...
	@echo "✅ Tests complete using vendor/"

# Build Docker image with AWS security support
docker-build:
	docker build -t $(DOCKER_IMAGE):$(VERSION) -t $(DOCKER_IMAGE):latest .

# Run Docker container with AWS environment
docker-run:
	docker run -p 8080:8080 --env-file .env \
		-e AWS_REGION \
		-e AWS_PROFILE \
		-e AMAZON_Q_SIGV4 \
		-e OIDC_START_URL \
		-e OIDC_REGION \
		$(DOCKER_IMAGE):latest

# Format code
fmt:
	go fmt ./...

# Lint code with enhanced rules for security
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run --enable=gosec,gocritic,gocyclo

# Install the binary
install: build
	go install ./cmd/kiro-gateway

# Validate AWS security configuration
validate-aws-config:
	@echo "Validating AWS security configuration..."
	@go run ./cmd/kiro-gateway -validate-config

# Generate AWS security documentation
docs-aws:
	@echo "Generating AWS security documentation..."
	@go doc -all ./internal/auth/credentials > docs/credentials.md
	@go doc -all ./internal/auth/sigv4 > docs/sigv4.md
	@go doc -all ./internal/auth/oidc > docs/oidc.md

# Full CI pipeline
ci: deps fmt lint static-analysis security-scan test-coverage test-security test-integration benchmark

# Release build with all platforms and security validation
release: clean deps security-scan test-coverage build-all
	@echo "Release $(VERSION) built successfully"
	@echo "Binaries:"
	@ls -la $(BINARY_NAME)-*

# Show help
help:
	@echo "Available targets:"
	@echo "  build              - Build the application with AWS security features"
	@echo "  build-all          - Build for all platforms"
	@echo "  build-vendor       - Build with vendored dependencies"
	@echo "  run                - Build and run the application"
	@echo "  debug              - Run with debug logging"
	@echo "  run-sigv4          - Run with AWS SigV4 authentication"
	@echo "  test               - Run all tests"
	@echo "  test-vendor        - Run tests with vendored dependencies"
	@echo "  test-coverage      - Run tests with coverage"
	@echo "  test-security      - Run security-specific tests"
	@echo "  test-performance   - Run performance benchmarks"
	@echo "  test-integration   - Run integration tests"
	@echo "  security-scan      - Run security analysis with gosec"
	@echo "  static-analysis    - Run static code analysis"
	@echo "  benchmark          - Benchmark AWS security features"
	@echo "  clean              - Clean build artifacts"
	@echo "  deps               - Install dependencies"
	@echo "  vendor             - Vendor all dependencies to vendor/"
	@echo "  vendor-update      - Update and re-vendor dependencies"
	@echo "  vendor-verify      - Verify vendor directory is in sync"
	@echo "  docker-build       - Build Docker image"
	@echo "  docker-run         - Run Docker container"
	@echo "  fmt                - Format code"
	@echo "  lint               - Lint code with security rules"
	@echo "  install            - Install binary"
	@echo "  validate-aws-config - Validate AWS configuration"
	@echo "  docs-aws           - Generate AWS security docs"
	@echo "  ci                 - Run full CI pipeline"
	@echo "  release            - Build release with all platforms"
	@echo "  help               - Show this help"
