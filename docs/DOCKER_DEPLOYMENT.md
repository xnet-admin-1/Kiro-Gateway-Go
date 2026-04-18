# Docker Deployment Guide

Complete guide for deploying Kiro Gateway in Docker containers with headless authentication.

## Quick Start

### 1. Build the Image

```bash
# Build with default settings
docker build -t kiro-gateway:latest .

# Build with version info
docker build \
  --build-arg VERSION=1.0.0 \
  --build-arg BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  --build-arg COMMIT_HASH=$(git rev-parse --short HEAD) \
  -t kiro-gateway:1.0.0 .
```

### 2. Run the Container

```bash
# Using environment file
docker run -d \
  --name kiro-gateway \
  -p 8080:8080 \
  --env-file .env \
  -v $(pwd)/.kiro/credentials:/app/.kiro/credentials:ro \
  kiro-gateway:latest

# Using docker-compose (recommended)
docker-compose up -d
```

### 3. Check Health

```bash
# Check container logs
docker logs -f kiro-gateway

# Check health endpoint
curl http://localhost:8080/health
```

## Docker Compose Deployment

### Configuration

1. **Copy environment template**:
   ```bash
   cp config/examples/config/examples/.env.docker .env
   ```

2. **Edit .env with your credentials**:
   ```bash
   # Required settings
   SSO_START_URL=https://your-org.awsapps.com/start
   SSO_ACCOUNT_ID=123456789012
   SSO_USERNAME=your-username
   SSO_PASSWORD=your-password
   MFA_TOTP_SECRET=YOUR-TOTP-SECRET
   PROFILE_ARN=arn:aws:codewhisperer:us-east-1:123456789012:profile/XXXXX
   ```

3. **Ensure credentials directory exists**:
   ```bash
   mkdir -p .kiro/credentials
   # Copy your encrypted credential files if you have them
   ```

### Start Services

```bash
# Start in foreground (see logs)
docker-compose up

# Start in background
docker-compose up -d

# View logs
docker-compose logs -f kiro-gateway

# Stop services
docker-compose down

# Stop and remove volumes
docker-compose down -v
```

## Alpine Container Details

### Base Image
- **Build**: `golang:1.23-alpine`
- **Runtime**: `alpine:latest`

### Installed Packages

**Build Stage**:
- `git` - Version control
- `gcc` - C compiler for CGO
- `musl-dev` - C standard library
- `sqlite-dev` - SQLite development files
- `ca-certificates` - SSL certificates

**Runtime Stage**:
- `ca-certificates` - SSL certificates
- `sqlite-libs` - SQLite runtime libraries
- `tzdata` - Timezone data
- `chromium` - Headless browser
- `chromium-chromedriver` - Browser automation
- `nss` - Network Security Services
- `freetype` - Font rendering
- `harfbuzz` - Text shaping
- `ttf-freefont` - Free fonts

### Container Size
- **Build image**: ~800MB (not shipped)
- **Runtime image**: ~450MB (includes Chromium)
- **Binary only**: ~25MB

### Security Features

1. **Non-root user**: Runs as `kiro` (UID 1001)
2. **Read-only credentials**: Mounted as read-only volume
3. **No sandbox**: Required for Chromium in containers
4. **Shared memory**: 2GB for browser stability

## Environment Variables

### Required

```bash
# SSO Configuration
SSO_START_URL=https://your-org.awsapps.com/start
SSO_REGION=us-east-1
SSO_ACCOUNT_ID=123456789012
SSO_ROLE_NAME=AdministratorAccess

# Authentication
SSO_USERNAME=your-username
SSO_PASSWORD=your-password
MFA_TOTP_SECRET=YOUR-TOTP-SECRET

# Profile
PROFILE_ARN=arn:aws:codewhisperer:us-east-1:123456789012:profile/XXXXX
```

### Optional

