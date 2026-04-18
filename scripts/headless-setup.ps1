# Headless Authentication Setup Script (PowerShell)
# Automates the setup of Kiro Gateway for headless environments on Windows

# Functions
function Write-Header {
    param([string]$Message)
    Write-Host "========================================" -ForegroundColor Blue
    Write-Host $Message -ForegroundColor Blue
    Write-Host "========================================" -ForegroundColor Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "✓ $Message" -ForegroundColor Green
}

function Write-Error {
    param([string]$Message)
    Write-Host "✗ $Message" -ForegroundColor Red
}

function Write-Warning {
    param([string]$Message)
    Write-Host "⚠ $Message" -ForegroundColor Yellow
}

function Write-Info {
    param([string]$Message)
    Write-Host "ℹ $Message" -ForegroundColor Cyan
}

# Check for AWS credentials
function Test-AWSCredentials {
    Write-Header "Checking AWS Credentials"
    
    try {
        $identity = aws sts get-caller-identity 2>$null | ConvertFrom-Json
        
        if ($identity) {
            Write-Success "AWS credentials found"
            Write-Info "Account: $($identity.Account)"
            Write-Info "Identity: $($identity.Arn)"
            return $true
        }
    }
    catch {
        Write-Error "No AWS credentials found"
        return $false
    }
    
    return $false
}

# Setup Method 1: IAM + SigV4
function Setup-SigV4 {
    Write-Header "Setting up IAM + SigV4 Authentication"
    
    if (-not (Test-AWSCredentials)) {
        Write-Error "AWS credentials required for SigV4 authentication"
        Write-Info "Please configure AWS credentials using one of:"
        Write-Info "  1. aws configure"
        Write-Info "  2. IAM role (EC2/ECS/Lambda)"
        Write-Info "  3. Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)"
        return $false
    }
    
    # Generate API key
    $apiKey = if ($env:PROXY_API_KEY) { $env:PROXY_API_KEY } else { 
        -join ((48..57) + (97..102) | Get-Random -Count 64 | ForEach-Object {[char]$_})
    }
    
    # Get AWS region
    $awsRegion = if ($env:AWS_REGION) { $env:AWS_REGION } else { "us-east-1" }
    
    # Get Profile ARN if set
    $profileArn = $env:PROFILE_ARN
    
    # Create .env file
    $envContent = @"
# Kiro Gateway - Headless Configuration (SigV4)
# Generated: $(Get-Date)

# Server Configuration
PORT=8090
PROXY_API_KEY=$apiKey

# Authentication - SigV4 with IAM credentials
AMAZON_Q_SIGV4=true
AWS_REGION=$awsRegion

# Use Q Developer endpoint with SigV4
Q_USE_SENDMESSAGE=true

# Profile ARN (if using Identity Center)
$(if ($profileArn) { "PROFILE_ARN=$profileArn" } else { "# PROFILE_ARN=arn:aws:codewhisperer:us-east-1:123456789012:profile/abc123" })

# Logging
LOG_LEVEL=info
DEBUG=false

# Retry Configuration
MAX_RETRIES=3
MAX_BACKOFF=30s
CONNECT_TIMEOUT=10s
READ_TIMEOUT=30s
OPERATION_TIMEOUT=60s
"@
    
    $envContent | Out-File -FilePath ".env" -Encoding UTF8
    
    Write-Success "Created .env file with SigV4 configuration"
    Write-Info "API Key: $apiKey"
    
    return $true
}

