# Setup Docker Secrets for Kiro Gateway
# This script helps configure Docker secrets for sensitive credentials

param(
    [switch]$Generate,
    [switch]$Remove,
    [switch]$List
)

$ErrorActionPreference = "Stop"

function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

function Generate-AdminKey {
    # Generate cryptographically secure admin API key
    $bytes = New-Object byte[] 32
    $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    $rng.GetBytes($bytes)
    $key = "kiro-" + [Convert]::ToBase64String($bytes).Replace("+", "-").Replace("/", "_").TrimEnd('=')
    return $key
}

function Create-DockerSecret {
    param(
        [string]$SecretName,
        [string]$SecretValue
    )
    
    try {
        # Check if secret already exists
        $existing = docker secret ls --filter "name=$SecretName" --format "{{.Name}}" 2>$null
        if ($existing -eq $SecretName) {
            Write-ColorOutput "⚠️  Secret '$SecretName' already exists. Remove it first with -Remove flag." "Yellow"
            return $false
        }
        
        # Create secret
        $SecretValue | docker secret create $SecretName - 2>&1 | Out-Null
        if ($LASTEXITCODE -eq 0) {
            Write-ColorOutput "✅ Created secret: $SecretName" "Green"
            return $true
        } else {
            Write-ColorOutput "❌ Failed to create secret: $SecretName" "Red"
            return $false
        }
    } catch {
        Write-ColorOutput "❌ Error creating secret: $_" "Red"
        return $false
    }
}

function Remove-DockerSecret {
    param(
        [string]$SecretName
    )
    
    try {
        docker secret rm $SecretName 2>&1 | Out-Null
        if ($LASTEXITCODE -eq 0) {
            Write-ColorOutput "✅ Removed secret: $SecretName" "Green"
            return $true
        } else {
            Write-ColorOutput "⚠️  Secret '$SecretName' not found or already removed" "Yellow"
            return $false
        }
    } catch {
        Write-ColorOutput "❌ Error removing secret: $_" "Red"
        return $false
    }
}

function List-DockerSecrets {
    Write-ColorOutput "`n=== Docker Secrets ===" "Cyan"
    docker secret ls
}

# Main script logic
if ($List) {
    List-DockerSecrets
    exit 0
}

if ($Remove) {
    Write-ColorOutput "`n🗑️  Removing Docker secrets..." "Yellow"
    Remove-DockerSecret "admin_api_key"
    Remove-DockerSecret "sso_password"
    Remove-DockerSecret "mfa_totp_secret"
    Remove-DockerSecret "sso_client_secret"
    Write-ColorOutput "`n✅ Secrets removed" "Green"
    exit 0
}

if ($Generate) {
    Write-ColorOutput "`n🔐 Setting up Docker secrets for Kiro Gateway..." "Cyan"
    Write-ColorOutput "This will create secure Docker secrets for sensitive credentials.`n" "White"
    
    # Check if Docker is running
    try {
        docker info 2>&1 | Out-Null
        if ($LASTEXITCODE -ne 0) {
            Write-ColorOutput "❌ Docker is not running. Please start Docker Desktop." "Red"
            exit 1
        }
    } catch {
        Write-ColorOutput "❌ Docker is not available. Please install Docker Desktop." "Red"
        exit 1
    }
    
    # Check if swarm mode is enabled (required for secrets)
    $swarmStatus = docker info --format "{{.Swarm.LocalNodeState}}" 2>$null
    if ($swarmStatus -ne "active") {
        Write-ColorOutput "⚠️  Docker Swarm is not active. Initializing..." "Yellow"
        docker swarm init 2>&1 | Out-Null
        if ($LASTEXITCODE -ne 0) {
            Write-ColorOutput "❌ Failed to initialize Docker Swarm" "Red"
            exit 1
        }
        Write-ColorOutput "✅ Docker Swarm initialized" "Green"
    }
    
    # Generate admin API key
    Write-ColorOutput "`n📝 Generating admin API key..." "Cyan"
    $adminKey = Generate-AdminKey
    Write-ColorOutput "Generated key: $adminKey" "White"
    Write-ColorOutput "⚠️  SAVE THIS KEY SECURELY - it won't be shown again!" "Yellow"
    
    # Prompt for SSO credentials
    Write-ColorOutput "`n📝 Enter SSO credentials (or press Enter to skip):" "Cyan"
    $ssoPassword = Read-Host "SSO Password" -AsSecureString
    $ssoPasswordPlain = [Runtime.InteropServices.Marshal]::PtrToStringAuto(
        [Runtime.InteropServices.Marshal]::SecureStringToBSTR($ssoPassword)
    )
    
    $mfaSecret = Read-Host "MFA TOTP Secret (base32)"
    
    # Prompt for OIDC credentials (optional, for headless OIDC mode)
    Write-ColorOutput "`n📝 Enter OIDC client secret (optional, for headless OIDC mode):" "Cyan"
    Write-ColorOutput "   Note: OIDC Client ID is not sensitive and should be in .env file" "Gray"
    Write-ColorOutput "   Leave blank if using browser automation instead" "Gray"
    $oidcClientSecret = Read-Host "OIDC Client Secret" -AsSecureString
    $oidcClientSecretPlain = [Runtime.InteropServices.Marshal]::PtrToStringAuto(
        [Runtime.InteropServices.Marshal]::SecureStringToBSTR($oidcClientSecret)
    )
    
    # Create secrets
    Write-ColorOutput "`n🔐 Creating Docker secrets..." "Cyan"
    
    $success = $true
    $success = $success -and (Create-DockerSecret "admin_api_key" $adminKey)
    
    if ($ssoPasswordPlain) {
        $success = $success -and (Create-DockerSecret "sso_password" $ssoPasswordPlain)
    }
    
    if ($mfaSecret) {
        $success = $success -and (Create-DockerSecret "mfa_totp_secret" $mfaSecret)
    }
    
    if ($oidcClientSecretPlain) {
        $success = $success -and (Create-DockerSecret "sso_client_secret" $oidcClientSecretPlain)
    }
    
    if ($success) {
        Write-ColorOutput "`n✅ Docker secrets configured successfully!" "Green"
        Write-ColorOutput "`nNext steps:" "Cyan"
        Write-ColorOutput "1. Update docker-compose.yml to use secrets (uncomment secrets section)" "White"
        Write-ColorOutput "2. Update application code to read from /run/secrets/" "White"
        Write-ColorOutput "3. Remove credentials from .env file" "White"
        Write-ColorOutput "4. Deploy with: docker stack deploy -c docker-compose.yml kiro" "White"
        
        Write-ColorOutput "`n📋 Admin API Key (save this):" "Yellow"
        Write-ColorOutput $adminKey "White"
    } else {
        Write-ColorOutput "`n❌ Some secrets failed to create" "Red"
        exit 1
    }
} else {
    Write-ColorOutput "Usage:" "Cyan"
    Write-ColorOutput "  .\setup-docker-secrets.ps1 -Generate  # Generate and create secrets" "White"
    Write-ColorOutput "  .\setup-docker-secrets.ps1 -List      # List existing secrets" "White"
    Write-ColorOutput "  .\setup-docker-secrets.ps1 -Remove    # Remove all secrets" "White"
}
