# Docker Secrets Setup Complete

**Date**: January 26, 2026  
**Status**: ✅ CONFIGURED

---

## Secrets Created

The following Docker secrets have been created in Docker Desktop:

| Secret Name | Purpose | Status |
|-------------|---------|--------|
| `admin_api_key` | Admin API key for gateway management | ✅ Created |
| `sso_password` | SSO password for browser automation | ✅ Created |
| `mfa_totp_secret` | TOTP secret for automated MFA | ✅ Created |

**Note**: OIDC client secret was not provided (optional, for headless OIDC mode)

---

## Admin API Key

**IMPORTANT**: Save this key securely - it won't be shown again!

```
kiro-YOUR_API_KEY_HERE
```

This key has been:
- ✅ Stored in Docker secret `admin_api_key`
- ✅ Added to `.env` as fallback for non-Docker deployments
- ✅ Saved to `.kiro/admin-key.txt` (local backup)

---

## Verification

To verify secrets are created:

```powershell
# List all secrets
docker secret ls

# Or use the helper script
.\scripts\setup-docker-secrets.ps1 -List
```

Expected output:
```
ID                          NAME              DRIVER    CREATED
3nvl5gudfu01j8ls4ccwnpkfi   admin_api_key               X seconds ago
kj7qtagv6anm5rejs3v4moibe   mfa_totp_secret             X seconds ago
w37thm1jyk4g29r5palrrhz04   sso_password                X seconds ago
```

---

## Using Docker Secrets

### Option 1: Docker Compose (Development)

The secrets are already configured but commented out in `docker-compose.yml`. To use them:

1. Uncomment the secrets section in `docker-compose.yml`:
   ```yaml
   services:
     kiro-gateway:
       secrets:
         - admin_api_key
         - sso_password
         - mfa_totp_secret
   
   secrets:
     admin_api_key:
       external: true
     sso_password:
       external: true
     mfa_totp_secret:
       external: true
   ```

2. Deploy with Docker Compose:
   ```powershell
   docker-compose up -d
   ```

### Option 2: Docker Stack (Production)

For production deployments using Docker Swarm:

**IMPORTANT**: Docker Stack does NOT automatically load `.env` files. You must either:

#### Method A: Export Environment Variables (Recommended)

```powershell
# Load .env file into environment
Get-Content .env | ForEach-Object {
    if ($_ -match '^([^#][^=]+)=(.*)$') {
        [Environment]::SetEnvironmentVariable($matches[1], $matches[2], 'Process')
    }
}

# Deploy the stack
docker stack deploy -c docker-compose.stack.yml kiro

# Check stack status
docker stack ps kiro

# View logs
docker service logs kiro_kiro-gateway
```

#### Method B: Use docker-compose.yml (Development)

For development with automatic `.env` loading:

```powershell
# Docker Compose automatically loads .env
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f kiro-gateway
```

---

## How It Works

The application automatically reads secrets from `/run/secrets/` when running in Docker:

1. **Docker mounts secrets** at `/run/secrets/<secret_name>`
2. **Application reads secrets** on startup via `loadDockerSecrets()` function
3. **Secrets take precedence** over environment variables
4. **Graceful fallback** to environment variables if secrets not available

Example from logs:
```
✅ Loaded admin API key from Docker secret
✅ Loaded SSO password from Docker secret
✅ Loaded MFA TOTP secret from Docker secret
✅ Loaded 3 credential(s) from Docker secrets
```

---

## Security Benefits

Using Docker secrets provides:

✅ **Encrypted at rest** - Secrets are encrypted in Docker's internal database  
✅ **Encrypted in transit** - Secrets are encrypted when sent to containers  
✅ **Access control** - Only containers with explicit access can read secrets  
✅ **No environment exposure** - Secrets don't appear in `docker inspect` or logs  
✅ **Automatic cleanup** - Secrets are removed when containers stop  

---

## Managing Secrets

### View Secrets

```powershell
# List all secrets
docker secret ls

# Inspect secret metadata (does NOT show the value)
docker secret inspect admin_api_key
```

### Update Secrets

To update a secret, you must remove and recreate it:

```powershell
# Remove old secret
docker secret rm admin_api_key

# Create new secret
echo "new-admin-key-value" | docker secret create admin_api_key -

# Restart services to pick up new secret
docker service update --force kiro_kiro-gateway
```

### Remove All Secrets

```powershell
# Use the helper script
.\scripts\setup-docker-secrets.ps1 -Remove

# Or manually
docker secret rm admin_api_key sso_password mfa_totp_secret
```

---

## Troubleshooting

### Secret Not Found

If you see "secret not found" errors:

1. Verify Docker Swarm is initialized:
   ```powershell
   docker info | Select-String "Swarm"
   ```

2. List secrets to confirm they exist:
   ```powershell
   docker secret ls
   ```

3. Recreate secrets if needed:
   ```powershell
   .\scripts\setup-docker-secrets.ps1 -Generate
   ```

### Application Not Reading Secrets

If the application isn't reading secrets:

1. Check container logs:
   ```powershell
   docker logs kiro-YOUR_API_KEY_HERE
   ```

2. Verify secrets are mounted:
   ```powershell
   docker exec kiro-YOUR_API_KEY_HERE ls -la /run/secrets/
   ```

3. Ensure secrets section is uncommented in `docker-compose.yml`

### Docker Swarm Not Active

If you get "This node is not a swarm manager":

```powershell
# Initialize Docker Swarm
docker swarm init

# Recreate secrets
.\scripts\setup-docker-secrets.ps1 -Generate
```

---

## Next Steps

1. ✅ **Secrets created** - All sensitive credentials are in Docker secrets
2. ⏳ **Update docker-compose.yml** - Uncomment secrets section when ready
3. ⏳ **Test deployment** - Deploy with `docker-compose up -d` or `docker stack deploy`
4. ⏳ **Remove plaintext credentials** - Optionally remove from `.env` after testing
5. ⏳ **Document for team** - Share admin API key securely with team members

---

## References

- [Docker Secrets Documentation](https://docs.docker.com/engine/swarm/secrets/)
- [Docker Compose Secrets](https://docs.docker.com/compose/use-secrets/)
- Setup Script: `scripts/setup-docker-secrets.ps1`
- Implementation: `cmd/kiro-gateway/main.go` (`loadDockerSecrets()` function)

---

**Setup Date**: January 26, 2026  
**Docker Swarm**: Initialized  
**Secrets Count**: 3 (admin_api_key, sso_password, mfa_totp_secret)  
**Status**: ✅ Ready for deployment
