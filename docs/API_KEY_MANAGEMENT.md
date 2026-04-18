# API Key Management System

Complete guide to managing API keys in Kiro Gateway.

## Overview

Kiro Gateway includes a comprehensive API key management system that allows you to:
- Generate multiple API keys for different users/applications
- Set expiration dates and permissions
- Track usage statistics
- Revoke or delete keys
- Manage keys through REST API endpoints

## Quick Start

### 1. Initial Setup

On first startup, the gateway automatically creates an initial admin API key:

```bash
./kiro-gateway.exe
```

Output:
```
Initializing API key manager...
No API keys found - creating initial admin API key...
✅ Admin API key saved to: .kiro/admin-key.txt
✅ Created initial admin API key: kiro-abc...xyz
   Full key: kiro-YOUR_API_KEY_HERE
   IMPORTANT: Save this key securely - it won't be shown again!
```

The admin key is saved to `.kiro/admin-key.txt` for easy access.

### 2. Using the Admin Key

Set the admin key in your environment:

```bash
# Windows
set ADMIN_API_KEY=kiro-YOUR_API_KEY_HERE

# Linux/Mac
export ADMIN_API_KEY=kiro-YOUR_API_KEY_HERE
```

Or add it to your `.env` file:

```env
ADMIN_API_KEY=kiro-YOUR_API_KEY_HERE
```

## API Endpoints

All API key management endpoints require admin authentication.

### Create API Key

**POST** `/v1/api-keys`

Create a new API key.

**Headers:**
```
Authorization: Bearer <admin-api-key>
Content-Type: application/json
```

**Request Body:**
```json
{
  "name": "Production App",
  "user_id": "app-prod-001",
  "expires_in": "30d",
  "permissions": ["chat.completions"],
  "metadata": {
    "environment": "production",
    "team": "engineering"
  }
}
```

**Fields:**
- `name` (required): Human-readable name for the key
- `user_id` (optional): User/application identifier (default: "default")
- `expires_in` (optional): Expiration duration (e.g., "30d", "1y", "24h")
- `permissions` (optional): Array of permissions (default: ["chat.completions"])
- `metadata` (optional): Custom key-value metadata

**Response:**
```json
{
  "key": "kiro-xyz123...",
  "name": "Production App",
  "user_id": "app-prod-001",
  "created_at": "2026-01-22T10:00:00Z",
  "expires_at": "2026-02-21T10:00:00Z",
  "permissions": ["chat.completions"],
  "metadata": {
    "environment": "production",
    "team": "engineering"
  }
}
```

**⚠️ IMPORTANT:** The full API key is only returned once during creation. Save it securely!

**Example:**
```bash
curl -X POST http://localhost:8080/v1/api-keys \
  -H "Authorization: Bearer $ADMIN_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Production App",
    "user_id": "app-prod-001",
    "expires_in": "30d",
    "permissions": ["chat.completions"]
  }'
```

### List API Keys

**GET** `/v1/api-keys`

List all API keys (keys are masked for security).

**Headers:**
```
Authorization: Bearer <admin-api-key>
```

**Query Parameters:**
- `user_id` (optional): Filter by user ID

**Response:**
```json
{
  "keys": [
    {
      "key_preview": "kiro-abc...xyz",
      "name": "Production App",
      "user_id": "app-prod-001",
      "created_at": "2026-01-22T10:00:00Z",
      "expires_at": "2026-02-21T10:00:00Z",
      "last_used_at": "2026-01-22T15:30:00Z",
      "is_active": true,
      "usage_count": 1523,
      "permissions": ["chat.completions"],
      "metadata": {
        "environment": "production"
      }
    }
  ],
  "count": 1
}
```

**Example:**
```bash
curl http://localhost:8080/v1/api-keys \
  -H "Authorization: Bearer $ADMIN_API_KEY"
```

### Get API Key Details

**GET** `/v1/api-keys/{key}`

Get details for a specific API key.

**Headers:**
```
Authorization: Bearer <admin-api-key>
```

**Response:**
```json
{
  "key_preview": "kiro-abc...xyz",
  "name": "Production App",
  "user_id": "app-prod-001",
  "created_at": "2026-01-22T10:00:00Z",
  "expires_at": "2026-02-21T10:00:00Z",
  "last_used_at": "2026-01-22T15:30:00Z",
  "is_active": true,
  "usage_count": 1523,
  "permissions": ["chat.completions"],
  "metadata": {
    "environment": "production"
  }
}
```

**Example:**
```bash
curl http://localhost:8080/v1/api-keys/kiro-abc123... \
  -H "Authorization: Bearer $ADMIN_API_KEY"
```

### Update API Key

**PATCH** `/v1/api-keys/{key}`

Update API key metadata.

**Headers:**
```
Authorization: Bearer <admin-api-key>
Content-Type: application/json
```

**Request Body:**
```json
{
  "name": "Production App (Updated)",
  "metadata": {
    "environment": "production",
    "version": "2.0"
  }
}
```

**Example:**
```bash
curl -X PATCH http://localhost:8080/v1/api-keys/kiro-abc123... \
  -H "Authorization: Bearer $ADMIN_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Production App (Updated)"
  }'
```

### Revoke API Key

**POST** `/v1/api-keys/{key}/revoke`

Revoke an API key (makes it inactive but keeps the record).

**Headers:**
```
Authorization: Bearer <admin-api-key>
```

**Response:** 204 No Content

**Example:**
```bash
curl -X POST http://localhost:8080/v1/api-keys/kiro-abc123.../revoke \
  -H "Authorization: Bearer $ADMIN_API_KEY"
```