# Setup Method 2: Bearer Token
function Setup-BearerToken {
    Write-Header "Setting up Bearer Token Authentication"
    
    # Check for AWS CLI
    if (-not (Get-Command aws -ErrorAction SilentlyContinue)) {
        Write-Error "AWS CLI not found. Please install AWS CLI first."
        return $false
    }
    
    # Try to get token from SSO cache
    $ssoCacheDir = "$env:USERPROFILE\.aws\sso\cache"
    
    if (Test-Path $ssoCacheDir) {
        $cacheFiles = Get-ChildItem -Path $ssoCacheDir -Filter "*.json" | Sort-Object LastWriteTime -Descending
        
        if ($cacheFiles.Count -gt 0) {
            $cacheFile = $cacheFiles[0].FullName
            Write-Info "Found SSO cache file: $cacheFile"
            
            try {
                $cacheData = Get-Content $cacheFile | ConvertFrom-Json
                $accessToken = $cacheData.accessToken
                $expiresAt = $cacheData.expiresAt
                
                if ($accessToken) {
                    Write-Success "Extracted bearer token from SSO cache"
                    Write-Info "Token expires at: $expiresAt"
                    
                    # Generate API key
                    $apiKey = if ($env:PROXY_API_KEY) { $env:PROXY_API_KEY } else { 
                        -join ((48..57) + (97..102) | Get-Random -Count 64 | ForEach-Object {[char]$_})
                    }
                    
                    # Get AWS region
                    $awsRegion = if ($env:AWS_REGION) { $env:AWS_REGION } else { "us-east-1" }
                    
                    # Get Profile ARN if set
                    $profileArn = $env:PROFILE_ARN
                    
                    # Create .env file
                    $envContent = @"
# Kiro Gateway - Headless Configuration (Bearer Token)
# Generated: $(Get-Date)

# Server Configuration
PORT=8090
PROXY_API_KEY=$apiKey

# Authentication - Bearer Token
AMAZON_Q_SIGV4=false
AWS_REGION=$awsRegion
BEARER_TOKEN=$accessToken

# Profile ARN (required for bearer token mode)
$(if ($profileArn) { "PROFILE_ARN=$profileArn" } else { "# PROFILE_ARN=arn:aws:codewhisperer:us-east-1:123456789012:profile/abc123" })

# Logging
LOG_LEVEL=info
DEBUG=false
"@
                    
                    $envContent | Out-File -FilePath ".env" -Encoding UTF8
                    
                    Write-Success "Created .env file with bearer token"
                    Write-Warning "Token will expire at: $expiresAt"
                    Write-Warning "You will need to refresh the token before expiration"
                    
                    return $true
                }
            }
            catch {
                Write-Error "Failed to parse SSO cache file: $_"
            }
        }
    }
    
    Write-Error "No valid SSO token found"
    Write-Info "Please run: aws sso login --profile YOUR_PROFILE"
    return $false
}

# Setup Method 3: CLI DB
function Setup-CLIDB {
    Write-Header "Setting up CLI DB Authentication"
    
    # Check for SSO cache
    $ssoCacheDir = "$env:USERPROFILE\.aws\sso\cache"
    
    if (-not (Test-Path $ssoCacheDir)) {
        Write-Error "AWS SSO cache directory not found"
        Write-Info "Please run: aws sso login --profile YOUR_PROFILE"
        return $false
    }
    
    # Find SSO cache file
    $cacheFiles = Get-ChildItem -Path $ssoCacheDir -Filter "*.json" | Sort-Object LastWriteTime -Descending
    
    if ($cacheFiles.Count -eq 0) {
        Write-Error "No SSO cache file found"
        Write-Info "Please run: aws sso login --profile YOUR_PROFILE"
        return $false
    }
    
    $cacheFile = $cacheFiles[0].FullName
    Write-Success "Found SSO cache file: $cacheFile"
    
    # Generate API key
    $apiKey = if ($env:PROXY_API_KEY) { $env:PROXY_API_KEY } else { 
        -join ((48..57) + (97..102) | Get-Random -Count 64 | ForEach-Object {[char]$_})
    }
    
    # Get AWS region
    $awsRegion = if ($env:AWS_REGION) { $env:AWS_REGION } else { "us-east-1" }
    
    # Get Profile ARN if set
    $profileArn = $env:PROFILE_ARN
    
    # Create .env file
    $envContent = @"
# Kiro Gateway - Headless Configuration (CLI DB)
# Generated: $(Get-Date)

# Server Configuration
PORT=8090
PROXY_API_KEY=$apiKey

# Authentication - CLI DB with auto-refresh
AUTH_TYPE=cli_db
CLI_DB_PATH=$cacheFile
AWS_REGION=$awsRegion

# Use CodeWhisperer endpoint with bearer token
Q_USE_SENDMESSAGE=false

# Profile ARN (required)
$(if ($profileArn) { "PROFILE_ARN=$profileArn" } else { "# PROFILE_ARN=arn:aws:codewhisperer:us-east-1:123456789012:profile/abc123" })

# Logging
LOG_LEVEL=info
DEBUG=false
"@
    
    $envContent | Out-File -FilePath ".env" -Encoding UTF8
    
    Write-Success "Created .env file with CLI DB configuration"
    Write-Info "Token will be automatically refreshed by AWS CLI"
    
    return $true
}

