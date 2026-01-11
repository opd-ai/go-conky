#!/bin/bash
# Build script for conky-go
# Usage: ./scripts/build.sh [debug|release]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BUILD_DIR="${PROJECT_ROOT}/build"
BINARY_NAME="conky-go"

cd "$PROJECT_ROOT"

# Parse build type argument
BUILD_TYPE="${1:-debug}"

echo "Building conky-go (${BUILD_TYPE})..."

# Create build directory
mkdir -p "$BUILD_DIR"

# Get version from git or use dev
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS="-X main.Version=${VERSION}"

case "$BUILD_TYPE" in
    debug)
        echo "Building debug binary..."
        go build -ldflags "$LDFLAGS" -o "${BUILD_DIR}/${BINARY_NAME}" ./cmd/conky-go
        ;;
    release)
        echo "Building release binary with optimizations..."
        CGO_ENABLED=0 go build -ldflags "$LDFLAGS -s -w" -trimpath -o "${BUILD_DIR}/${BINARY_NAME}" ./cmd/conky-go
        ;;
    *)
        echo "Unknown build type: $BUILD_TYPE"
        echo "Usage: $0 [debug|release]"
        exit 1
        ;;
esac

echo "Build complete: ${BUILD_DIR}/${BINARY_NAME}"
echo "Version: ${VERSION}"
