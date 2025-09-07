# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=$(shell go env GOPATH)/bin/golangci-lint

# Application parameters
BINARY_NAME=k8s-memory-watch
BINARY_PATH=./cmd/$(BINARY_NAME)
BUILD_DIR=./build

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: local-setup
local-setup: ## Set up local development environment
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/air-verse/air@latest
	@echo "Development tools installed successfully"

.PHONY: build
build: ## Build the application
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(BINARY_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

.PHONY: build-linux
build-linux: ## Build for Linux
	@echo "Building $(BINARY_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(BINARY_PATH)

.PHONY: build-docker
build-docker: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t $(BINARY_NAME):$(VERSION) -t $(BINARY_NAME):latest .

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out

.PHONY: update
update: ## Update dependencies
	@echo "Updating dependencies..."
	@$(GOMOD) tidy
	@$(GOMOD) download

.PHONY: add-package
add-package: ## Add a package (usage: make add-package package=github.com/example/pkg)
ifndef package
	@echo "Usage: make add-package package=github.com/example/pkg"
	@exit 1
endif
	@echo "Adding package: $(package)"
	@$(GOGET) $(package)
	@$(GOMOD) tidy

.PHONY: up
up: build ## Build and run the application
	@echo "Starting $(BINARY_NAME)..."
	@$(BUILD_DIR)/$(BINARY_NAME)

.PHONY: dev
dev: ## Run with hot reload (requires air)
	@echo "Starting development server with hot reload..."
	@air

.PHONY: down
down: ## Stop the application (if running as daemon)
	@echo "Stopping application..."
	@pkill -f $(BINARY_NAME) || true

.PHONY: check-typing
check-typing: ## Run type checking (Go has built-in type checking)
	@echo "Running type checking..."
	@$(GOCMD) vet ./...

.PHONY: check-format
check-format: ## Check code formatting
	@echo "Checking code formatting..."
	@files=$$($(GOFMT) -l .); if [ -n "$$files" ]; then \
		echo "The following files are not properly formatted:"; \
		echo "$$files"; \
		exit 1; \
	fi
	@echo "Code formatting is correct"

.PHONY: check-style
check-style: ## Check code style with golangci-lint
	@echo "Running linter..."
	@$(GOLINT) run

.PHONY: reformat
reformat: ## Format code according to Go standards
	@echo "Formatting code..."
	@$(GOFMT) -w .
	@$(shell go env GOPATH)/bin/goimports -w .
	@echo "Code formatting complete"

.PHONY: test-unit
test-unit: ## Run unit tests
	@echo "Running unit tests..."
	@$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html

.PHONY: test-e2e
test-e2e: ## Run end-to-end tests
	@echo "Running end-to-end tests..."
	@$(GOTEST) -v -tags=e2e ./test/integration/...

.PHONY: test-coverage
test-coverage: test-unit ## Show test coverage
	@echo "Test coverage report:"
	@$(GOCMD) tool cover -func=coverage.out

.PHONY: validate
validate: check-typing check-format check-style test-unit ## Run all validation checks
	@echo "All validation checks passed ✅"

.PHONY: install-deps
install-deps: ## Install Go dependencies
	@echo "Installing dependencies..."
	@$(GOMOD) download
	@$(GOMOD) verify

.PHONY: security-scan
security-scan: ## Run security scan with govulncheck
	@echo "Running security scan..."
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@govulncheck ./...

# Docker targets
.PHONY: docker-run
docker-run: build-docker ## Build and run Docker container
	@echo "Running Docker container..."
	@docker run --rm -it $(BINARY_NAME):latest

.PHONY: docker-compose-up
docker-compose-up: ## Start with docker-compose
	@echo "Starting with docker-compose..."
	@docker-compose up --build

.PHONY: docker-compose-down
docker-compose-down: ## Stop docker-compose
	@echo "Stopping docker-compose..."
	@docker-compose down
