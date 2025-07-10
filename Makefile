# Harbinger Makefile

# Variables
BINARY_NAME := harbinger
MAIN_PATH := ./cmd
BUILD_DIR := build
COVERAGE_FILE := coverage.out
COVERAGE_HTML := coverage.html

# Go related variables
GOBASE := $(shell pwd)
GOBIN := $(GOBASE)/$(BUILD_DIR)
GOFILES := $(shell find . -name "*.go" -type f)
GOPACKAGES := $(shell go list ./...)

# Build flags
LDFLAGS := -ldflags "-s -w"
BUILD_FLAGS := -trimpath

# Default target
.DEFAULT_GOAL := help

## help: Show this help message
.PHONY: help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ": "}; {printf "  %-15s %s\n", $$2, $$3}'

## build: Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: ./$(BINARY_NAME)"

## build-all: Build for multiple platforms
.PHONY: build-all
build-all: clean
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	
	@echo "Building for Linux amd64..."
	@GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	
	@echo "Building for Darwin amd64..."
	@GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	
	@echo "Building for Darwin arm64..."
	@GOOS=darwin GOARCH=arm64 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	
	@echo "Building for Windows amd64..."
	@GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	
	@echo "All builds complete in $(BUILD_DIR)/"

## install: Install the binary to $GOPATH/bin
.PHONY: install
install:
	@echo "Installing $(BINARY_NAME)..."
	@go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)
	@mkdir -p $(shell go env GOPATH)/bin
	@cp $(BINARY_NAME) $(shell go env GOPATH)/bin/$(BINARY_NAME)
	@rm -f $(BINARY_NAME)
	@echo "Installation complete"
	@echo "Run this command to add harbinger to your path:"
	@echo "export PATH=$(shell go env GOPATH)/bin:$PATH"

## uninstall: Remove the binary from $GOPATH/bin
.PHONY: uninstall
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(shell go env GOPATH)/bin/$(BINARY_NAME)
	@echo "Uninstall complete"

## clean: Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(BINARY_NAME)
	@rm -rf $(BUILD_DIR)
	@rm -f $(COVERAGE_FILE)
	@rm -f $(COVERAGE_HTML)
	@echo "Clean complete"

## test: Run tests
.PHONY: test
test:
	@echo "Running tests..."
	@go test -v $(GOPACKAGES)

## test-coverage: Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=$(COVERAGE_FILE) -covermode=atomic $(GOPACKAGES)
	@go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

## test-race: Run tests with race detector
.PHONY: test-race
test-race:
	@echo "Running tests with race detector..."
	@go test -v -race $(GOPACKAGES)

## lint: Run linter
.PHONY: lint
lint:
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install with:"; \
		echo "  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin"; \
	fi

## fmt: Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@go fmt $(GOPACKAGES)
	@echo "Formatting complete"

## vet: Run go vet
.PHONY: vet
vet:
	@echo "Running go vet..."
	@go vet $(GOPACKAGES)

## mod: Download and tidy modules
.PHONY: mod
mod:
	@echo "Downloading modules..."
	@go mod download
	@echo "Tidying modules..."
	@go mod tidy

## dev: Run the application in development mode
.PHONY: dev
dev: build
	@echo "Running $(BINARY_NAME) in development mode..."
	@./$(BINARY_NAME) monitor --interval 10s

## docker-build: Build Docker image
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(BINARY_NAME):latest .

## release: Create a new release (requires VERSION parameter)
.PHONY: release
release:
ifndef VERSION
	$(error VERSION is not set. Usage: make release VERSION=v1.0.0)
endif
	@echo "Creating release $(VERSION)..."
	@git tag -a $(VERSION) -m "Release $(VERSION)"
	@echo "Release $(VERSION) created. Push with: git push origin $(VERSION)"

## check: Run all checks (fmt, vet, lint, test)
.PHONY: check
check: fmt vet lint test
	@echo "All checks passed!"

## run: Build and run the application
.PHONY: run
run: build
	@./$(BINARY_NAME)

.PHONY: all
all: clean mod fmt vet lint test build