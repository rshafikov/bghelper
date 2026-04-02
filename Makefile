# bghelper Makefile
# Build and release automation

# Variables
BINARY_NAME=bgh
MAIN_PATH=./cmd/bghelper
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build directory
BUILD_DIR=dist

.PHONY: all build clean test coverage install uninstall run fmt lint help

all: clean test build

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)

## build-all: Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)

## test: Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

## coverage: Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.txt -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated: coverage.html"

## install: Install binary to GOPATH/bin
install:
	@echo "Installing..."
	$(GOCMD) install $(LDFLAGS) $(MAIN_PATH)

## uninstall: Remove binary from GOPATH/bin
uninstall:
	@echo "Uninstalling..."
	rm -f $(shell $(GOCMD) env GOPATH)/bin/$(BINARY_NAME)

## run: Run the application
run:
	$(GOCMD) run $(MAIN_PATH)

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

## lint: Run linter
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download

## tidy: Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

## release: Create a release using goreleaser
release:
	@echo "Creating release..."
	@which goreleaser > /dev/null || (echo "goreleaser not installed. Run: go install github.com/goreleaser/goreleaser/v2@latest" && exit 1)
	goreleaser release --clean

## release-snapshot: Create a snapshot release (for testing)
release-snapshot:
	@echo "Creating snapshot release..."
	@which goreleaser > /dev/null || (echo "goreleaser not installed. Run: go install github.com/goreleaser/goreleaser/v2@latest" && exit 1)
	goreleaser release --clean --snapshot

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':'
