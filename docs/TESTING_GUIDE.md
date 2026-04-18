# Kiro Gateway Testing Guide

## Quick Start

```powershell
# Run all tests
.\scripts\test_gateway.ps1

# Run specific test suite
.\scripts\test_gateway.ps1 -TestSuite direct
.\scripts\test_gateway.ps1 -TestSuite openai
.\scripts\test_gateway.ps1 -TestSuite anthropic

# Run with options
.\scripts\test_gateway.ps1 -TestSuite direct -IncludeVision $true -IncludeText $false
.\scripts\test_gateway.ps1 -TestSuite openai -Verbose
.\scripts\test_gateway.ps1 -Timeout 180
```

## Test Suites

### Available Test Suites

| Suite | Description | Tests |
|-------|-------------|-------|
| `all` | Run all tests (default) | Everything below |
| `direct` | Direct API (Q Developer native) | Text + Vision |
| `openai` | OpenAI-compatible API | Text + Vision |
| `anthropic` | Anthropic-compatible API | Text + Vision |
| `health` | Health checks | Gateway health, metrics |
| `keys` | API key validation | Invalid key, missing key |
| `streaming` | Streaming responses | SSE format validation |
| `mcp` | MCP integration | (Future) |

### Test Options

#### `-TestSuite <suite>`
Which test suite to run. Default: `all`

```powershell
.\scripts\test_gateway.ps1 -TestSuite direct
```

#### `-IncludeVision <bool>`
Include vision/multimodal tests. Default: `$true`

```powershell
# Only vision tests
.\scripts\test_gateway.ps1 -TestSuite direct -IncludeVision $true -IncludeText $false

# Skip vision tests
.\scripts\test_gateway.ps1 -IncludeVision $false
```

#### `-IncludeText <bool>`
Include text-only tests. Default: `$true`

```powershell
# Only text tests
.\scripts\test_gateway.ps1 -IncludeText $true -IncludeVision $false
```

#### `-Timeout <seconds>`
Request timeout in seconds. Default: `120`

```powershell
# Longer timeout for slow connections
.\scripts\test_gateway.ps1 -Timeout 180
```

#### `-BaseUrl <url>`
Gateway base URL. Default: `http://localhost:8080`

```powershell
# Test remote gateway
.\scripts\test_gateway.ps1 -BaseUrl "https://gateway.example.com"
```

#### `-Verbose`
Show detailed output including response content

```powershell
.\scripts\test_gateway.ps1 -Verbose
```

## Common Test Scenarios

### 1. Quick Health Check
```powershell
.\scripts\test_gateway.ps1 -TestSuite health
```

### 2. Test Direct API Only
```powershell
# Text and vision
.\scripts\test_gateway.ps1 -TestSuite direct

# Text only
.\scripts\test_gateway.ps1 -TestSuite direct -IncludeVision $false

# Vision only
.\scripts\test_gateway.ps1 -TestSuite direct -IncludeText $false
```

### 3. Test OpenAI API Only
```powershell
# All OpenAI tests
.\scripts\test_gateway.ps1 -TestSuite openai

# With verbose output
.\scripts\test_gateway.ps1 -TestSuite openai -Verbose
```

### 4. Test Anthropic API Only
```powershell
.\scripts\test_gateway.ps1 -TestSuite anthropic
```

### 5. Test API Key Validation
```powershell
.\scripts\test_gateway.ps1 -TestSuite keys
```

### 6. Test Streaming
```powershell
.\scripts\test_gateway.ps1 -TestSuite streaming
```

### 7. Full Test Suite with Verbose Output
```powershell
.\scripts\test_gateway.ps1 -TestSuite all -Verbose
```

### 8. Test Specific Combinations
```powershell
# Direct API vision only with verbose output
.\scripts\test_gateway.ps1 -TestSuite direct -IncludeText $false -Verbose

# OpenAI text only with longer timeout
.\scripts\test_gateway.ps1 -TestSuite openai -IncludeVision $false -Timeout 180
```

## Test Script Features

### Improvements Over Old Script

1. **Retry Logic**: Automatically retries failed requests (up to 2 retries)
2. **Better Timeout Handling**: Configurable timeouts with proper error handling
3. **Event Stream Parsing**: Correctly extracts content from streaming responses
4. **Delay Between Requests**: Prevents context cancellations from rapid execution
5. **Flexible Test Selection**: Run only what you need
6. **Verbose Mode**: See actual response content for debugging
7. **Better Error Messages**: Clear indication of what failed and why

### Test Delays

The script includes automatic delays between requests to prevent issues:

- **Text requests**: 500ms delay
- **Vision requests**: 2s delay (multimodal takes longer)

### Retry Strategy

- **Max retries**: 2 (3 total attempts)
- **Retry delay**: 2 seconds
- **Retries on**: Network errors, timeouts, 5xx errors
- **No retry on**: 4xx errors (client errors)

## Understanding Test Results

### Output Format

```
========================================
  Test Results Summary
========================================

Total Tests: 15
✅ Passed: 14
❌ Failed: 1
Pass Rate: 93.3%
```

### Exit Codes

- `0`: All tests passed
- `1`: One or more tests failed

### Using in CI/CD

```powershell
# Run tests and capture exit code
.\scripts\test_gateway.ps1 -TestSuite all
if ($LASTEXITCODE -ne 0) {
    Write-Error "Tests failed"
    exit 1
}
```

## Troubleshooting

### Tests Timing Out

```powershell
# Increase timeout
.\scripts\test_gateway.ps1 -Timeout 180
```

### Vision Tests Failing

```powershell
# Check if image exists
Test-Path "test-data/test_diagram.png"

# Run only vision tests with verbose output
.\scripts\test_gateway.ps1 -TestSuite direct -IncludeText $false -Verbose
```

### API Key Issues

```powershell
# Check API key file
Get-Content ".kiro/admin-key.txt"

# Test key validation
.\scripts\test_gateway.ps1 -TestSuite keys
```

### Gateway Not Running

```powershell
# Check health
.\scripts\test_gateway.ps1 -TestSuite health

# Start gateway
.\dist\kiro-gateway.exe
```

## Test Coverage

### Direct API
- ✅ Text requests (multiple models)
- ✅ Vision requests (multiple models)
- ✅ Origin field validation
- ✅ Event stream parsing

### OpenAI API
- ✅ Text requests (multiple models)
- ✅ Vision requests (multiple models)
- ✅ JSON response format
- ✅ Streaming responses

### Anthropic API
- ✅ Text requests (multiple models)
- ✅ Vision requests (multiple models)
- ✅ JSON response format
- ✅ Anthropic-specific headers

### Infrastructure
- ✅ Health checks
- ✅ Metrics endpoint
- ✅ API key validation
- ✅ Error handling
- ✅ Streaming format

## Performance Testing

For performance testing, use the dedicated performance test script:

```powershell
.\scripts\test\test_performance.ps1
```

See `docs/PERFORMANCE_TESTING.md` for details.

## Integration Testing

For full integration tests including MCP:

```powershell
.\scripts\test\test_integration.ps1
```

See `docs/INTEGRATION_TESTING.md` for details.
