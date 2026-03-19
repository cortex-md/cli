.PHONY: fmt lint test build build-dev clean install release help

BIN_DIR := bin
BIN_NAME := cortex
CMD_DIR := ./cmd/cortex

GO := go
VERSION ?= 0.1.0
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS_BASE := -s -w \
	-X github.com/cortex/cli/internal/buildinfo.Version=$(VERSION) \
	-X github.com/cortex/cli/internal/buildinfo.BuildTime=$(BUILD_TIME) \
	-X github.com/cortex/cli/internal/buildinfo.GitCommit=$(GIT_COMMIT)

LDFLAGS_PROD := $(LDFLAGS_BASE) \
	-X github.com/cortex/cli/internal/buildinfo.GitHubClientID=$(GITHUB_CLIENT_ID)

help:
	@echo "Cortex CLI Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  build-dev  Build development binary (no GitHub auth)"
	@echo "  build      Build production binary (requires GITHUB_CLIENT_ID)"
	@echo "  install    Install binary to GOPATH/bin"
	@echo "  release    Build release binaries for all platforms"
	@echo "  test       Run tests"
	@echo "  fmt        Format Go code"
	@echo "  lint       Run linters (requires golangci-lint)"
	@echo "  clean      Remove build artifacts"
	@echo "  help       Show this help message"
	@echo ""
	@echo "Build with GitHub OAuth:"
	@echo "  make build GITHUB_CLIENT_ID=Ov23li..."
	@echo ""
	@echo "Build for development (no login):"
	@echo "  make build-dev"

fmt:
	@echo "Formatting Go code..."
	@$(GO) fmt ./...

lint:
	@echo "Running linters..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found, skipping"; \
	fi

test:
	@echo "Running tests..."
	@$(GO) test -v -race -cover ./...

build-dev:
	@echo "Building $(BIN_NAME) (development)..."
	@mkdir -p $(BIN_DIR)
	@$(GO) build -ldflags "$(LDFLAGS_BASE)" -o $(BIN_DIR)/$(BIN_NAME) $(CMD_DIR)
	@echo "Binary built: $(BIN_DIR)/$(BIN_NAME)"
	@echo "Note: Login disabled in dev builds"

build:
ifndef GITHUB_CLIENT_ID
	$(error GITHUB_CLIENT_ID is required. Usage: make build GITHUB_CLIENT_ID=your_client_id)
endif
	@echo "Building $(BIN_NAME) (production)..."
	@mkdir -p $(BIN_DIR)
	@$(GO) build -ldflags "$(LDFLAGS_PROD)" -o $(BIN_DIR)/$(BIN_NAME) $(CMD_DIR)
	@echo "Binary built: $(BIN_DIR)/$(BIN_NAME)"

clean:
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR)
	@$(GO) clean

install:
ifndef GITHUB_CLIENT_ID
	$(error GITHUB_CLIENT_ID is required. Usage: make install GITHUB_CLIENT_ID=your_client_id)
endif
	@echo "Installing to GOPATH/bin..."
	@$(GO) install -ldflags "$(LDFLAGS_PROD)" $(CMD_DIR)
	@echo "Installed successfully"

release: clean
ifndef GITHUB_CLIENT_ID
	$(error GITHUB_CLIENT_ID is required for release builds)
endif
	@echo "Building release binaries..."
	@mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=arm64 $(GO) build -ldflags "$(LDFLAGS_PROD)" -o $(BIN_DIR)/$(BIN_NAME)-darwin-arm64 $(CMD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS_PROD)" -o $(BIN_DIR)/$(BIN_NAME)-darwin-amd64 $(CMD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS_PROD)" -o $(BIN_DIR)/$(BIN_NAME)-linux-amd64 $(CMD_DIR)
	GOOS=linux GOARCH=arm64 $(GO) build -ldflags "$(LDFLAGS_PROD)" -o $(BIN_DIR)/$(BIN_NAME)-linux-arm64 $(CMD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS_PROD)" -o $(BIN_DIR)/$(BIN_NAME)-windows-amd64.exe $(CMD_DIR)
	@echo "Release binaries built in $(BIN_DIR)/"
