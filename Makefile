# ==============================================================================
# Rental API - Makefile
# ==============================================================================

SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c

# Variables
BIN_DIR        ?= bin
API_BIN        ?= $(BIN_DIR)/api
MIGRATION_BIN  ?= $(BIN_DIR)/migration
API_ENTRY      ?= ./cmd/api
MIGRATION_ENTRY?= ./cmd/migration
COVERAGE_DIR   ?= coverage
COVERAGE_FILE  ?= $(COVERAGE_DIR)/coverage.out
COVERAGE_HTML  ?= $(COVERAGE_DIR)/coverage.html

GO             ?= go
GOFLAGS        ?=

# Git information
GIT_COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_TAG        ?= $(shell git describe --tags --always 2>/dev/null || echo "dev")
BUILD_TIME     ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

LDFLAGS        := -w -s \
	-X 'main.GitCommit=$(GIT_COMMIT)' \
	-X 'main.GitTag=$(GIT_TAG)' \
	-X 'main.BuildTime=$(BUILD_TIME)'

# Default target
.DEFAULT_GOAL := help

# ==============================================================================
# Help
# ==============================================================================

.PHONY: help
help: ## Show this help menu
	@echo "========================================================================"
	@echo "Rental API - Available Make Commands"
	@echo "========================================================================"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""

# ==============================================================================
# Development & Running
# ==============================================================================

.PHONY: run
run: ## Run the API server directly (cmd/api)
	$(GO) run $(API_ENTRY)/main.go

.PHONY: seed
seed: ## Run database seeder migration (cmd/migration)
	$(GO) run $(MIGRATION_ENTRY)/main.go

.PHONY: env-init
env-init: ## Initialize .env from .env.example if .env does not exist
	@if [ ! -f .env ]; then \
		cp .env.example .env && echo "Created .env from .env.example"; \
	else \
		echo ".env already exists"; \
	fi

# ==============================================================================
# Building
# ==============================================================================

.PHONY: build
build: build-api build-migration ## Build all binaries (API server and migration runner)

.PHONY: build-api
build-api: ## Build the API server binary into bin/api
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(API_BIN) $(API_ENTRY)
	@echo "Built $(API_BIN)"

.PHONY: build-migration
build-migration: ## Build the database migration binary into bin/migration
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(MIGRATION_BIN) $(MIGRATION_ENTRY)
	@echo "Built $(MIGRATION_BIN)"

# ==============================================================================
# Code Generation
# ==============================================================================

.PHONY: generate
generate: generate-ent generate-graphql ## Run all code generators (Ent ORM and GraphQL gqlgen)

.PHONY: generate-ent
generate-ent: ## Run Ent ORM code generation
	$(GO) generate ./ent/...

.PHONY: generate-graphql
generate-graphql: ## Run gqlgen GraphQL code generation
	$(GO) run github.com/99designs/gqlgen generate

# ==============================================================================
# Testing & Code Quality
# ==============================================================================

.PHONY: test
test: ## Run unit tests
	$(GO) test ./...

.PHONY: test-verbose
test-verbose: ## Run unit tests with verbose logging
	$(GO) test -v ./...

.PHONY: test-coverage
test-coverage: ## Run unit tests and generate coverage report
	@mkdir -p $(COVERAGE_DIR)
	$(GO) test -coverprofile=$(COVERAGE_FILE) ./...
	$(GO) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

.PHONY: fmt
fmt: ## Format all Go source files
	$(GO) fmt ./...

.PHONY: vet
vet: ## Run Go static analysis (go vet)
	$(GO) vet ./...

.PHONY: lint
lint: ## Run linter (golangci-lint if installed, fallback to go vet)
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, falling back to go vet..."; \
		$(GO) vet ./...; \
	fi

.PHONY: tidy
tidy: ## Download dependencies and tidy go.mod/go.sum
	$(GO) mod tidy

.PHONY: verify
verify: fmt vet test ## Format code, run static analysis, and execute tests

# ==============================================================================
# Cleanup
# ==============================================================================

.PHONY: clean
clean: ## Remove build artifacts and test coverage reports
	rm -rf $(BIN_DIR) $(COVERAGE_DIR) coverage.out coverage.html
	@echo "Cleaned build artifacts and coverage files"