### Delete API Key

**DELETE** `/v1/api-keys/{key}`

Permanently delete an API key.

**Headers:**
```
Authorization: Bearer <admin-api-key>
```

**Response:** 204 No Content

**Example:**
```bash
curl -X DELETE http://localhost:8080/v1/api-keys/kiro-abc123... \
  -H "Authorization: Bearer $ADMIN_API_KEY"
```

## Expiration Formats

The `expires_in` field supports the following formats:

- **Hours:** `24h`, `48h`
- **Days:** `7d`, `30d`, `90d`
- **Weeks:** `1w`, `4w`
- **Months:** `1m`, `6m`, `12m`
- **Years:** `1y`, `2y`
- **Standard Go Duration:** `72h30m`

Examples:
```json
{"expires_in": "30d"}   // 30 days
{"expires_in": "1y"}    // 1 year
{"expires_in": "24h"}   // 24 hours
{"expires_in": null}    // Never expires
```

## Permissions

API keys can have the following permissions:

- `chat.completions` - Access to chat completions endpoint
- `models` - Access to models endpoint
- `async.jobs` - Access to async job endpoints
- `admin` - Full admin access (can manage API keys)
- `*` - Wildcard (all permissions)

Default permissions: `["chat.completions"]`

## Using API Keys

Once created, API keys can be used to authenticate requests:

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer kiro-abc123..." \
  -H "Content-Type: application/json" \
  -d '{
    "model": "anthropic.claude-sonnet-4-5",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

## Storage

API keys are stored in the directory specified by `API_KEY_STORAGE_DIR` (default: `.kiro/api-keys/`).

Each key is stored as a separate JSON file with restricted permissions (0600).

**Directory Structure:**
```
.kiro/
├── api-keys/
│   ├── kiro-abc123....json
│   ├── kiro-xyz789....json
│   └── ...
└── admin-key.txt
```

## Security Best Practices

1. **Protect Admin Key:** The admin key has full access. Store it securely and never commit it to version control.

2. **Use Expiration:** Set expiration dates for API keys when possible.

3. **Rotate Keys:** Regularly rotate API keys, especially for production applications.

4. **Least Privilege:** Grant only the permissions needed for each key.

5. **Monitor Usage:** Check `usage_count` and `last_used_at` to detect unusual activity.

6. **Revoke Unused Keys:** Revoke or delete keys that are no longer needed.

7. **Backup Storage:** Backup the `.kiro/api-keys/` directory regularly.

## Automatic Cleanup

The gateway automatically cleans up expired API keys every hour. Expired keys are permanently deleted.

To disable automatic cleanup, you can modify the cleanup interval in the code or manually manage expired keys.

## Migration from Single API Key

If you're migrating from the single `PROXY_API_KEY` configuration:

1. The gateway still supports `PROXY_API_KEY` for backward compatibility
2. When API key manager is enabled, it takes precedence
3. You can gradually migrate by:
   - Creating new API keys for each application
   - Updating applications to use new keys
   - Eventually removing `PROXY_API_KEY` from configuration

## Troubleshooting

### "Invalid or expired API Key"

- Check that the key hasn't expired
- Verify the key is active (not revoked)
- Ensure you're using the full key (not the masked preview)

### "Admin access required"

- Use the admin API key for management endpoints
- Or create a key with `admin` permission

### "API key not found"

- The key may have been deleted or expired
- Check the `.kiro/api-keys/` directory for the key file

## Configuration

Environment variables:

```env
# Admin API key (set automatically on first run)
ADMIN_API_KEY=kiro-...

# API key storage directory
API_KEY_STORAGE_DIR=.kiro/api-keys

# Legacy single API key (optional, for backward compatibility)
PROXY_API_KEY=your-legacy-key
```

## Example Workflow

### 1. Create API Key for Production App

```bash
curl -X POST http://localhost:8080/v1/api-keys \
  -H "Authorization: Bearer $ADMIN_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Production App",
    "user_id": "prod-app-001",
    "expires_in": "90d",
    "permissions": ["chat.completions"],
    "metadata": {"environment": "production"}
  }'
```

Save the returned key: `kiro-xyz123...`

### 2. Use the Key in Your Application

```python
import openai

client = openai.OpenAI(
    api_key="kiro-xyz123...",
    base_url="http://localhost:8080/v1"
)

response = client.chat.completions.create(
    model="anthropic.claude-sonnet-4-5",
    messages=[{"role": "user", "content": "Hello!"}]
)
```

### 3. Monitor Usage

```bash
curl http://localhost:8080/v1/api-keys/kiro-xyz123... \
  -H "Authorization: Bearer $ADMIN_API_KEY"
```

### 4. Rotate Key Before Expiration

```bash
# Create new key
curl -X POST http://localhost:8080/v1/api-keys \
  -H "Authorization: Bearer $ADMIN_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Production App (Rotated)",
    "user_id": "prod-app-001",
    "expires_in": "90d"
  }'

# Update application with new key

# Revoke old key
curl -X POST http://localhost:8080/v1/api-keys/kiro-xyz123.../revoke \
  -H "Authorization: Bearer $ADMIN_API_KEY"
```

## Summary

The API key management system provides:
- ✅ Multiple API keys per gateway instance
- ✅ Expiration and automatic cleanup
- ✅ Usage tracking and statistics
- ✅ Granular permissions
- ✅ Secure file-based storage
- ✅ REST API for management
- ✅ Backward compatibility with single key
- ✅ Admin authentication for management endpoints

For more information, see the main [README.md](README.md) and [QUICK_REFERENCE.md](QUICK_REFERENCE.md).
