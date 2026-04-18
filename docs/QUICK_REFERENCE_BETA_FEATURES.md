# Beta Features Quick Reference

## Environment Variables

```bash
# Beta Features
ENABLE_EXTENDED_CONTEXT=false   # 1M context (Sonnet 4.5 only)
ENABLE_EXTENDED_THINKING=true   # Multi-hour reasoning
WARN_ON_BETA_FEATURES=true      # Log beta warnings
```

## Model Limits at a Glance

| Model | Context | Extended | Max Out | Images | Thinking |
|-------|---------|----------|---------|--------|----------|
| Sonnet 4.5 | 200K | 1M* | 32K/8K* | ✅ | ✅ |
| Opus 4.5 | 200K | ❌ | 32K | ✅ | ✅ |
| Haiku 4.5 | 200K | ❌ | 32K | ✅ | ❌ |
| 3.7 Sonnet | 200K | ❌ | 32K | ✅ | ❌ |

*Beta feature, requires `ENABLE_EXTENDED_CONTEXT=true`

## Model IDs

```
Sonnet 4.5:  claude-sonnet-4, claude-sonnet-4-20250929
Opus 4.5:    claude-opus-4-5, claude-opus-4-20251124
Haiku 4.5:   claude-haiku-4-5, claude-haiku-4-20251015
3.7 Sonnet:  claude-3.7-sonnet, claude-3-7-sonnet-20250224
```

## Context Window Modes

### Standard (Default)
```
Context: 200K tokens
Max Output: 32K tokens
Max Input: 168K tokens
```

### Extended (Beta - Sonnet 4.5)
```
Context: 1M tokens
Max Output: 8K tokens (reduced)
Max Input: 992K tokens
```

## Quick Test

```bash
# Standard context
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer $PROXY_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-sonnet-4",
    "messages": [{"role": "user", "content": "Hello"}],
    "max_tokens": 1000
  }'

# Extended context (requires ENABLE_EXTENDED_CONTEXT=true)
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer $PROXY_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-sonnet-4",
    "messages": [{"role": "user", "content": "Test extended"}],
    "max_tokens": 8000
  }'
```

## Common Errors

### Max Tokens Exceeded
```json
{
  "error": {
    "field": "max_tokens",
    "limit": 32000,
    "actual": 40000
  }
}
```
**Fix**: Use ≤32K (standard) or ≤8K (extended)

### Extended Context Not Supported
```json
{
  "error": {
    "message": "model does not support extended context"
  }
}
```
**Fix**: Use Sonnet 4.5 or disable extended context

### Context Window Exceeded
```json
{
  "error": {
    "field": "messages",
    "limit": 168000,
    "actual": 200000
  }
}
```
**Fix**: Reduce message size or enable extended context

## Log Messages

### Beta Warning
```
⚠️  BETA FEATURE: Extended context window (1M tokens) enabled for claude-sonnet-4
```

### Model Selection
```
Model: claude-sonnet-4, Stream: false, Messages: 1
```

### Validation Error
```
Validation error: max_tokens: max_tokens exceeds maximum (limit: 32000, actual: 40000)
```

## Best Practices

1. **Start Standard**: Use 200K context unless you need more
2. **Monitor Warnings**: Keep beta warnings enabled
3. **Adjust Max Tokens**: 8K for extended, 32K for standard
4. **Choose Right Model**:
   - Sonnet 4.5: Balanced, all features
   - Opus 4.5: Most intelligent
   - Haiku 4.5: Fastest, cheapest

## Testing

```bash
# Run test suite
./test_beta_features.ps1

# Enable extended context
$env:ENABLE_EXTENDED_CONTEXT="true"
./kiro-gateway.exe

# Check logs
tail -f gateway.log
```

## Documentation

- Full Guide: `BETA_FEATURES_GUIDE.md`
- Implementation: `MODEL_SPECIFIC_LIMITS_COMPLETE.md`
- API Specs: `docs/api/docs/api/AWS_Q_DEVELOPER_API_SPECIFICATIONS.md`
