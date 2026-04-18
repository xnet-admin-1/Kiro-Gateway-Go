# Beta Features Configuration Guide

This guide explains how to configure and use beta features in the Kiro Gateway.

## Overview

The gateway supports model-specific limits and beta features that can be enabled via environment variables. These features allow you to take advantage of cutting-edge capabilities while maintaining proper validation and warnings.

## Beta Features

### 1. Extended Context Window (1M Tokens)

**Status**: Beta (Sonnet 4.5 only)

The extended context window allows you to use up to 1 million tokens of context with Claude Sonnet 4.5, compared to the standard 200K token limit.

**Configuration**:
```bash
ENABLE_EXTENDED_CONTEXT=true
```

**Limitations**:
- Only available for Claude Sonnet 4.5 models
- Max output tokens reduced to 8K (from 32K) when using extended context
- May have higher latency due to larger context processing

**Supported Models**:
- `claude-sonnet-4`
- `claude-sonnet-4-20250929`

**Example**:
```bash
# Enable extended context
export ENABLE_EXTENDED_CONTEXT=true

# Start gateway
./kiro-gateway.exe
```

### 2. Extended Thinking

**Status**: Beta (Sonnet 4.5 and Opus 4.5)

Extended thinking enables multi-hour reasoning for complex problems that require deep analysis.

**Configuration**:
```bash
ENABLE_EXTENDED_THINKING=true  # Default: true
```

**Supported Models**:
- `claude-sonnet-4` / `claude-sonnet-4-20250929`
- `claude-opus-4-5` / `claude-opus-4-20251124`

**Not Supported**:
- Claude Haiku 4.5 (optimized for speed, not extended reasoning)
- Claude 3.7 Sonnet (legacy model)

### 3. Beta Feature Warnings

**Status**: Enabled by default

When enabled, the gateway logs warnings when beta features are used, helping you track experimental feature usage.

**Configuration**:
```bash
WARN_ON_BETA_FEATURES=true  # Default: true
```

**Example Warning**:
```
[req_1234567890] ⚠️  BETA FEATURE: Extended context window (1M tokens) enabled for claude-sonnet-4
```

## Model-Specific Limits

The gateway automatically enforces different limits based on the selected model:

### Claude Sonnet 4.5 (Balanced)

```
Model IDs: claude-sonnet-4, claude-sonnet-4-20250929
Context Window: 200K tokens (standard) / 1M tokens (extended beta)
Max Output: 32K tokens (standard) / 8K tokens (extended)
Multimodal: ✅ Yes
Extended Thinking: ✅ Yes
Extended Context: ✅ Yes (beta)
Speed: Medium
Intelligence: High
Cost: Medium
```

### Claude Opus 4.5 (Most Intelligent)

```
Model IDs: claude-opus-4-5, claude-opus-4-20251124
Context Window: 200K tokens
Max Output: 32K tokens
Multimodal: ✅ Yes
Extended Thinking: ✅ Yes
Extended Context: ❌ No
Speed: Slow
Intelligence: Highest
Cost: High
```

### Claude Haiku 4.5 (Fastest)

```
Model IDs: claude-haiku-4-5, claude-haiku-4-20251015
Context Window: 200K tokens
Max Output: 32K tokens
Multimodal: ✅ Yes
Extended Thinking: ❌ No
Extended Context: ❌ No
Speed: Fast
Intelligence: Good
Cost: Low
```

### Claude 3.7 Sonnet (Legacy)

```
Model IDs: claude-3.7-sonnet, claude-3-7-sonnet-20250224
Context Window: 200K tokens
Max Output: 32K tokens
Multimodal: ✅ Yes
Extended Thinking: ❌ No
Extended Context: ❌ No
Speed: Medium
Intelligence: Good
Cost: Medium
```

## Environment Variables Reference

### Beta Features

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `ENABLE_EXTENDED_CONTEXT` | boolean | `false` | Enable 1M token context window (Sonnet 4.5 only) |
| `ENABLE_EXTENDED_THINKING` | boolean | `true` | Enable multi-hour extended thinking |
| `WARN_ON_BETA_FEATURES` | boolean | `true` | Log warnings when beta features are used |

### Validation

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `ENFORCE_STRICT_LIMITS` | boolean | `true` | Enforce AWS Q Developer limits before API calls |

## Complete Configuration Example

```bash
# .env file

# API Configuration
PORT=8080
PROXY_API_KEY=your-api-key-here

# AWS Configuration
AWS_REGION=us-east-1
AWS_PROFILE=xnet-admin

# Authentication Mode
AMAZON_Q_SIGV4=false  # Use bearer token
# AMAZON_Q_SIGV4=true  # Use SigV4 (required for multimodal)

# Bearer Token (for CodeWhisperer API)
BEARER_TOKEN=your-bearer-token-here

# Q Developer Mode (for multimodal support)
Q_USE_SENDMESSAGE=false  # Auto-switches when images detected
# Q_USE_SENDMESSAGE=true  # Always use Q Developer endpoint

# Beta Features
ENABLE_EXTENDED_CONTEXT=false  # Set to true for 1M context (Sonnet 4.5)
ENABLE_EXTENDED_THINKING=true  # Enabled by default
WARN_ON_BETA_FEATURES=true     # Log beta feature usage

# Rate Limiting
MAX_CONNECTIONS=100
KEEP_ALIVE_CONNECTIONS=10
```

