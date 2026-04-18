# Alpine Container Deployment Guide

This guide covers deploying Kiro Gateway in an Alpine Linux container with full browser automation support for headless authentication.

## Overview

The Alpine container includes:
- **Kiro Gateway** - Statically compiled Go binary
- **Chromium Browser** - For automated OIDC authentication
- **SQLite** - Encrypted credential storage
- **Security hardening** - Non-root user, minimal attack surface

## Container Specifications

- **Base Image**: Alpine Linux (latest)
- **Size**: ~1.2GB (includes Chromium and dependencies)
- **User**: Non-root (`kiro:kiro`, UID/GID 1001)
- **Architecture**: linux/amd64 (statically linked)
- **Browser**: Chromium 144+ with ChromeDriver

## Building the Container

### Quick Build

```powershell
# Build with default settings
docker build -t kiro-gateway:alpine-latest .

# Build with version tags
docker build -t kiro-gateway:alpine-latest \
  --build-arg VERSION=1.0.0 \
  --build-arg BUILD_TIME=$(Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ") \
  --build-arg COMMIT_HASH=$(git rev-parse --short HEAD) \
  .
```

### Using Build Script

```powershell
# PowerShell
.\scripts\build-docker.ps1 -Version "1.0.0" -Tag "alpine-latest"

# Bash
./scripts/build-docker.sh
```

## Running the Container

### Using Docker Compose (Recommended)

```yaml
# docker-compose.yml
version: '3.8'

services:
  kiro-gateway:
    image: kiro-gateway:alpine-latest
    ports:
      - "8080:8080"
    environment:
      # AWS Configuration
      - AWS_REGION=us-east-1
      - AMAZON_Q_SIGV4=true
      - Q_USE_SENDMESSAGE=true
      
      # Headless Authentication
      - HEADLESS_MODE=true
      - AUTOMATE_AUTH=true
      - SSO_START_URL=https://your-org.awsapps.com/start
      - SSO_REGION=us-east-1
      - SSO_USERNAME=your-username
      - SSO_PASSWORD=your-password
      - MFA_TOTP_SECRET=your-totp-secret
      
      # Browser Settings (pre-configured)
      - CHROME_BIN=/usr/bin/chromium-browser
      - CHROME_NO_SANDBOX=true
    volumes:
      - kiro-data:/app/data
      - kiro-logs:/app/logs
    security_opt:
      - seccomp:unconfined
    shm_size: '2gb'
    restart: unless-stopped

volumes:
  kiro-data:
  kiro-logs:
```

Start the service:

```powershell
docker-compose up -d
```

### Using Docker Run

```powershell
docker run -d \
  --name kiro-gateway \
  -p 8080:8080 \
  --env-file .env \
  --security-opt seccomp:unconfined \
  --shm-size 2g \
  -v kiro-data:/app/data \
  -v kiro-logs:/app/logs \
  kiro-gateway:alpine-latest
```

## Environment Variables

### Required Variables

```bash
# AWS Configuration
AWS_REGION=us-east-1
AMAZON_Q_SIGV4=true
Q_USE_SENDMESSAGE=true

# SSO Configuration (for headless mode)
SSO_START_URL=https://your-org.awsapps.com/start
SSO_REGION=us-east-1
SSO_USERNAME=your-username
SSO_PASSWORD=your-password
```

### Optional Variables

```bash
# MFA (if enabled)
MFA_TOTP_SECRET=your-base32-secret

# Performance
MAX_RETRIES=3
CONNECT_TIMEOUT=10s
READ_TIMEOUT=30s

# Debug
DEBUG=false
LOG_LEVEL=info

# Browser (pre-configured in container)
CHROME_BIN=/usr/bin/chromium-browser
CHROME_NO_SANDBOX=true
HEADLESS_MODE=true
```

## Browser Automation

The container includes Chromium for automated authentication:

### Features
- **Headless mode** - No GUI required
- **Auto-detection** - Finds Chrome at `/usr/bin/chromium-browser`
- **TOTP support** - Automated MFA code generation
- **Screenshot capture** - Debugging screenshots in `/app/logs/screenshots`

### Chrome Configuration

The container sets these environment variables:

```bash
CHROME_BIN=/usr/bin/chromium-browser
CHROME_PATH=/usr/lib/chromium/
CHROME_NO_SANDBOX=true  # Required for non-root user
```

### Security Options

Required Docker security options for browser automation:

```yaml
security_opt:
  - seccomp:unconfined  # Allows Chrome to run
shm_size: '2gb'         # Shared memory for Chrome
```

## Volumes and Persistence

### Data Volumes

```yaml
volumes:
  # Application data (database, credentials)
  - kiro-data:/app/data
  
  # Logs and screenshots
  - kiro-logs:/app/logs
  
  # Optional: Mount credentials directory
  - ./.kiro/credentials:/app/.kiro/credentials:ro
```

### Directory Structure

```
/app/
├── kiro-gateway           # Binary
├── .kiro/
│   ├── credentials/       # Encrypted credentials
│   └── api-keys/          # API key storage
├── logs/
│   ├── gateway.log        # Application logs
│   ├── gateway-error.log  # Error logs
│   └── screenshots/       # Browser automation screenshots
└── data/
    └── kiro-gateway.db    # SQLite database
```

## Health Checks

The container includes a health check:

```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1
```

Check health status:

```powershell
# Docker
docker inspect --format='{{.State.Health.Status}}' kiro-gateway

# Docker Compose
docker-compose ps
```

## Troubleshooting

### Container Won't Start

```powershell
# Check logs
docker logs kiro-gateway

# Check environment
docker exec kiro-gateway env | grep -E "SSO|AWS|CHROME"
```

