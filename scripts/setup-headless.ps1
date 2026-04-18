# Kiro Gateway - Headless Mode Setup Script
# This script helps configure headless authentication

param(
    [Parameter(Mandatory=$true)]
    [string]$StartURL,
    
    [Parameter(Mandatory=$true)]
    [string]$AccountID,
    
    [Parameter(Mandatory=$true)]
    [string]$RoleName,
    
    [Parameter(Mandatory=$false)]
    [string]$Region = "us-east-1",
    
    [Parameter(Mandatory=$false)]
    [string]$EnvFile = ".env"
)

Write-Host "╔════════════════════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║          Kiro Gateway - Headless Mode Setup                           ║" -ForegroundColor Cyan
Write-Host "╚════════════════════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""

# Validate inputs
if (-not $StartURL.StartsWith("https://")) {
    Write-Host "❌ Error: Start URL must begin with https://" -ForegroundColor Red
    exit 1
}

if ($AccountID -notmatch '^\d{12}$') {
    Write-Host "❌ Error: Account ID must be a 12-digit number" -ForegroundColor Red
    exit 1
}

Write-Host "Configuration:" -ForegroundColor Yellow
Write-Host "  Start URL:  $StartURL" -ForegroundColor White
Write-Host "  Account ID: $AccountID" -ForegroundColor White
Write-Host "  Role Name:  $RoleName" -ForegroundColor White
Write-Host "  Region:     $Region" -ForegroundColor White
Write-Host "  Env File:   $EnvFile" -ForegroundColor White
Write-Host ""

# Check if .env file exists
if (Test-Path $EnvFile) {
    Write-Host "⚠️  Warning: $EnvFile already exists" -ForegroundColor Yellow
    $response = Read-Host "Do you want to update it? (y/n)"
    if ($response -ne "y") {
        Write-Host "Setup cancelled" -ForegroundColor Yellow
        exit 0
    }
}

# Create or update .env file
Write-Host "📝 Creating configuration..." -ForegroundColor Cyan

$envContent = @"
# Kiro Gateway - Headless Mode Configuration
# Generated: $(Get-Date -Format "yyyy-MM-dd HH:mm:ss")

# ============================================================================
# HEADLESS MODE
# ============================================================================
HEADLESS_MODE=true

# SSO Configuration
SSO_START_URL=$StartURL
SSO_REGION=$Region
AWS_SSO_ACCOUNT_ID=$AccountID
AWS_SSO_ROLE_NAME=$RoleName

# ============================================================================
# Q DEVELOPER CONFIGURATION
# ============================================================================
Q_USE_SENDMESSAGE=true
AMAZON_Q_SIGV4=true
AWS_REGION=$Region

# ============================================================================
# SERVER CONFIGURATION
# ============================================================================
PORT=8090

# ============================================================================
# TIMEOUT CONFIGURATION
# ============================================================================
FIRST_TOKEN_TIMEOUT=30s
MULTIMODAL_FIRST_TOKEN_TIMEOUT=90s

# ============================================================================
# LOGGING
# ============================================================================
LOG_LEVEL=INFO
DEBUG=false
"@

# Write to file
$envContent | Out-File -FilePath $EnvFile -Encoding UTF8

Write-Host "✅ Configuration saved to $EnvFile" -ForegroundColor Green
Write-Host ""

# Display next steps
Write-Host "╔════════════════════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║                          Next Steps                                    ║" -ForegroundColor Cyan
Write-Host "╚════════════════════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""
Write-Host "1. Build the gateway:" -ForegroundColor Yellow
Write-Host "   go build -o kiro-gateway.exe ./cmd/kiro-gateway" -ForegroundColor White
Write-Host ""
Write-Host "2. Start the gateway:" -ForegroundColor Yellow
Write-Host "   .\kiro-gateway.exe" -ForegroundColor White
Write-Host ""
Write-Host "3. Follow the on-screen instructions to authorize the device" -ForegroundColor Yellow
Write-Host "   - Visit the provided URL" -ForegroundColor White
Write-Host "   - Enter the displayed code" -ForegroundColor White
Write-Host "   - Authorize the device" -ForegroundColor White
Write-Host ""
Write-Host "4. Once authorized, tokens will be cached for future runs" -ForegroundColor Yellow
Write-Host ""
Write-Host "📚 For more information, see:" -ForegroundColor Cyan
Write-Host "   docs/HEADLESS_AUTHENTICATION.md" -ForegroundColor White
Write-Host ""
