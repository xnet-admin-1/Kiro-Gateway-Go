# Agents and Automation Guide

This guide covers automated workflows, testing, and operational procedures for the Kiro Gateway.

## Table of Contents

1. [Test Automation](#test-automation)
2. [Build Automation](#build-automation)
3. [Deployment Automation](#deployment-automation)
4. [Monitoring and Alerts](#monitoring-and-alerts)
5. [Maintenance Tasks](#maintenance-tasks)

## Test Automation

### Automated Test Scripts

The gateway includes comprehensive test scripts for validation:

#### Health Check
```powershell
.\test_gateway_health.ps1
```

**What it tests:**
- Gateway is running and responding
- Authentication is working
- AWS profile is configured
- Correct API mode is active

**When to run:**
- After starting the gateway
- After configuration changes
- Before deploying to production

#### API Key Management
```powershell
.\test_api_keys.ps1
```

**What it tests:**
- Admin key exists and works
- Key creation and listing
- Key revocation
- Key deletion
- Key expiration

**When to run:**
- After initial setup
- When troubleshooting auth issues
- Before key rotation

#### Vision/Multimodal
```powershell
.\test_visual.ps1
```

**What it tests:**
- Single image processing
- Multiple image processing
- External image URLs
- Image format support
- Response parsing

**When to run:**
- After enabling Q Developer SigV4 mode
- When testing vision features
- After event stream parser changes

#### Streaming
```powershell
.\test_streaming.ps1
```

**What it tests:**
- SSE streaming responses
- Event parsing
- Token counting
- Stream interruption handling

**When to run:**
- After client changes
- When testing streaming features
- Performance validation

#### Both Modes
```powershell
.\test_both_modes.ps1
```

**What it tests:**
- CodeWhisperer mode (bearer token)
- Q Developer mode (SigV4)
- Mode switching
- Configuration validation

**When to run:**
- After authentication changes
- When validating both endpoints
- Integration testing

### Continuous Integration

#### GitHub Actions Example

```yaml
name: Test Gateway

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Run tests
      run: go test -v ./...
    
    - name: Run integration tests
      run: go test -v ./tests/integration/...
      env:
        AWS_REGION: us-east-1
    
    - name: Run security tests
      run: go test -v ./tests/security/...
    
    - name: Build
      run: go build -v ./cmd/kiro-gateway

  lint:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - uses: golangci/golangci-lint-action@v3
```

## Build Automation

### Build Scripts

#### Simple Build (Windows)
```powershell
.\build_simple.ps1
```

**What it does:**
- Builds for all platforms (Linux, macOS, Windows)
- Creates binaries in `dist/` directory
- Validates each binary
- Shows build summary

**When to use:**
- Quick builds during development
- Creating release binaries
- Cross-platform testing

#### Full Build (Windows)
```powershell
.\build.ps1
```

**What it does:**
- Runs all tests
- Builds for all platforms
- Creates vendored version
- Generates checksums
- Creates release package

**When to use:**
- Production releases
- Full validation
- Creating distribution packages

#### Build with Vendoring
```powershell
.\vendor.ps1
.\build.ps1
```

**What it does:**
- Downloads all dependencies
- Creates vendor directory
- Builds with vendored deps
- Ensures reproducible builds

**When to use:**
- Air-gapped environments
- Reproducible builds
- Dependency auditing

### Automated Release Process

```powershell
# 1. Update version
$version = "v1.0.0"

# 2. Run full test suite
go test ./...

# 3. Build all platforms
.\build_simple.ps1

# 4. Create release package
$releaseDir = "release-$version"
New-Item -ItemType Directory -Path $releaseDir
Copy-Item dist\* $releaseDir\
Copy-Item README.md $releaseDir\
Copy-Item QUICKSTART.md $releaseDir\
Copy-Item .env.example $releaseDir\

# 5. Create archive
Compress-Archive -Path $releaseDir -DestinationPath "kiro-gateway-$version.zip"

# 6. Generate checksums
Get-FileHash "kiro-gateway-$version.zip" -Algorithm SHA256 > checksums.txt
```

## Deployment Automation

### Docker Deployment

#### Build and Push
```bash
#!/bin/bash
VERSION="1.0.0"
REGISTRY="your-registry.com"
IMAGE="kiro-gateway"

# Build
docker build -t $IMAGE:$VERSION .
docker tag $IMAGE:$VERSION $IMAGE:latest

# Push
docker push $REGISTRY/$IMAGE:$VERSION
docker push $REGISTRY/$IMAGE:latest
```

#### Docker Compose
```yaml
version: '3.8'

services:
  kiro-gateway:
    image: kiro-gateway:latest
    ports:
      - "8080:8080"
    environment:
      - AWS_REGION=us-east-1
      - AMAZON_Q_SIGV4=true
      - Q_USE_SENDMESSAGE=true
    volumes:
      - ./config:/config
      - ~/.aws:/root/.aws:ro
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

### Kubernetes Deployment

#### Deployment Script
```bash
#!/bin/bash
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/secret.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/ingress.yaml

# Wait for rollout
kubectl rollout status deployment/kiro-gateway -n kiro

# Verify
kubectl get pods -n kiro
kubectl logs -f deployment/kiro-gateway -n kiro
```

#### Rolling Update
```bash
#!/bin/bash
VERSION="1.0.1"

# Update image
kubectl set image deployment/kiro-gateway \
  kiro-gateway=kiro-gateway:$VERSION \
  -n kiro

# Monitor rollout
kubectl rollout status deployment/kiro-gateway -n kiro

# Rollback if needed
# kubectl rollout undo deployment/kiro-gateway -n kiro
```

### AWS ECS Deployment

```bash
#!/bin/bash
CLUSTER="kiro-cluster"
SERVICE="kiro-gateway"
TASK_DEF="kiro-gateway-task"

# Update task definition
aws ecs register-task-definition \
  --cli-input-json file://task-definition.json

# Update service
aws ecs update-service \
  --cluster $CLUSTER \
  --service $SERVICE \
  --task-definition $TASK_DEF \
  --force-new-deployment

# Wait for deployment
aws ecs wait services-stable \
  --cluster $CLUSTER \
  --services $SERVICE
```

## Monitoring and Alerts

### Health Check Automation

```powershell
# health-check.ps1
$endpoint = "http://localhost:8080/health"
$maxRetries = 3
$retryDelay = 5

for ($i = 1; $i -le $maxRetries; $i++) {
    try {
        $response = Invoke-WebRequest -Uri $endpoint -UseBasicParsing
        if ($response.StatusCode -eq 200) {
            Write-Host "✅ Gateway is healthy"
            exit 0
        }
    } catch {
        Write-Host "⚠️ Attempt $i failed: $($_.Exception.Message)"
        if ($i -lt $maxRetries) {
            Start-Sleep -Seconds $retryDelay
        }
    }
}

Write-Host "❌ Gateway is unhealthy after $maxRetries attempts"
exit 1
```

### Prometheus Metrics (Future)

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'kiro-gateway'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

### CloudWatch Alarms

```bash
# Create CloudWatch alarm for ECS
aws cloudwatch put-metric-alarm \
  --alarm-name kiro-YOUR_API_KEY_HERE \
  --alarm-description "Alert when error rate exceeds 5%" \
  --metric-name ErrorRate \
  --namespace KiroGateway \
  --statistic Average \
  --period 300 \
  --threshold 5 \
  --comparison-operator GreaterThanThreshold \
  --evaluation-periods 2 \
  --alarm-actions arn:aws:sns:us-east-1:ACCOUNT:alerts
```

## Maintenance Tasks

### Daily Tasks

#### 1. Check Gateway Health
```powershell
.\test_gateway_health.ps1
```

#### 2. Review Logs
```powershell
# Check for errors
Get-Content gateway.log | Select-String "ERROR"

# Check for warnings
Get-Content gateway.log | Select-String "WARN"

# Monitor request rate
Get-Content gateway.log | Select-String "POST /v1/chat/completions" | Measure-Object
```

#### 3. Monitor API Key Usage
```powershell
# List all keys
curl http://localhost:8080/v1/api-keys \
  -H "Authorization: Bearer ADMIN_KEY"

# Check for expiring keys (within 7 days)
# Review output and rotate if needed
```

### Weekly Tasks

#### 1. Rotate API Keys
```powershell
# Create new key
$newKey = curl -X POST http://localhost:8080/v1/api-keys \
  -H "Authorization: Bearer ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{"name":"Weekly Key","expires_in":"7d"}'

# Update applications with new key
# Revoke old key after migration
curl -X POST http://localhost:8080/v1/api-keys/OLD_KEY/revoke \
  -H "Authorization: Bearer ADMIN_KEY"
```

#### 2. Review Performance
```powershell
# Run performance tests
go test -bench=. ./tests/performance/...

# Check connection pool metrics
# Review gateway logs for pool statistics
```

#### 3. Update Dependencies
```bash
# Check for updates
go list -u -m all

# Update dependencies
go get -u ./...
go mod tidy

# Run tests
go test ./...
```

### Monthly Tasks

#### 1. Security Audit
```bash
# Run security tests
go test ./tests/security/...

# Check for vulnerabilities
go list -json -m all | nancy sleuth

# Review API key permissions
# Audit access logs
```

#### 2. Backup Configuration
```powershell
# Backup .env files
$backupDir = "backups\$(Get-Date -Format 'yyyy-MM-dd')"
New-Item -ItemType Directory -Path $backupDir -Force
Copy-Item .env* $backupDir\

# Backup API keys database
Copy-Item .kiro\* $backupDir\.kiro\ -Recurse
```

#### 3. Performance Review
```bash
# Run comprehensive performance tests
go test -bench=. -benchmem ./tests/performance/...

# Analyze results
# Identify bottlenecks
# Plan optimizations
```

### Quarterly Tasks

#### 1. Major Version Updates
```bash
# Update Go version
# Update AWS SDK
# Update dependencies
# Run full test suite
# Update documentation
```

#### 2. Disaster Recovery Test
```bash
# Test backup restoration
# Test failover procedures
# Verify monitoring alerts
# Update runbooks
```

#### 3. Capacity Planning
```bash
# Review usage metrics
# Analyze growth trends
# Plan infrastructure scaling
# Update resource limits
```

## Automation Best Practices

### 1. Idempotency
- All automation scripts should be idempotent
- Running multiple times should produce same result
- Check state before making changes

### 2. Error Handling
- Always check exit codes
- Log all operations
- Implement retry logic
- Provide clear error messages

### 3. Validation
- Validate inputs before processing
- Test in non-production first
- Verify results after operations
- Maintain rollback procedures

### 4. Documentation
- Document all automation scripts
- Include usage examples
- Explain prerequisites
- Note any side effects

### 5. Security
- Never commit credentials
- Use environment variables
- Rotate secrets regularly
- Audit access logs

## Troubleshooting Automation

### Script Fails to Run

**Check:**
- Execution policy (Windows): `Set-ExecutionPolicy RemoteSigned`
- File permissions (Linux/Mac): `chmod +x script.sh`
- Dependencies installed
- Environment variables set

### Tests Fail

**Check:**
- Gateway is running
- Correct port configured
- API keys are valid
- AWS credentials configured
- Network connectivity

### Build Fails

**Check:**
- Go version (1.21+)
- Dependencies downloaded: `go mod download`
- Disk space available
- Build tools installed

### Deployment Fails

**Check:**
- Container registry accessible
- Kubernetes cluster reachable
- AWS credentials valid
- Resource quotas not exceeded
- Configuration files present

## Additional Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Docker Documentation](https://docs.docker.com/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [AWS ECS Documentation](https://docs.aws.amazon.com/ecs/)
- [Go Testing Documentation](https://golang.org/pkg/testing/)

---

**Last Updated:** January 24, 2026  
**Maintained By:** Kiro Gateway Team
