# Build Docker Image for Kiro Gateway
# Supports both local and CI/CD builds

param(
    [string]$Version = "dev",
    [string]$Tag = "latest",
    [switch]$Push = $false,
    [string]$Registry = "",
    [switch]$NoBuildCache = $false
)

# Get build metadata
$BuildTime = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$CommitHash = if (Get-Command git -ErrorAction SilentlyContinue) {
    git rev-parse --short HEAD 2>$null
} else {
    "unknown"
}

Write-Host "Building Kiro Gateway Docker Image" -ForegroundColor Cyan
Write-Host "=================================" -ForegroundColor Cyan
Write-Host "Version:     $Version" -ForegroundColor Green
Write-Host "Tag:         $Tag" -ForegroundColor Green
Write-Host "Build Time:  $BuildTime" -ForegroundColor Green
Write-Host "Commit Hash: $CommitHash" -ForegroundColor Green
Write-Host ""

# Construct image name
$ImageName = if ($Registry) {
    "$Registry/kiro-gateway:$Tag"
} else {
    "kiro-gateway:$Tag"
}

Write-Host "Image Name: $ImageName" -ForegroundColor Yellow
Write-Host ""

# Build arguments
$BuildArgs = @(
    "--build-arg", "VERSION=$Version",
    "--build-arg", "BUILD_TIME=$BuildTime",
    "--build-arg", "COMMIT_HASH=$CommitHash"
)

if ($NoBuildCache) {
    $BuildArgs += "--no-cache"
}

# Build command
$BuildCmd = @("docker", "build") + $BuildArgs + @("-t", $ImageName, ".")

Write-Host "Executing: $($BuildCmd -join ' ')" -ForegroundColor Gray
Write-Host ""

# Execute build
try {
    & $BuildCmd[0] $BuildCmd[1..($BuildCmd.Length-1)]
    
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Build failed with exit code $LASTEXITCODE" -ForegroundColor Red
        exit $LASTEXITCODE
    }
    
    Write-Host ""
    Write-Host "✅ Build successful!" -ForegroundColor Green
    Write-Host ""
    
    # Show image info
    Write-Host "Image Information:" -ForegroundColor Cyan
    docker images $ImageName --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}"
    Write-Host ""
    
    # Tag with version if different from tag
    if ($Tag -ne $Version -and $Version -ne "dev") {
        $VersionImageName = if ($Registry) {
            "$Registry/kiro-gateway:$Version"
        } else {
            "kiro-gateway:$Version"
        }
        
        Write-Host "Tagging as version: $VersionImageName" -ForegroundColor Yellow
        docker tag $ImageName $VersionImageName
    }
    
    # Push if requested
    if ($Push) {
        Write-Host ""
        Write-Host "Pushing image to registry..." -ForegroundColor Yellow
        
        docker push $ImageName
        
        if ($LASTEXITCODE -ne 0) {
            Write-Host "Push failed with exit code $LASTEXITCODE" -ForegroundColor Red
            exit $LASTEXITCODE
        }
        
        # Push version tag if created
        if ($Tag -ne $Version -and $Version -ne "dev") {
            docker push $VersionImageName
        }
        
        Write-Host "✅ Push successful!" -ForegroundColor Green
    }
    
    Write-Host ""
    Write-Host "Next Steps:" -ForegroundColor Cyan
    Write-Host "  1. Test locally:  docker run -p 8080:8080 --env-file .env $ImageName" -ForegroundColor White
    Write-Host "  2. Use compose:   docker-compose up -d" -ForegroundColor White
    Write-Host "  3. Check health:  curl http://localhost:8080/health" -ForegroundColor White
    Write-Host ""
    
} catch {
    Write-Host "Build failed: $_" -ForegroundColor Red
    exit 1
}
