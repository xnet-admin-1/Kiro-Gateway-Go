#!/usr/bin/env pwsh
# Build script for TOTP Manager desktop application

param(
    [string]$Platform = "windows",
    [string]$Arch = "amd64",
    [switch]$Release
)

$ErrorActionPreference = "Stop"

Write-Host "Building Kiro Gateway TOTP Manager..." -ForegroundColor Cyan
Write-Host ""

# Navigate to totp-manager directory
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Split-Path -Parent $scriptDir
$totpManagerDir = Join-Path $projectRoot "cmd\totp-manager"

if (-not (Test-Path $totpManagerDir)) {
    Write-Host "Error: TOTP Manager directory not found at $totpManagerDir" -ForegroundColor Red
    exit 1
}

Set-Location $totpManagerDir

# Check if Go is installed
try {
    $goVersion = go version
    Write-Host "Using $goVersion" -ForegroundColor Green
} catch {
    Write-Host "Error: Go is not installed or not in PATH" -ForegroundColor Red
    exit 1
}

# Download dependencies
Write-Host "Downloading dependencies..." -ForegroundColor Yellow
go mod download
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to download dependencies" -ForegroundColor Red
    exit 1
}

# Set build variables
$env:GOOS = $Platform
$env:GOARCH = $Arch

$outputName = "totp-keys"
if ($Platform -eq "windows") {
    $outputName += ".exe"
}

$outputPath = Join-Path $projectRoot "dist" $outputName

# Create dist directory
$distDir = Join-Path $projectRoot "dist"
if (-not (Test-Path $distDir)) {
    New-Item -ItemType Directory -Path $distDir | Out-Null
}

# Build flags
$buildFlags = @()
if ($Release) {
    # -H windowsgui hides the console window on Windows
    $buildFlags += "-ldflags=`"-s -w -H windowsgui`""
    Write-Host "Building release version (optimized, no console)..." -ForegroundColor Yellow
} else {
    Write-Host "Building debug version..." -ForegroundColor Yellow
}

# Build the application
Write-Host "Building for $Platform/$Arch..." -ForegroundColor Yellow
$buildCmd = "go build $($buildFlags -join ' ') -o `"$outputPath`" ."
Invoke-Expression $buildCmd

if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed" -ForegroundColor Red
    exit 1
}

# Get file size
$fileSize = (Get-Item $outputPath).Length / 1MB
Write-Host ""
Write-Host "Build successful!" -ForegroundColor Green
Write-Host "Output: $outputPath" -ForegroundColor Cyan
Write-Host "Size: $([math]::Round($fileSize, 2)) MB" -ForegroundColor Cyan
Write-Host ""

# Platform-specific instructions
if ($Platform -eq "windows") {
    Write-Host "To run the application:" -ForegroundColor Yellow
    Write-Host "  .\dist\totp-keys.exe" -ForegroundColor Gray
} else {
    Write-Host "To run the application:" -ForegroundColor Yellow
    Write-Host "  ./dist/totp-keys" -ForegroundColor Gray
}

Write-Host ""
Write-Host "First-time setup:" -ForegroundColor Yellow
Write-Host "  1. Launch the application" -ForegroundColor Gray
Write-Host "  2. Go to Configuration tab" -ForegroundColor Gray
Write-Host "  3. Enter gateway URL (http://localhost:8080)" -ForegroundColor Gray
Write-Host "  4. Enter admin API key" -ForegroundColor Gray
Write-Host "  5. Click 'Save Configuration'" -ForegroundColor Gray
Write-Host ""

# Return to original directory
Set-Location $projectRoot
