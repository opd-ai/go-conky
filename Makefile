.PHONY: build test clean install deps lint coverage bench integration fmt vet dist checksums build-linux build-windows build-darwin build-all dist-linux dist-windows dist-darwin dist-all

BINARY_NAME=conky-go
BUILD_DIR=build
DIST_DIR=dist
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
	@rm -rf $(DIST_DIR)
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
	@echo "  build         - Build the binary for native platform"
	@echo "  test          - Run tests with race detection"
	@echo "  bench         - Run benchmarks"
	@echo "  integration   - Run integration tests"
	@echo "  clean         - Clean build artifacts"
	@echo "  install       - Install binary to /usr/local/bin"
	@echo "  deps          - Download and verify dependencies"
	@echo "  lint          - Run golangci-lint"
	@echo "  coverage      - Generate test coverage report"
	@echo "  fmt           - Format code with go fmt"
	@echo "  vet           - Run go vet"
	@echo "  run           - Build and run with ~/.conkyrc"
	@echo "  dist          - Build distribution package for native platform"
	@echo "  checksums     - Generate checksums for distribution files"
	@echo ""
	@echo "Cross-platform build targets:"
	@echo "  build-linux   - Build for Linux (amd64, arm64)"
	@echo "  build-windows - Build for Windows (amd64)"
	@echo "  build-darwin  - Build for macOS (amd64, arm64)"
	@echo "  build-android - Build for Android (arm64)"
	@echo "  build-all     - Build for all platforms"
	@echo "  dist-linux    - Create Linux distribution packages"
	@echo "  dist-windows  - Create Windows distribution package"
	@echo "  dist-darwin   - Create macOS distribution packages"
	@echo "  dist-all      - Create distribution packages for all platforms"
	@echo ""
	@echo "  help          - Print this help message"

# Build distribution package for native platform
dist: clean
	@echo "Building distribution package..."
	@mkdir -p $(DIST_DIR)
	@set -e; \
	BINARY="$(BINARY_NAME)-$$(go env GOOS)-$$(go env GOARCH)"; \
	echo "Building $(DIST_DIR)/$$BINARY..."; \
	go build $(LDFLAGS) -o $(DIST_DIR)/$$BINARY ./cmd/conky-go; \
	cp README.md LICENSE $(DIST_DIR)/; \
	tar -czvf "$(DIST_DIR)/$$BINARY.tar.gz" -C $(DIST_DIR) "$$BINARY" README.md LICENSE; \
	rm -f "$(DIST_DIR)/$$BINARY" "$(DIST_DIR)/README.md" "$(DIST_DIR)/LICENSE"
	@$(MAKE) checksums

# Generate checksums for distribution files
checksums:
	@echo "Generating checksums..."
	@cd $(DIST_DIR) && if ls *.tar.gz >/dev/null 2>&1; then sha256sum *.tar.gz > checksums.txt && echo "Checksums written to $(DIST_DIR)/checksums.txt"; else echo "No .tar.gz files found in $(DIST_DIR); skipping checksum generation."; fi

# Cross-platform build targets
.PHONY: build-linux build-windows build-darwin build-android build-all

build-linux:
	@echo "Building for Linux (amd64)..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/conky-go
	@echo "Building for Linux (arm64)..."
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/conky-go

build-windows:
	@echo "Building for Windows (amd64)..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/conky-go

build-darwin:
	@echo "Building for macOS (amd64)..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/conky-go
	@echo "Building for macOS (arm64)..."
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/conky-go

build-android:
	@echo "Building for Android (arm64)..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=android GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-android-arm64 ./cmd/conky-go

build-all: build-linux build-windows build-darwin build-android
	@echo "All platform builds complete."

# Cross-platform distribution packages
.PHONY: dist-all dist-linux dist-windows dist-darwin

dist-linux:
	@echo "Building Linux distribution packages..."
	@mkdir -p $(DIST_DIR)
	@$(MAKE) build-linux
	@for arch in amd64 arm64; do \
		echo "Packaging $(BINARY_NAME)-linux-$$arch..."; \
		cp README.md LICENSE $(BUILD_DIR)/; \
		tar -czvf "$(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-$$arch.tar.gz" \
			-C $(BUILD_DIR) "$(BINARY_NAME)-linux-$$arch" README.md LICENSE; \
		rm -f "$(BUILD_DIR)/README.md" "$(BUILD_DIR)/LICENSE"; \
	done

dist-windows:
	@echo "Building Windows distribution package..."
	@mkdir -p $(DIST_DIR)
	@$(MAKE) build-windows
	@cp README.md LICENSE $(BUILD_DIR)/
	@cd $(BUILD_DIR) && zip -q ../$(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip \
		$(BINARY_NAME)-windows-amd64.exe README.md LICENSE
	@rm -f "$(BUILD_DIR)/README.md" "$(BUILD_DIR)/LICENSE"

dist-darwin:
	@echo "Building macOS distribution packages..."
	@mkdir -p $(DIST_DIR)
	@$(MAKE) build-darwin
	@for arch in amd64 arm64; do \
		echo "Packaging $(BINARY_NAME)-darwin-$$arch..."; \
		cp README.md LICENSE $(BUILD_DIR)/; \
		tar -czvf "$(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-$$arch.tar.gz" \
			-C $(BUILD_DIR) "$(BINARY_NAME)-darwin-$$arch" README.md LICENSE; \
		rm -f "$(BUILD_DIR)/README.md" "$(BUILD_DIR)/LICENSE"; \
	done

dist-all: dist-linux dist-windows dist-darwin
	@echo "All distribution packages complete."
	@$(MAKE) checksums
