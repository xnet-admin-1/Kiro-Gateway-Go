#!/bin/bash
# Build script for TOTP Manager desktop application

set -e

PLATFORM="${1:-linux}"
ARCH="${2:-amd64}"
RELEASE="${3:-false}"

echo "Building Kiro Gateway TOTP Manager..."
echo ""

# Navigate to totp-manager directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
TOTP_MANAGER_DIR="$PROJECT_ROOT/cmd/totp-manager"

if [ ! -d "$TOTP_MANAGER_DIR" ]; then
    echo "Error: TOTP Manager directory not found at $TOTP_MANAGER_DIR"
    exit 1
fi

cd "$TOTP_MANAGER_DIR"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed or not in PATH"
    exit 1
fi

GO_VERSION=$(go version)
echo "Using $GO_VERSION"

# Download dependencies
echo "Downloading dependencies..."
go mod download

# Set build variables
export GOOS="$PLATFORM"
export GOARCH="$ARCH"

OUTPUT_NAME="totp-manager"
if [ "$PLATFORM" = "windows" ]; then
    OUTPUT_NAME="${OUTPUT_NAME}.exe"
fi

OUTPUT_PATH="$PROJECT_ROOT/dist/$OUTPUT_NAME"

# Create dist directory
mkdir -p "$PROJECT_ROOT/dist"

# Build flags
BUILD_FLAGS=""
if [ "$RELEASE" = "true" ]; then
    BUILD_FLAGS="-ldflags=-s -w"
    echo "Building release version (optimized)..."
else
    echo "Building debug version..."
fi

# Build the application
echo "Building for $PLATFORM/$ARCH..."
go build $BUILD_FLAGS -o "$OUTPUT_PATH" .

# Get file size
FILE_SIZE=$(du -h "$OUTPUT_PATH" | cut -f1)
echo ""
echo "Build successful!"
echo "Output: $OUTPUT_PATH"
echo "Size: $FILE_SIZE"
echo ""

# Platform-specific instructions
if [ "$PLATFORM" = "windows" ]; then
    echo "To run the application:"
    echo "  ./dist/totp-manager.exe"
else
    echo "To run the application:"
    echo "  ./dist/totp-manager"
    chmod +x "$OUTPUT_PATH"
fi

echo ""
echo "First-time setup:"
echo "  1. Launch the application"
echo "  2. Go to Configuration tab"
echo "  3. Enter gateway URL (http://localhost:8080)"
echo "  4. Enter admin API key"
echo "  5. Click 'Save Configuration'"
echo ""

# Return to original directory
cd "$PROJECT_ROOT"
