#!/bin/bash
# Build Docker Image for Kiro Gateway
# Supports both local and CI/CD builds

set -e

# Default values
VERSION="${VERSION:-dev}"
TAG="${TAG:-latest}"
PUSH="${PUSH:-false}"
REGISTRY="${REGISTRY:-}"
NO_BUILD_CACHE="${NO_BUILD_CACHE:-false}"

# Get build metadata
BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
COMMIT_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo "Building Kiro Gateway Docker Image"
echo "================================="
echo "Version:     $VERSION"
echo "Tag:         $TAG"
echo "Build Time:  $BUILD_TIME"
echo "Commit Hash: $COMMIT_HASH"
echo ""

# Construct image name
if [ -n "$REGISTRY" ]; then
    IMAGE_NAME="$REGISTRY/kiro-gateway:$TAG"
else
    IMAGE_NAME="kiro-gateway:$TAG"
fi

echo "Image Name: $IMAGE_NAME"
echo ""

# Build arguments
BUILD_ARGS=(
    --build-arg "VERSION=$VERSION"
    --build-arg "BUILD_TIME=$BUILD_TIME"
    --build-arg "COMMIT_HASH=$COMMIT_HASH"
)

if [ "$NO_BUILD_CACHE" = "true" ]; then
    BUILD_ARGS+=(--no-cache)
fi

# Build command
echo "Executing: docker build ${BUILD_ARGS[*]} -t $IMAGE_NAME ."
echo ""

# Execute build
docker build "${BUILD_ARGS[@]}" -t "$IMAGE_NAME" .

echo ""
echo "✅ Build successful!"
echo ""

# Show image info
echo "Image Information:"
docker images "$IMAGE_NAME" --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}"
echo ""

# Tag with version if different from tag
if [ "$TAG" != "$VERSION" ] && [ "$VERSION" != "dev" ]; then
    if [ -n "$REGISTRY" ]; then
        VERSION_IMAGE_NAME="$REGISTRY/kiro-gateway:$VERSION"
    else
        VERSION_IMAGE_NAME="kiro-gateway:$VERSION"
    fi
    
    echo "Tagging as version: $VERSION_IMAGE_NAME"
    docker tag "$IMAGE_NAME" "$VERSION_IMAGE_NAME"
fi

# Push if requested
if [ "$PUSH" = "true" ]; then
    echo ""
    echo "Pushing image to registry..."
    
    docker push "$IMAGE_NAME"
    
    # Push version tag if created
    if [ "$TAG" != "$VERSION" ] && [ "$VERSION" != "dev" ]; then
        docker push "$VERSION_IMAGE_NAME"
    fi
    
    echo "✅ Push successful!"
fi

echo ""
echo "Next Steps:"
echo "  1. Test locally:  docker run -p 8080:8080 --env-file .env $IMAGE_NAME"
echo "  2. Use compose:   docker-compose up -d"
echo "  3. Check health:  curl http://localhost:8080/health"
echo ""
