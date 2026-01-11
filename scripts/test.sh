#!/bin/bash
# Test script for conky-go
# Usage: ./scripts/test.sh [unit|race|coverage|all]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

# Parse test type argument
TEST_TYPE="${1:-all}"

echo "Running tests (${TEST_TYPE})..."

case "$TEST_TYPE" in
    unit)
        echo "Running unit tests..."
        go test -v ./...
        ;;
    race)
        echo "Running tests with race detection..."
        go test -v -race ./...
        ;;
    coverage)
        echo "Running tests with coverage..."
        go test -coverprofile=coverage.out ./...
        go tool cover -func=coverage.out
        echo ""
        echo "Generating HTML coverage report..."
        go tool cover -html=coverage.out -o coverage.html
        echo "Coverage report: coverage.html"
        ;;
    all)
        echo "Running all tests with race detection..."
        go test -v -race ./...
        echo ""
        echo "Running tests with coverage..."
        go test -coverprofile=coverage.out ./...
        go tool cover -func=coverage.out
        ;;
    *)
        echo "Unknown test type: $TEST_TYPE"
        echo "Usage: $0 [unit|race|coverage|all]"
        exit 1
        ;;
esac

echo ""
echo "Tests complete!"