### Browser Automation Issues

```powershell
# Verify Chrome installation
docker exec kiro-gateway /usr/bin/chromium-browser --version

# Check Chrome path detection
docker exec kiro-gateway ls -la /usr/bin/chromium-browser

# View browser logs
docker exec kiro-gateway cat /app/logs/gateway.log | grep BROWSER
```

### Permission Issues

```powershell
# Verify running as non-root
docker exec kiro-gateway whoami  # Should output: kiro

# Check directory permissions
docker exec kiro-gateway ls -la /app
```

### Memory Issues

If Chrome crashes with "out of memory":

```yaml
# Increase shared memory
shm_size: '4gb'

# Or use tmpfs
tmpfs:
  - /dev/shm:size=4g
```

## Testing the Container

Run the test suite:

```powershell
.\scripts\test-alpine-container.ps1
```

This verifies:
- ✅ Chrome installation
- ✅ Binary functionality
- ✅ Image size
- ✅ Non-root user
- ✅ Directory structure

## Production Deployment

### Best Practices

1. **Use secrets management**
   ```yaml
   secrets:
     sso_password:
       external: true
   environment:
     SSO_PASSWORD_FILE: /run/secrets/sso_password
   ```

2. **Enable resource limits**
   ```yaml
   deploy:
     resources:
       limits:
         cpus: '2'
         memory: 4G
       reservations:
         cpus: '1'
         memory: 2G
   ```

3. **Configure logging**
   ```yaml
   logging:
     driver: "json-file"
     options:
       max-size: "10m"
       max-file: "3"
   ```

4. **Use health checks**
   ```yaml
   healthcheck:
     test: ["CMD", "wget", "--spider", "http://localhost:8080/health"]
     interval: 30s
     timeout: 3s
     retries: 3
     start_period: 40s
   ```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kiro-gateway
spec:
  replicas: 2
  selector:
    matchLabels:
      app: kiro-gateway
  template:
    metadata:
      labels:
        app: kiro-gateway
    spec:
      securityContext:
        runAsUser: 1001
        runAsGroup: 1001
        fsGroup: 1001
      containers:
      - name: kiro-gateway
        image: kiro-gateway:alpine-latest
        ports:
        - containerPort: 8080
        env:
        - name: AWS_REGION
          value: "us-east-1"
        - name: HEADLESS_MODE
          value: "true"
        envFrom:
        - secretRef:
            name: kiro-gateway-secrets
        volumeMounts:
        - name: data
          mountPath: /app/data
        - name: logs
          mountPath: /app/logs
        - name: dshm
          mountPath: /dev/shm
        resources:
          limits:
            memory: "4Gi"
            cpu: "2"
          requests:
            memory: "2Gi"
            cpu: "1"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: kiro-data
      - name: logs
        emptyDir: {}
      - name: dshm
        emptyDir:
          medium: Memory
          sizeLimit: 2Gi
```

## Security Considerations

### Container Security

- ✅ **Non-root user** - Runs as `kiro:kiro` (UID 1001)
- ✅ **Minimal base** - Alpine Linux reduces attack surface
- ✅ **Static binary** - No dynamic dependencies
- ✅ **Read-only filesystem** - Mount root as read-only where possible
- ✅ **No privileged mode** - Uses `seccomp:unconfined` only

### Credential Security

- ✅ **Encrypted storage** - SQLite database with AES-256-GCM
- ✅ **Environment variables** - Use secrets management
- ✅ **No credential logging** - Sensitive data redacted
- ✅ **Secure deletion** - Credentials wiped on cleanup

### Network Security

```yaml
# Restrict network access
networks:
  kiro-network:
    driver: bridge
    ipam:
      config:
        - subnet: 172.28.0.0/16

# Use internal network for backend services
services:
  kiro-gateway:
    networks:
      - kiro-network
      - external
```

## Monitoring

### Prometheus Metrics

```yaml
# Expose metrics endpoint
environment:
  - ENABLE_METRICS=true
  - METRICS_PORT=9090

ports:
  - "9090:9090"  # Metrics
```

### Log Aggregation

```yaml
# Forward logs to external system
logging:
  driver: "fluentd"
  options:
    fluentd-address: "localhost:24224"
    tag: "kiro-gateway"
```

## Performance Tuning

### Browser Performance

```yaml
environment:
  # Reduce browser overhead
  - CHROME_NO_SANDBOX=true
  - CHROME_DISABLE_GPU=true
  - CHROME_DISABLE_DEV_SHM=false
```

### Connection Pooling

```yaml
environment:
  - MAX_CONNECTIONS=100
  - IDLE_TIMEOUT=5m
  - CONNECTION_TIMEOUT=30s
```

## Backup and Recovery

### Backup Credentials

```powershell
# Backup encrypted credentials
docker cp kiro-gateway:/app/.kiro/credentials ./backup/

# Backup database
docker cp kiro-gateway:/app/data/kiro-gateway.db ./backup/
```

### Restore

```powershell
# Restore credentials
docker cp ./backup/credentials kiro-gateway:/app/.kiro/

# Restore database
docker cp ./backup/kiro-gateway.db kiro-gateway:/app/data/
```

## Updates and Maintenance

### Update Container

```powershell
# Pull new image
docker pull kiro-gateway:alpine-latest

# Restart with new image
docker-compose up -d --force-recreate
```

### Rolling Updates

```yaml
# docker-compose.yml
deploy:
  update_config:
    parallelism: 1
    delay: 10s
    order: start-first
```

## Support

For issues or questions:
- Check logs: `docker logs kiro-gateway`
- Review documentation: `/docs`
- Run tests: `.\scripts\test-alpine-container.ps1`

## License

See main project LICENSE file.
