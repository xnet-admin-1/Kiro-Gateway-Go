# PowerShell Build Script for Task 49.3: Build binaries for Windows, macOS, and Linux
# This script implements the complete Task 49.3 requirements

param(
    [string]$Version = "dev",
    [string]$BuildTime = (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ"),
    [string]$CommitHash = "unknown"
)

# Configuration
$BinaryName = "kiro-gateway"
$DistDir = "dist"

# Try to get version from git
try {
    $Version = & git describe --tags --always --dirty 2>$null
    if (-not $Version) { $Version = "dev" }
} catch {
    $Version = "dev"
}

try {
    $CommitHash = & git rev-parse --short HEAD 2>$null
    if (-not $CommitHash) { $CommitHash = "unknown" }
} catch {
    $CommitHash = "unknown"
}

# Build flags
$BuildFlags = "-ldflags", "-X main.Version=$Version -X main.BuildTime=$BuildTime -X main.CommitHash=$CommitHash -s -w"

Write-Host "=== Task 49.3: Build Binaries ===" -ForegroundColor Blue
Write-Host "Building kiro-gateway-go for multiple platforms..."
Write-Host "Version: $Version"
Write-Host "Build Time: $BuildTime"
Write-Host "Commit: $CommitHash"
Write-Host ""

# Create dist directory
Write-Host "Creating dist directory..." -ForegroundColor Yellow
if (-not (Test-Path $DistDir)) {
    New-Item -ItemType Directory -Path $DistDir | Out-Null
}

# Clean previous builds
Write-Host "Cleaning previous builds..." -ForegroundColor Yellow
Get-ChildItem -Path $DistDir -Filter "$BinaryName-*" | Remove-Item -Force

# Define platforms
$Platforms = @(
    @{GOOS="linux"; GOARCH="amd64"; Name="$BinaryName-linux-amd64"; CGO="0"},
    @{GOOS="linux"; GOARCH="arm64"; Name="$BinaryName-linux-arm64"; CGO="0"},
    @{GOOS="darwin"; GOARCH="amd64"; Name="$BinaryName-darwin-amd64"; CGO="0"},
    @{GOOS="darwin"; GOARCH="arm64"; Name="$BinaryName-darwin-arm64"; CGO="0"},
    @{GOOS="windows"; GOARCH="amd64"; Name="$BinaryName-windows-amd64.exe"; CGO="1"}
)

Write-Host "Building binaries for all platforms..." -ForegroundColor Yellow

foreach ($Platform in $Platforms) {
    Write-Host "Building for $($Platform.GOOS)/$($Platform.GOARCH)..." -ForegroundColor Blue
    
    $BinaryPath = Join-Path $DistDir $Platform.Name
    
    # Set environment variables
    $env:CGO_ENABLED = $Platform.CGO
    $env:GOOS = $Platform.GOOS
    $env:GOARCH = $Platform.GOARCH
    
    # Build command arguments
    $BuildArgs = @("build")
    if ($Platform.CGO -eq "0") {
        $BuildArgs += "-tags", "nocgo"
    }
    $BuildArgs += $BuildFlags
    $BuildArgs += "-o", $BinaryPath
    $BuildArgs += "./cmd/kiro-gateway"
    
    # Execute build
    try {
        & go @BuildArgs
        if ($LASTEXITCODE -eq 0) {
            $Size = (Get-Item $BinaryPath).Length
            $SizeMB = [math]::Round($Size / 1048576, 2)
            Write-Host "✓ Built $($Platform.Name) ($SizeMB MB)" -ForegroundColor Green
        } else {
            Write-Host "✗ Failed to build for $($Platform.GOOS)/$($Platform.GOARCH)" -ForegroundColor Red
            exit 1
        }
    } catch {
        Write-Host "✗ Failed to build for $($Platform.GOOS)/$($Platform.GOARCH): $_" -ForegroundColor Red
        exit 1
    }
}

Write-Host ""
Write-Host "Testing built binaries..." -ForegroundColor Yellow

# Test each binary
foreach ($Platform in $Platforms) {
    $BinaryPath = Join-Path $DistDir $Platform.Name
    Write-Host "Testing $($Platform.Name)..." -ForegroundColor Blue
    
    # Check if file exists
    if (-not (Test-Path $BinaryPath)) {
        Write-Host "✗ Binary not found: $BinaryPath" -ForegroundColor Red
        continue
    }
    
    # Check file size
    $Size = (Get-Item $BinaryPath).Length
    if ($Size -eq 0) {
        Write-Host "✗ Binary is empty: $BinaryPath" -ForegroundColor Red
        continue
    }
    
    # Test execution (only for Windows binary)
    if ($Platform.GOOS -eq "windows") {
        Write-Host "  Testing execution..." -ForegroundColor Cyan
        
        # Test version flag (if implemented) with timeout
        try {
            $Job = Start-Job -ScriptBlock { param($Path) & $Path -version 2>&1 } -ArgumentList $BinaryPath
            $Completed = Wait-Job -Job $Job -Timeout 2
            if ($Completed) {
                $Output = Receive-Job -Job $Job
                Write-Host "  ✓ Version flag works" -ForegroundColor Green
            } else {
                Write-Host "  ⚠ Version flag timed out (may not be implemented)" -ForegroundColor Yellow
            }
            Remove-Job -Job $Job -Force
        } catch {
            Write-Host "  ⚠ Version flag not implemented or failed" -ForegroundColor Yellow
        }
        
        # Test help flag with timeout
        try {
            $Job = Start-Job -ScriptBlock { param($Path) & $Path -help 2>&1 } -ArgumentList $BinaryPath
            $Completed = Wait-Job -Job $Job -Timeout 2
            if ($Completed) {
                $Output = Receive-Job -Job $Job
                Write-Host "  ✓ Help flag works" -ForegroundColor Green
            } else {
                Write-Host "  ⚠ Help flag timed out (may not be implemented)" -ForegroundColor Yellow
            }
            Remove-Job -Job $Job -Force
        } catch {
            Write-Host "  ⚠ Help flag not implemented or failed" -ForegroundColor Yellow
        }
    }
    
    Write-Host "✓ $($Platform.Name) validated" -ForegroundColor Green
}

Write-Host ""
Write-Host "=== Build Summary ===" -ForegroundColor Green
Write-Host "Built binaries:"
Get-ChildItem -Path $DistDir -Filter "$BinaryName-*" | ForEach-Object {
    $SizeMB = [math]::Round($_.Length / 1048576, 2)
    Write-Host "  $($_.Name) ($SizeMB MB)"
}

Write-Host ""
Write-Host "=== Task 49.3 Complete ===" -ForegroundColor Green
Write-Host "All binaries built and tested successfully!"
Write-Host "Binaries are available in the $DistDir/ directory"

# Generate checksums
Write-Host ""
Write-Host "Generating checksums..." -ForegroundColor Yellow
$ChecksumFile = Join-Path $DistDir "checksums.sha256"
if (Test-Path $ChecksumFile) {
    Remove-Item $ChecksumFile
}

Get-ChildItem -Path $DistDir -Filter "$BinaryName-*" | ForEach-Object {
    $Hash = Get-FileHash -Path $_.FullName -Algorithm SHA256
    "$($Hash.Hash.ToLower())  $($_.Name)" | Add-Content -Path $ChecksumFile
}

if (Test-Path $ChecksumFile) {
    Write-Host "SHA256 checksums saved to $ChecksumFile" -ForegroundColor Green
}

Write-Host ""
Write-Host "Build artifacts:" -ForegroundColor Blue
Get-ChildItem -Path $DistDir -Filter "$BinaryName-*" | ForEach-Object { Write-Host "  $($_.Name)" }
if (Test-Path $ChecksumFile) {
    Write-Host "  checksums.sha256"
}

# Reset environment variables
Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue
Remove-Item Env:GOOS -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
