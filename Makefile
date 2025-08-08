.PHONY: build run test clean docker-up docker-down docker-rebuild

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build directory
BUILD_DIR=build

# Services
SERVICES=auction-service analytics-service bidding-service

# Default target
all: build

# Build all services
build:
	@echo "Building all services..."
	@mkdir -p $(BUILD_DIR)
	@for service in $(SERVICES); do \
		echo "Building $$service..."; \
		$(GOBUILD) -o $(BUILD_DIR)/$$service ./cmd/$$service; \
	done

# Build specific service
build-service:
	@if [ -z "$(SERVICE)" ]; then \
		echo "Usage: make build-service SERVICE=<service-name>"; \
		exit 1; \
	fi
	@echo "Building $(SERVICE)..."
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) -o $(BUILD_DIR)/$(SERVICE) ./cmd/$(SERVICE)

# Run tests
test:
	@echo "Running tests..."
	@$(GOTEST) -v -race ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Run specific service locally
run-auction-service:
	@echo "Running auction service..."
	@$(GOCMD) run ./cmd/auction-service

run-analytics-service:
	@echo "Running analytics service..."
	@$(GOCMD) run ./cmd/analytics-service

run-auction-manager:
	@echo "Running bidding service..."
	@$(GOCMD) run ./cmd/bidding-service

# Docker commands
docker-up:
	@echo "Starting services with Docker Compose..."
	@docker-compose -f deployments/docker-compose.yml up -d

docker-down:
	@echo "Stopping services..."
	@docker-compose -f deployments/docker-compose.yml down

docker-rebuild:
	@echo "Rebuilding and starting services..."
	@docker-compose -f deployments/docker-compose.yml down
	@docker-compose -f deployments/docker-compose.yml build --no-cache
	@docker-compose -f deployments/docker-compose.yml up -d

# View logs
docker-logs:
	@docker-compose -f deployments/docker-compose.yml logs -f

docker-logs-service:
	@if [ -z "$(SERVICE)" ]; then \
		echo "Usage: make docker-logs-service SERVICE=<service-name>"; \
		exit 1; \
	fi
	@docker-compose -f deployments/docker-compose.yml logs -f $(SERVICE)

# Clean
clean:
	@echo "Cleaning..."
	@$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@$(GOMOD) download
	@$(GOMOD) tidy

# Lint code (requires golangci-lint)
lint:
	@echo "Running linter..."
	@golangci-lint run

# Format code
fmt:
	@echo "Formatting code..."
	@$(GOCMD) fmt ./...

# Generate mocks (requires mockgen)
generate-mocks:
	@echo "Generating mocks..."
	@$(GOCMD) generate ./...

# Database migration (simple version)
migrate-up:
	@echo "Running database migrations..."
	@mysql -h localhost -u auction_user -pauction_pass auction_db < scripts/init.sql

# Development setup
dev-setup: deps
	@echo "Setting up development environment..."
	@docker-compose -f deployments/docker-compose.yml up -d redis mysql
	@sleep 10
	@make migrate-up

# Integration tests (requires running services)
test-integration:
	@echo "Running integration tests..."
	@$(GOTEST) -v -tags=integration ./tests/integration/...

# Load test (requires wrk or similar tool)
load-test:
	@echo "Running load tests..."
	@echo "Note: This requires wrk to be installed"
	@wrk -t12 -c400 -d30s --script=tests/load/bid_test.lua http://localhost:8080

# Help
help:
	@echo "Available targets:"
	@echo "  build              - Build all services"
	@echo "  build-service      - Build specific service (usage: make build-service SERVICE=auction-service)"
	@echo "  test               - Run unit tests"
	@echo "  test-coverage      - Run tests with coverage"
	@echo "  test-integration   - Run integration tests"
	@echo "  run-auction-service     - Run auction service locally"
	@echo "  run-analytics-service   - Run analytics service locally"
	@echo "  run-auction-manager     - Run auction manager locally"
	@echo "  docker-up          - Start all services with Docker"
	@echo "  docker-down        - Stop all services"
	@echo "  docker-rebuild     - Rebuild and restart services"
	@echo "  docker-logs        - View all service logs"
	@echo "  docker-logs-service     - View specific service logs (usage: make docker-logs-service SERVICE=auction-service-1)"
	@echo "  clean              - Clean build artifacts"
	@echo "  deps               - Download dependencies"
	@echo "  lint               - Run linter"
	@echo "  fmt                - Format code"
	@echo "  dev-setup          - Setup development environment"
	@echo "  migrate-up         - Run database migrations"
	@echo "  load-test          - Run load tests"
	@echo "  help               - Show this help"