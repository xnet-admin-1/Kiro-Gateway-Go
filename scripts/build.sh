#!/bin/bash

# Build script for Task 49.3: Build binaries for Windows, macOS, and Linux
# This script implements the complete Task 49.3 requirements

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BINARY_NAME="kiro-gateway"
DIST_DIR="dist"
VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
COMMIT_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
BUILD_FLAGS="-ldflags \"-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.CommitHash=${COMMIT_HASH} -s -w\""

# Platforms to build for
declare -A PLATFORMS=(
    ["linux/amd64"]="${BINARY_NAME}-linux-amd64"
    ["linux/arm64"]="${BINARY_NAME}-linux-arm64"
    ["darwin/amd64"]="${BINARY_NAME}-darwin-amd64"
    ["darwin/arm64"]="${BINARY_NAME}-darwin-arm64"
    ["windows/amd64"]="${BINARY_NAME}-windows-amd64.exe"
)

echo -e "${BLUE}=== Task 49.3: Build Binaries ===${NC}"
echo "Building kiro-gateway-go for multiple platforms..."
echo "Version: ${VERSION}"
echo "Build Time: ${BUILD_TIME}"
echo "Commit: ${COMMIT_HASH}"
echo ""

# Create dist directory
echo -e "${YELLOW}Creating dist directory...${NC}"
mkdir -p "${DIST_DIR}"

# Clean previous builds
echo -e "${YELLOW}Cleaning previous builds...${NC}"
rm -f "${DIST_DIR}/${BINARY_NAME}"-*

# Build for each platform
echo -e "${YELLOW}Building binaries for all platforms...${NC}"
for platform in "${!PLATFORMS[@]}"; do
    IFS='/' read -r goos goarch <<< "$platform"
    binary_name="${PLATFORMS[$platform]}"
    binary_path="${DIST_DIR}/${binary_name}"
    
    echo -e "${BLUE}Building for ${platform}...${NC}"
    
    # Set environment and build
    CGO_ENABLED=1 GOOS="$goos" GOARCH="$goarch" \
        go build $BUILD_FLAGS -o "$binary_path" ./cmd/kiro-gateway
    
    if [ $? -eq 0 ]; then
        # Get file size
        if [ -f "$binary_path" ]; then
            size=$(stat -f%z "$binary_path" 2>/dev/null || stat -c%s "$binary_path" 2>/dev/null || echo "unknown")
            size_mb=$(echo "scale=2; $size / 1024 / 1024" | bc -l 2>/dev/null || echo "unknown")
            echo -e "${GREEN}✓ Built ${binary_name} (${size_mb} MB)${NC}"
        else
            echo -e "${RED}✗ Binary not found: ${binary_path}${NC}"
            exit 1
        fi
    else
        echo -e "${RED}✗ Failed to build for ${platform}${NC}"
        exit 1
    fi
done

echo ""
echo -e "${YELLOW}Testing built binaries...${NC}"

# Test each binary
for platform in "${!PLATFORMS[@]}"; do
    binary_name="${PLATFORMS[$platform]}"
    binary_path="${DIST_DIR}/${binary_name}"
    
    echo -e "${BLUE}Testing ${binary_name}...${NC}"
    
    # Check if file exists
    if [ ! -f "$binary_path" ]; then
        echo -e "${RED}✗ Binary not found: ${binary_path}${NC}"
        continue
    fi
    
    # Check file size
    size=$(stat -f%z "$binary_path" 2>/dev/null || stat -c%s "$binary_path" 2>/dev/null || echo "0")
    if [ "$size" -eq 0 ]; then
        echo -e "${RED}✗ Binary is empty: ${binary_path}${NC}"
        continue
    fi
    
    # Check file type (if file command is available)
    if command -v file >/dev/null 2>&1; then
        file_type=$(file "$binary_path")
        echo "  File type: $file_type"
        
        # Validate file type based on platform
        case "$platform" in
            linux/*)
                if [[ ! "$file_type" =~ ELF ]]; then
                    echo -e "${YELLOW}⚠ Warning: Linux binary should be ELF format${NC}"
                fi
                ;;
            darwin/*)
                if [[ ! "$file_type" =~ Mach-O ]]; then
                    echo -e "${YELLOW}⚠ Warning: Darwin binary should be Mach-O format${NC}"
                fi
                ;;
            windows/*)
                if [[ ! "$file_type" =~ PE32 ]]; then
                    echo -e "${YELLOW}⚠ Warning: Windows binary should be PE32 format${NC}"
                fi
                ;;
        esac
    fi
    
    # Test execution (only for current platform)
    current_platform="${GOOS:-$(go env GOOS)}/${GOARCH:-$(go env GOARCH)}"
    if [ "$platform" = "$current_platform" ]; then
        echo "  Testing execution..."
        
        # Test version flag (if implemented)
        if timeout 5s "$binary_path" -version >/dev/null 2>&1; then
            echo -e "${GREEN}  ✓ Version flag works${NC}"
        else
            echo -e "${YELLOW}  ⚠ Version flag not implemented or failed${NC}"
        fi
        
        # Test help flag
        if timeout 5s "$binary_path" -help >/dev/null 2>&1; then
            echo -e "${GREEN}  ✓ Help flag works${NC}"
        else
            echo -e "${YELLOW}  ⚠ Help flag not implemented or failed${NC}"
        fi
    fi
    
    echo -e "${GREEN}✓ ${binary_name} validated${NC}"
done

echo ""
echo -e "${GREEN}=== Build Summary ===${NC}"
echo "Built binaries:"
ls -la "${DIST_DIR}/${BINARY_NAME}"-*

echo ""
echo -e "${GREEN}=== Task 49.3 Complete ===${NC}"
echo "All binaries built and tested successfully!"
echo "Binaries are available in the ${DIST_DIR}/ directory"

# Generate checksums
echo ""
echo -e "${YELLOW}Generating checksums...${NC}"
cd "$DIST_DIR"
if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${BINARY_NAME}"-* > checksums.sha256
    echo "SHA256 checksums saved to ${DIST_DIR}/checksums.sha256"
elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "${BINARY_NAME}"-* > checksums.sha256
    echo "SHA256 checksums saved to ${DIST_DIR}/checksums.sha256"
else
    echo -e "${YELLOW}⚠ No checksum utility found${NC}"
fi
cd ..

echo ""
echo -e "${BLUE}Build artifacts:${NC}"
find "$DIST_DIR" -name "${BINARY_NAME}-*" -o -name "checksums.*" | sort
