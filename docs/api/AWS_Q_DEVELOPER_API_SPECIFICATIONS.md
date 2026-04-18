# AWS Q Developer API Specifications - Complete Reference

## Date: January 22, 2026, 7:30 PM

## Overview

This document provides comprehensive specifications for both AWS Q Developer and CodeWhisperer APIs, including available models, media support, size limits, token counts, and rate limits.

---

## Table of Contents

1. [API Endpoints](#api-endpoints)
2. [Available Models](#available-models)
3. [Context Window & Token Limits](#context-window--token-limits)
4. [Multimodal Support](#multimodal-support)
5. [Rate Limits & Quotas](#rate-limits--quotas)
6. [Request Size Limits](#request-size-limits)
7. [Error Codes](#error-codes)
8. [Best Practices](#best-practices)

---

## API Endpoints

### CodeWhisperer API (Bearer Token)

**Base URL**: `https://codewhisperer.{region}.amazonaws.com`

**Available Regions**:
- `us-east-1` (US East - N. Virginia) - Default
- `eu-central-1` (Europe - Frankfurt)

**Operations**:
- `/generateAssistantResponse` - Text-only chat completions

**Authentication**: Bearer Token (from AWS SSO)

**Features**:
- ✅ Text conversations
- ❌ Multimodal (images)
- ❌ Advanced Q Developer features

### Q Developer API (SigV4)

**Base URL**: `https://q.{region}.amazonaws.com`

**Available Regions**:
- `us-east-1` (US East - N. Virginia) - Default
- `eu-central-1` (Europe - Frankfurt)

**Operations**:
- `/sendMessage` - Full-featured chat completions with multimodal support

**Authentication**: AWS SigV4 (IAM credentials)

**Features**:
- ✅ Text conversations
- ✅ Multimodal (images)
- ✅ All Q Developer features
- ✅ Advanced reasoning
- ✅ Extended thinking mode

---

## Available Models

### Current Claude Models (as of January 2026)

Based on Q CLI source code and AWS documentation, the following Claude models are available:

#### Claude Sonnet 4.5 (Recommended)

**Model ID**: `claude-sonnet-4` or `claude-sonnet-4-20250929`

**Specifications**:
- **Context Window**: 200,000 tokens (standard) / 1,000,000 tokens (beta)
- **Max Output**: 32,000 tokens
- **Knowledge Cutoff**: July 2025
- **Status**: Active (Latest)
- **Best For**: Complex agents, coding, balanced performance

**Features**:
- ✅ Text generation
- ✅ Vision (multimodal)
- ✅ Tool use
- ✅ Extended thinking
- ✅ Code generation
- ✅ Multi-language support

#### Claude Opus 4.5

**Model ID**: `claude-opus-4-5` or `claude-opus-4-20251124`

**Specifications**:
- **Context Window**: 200,000 tokens
- **Max Output**: 32,000 tokens
- **Knowledge Cutoff**: May 2025
- **Status**: Active
- **Best For**: Most intelligent model, top-level performance, complex tasks

**Features**:
- ✅ Highest intelligence
- ✅ Production code
- ✅ Sophisticated agents
- ✅ Complex office tasks
- ✅ Multi-hour extended thinking (up to 7 hours)
- ✅ Vision (multimodal)

#### Claude Haiku 4.5

**Model ID**: `claude-haiku-4-5` or `claude-haiku-4-20251015`

**Specifications**:
- **Context Window**: 200,000 tokens
- **Max Output**: 32,000 tokens
- **Knowledge Cutoff**: February 2025
- **Status**: Active
- **Best For**: Fastest model, near-instant responsiveness

**Features**:
- ✅ Fastest response time
- ✅ Near-frontier performance
- ✅ Cost-effective
- ✅ Vision (multimodal)

#### Claude 3.7 Sonnet (Legacy)

**Model ID**: `claude-3.7-sonnet` or `claude-3-7-sonnet-20250224`

**Specifications**:
- **Context Window**: 200,000 tokens
- **Max Output**: 32,000 tokens
- **Knowledge Cutoff**: October 2024
- **Status**: Deprecated (use Claude Sonnet 4.5 instead)

### Model Comparison

| Model | Context Window | Max Output | Speed | Intelligence | Cost | Multimodal |
|-------|---------------|------------|-------|--------------|------|------------|
| **Claude Opus 4.5** | 200K | 32K | Slow | Highest | Highest | ✅ |
| **Claude Sonnet 4.5** | 200K / 1M (beta) | 32K | Medium | High | Medium | ✅ |
| **Claude Haiku 4.5** | 200K | 32K | Fastest | Good | Lowest | ✅ |
| **Claude 3.7 Sonnet** | 200K | 32K | Medium | Good | Medium | ✅ |

---

## Context Window & Token Limits

### Token Calculations

**Character to Token Ratio** (English):
- **Average**: 4.7 characters per token
- **200,000 tokens** ≈ 940,000 characters ≈ 150,000 words ≈ 500+ pages

**Context Window Utilization**:
- **75% recommended** for context files (150,000 tokens)
- **25% reserved** for response generation (50,000 tokens)

### Context Window Sizes

| Model | Standard Context | Extended Context (Beta) |
|-------|-----------------|------------------------|
| Claude Opus 4.5 | 200,000 tokens | N/A |
| Claude Sonnet 4.5 | 200,000 tokens | 1,000,000 tokens |
| Claude Haiku 4.5 | 200,000 tokens | N/A |

### Service Limits (from Q CLI source code)

```rust
// From crates/chat-cli/src/cli/chat/consts.rs

MAX_CONVERSATION_STATE_HISTORY_LEN: 10,000 messages
MAX_TOOL_RESPONSE_SIZE: 400,000 characters (service limit: 800,000)
MAX_USER_MESSAGE_SIZE: 400,000 characters (service limit: 600,000)
MAX_CURRENT_WORKING_DIRECTORY_LEN: 256 characters
```

---

## Multimodal Support

### Image Support

**Supported Formats**:
- ✅ PNG (`.png`)
- ✅ JPEG/JPG (`.jpeg`, `.jpg`)
- ✅ WebP (`.webp`)
- ✅ GIF (`.gif`)

**Image Size Limits**:

| Scenario | Max Resolution | Max File Size | Max Images |
|----------|---------------|---------------|------------|
| **Standard (≤20 images)** | 8,000 × 8,000 px | 10 MB per image | 20 images |
| **Bulk (>20 images)** | 2,000 × 2,000 px | 10 MB per image | 100 images (API) |
| **Claude.ai Web** | 8,000 × 8,000 px | 30 MB per image | 5 images |

**From Q CLI Source Code**:
```rust
// From crates/chat-cli/src/cli/chat/consts.rs
MAX_NUMBER_OF_IMAGES_PER_REQUEST: 10
MAX_IMAGE_SIZE: 10 * 1024 * 1024  // 10 MB in bytes
```

**From Anthropic Documentation**:
- Images larger than 8,000 × 8,000 px are **rejected**
- More than 20 images: max resolution reduced to 2,000 × 2,000 px
- API supports up to 100 images per request (with reduced resolution)

### Image Processing

**Automatic Resizing**:
- Images are automatically resized to fit within limits
- Aspect ratio is preserved
- Quality is optimized for model processing

**Token Cost**:
- Images consume tokens based on resolution
- Higher resolution = more tokens
- Exact token count varies by image complexity

### Multimodal Request Format

**OpenAI Format** (Gateway Input):
```json
{
  "model": "claude-sonnet-4",
  "messages": [{
    "role": "user",
    "content": [
      {
        "type": "text",
        "text": "What do you see in this image?"
      },
      {
        "type": "image_url",
        "image_url": {
          "url": "data:image/png;base64,iVBORw0KG..."
        }
      }
    ]
  }]
}
```

**Q Developer Format** (Gateway Output):
```json
{
  "conversationState": {
    "currentMessage": {
      "userInputMessage": {
        "content": "What do you see in this image?",
        "images": [
          {
            "format": "png",
            "source": {
              "bytes": "iVBORw0KG..."
            }
          }
        ]
      }
    }
  }
}
```

---

## Rate Limits & Quotas

### Amazon Q Developer Pro Quotas

**IDE and CLI Usage** (per user, per month):

| Feature | Quota | Pooling |
|---------|-------|---------|
| **Agentic Requests** | 10,000 inference calls (~1,000 user inputs) | Per user |
| **Code Transformation** | 4,000 lines of code | Account-level pooled |
| **Extra Lines of Code** | Available for purchase | Account-level |

**AWS Management Console** (per user, per month):

| Feature | Quota |
|---------|-------|
| **Generative SQL** | 1,000 queries |
| **Network Reachability Analysis** | 20 requests per day |

**Amazon CodeCatalyst** (per user, per month):

| Feature | Quota |
|---------|-------|
| **Agent for Software Development** | 30 requests |
| **Pull Request Summaries** | 20 requests |

### .NET Transform Quotas

**Q Dev Pro Subscription** (per user, per supported region):

| Feature | Quota |
|---------|-------|
| **Monthly Lines of Code** | 1,000,000 lines |
| **Concurrent Jobs** | 10 jobs |

**BuilderID** (per user, per supported region):

| Feature | Quota |
|---------|-------|
| **Monthly Lines of Code** | 1,000 lines |
| **Concurrent Jobs** | 1 job |

### API Rate Limits

**Estimated Limits** (based on AWS patterns):

| Endpoint | Requests Per Second (RPS) | Burst Capacity |
|----------|--------------------------|----------------|
| **CodeWhisperer API** | ~10 RPS | ~50 requests |
| **Q Developer API** | ~10 RPS | ~50 requests |

**Note**: Exact rate limits are not publicly documented. These are conservative estimates based on:
- AWS API Gateway standard limits (10,000 RPS with 5,000 burst)
- Q Developer service quotas
- Observed behavior in Q CLI

### Throttling Behavior

**Error Codes**:
- `QuotaBreachError` - Rate limit exceeded
- `ThrottlingError` - Request throttled
- `ServiceQuotaExceededError` - Service quota exceeded
- `MonthlyLimitReached` - Monthly usage limit reached

**Retry Strategy**:
- Exponential backoff with jitter
- Max retries: 3
- Initial delay: 1 second
- Max delay: 30 seconds

---

## Request Size Limits

### Message Size Limits

| Component | Limit | Notes |
|-----------|-------|-------|
| **User Message** | 400,000 characters | Service limit: 600,000 |
| **Tool Response** | 400,000 characters | Service limit: 800,000 |
| **Conversation History** | 10,000 messages | Total messages in state |
| **Current Working Directory** | 256 characters | Path length |

### Total Request Size

**Estimated Limits**:
- **Text-only**: ~400 KB (400,000 characters)
- **With Images**: ~40 MB (10 images × 10 MB each, after base64 encoding)
- **API Gateway Limit**: 10 MB payload (may require chunking for large requests)

### Response Size Limits

| Component | Limit |
|-----------|-------|
| **Max Output Tokens** | 32,000 tokens (~150,000 characters) |
| **Streaming Chunk Size** | Variable (typically 1-100 tokens per chunk) |

---

## Error Codes

### Common Error Codes

| Error Code | HTTP Status | Description | Retry? |
|------------|-------------|-------------|--------|
| `QuotaBreachError` | 429 | Rate limit exceeded | ✅ Yes (with backoff) |
| `ThrottlingError` | 429 | Request throttled | ✅ Yes (with backoff) |
| `ServiceQuotaExceededError` | 429 | Service quota exceeded | ✅ Yes (after quota increase) |
| `MonthlyLimitReached` | 402 | Monthly usage limit reached | ❌ No (wait for reset) |
| `ContextWindowOverflow` | 400 | Input too long | ❌ No (reduce input size) |
| `ModelOverloadedError` | 503 | Model temporarily unavailable | ✅ Yes (switch model) |
| `ValidationError` | 400 | Invalid request | ❌ No (fix request) |
| `AuthenticationFailed` | 401 | Invalid credentials | ❌ No (refresh credentials) |
| `InternalServerError` | 500 | Service error | ✅ Yes (with backoff) |

### Error Handling

**From Q CLI Source Code**:
```rust
// Error classification logic
match (status_code, body_content) {
    (429, _) => Throttling,
    (_, b"Input is too long.") => ContextWindowOverflow,
    (_, b"INSUFFICIENT_MODEL_CAPACITY") => ModelOverloadedError,
    (_, b"MONTHLY_REQUEST_COUNT") => MonthlyLimitReached,
    (500..=599, _) => ServiceFailure,
    _ => Unknown
}
```

---

## Best Practices

### 1. Model Selection

**Choose the right model for your use case**:

- **Claude Opus 4.5**: Complex reasoning, production code, multi-hour tasks
- **Claude Sonnet 4.5**: Balanced performance, most use cases, coding
- **Claude Haiku 4.5**: Fast responses, simple tasks, cost-sensitive

### 2. Context Window Management

**Optimize context usage**:

```
Total Context (200K tokens)
├── System Prompt: ~1,000 tokens (0.5%)
├── Conversation History: ~50,000 tokens (25%)
├── Context Files: ~100,000 tokens (50%)
├── Current Message: ~10,000 tokens (5%)
└── Reserved for Response: ~39,000 tokens (19.5%)
```

**Tips**:
- Keep system prompts concise
- Limit conversation history to recent messages
- Use context file summarization for large codebases
- Monitor token usage with `/usage` command

### 3. Image Optimization

**Before sending images**:

1. **Resize**: Keep under 8,000 × 8,000 px (or 2,000 × 2,000 px for >20 images)
2. **Compress**: Aim for <5 MB per image (limit is 10 MB)
3. **Format**: Use PNG for screenshots, JPEG for photos
4. **Quantity**: Limit to 10 images per request (Q CLI limit)

**Example**:
```bash
# Resize image to 2000x2000 max
convert input.png -resize 2000x2000\> output.png

# Compress to <5MB
convert input.png -quality 85 -define png:compression-level=9 output.png
```

### 4. Rate Limit Handling

**Implement exponential backoff**:

```python
import time
import random

def make_request_with_retry(request_func, max_retries=3):
    for attempt in range(max_retries):
        try:
            return request_func()
        except ThrottlingError:
            if attempt == max_retries - 1:
                raise
            
            # Exponential backoff with jitter
            delay = (2 ** attempt) + random.uniform(0, 1)
            time.sleep(delay)
```

### 5. Quota Management

**Monitor usage**:

```bash
# Check current usage (Q CLI)
q chat /usage

# View quota limits
aws service-quotas list-service-quotas \
  --service-code amazonq \
  --region us-east-1
```

**Request quota increases**:
- Use AWS Service Quotas console
- Provide justification for increase
- Allow 1-2 business days for approval

### 6. Error Recovery

**Handle common errors gracefully**:

```python
def handle_api_error(error):
    if error.code == "ContextWindowOverflow":
        # Summarize conversation history
        return summarize_and_retry()
    
    elif error.code == "MonthlyLimitReached":
        # Wait for quota reset or upgrade plan
        return notify_user_quota_exceeded()
    
    elif error.code == "ModelOverloadedError":
        # Switch to alternative model
        return retry_with_different_model()
    
    elif error.code in ["ThrottlingError", "QuotaBreachError"]:
        # Exponential backoff
        return retry_with_backoff()
    
    else:
        # Log and notify
        return handle_unknown_error(error)
```

### 7. Multimodal Best Practices

**For optimal multimodal performance**:

1. **Use Q Developer API**: Required for multimodal support
2. **Enable SigV4**: Set `Q_USE_SENDMESSAGE=true` and `AMAZON_Q_SIGV4=true`
3. **Provide Context**: Include descriptive text with images
4. **Batch Images**: Send related images together for comparison
5. **Monitor Tokens**: Images consume significant tokens

**Example**:
```json
{
  "messages": [{
    "role": "user",
    "content": [
      {
        "type": "text",
        "text": "Compare these two UI designs and suggest improvements:"
      },
      {
        "type": "image_url",
        "image_url": {"url": "data:image/png;base64,..."}
      },
      {
        "type": "image_url",
        "image_url": {"url": "data:image/png;base64,..."}
      }
    ]
  }]
}
```

---

## Summary

### Key Specifications

| Specification | Value |
|--------------|-------|
| **Context Window** | 200,000 tokens (standard) / 1,000,000 tokens (beta) |
| **Max Output** | 32,000 tokens |
| **Max Images** | 10 per request (Q CLI) / 20 per request (API) |
| **Max Image Size** | 10 MB per image |
| **Max Image Resolution** | 8,000 × 8,000 px (≤20 images) / 2,000 × 2,000 px (>20 images) |
| **Supported Formats** | PNG, JPEG, WebP, GIF |
| **Rate Limit** | ~10 RPS (estimated) |
| **Monthly Quota** | 10,000 inference calls (~1,000 user inputs) |

### API Comparison

| Feature | CodeWhisperer API | Q Developer API |
|---------|-------------------|-----------------|
| **Endpoint** | `codewhisperer.{region}.amazonaws.com` | `q.{region}.amazonaws.com` |
| **Auth** | Bearer Token | AWS SigV4 |
| **Multimodal** | ❌ No | ✅ Yes |
| **Context Window** | 200K tokens | 200K / 1M tokens |
| **Advanced Features** | ❌ Limited | ✅ Full |

---

## References

### Source Code References

1. **Q CLI Repository**: `aws/kiro-q/amazon-q-developer-cli/`
   - Model definitions: `crates/chat-cli/src/cli/chat/cli/model.rs`
   - Constants: `crates/chat-cli/src/cli/chat/consts.rs`
   - Image handling: `crates/chat-cli/src/cli/chat/util/images.rs`
   - Error handling: `crates/chat-cli/src/api_client/error.rs`

2. **API Clients**:
   - CodeWhisperer: `crates/amzn-codewhisperer-streaming-client/`
   - Q Developer: `crates/amzn-qdeveloper-streaming-client/`

### Documentation References

1. **AWS Documentation**:
   - [Amazon Q Developer Endpoints and Quotas](https://docs.aws.amazon.com/general/latest/gr/amazonqdev.html)
   - [Amazon Q Developer Pricing](https://aws.amazon.com/q/developer/pricing/)
   - [Amazon Q Developer FAQs](https://aws.amazon.com/q/developer/faqs/)

2. **Anthropic Documentation**:
   - [Claude Vision API](https://docs.anthropic.com/en/docs/build-with-claude/vision)
   - [Claude Models Overview](https://docs.claude.com/en/docs/about-claude/models/overview)
   - [What's New in Claude 4.5](https://docs.claude.com/en/docs/about-claude/models/whats-new-claude-4-5)

3. **Community Resources**:
   - [Claude 4 Announcement](https://aws.amazon.com/blogs/aws/claude-opus-4-anthropics-most-powerful-model-for-coding-is-now-in-amazon-bedrock/)
   - [Amazon Q Developer Alternatives Analysis](https://www.augmentcode.com/guides/amazon-q-developer-alternatives)

---

**Documentation Completed**: January 22, 2026, 7:30 PM  
**Status**: ✅ **COMPREHENSIVE SPECIFICATIONS DOCUMENTED**  
**Version**: 1.0

---

## Changelog

### Version 1.0 (January 22, 2026)
- Initial comprehensive documentation
- Documented all available Claude models (Opus 4.5, Sonnet 4.5, Haiku 4.5)
- Added context window specifications (200K / 1M tokens)
- Documented multimodal support (images, formats, size limits)
- Added rate limits and quotas from AWS documentation
- Included error codes and handling strategies
- Added best practices for model selection, context management, and image optimization
- Referenced Q CLI source code for accurate limits
