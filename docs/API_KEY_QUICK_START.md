# API Key Management - Quick Start

## 1. Start Gateway

```bash
./kiro-gateway.exe
```

On first run, an admin key is automatically created and saved to `.kiro/admin-key.txt`.

## 2. Get Admin Key

```bash
# Windows
type .kiro\admin-key.txt

# Linux/Mac
cat .kiro/admin-key.txt
```

Set it in your environment:
```bash
# Windows
set ADMIN_API_KEY=kiro-your-admin-key-here

# Linux/Mac
export ADMIN_API_KEY=kiro-your-admin-key-here
```

## 3. Create API Key

```bash
curl -X POST http://localhost:8080/v1/api-keys \
  -H "Authorization: Bearer $ADMIN_API_KEY" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"My App\",\"expires_in\":\"30d\"}"
```

Save the returned key!

## 4. Use API Key

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer kiro-your-api-key" \
  -H "Content-Type: application/json" \
  -d "{\"model\":\"anthropic.claude-sonnet-4-5\",\"messages\":[{\"role\":\"user\",\"content\":\"Hello!\"}]}"
```

## Common Commands

### List All Keys
```bash
curl http://localhost:8080/v1/api-keys \
  -H "Authorization: Bearer $ADMIN_API_KEY"
```

### Get Key Details
```bash
curl http://localhost:8080/v1/api-keys/kiro-your-key \
  -H "Authorization: Bearer $ADMIN_API_KEY"
```

### Revoke Key
```bash
curl -X POST http://localhost:8080/v1/api-keys/kiro-your-key/revoke \
  -H "Authorization: Bearer $ADMIN_API_KEY"
```

### Delete Key
```bash
curl -X DELETE http://localhost:8080/v1/api-keys/kiro-your-key \
  -H "Authorization: Bearer $ADMIN_API_KEY"
```

## Expiration Formats

- `24h` - 24 hours
- `7d` - 7 days
- `30d` - 30 days
- `1y` - 1 year
- `null` - Never expires

## Test Script

```bash
./test_api_keys.ps1
```

## Full Documentation

See [API_KEY_MANAGEMENT.md](API_KEY_MANAGEMENT.md) for complete documentation.
