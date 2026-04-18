#!/usr/bin/env pwsh
# Get TOTP code from running Kiro Gateway container

param(
    [string]$ContainerName = "kiro-gateway-go-kiro-gateway-1",
    [string]$GatewayURL = "http://localhost:8080",
    [string]$ApiKey = $env:PROXY_API_KEY
)

Write-Host "Retrieving TOTP code from Kiro Gateway..." -ForegroundColor Cyan
Write-Host ""

# Check if API key is provided
if (-not $ApiKey) {
    Write-Host "Error: API key not provided" -ForegroundColor Red
    Write-Host ""
    Write-Host "Provide API key via:" -ForegroundColor Yellow
    Write-Host "  1. Parameter: -ApiKey YOUR_KEY" -ForegroundColor Gray
    Write-Host "  2. Environment: `$env:PROXY_API_KEY = 'YOUR_KEY'" -ForegroundColor Gray
    Write-Host "  3. From container: docker exec <container> cat /app/.kiro/api-keys/*.json" -ForegroundColor Gray
    exit 1
}

try {
    # Try to get TOTP from the gateway endpoint with authentication
    $headers = @{
        "Authorization" = "Bearer $ApiKey"
    }
    
    $response = Invoke-RestMethod -Uri "$GatewayURL/totp" -Method Get -Headers $headers -ErrorAction Stop
    
    Write-Host "TOTP Code: " -NoNewline -ForegroundColor Green
    Write-Host $response.code -ForegroundColor Yellow -BackgroundColor Black
    Write-Host ""
    Write-Host "Expires in: $($response.expires_in) seconds" -ForegroundColor Gray
    Write-Host "Timestamp: $($response.timestamp)" -ForegroundColor Gray
    Write-Host ""
    Write-Host "Use this code to authenticate other clients/devices" -ForegroundColor Cyan
    
} catch {
    $statusCode = $_.Exception.Response.StatusCode.value__
    
    if ($statusCode -eq 401) {
        Write-Host "Authentication failed: Invalid API key" -ForegroundColor Red
        Write-Host ""
        Write-Host "Get a valid API key:" -ForegroundColor Yellow
        Write-Host "  docker exec $ContainerName cat /app/.kiro/api-keys/*.json" -ForegroundColor Gray
    } elseif ($statusCode -eq 503) {
        Write-Host "TOTP secret not configured in gateway" -ForegroundColor Red
        Write-Host ""
        Write-Host "Configure MFA_TOTP_SECRET environment variable" -ForegroundColor Yellow
    } else {
        Write-Host "Failed to retrieve TOTP code from gateway endpoint" -ForegroundColor Red
        Write-Host "Error: $_" -ForegroundColor Red
    }
    
    Write-Host ""
    Write-Host "Trying to get code directly from container..." -ForegroundColor Yellow
    
    try {
        # Fallback: exec into container and curl the endpoint with auth
        $containerResponse = docker exec $ContainerName sh -c "curl -s -H 'Authorization: Bearer $ApiKey' http://localhost:8080/totp" 2>$null
        
        if ($containerResponse) {
            $data = $containerResponse | ConvertFrom-Json
            
            if ($data.code) {
                Write-Host "TOTP Code: " -NoNewline -ForegroundColor Green
                Write-Host $data.code -ForegroundColor Yellow -BackgroundColor Black
                Write-Host ""
                Write-Host "Expires in: $($data.expires_in) seconds" -ForegroundColor Gray
                Write-Host "Timestamp: $($data.timestamp)" -ForegroundColor Gray
            } else {
                Write-Host "Failed to retrieve TOTP code from container" -ForegroundColor Red
                Write-Host "Response: $containerResponse" -ForegroundColor Gray
                exit 1
            }
        } else {
            Write-Host "Failed to retrieve TOTP code from container" -ForegroundColor Red
            exit 1
        }
    } catch {
        Write-Host "Failed to retrieve TOTP code from container" -ForegroundColor Red
        Write-Host "Error: $_" -ForegroundColor Red
        Write-Host ""
        Write-Host "Make sure:" -ForegroundColor Yellow
        Write-Host "  1. Container is running: docker ps" -ForegroundColor Gray
        Write-Host "  2. API key is valid" -ForegroundColor Gray
        Write-Host "  3. MFA_TOTP_SECRET is configured" -ForegroundColor Gray
        Write-Host "  4. Gateway is healthy: curl http://localhost:8080/health" -ForegroundColor Gray
        exit 1
    }
}
