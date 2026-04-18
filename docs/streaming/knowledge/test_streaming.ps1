# Test streaming mode
$headers = @{
    "Authorization" = "Bearer test-bearer-key-12345"
    "Content-Type" = "application/json"
}

$body = @{
    model = "anthropic.claude-3-5-sonnet-20241022-v2:0"
    messages = @(
        @{
            role = "user"
            content = "Count to 5"
        }
    )
    stream = $true
} | ConvertTo-Json -Depth 10

Write-Host "Testing streaming mode..."
Write-Host "Request body: $body"
Write-Host ""

# Use curl for streaming
curl.exe -X POST http://localhost:8080/v1/chat/completions `
    -H "Authorization: Bearer test-bearer-key-12345" `
    -H "Content-Type: application/json" `
    -d $body `
    --no-buffer