## Usage Examples

### Example 1: Standard Configuration (200K Context)

```bash
# .env
ENABLE_EXTENDED_CONTEXT=false
ENABLE_EXTENDED_THINKING=true
WARN_ON_BETA_FEATURES=true
```

**Request**:
```json
{
  "model": "claude-sonnet-4",
  "messages": [
    {"role": "user", "content": "Explain quantum computing"}
  ],
  "max_tokens": 4000
}
```

**Validation**:
- ✅ Context window: 200K tokens
- ✅ Max output: 32K tokens (4K requested)
- ✅ Extended thinking: Available

### Example 2: Extended Context (1M Tokens)

```bash
# .env
ENABLE_EXTENDED_CONTEXT=true
ENABLE_EXTENDED_THINKING=true
WARN_ON_BETA_FEATURES=true
```

**Request**:
```json
{
  "model": "claude-sonnet-4",
  "messages": [
    {"role": "user", "content": "Analyze this 500K token document..."}
  ],
  "max_tokens": 8000
}
```

**Validation**:
- ✅ Context window: 1M tokens (beta)
- ✅ Max output: 8K tokens (reduced for extended context)
- ⚠️ Warning logged: "BETA FEATURE: Extended context window (1M tokens) enabled"

### Example 3: Unsupported Feature

```bash
# .env
ENABLE_EXTENDED_CONTEXT=true
```

**Request**:
```json
{
  "model": "claude-haiku-4-5",
  "messages": [
    {"role": "user", "content": "Hello"}
  ]
}
```

**Response**:
```json
{
  "error": {
    "message": "model does not support extended context window (beta)",
    "type": "validation_error",
    "code": 400,
    "field": "model"
  }
}
```

## Validation Behavior

### With Extended Context Disabled (Default)

```
Context Window: 200K tokens
Max Output: 32K tokens
Total Input Limit: 168K tokens (200K - 32K)
```

### With Extended Context Enabled (Beta)

```
Context Window: 1M tokens (Sonnet 4.5 only)
Max Output: 8K tokens (reduced)
Total Input Limit: 992K tokens (1M - 8K)
```

## Error Messages

### Extended Context Not Supported

```json
{
  "error": {
    "message": "model does not support extended context window (beta)",
    "type": "validation_error",
    "code": 400,
    "field": "model"
  }
}
```

### Context Window Exceeded

```json
{
  "error": {
    "message": "total message size exceeds context window (using extended context)",
    "type": "validation_error",
    "code": 400,
    "field": "messages",
    "limit": 992000,
    "actual": 1000000
  }
}
```

### Max Tokens Exceeded

```json
{
  "error": {
    "message": "max_tokens exceeds maximum output tokens for this model (using extended context)",
    "type": "validation_error",
    "code": 400,
    "field": "max_tokens",
    "limit": 8000,
    "actual": 32000
  }
}
```

## Best Practices

1. **Start with Standard Context**: Use the default 200K context window unless you specifically need extended context
2. **Monitor Beta Warnings**: Keep `WARN_ON_BETA_FEATURES=true` to track experimental feature usage
3. **Adjust Max Tokens**: When using extended context, remember max output is reduced to 8K tokens
4. **Choose the Right Model**: 
   - Sonnet 4.5: Balanced performance, supports all features
   - Opus 4.5: Maximum intelligence, no extended context
   - Haiku 4.5: Maximum speed, no extended features
5. **Test Thoroughly**: Beta features may have different behavior or limitations

## Troubleshooting

### Extended Context Not Working

**Problem**: Extended context not being used despite `ENABLE_EXTENDED_CONTEXT=true`

**Solutions**:
1. Verify you're using a supported model (Sonnet 4.5)
2. Check gateway logs for beta feature warnings
3. Ensure environment variable is properly set

### Max Tokens Validation Error

**Problem**: Getting "max_tokens exceeds maximum" error

**Solutions**:
1. With extended context: Use ≤8K tokens
2. With standard context: Use ≤32K tokens
3. Check which context mode is active in logs

### Model Not Supported Error

**Problem**: Getting "model does not support" error

**Solutions**:
1. Check model ID is correct
2. Verify feature support in model limits table above
3. Use a different model that supports the feature

## Monitoring

The gateway logs detailed information about beta feature usage:

```
[req_1234567890] Model: claude-sonnet-4, Stream: false, Messages: 1
[req_1234567890] ⚠️  BETA FEATURE: Extended context window (1M tokens) enabled for claude-sonnet-4
[req_1234567890] Using API endpoint: /sendMessage (Q Developer: true, Auth: SigV4)
```

Monitor these logs to:
- Track beta feature adoption
- Identify potential issues
- Optimize model selection
- Plan for production deployment

## Future Features

Features planned for future releases:
- Extended context for Opus 4.5
- Additional model variants
- Custom context window sizes
- Advanced rate limiting per model
- Cost tracking per model tier

## Support

For issues or questions about beta features:
1. Check gateway logs for detailed error messages
2. Review validation error responses
3. Consult AWS Q Developer documentation
4. File an issue with logs and configuration