```bash
# Server
PORT=8080
PROXY_API_KEY=your-api-key

# AWS
AWS_REGION=us-east-1
AMAZON_Q_SIGV4=true
Q_USE_SENDMESSAGE=true

# Headless Mode
HEADLESS_MODE=true
AUTOMATE_AUTH=true

# Debug
DEBUG=false
LOG_LEVEL=info
```

## Volume Mounts

### Credentials (Read-Only)
```bash
-v $(pwd)/.kiro/credentials:/app/.kiro/credentials:ro
```
Mount encrypted credential files for secure access.

### Data (Read-Write)
```bash
-v kiro-data:/app/data
```
Persistent storage for database and application data.

### Logs (Read-Write)
```bash
-v kiro-logs:/app/logs
```
Persistent storage for logs and screenshots.

## Networking

### Ports
- **8080**: HTTP API endpoint (default)

### Health Check
- **Endpoint**: `http://localhost:8080/health`
- **Interval**: 30 seconds
- **Timeout**: 3 seconds
- **Retries**: 3
- **Start Period**: 40 seconds (allows time for initial auth)

## Browser Automation in Docker

### Chromium Configuration

The container includes Chromium for headless browser automation:

```dockerfile
ENV CHROME_BIN=/usr/bin/chromium-browser
ENV CHROME_PATH=/usr/lib/chromium/
ENV CHROME_NO_SANDBOX=true
```

### Shared Memory

Chromium requires shared memory for stability:

```yaml
shm_size: '2gb'
```

### Security Options

Disable seccomp for browser compatibility:

```yaml
security_opt:
  - seccomp:unconfined
```

## Troubleshooting

### Container Won't Start

1. **Check logs**:
   ```bash
   docker logs kiro-gateway
   ```

2. **Verify environment variables**:
   ```bash
   docker exec kiro-gateway env | grep SSO
   ```

3. **Check credentials mount**:
   ```bash
   docker exec kiro-gateway ls -la /app/.kiro/credentials
   ```

### Authentication Fails

1. **Verify credentials**:
   - Check SSO_USERNAME, SSO_PASSWORD, MFA_TOTP_SECRET
   - Ensure TOTP secret is current

2. **Check browser automation**:
   ```bash
   # View screenshots
   docker exec kiro-gateway ls -la /app/logs/screenshots
   
   # Copy screenshots out
   docker cp kiro-gateway:/app/logs/screenshots ./screenshots
   ```

3. **Enable debug logging**:
   ```bash
   docker-compose down
   # Edit .env: DEBUG=true, LOG_LEVEL=debug
   docker-compose up
   ```

### Browser Issues

1. **Chromium not found**:
   ```bash
   docker exec kiro-gateway which chromium-browser
   docker exec kiro-gateway chromium-browser --version
   ```

2. **Shared memory too small**:
   ```yaml
   # Increase in docker-compose.yml
   shm_size: '4gb'
   ```

3. **Sandbox errors**:
   ```bash
   # Verify no-sandbox is set
   docker exec kiro-gateway env | grep CHROME
   ```

### Performance Issues

1. **Increase resources**:
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

2. **Check container stats**:
   ```bash
   docker stats kiro-gateway
   ```

## Production Deployment

### Best Practices

1. **Use secrets management**:
   ```yaml
   secrets:
     sso_password:
       external: true
     mfa_secret:
       external: true
   ```

2. **Enable monitoring**:
   ```yaml
   logging:
     driver: "json-file"
     options:
       max-size: "10m"
       max-file: "3"
   ```

3. **Use health checks**:
   ```yaml
   healthcheck:
     test: ["CMD", "wget", "--spider", "http://localhost:8080/health"]
     interval: 30s
     timeout: 3s
     retries: 3
   ```

4. **Set resource limits**:
   ```yaml
   deploy:
     resources:
       limits:
         memory: 4G
   ```

### Kubernetes Deployment

