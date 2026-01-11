.PHONY: build test clean install deps lint coverage bench integration fmt vet

BINARY_NAME=conky-go
BUILD_DIR=build
GO_FILES=$(shell find . -name "*.go" -not -path "./vendor/*")
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

# Default target
all: build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/conky-go

# Run tests with race detection
test:
	@echo "Running tests..."
	@go test -v -race ./...

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

# Run integration tests
integration:
	@echo "Running integration tests..."
	@go test -v -tags=integration ./test/...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@go clean -testcache

# Install binary to system
install: build
	@echo "Installing $(BINARY_NAME)..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)

# Download and verify dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod verify
	@go mod tidy

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

# Generate test coverage report
coverage:
	@echo "Generating coverage report..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run go vet
vet:
	@echo "Running vet..."
	@go vet ./...

# Development helpers
.PHONY: run
run: build
	@$(BUILD_DIR)/$(BINARY_NAME) -c ~/.conkyrc

# Print help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build       - Build the binary"
	@echo "  test        - Run tests with race detection"
	@echo "  bench       - Run benchmarks"
	@echo "  integration - Run integration tests"
	@echo "  clean       - Clean build artifacts"
	@echo "  install     - Install binary to /usr/local/bin"
	@echo "  deps        - Download and verify dependencies"
	@echo "  lint        - Run golangci-lint"
	@echo "  coverage    - Generate test coverage report"
	@echo "  fmt         - Format code with go fmt"
	@echo "  vet         - Run go vet"
	@echo "  run         - Build and run with ~/.conkyrc"
	@echo "  help        - Print this help message"