# Show menu
function Show-Menu {
    Write-Header "Kiro Gateway - Headless Authentication Setup"
    Write-Host ""
    Write-Host "Select authentication method:" -ForegroundColor White
    Write-Host ""
    Write-Host "1) IAM + SigV4 (Recommended for production)" -ForegroundColor White
    Write-Host "   - Uses AWS IAM credentials" -ForegroundColor Gray
    Write-Host "   - Automatic credential rotation" -ForegroundColor Gray
    Write-Host "   - Works with IAM roles, instance profiles, IRSA" -ForegroundColor Gray
    Write-Host ""
    Write-Host "2) Bearer Token (Simple, but requires manual renewal)" -ForegroundColor White
    Write-Host "   - Uses pre-generated bearer token" -ForegroundColor Gray
    Write-Host "   - Token expires in ~1 hour" -ForegroundColor Gray
    Write-Host "   - Good for testing" -ForegroundColor Gray
    Write-Host ""
    Write-Host "3) CLI DB (Auto-refresh, requires AWS CLI)" -ForegroundColor White
    Write-Host "   - Uses AWS SSO cache" -ForegroundColor Gray
    Write-Host "   - Automatic token refresh" -ForegroundColor Gray
    Write-Host "   - Good for development" -ForegroundColor Gray
    Write-Host ""
    Write-Host "4) Exit" -ForegroundColor White
    Write-Host ""
}

# Main script
function Main {
    # Check for required tools
    if (-not (Get-Command aws -ErrorAction SilentlyContinue)) {
        Write-Warning "AWS CLI not found. Some features may not work."
        Write-Info "Install from: https://aws.amazon.com/cli/"
    }
    
    # Main loop
    while ($true) {
        Clear-Host
        Show-Menu
        
        $choice = Read-Host "Enter choice [1-4]"
        
        switch ($choice) {
            "1" {
                if (Setup-SigV4) {
                    Write-Success "Setup complete!"
                    Write-Info "Start gateway with: .\kiro-gateway.exe"
                    break
                }
            }
            "2" {
                if (Setup-BearerToken) {
                    Write-Success "Setup complete!"
                    Write-Info "Start gateway with: .\kiro-gateway.exe"
                    break
                }
            }
            "3" {
                if (Setup-CLIDB) {
                    Write-Success "Setup complete!"
                    Write-Info "Start gateway with: .\kiro-gateway.exe"
                    break
                }
            }
            "4" {
                Write-Info "Exiting..."
                return
            }
            default {
                Write-Error "Invalid choice. Please select 1-4."
            }
        }
        
        if ($choice -ne "4") {
            Write-Host ""
            Read-Host "Press Enter to continue"
        }
    }
    
    # Test configuration
    Write-Host ""
    $testChoice = Read-Host "Would you like to test the configuration? (y/n)"
    
    if ($testChoice -eq "y" -or $testChoice -eq "Y") {
        Write-Header "Testing Configuration"
        
        # Build gateway if needed
        if (-not (Test-Path ".\kiro-gateway.exe")) {
            Write-Info "Building gateway..."
            go build -o kiro-gateway.exe .\cmd\kiro-gateway
            Write-Success "Gateway built successfully"
        }
        
        # Start gateway in background
        Write-Info "Starting gateway..."
        $gateway = Start-Process -FilePath ".\kiro-gateway.exe" -PassThru -WindowStyle Hidden
        
        # Wait for startup
        Start-Sleep -Seconds 5
        
        # Test health endpoint
        try {
            $health = Invoke-RestMethod -Uri "http://localhost:8090/health" -Method Get
            Write-Success "Gateway is running"
            
            # Get API key
            $apiKey = (Get-Content .env | Select-String "PROXY_API_KEY=").ToString().Split("=")[1]
            
            # Test chat endpoint
            Write-Info "Testing chat endpoint..."
            $body = @{
                model = "claude-3-5-sonnet-20241022-v2"
                messages = @(
                    @{
                        role = "user"
                        content = "Say hello"
                    }
                )
                max_tokens = 50
            } | ConvertTo-Json -Depth 10
            
            $response = Invoke-RestMethod -Uri "http://localhost:8090/v1/chat/completions" `
                -Method Post `
                -Headers @{
                    "Content-Type" = "application/json"
                    "Authorization" = "Bearer $apiKey"
                } `
                -Body $body
            
            if ($response.choices[0].message.content) {
                Write-Success "Chat endpoint working!"
                Write-Info "Response: $($response.choices[0].message.content)"
            }
            else {
                Write-Error "Chat endpoint test failed"
            }
        }
        catch {
            Write-Error "Gateway test failed: $_"
        }
        finally {
            # Stop gateway
            Write-Info "Stopping gateway..."
            Stop-Process -Id $gateway.Id -Force -ErrorAction SilentlyContinue
            Write-Success "Test complete"
        }
    }
    
    Write-Host ""
    Write-Success "All done! Your gateway is ready for headless operation."
    Write-Info "Configuration saved to: .env"
    Write-Info "Start gateway with: .\kiro-gateway.exe"
}

# Run main function
Main