Example Kubernetes manifests:

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kiro-gateway
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kiro-gateway
  template:
    metadata:
      labels:
        app: kiro-gateway
    spec:
      containers:
      - name: kiro-gateway
        image: kiro-gateway:latest
        ports:
        - containerPort: 8080
        env:
        - name: HEADLESS_MODE
          value: "true"
        - name: AUTOMATE_AUTH
          value: "true"
        envFrom:
        - secretRef:
            name: kiro-gateway-secrets
        volumeMounts:
        - name: credentials
          mountPath: /app/.kiro/credentials
          readOnly: true
        - name: data
          mountPath: /app/data
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
          initialDelaySeconds: 40
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 40
          periodSeconds: 10
      volumes:
      - name: credentials
        secret:
          secretName: kiro-credentials
      - name: data
        persistentVolumeClaim:
          claimName: kiro-data
      - name: dshm
        emptyDir:
          medium: Memory
          sizeLimit: 2Gi
      securityContext:
        runAsUser: 1001
        runAsGroup: 1001
        fsGroup: 1001
```

### Docker Swarm

```bash
# Initialize swarm
docker swarm init

# Create secrets
echo "your-password" | docker secret create sso_password -
echo "your-totp-secret" | docker secret create mfa_secret -

# Deploy stack
docker stack deploy -c docker-compose.yml kiro
```

## Monitoring

### Logs

```bash
# Follow logs
docker-compose logs -f kiro-gateway

# Last 100 lines
docker-compose logs --tail=100 kiro-gateway

# Since timestamp
docker-compose logs --since 2024-01-24T10:00:00 kiro-gateway
```

### Metrics

```bash
# Container stats
docker stats kiro-gateway

# Detailed inspect
docker inspect kiro-gateway
```

### Health Endpoint

```bash
# Check health
curl http://localhost:8080/health

# Expected response
{
  "status": "healthy",
  "timestamp": "2026-01-24T10:30:00Z",
  "version": "1.0.0"
}
```

## Backup and Recovery

### Backup Data

```bash
# Backup database
docker cp kiro-gateway:/app/kiro-gateway.db ./backup/

# Backup credentials
docker cp kiro-gateway:/app/.kiro/credentials ./backup/

# Backup using volume
docker run --rm \
  -v kiro-data:/data \
  -v $(pwd)/backup:/backup \
  alpine tar czf /backup/kiro-data.tar.gz /data
```

### Restore Data

```bash
# Restore database
docker cp ./backup/kiro-gateway.db kiro-gateway:/app/

# Restore using volume
docker run --rm \
  -v kiro-data:/data \
  -v $(pwd)/backup:/backup \
  alpine tar xzf /backup/kiro-data.tar.gz -C /
```

## Updates

### Rolling Update

```bash
# Pull new image
docker pull kiro-gateway:latest

# Recreate container
docker-compose up -d --force-recreate kiro-gateway

# Or with zero downtime
docker-compose up -d --no-deps --build kiro-gateway
```

### Version Pinning

```yaml
services:
  kiro-gateway:
    image: kiro-gateway:1.0.0  # Pin to specific version
```

## Security Considerations

1. **Secrets**: Never commit `.env` with real credentials
2. **Volumes**: Mount credentials as read-only
3. **Network**: Use internal networks for service communication
4. **User**: Container runs as non-root user (UID 1001)
5. **Updates**: Keep base images and dependencies updated

## Performance Tuning

### Resource Allocation

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

### Browser Optimization

```bash
# Increase shared memory for better browser performance
shm_size: '4gb'

# Adjust Chromium flags if needed
ENV CHROME_FLAGS="--disable-gpu --no-sandbox --disable-dev-shm-usage"
```

## Support

For issues or questions:
- Check logs: `docker-compose logs -f`
- Review screenshots: `docker cp kiro-gateway:/app/logs/screenshots ./`
- Enable debug mode: `DEBUG=true LOG_LEVEL=debug`

---

*Last Updated: January 24, 2026*
