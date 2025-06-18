# Foundation Inquiry Slack Bot - Makefile

.PHONY: help build run run-dev test test-config test-e2e fmt vet lint clean setup quick-start ci db-clean db-backup install-tools

# Default target
help:
	@echo "Foundation Inquiry Slack Bot - Available Commands:"
	@echo ""
	@echo "Building and Running:"
	@echo "  build      - Build the application binary"
	@echo "  run        - Run directly with go run (use ENV_FILE to specify env file, e.g., make run ENV_FILE=.env.test)"
	@echo "  run-dev    - Run in development mode with detailed logging"
	@echo ""
	@echo "Testing:"
	@echo "  test       - Run all Go tests"
	@echo "  test-e2e   - Run end-to-end tests (requires server running)"
	@echo ""
	@echo "Development Tools:"
	@echo "  fmt        - Format Go code with gofmt"
	@echo "  vet        - Run go vet"
	@echo "  lint       - Run golangci-lint (install with make install-tools)"
	@echo "  ci         - Run full CI pipeline (fmt, vet, lint, test, build)"
	@echo ""
	@echo "Database Management:"
	@echo "  db-clean   - Clean database files in data/ directory"
	@echo "  db-backup  - Backup current database with timestamp"
	@echo ""
	@echo "Setup:"
	@echo "  setup      - Set up development environment"
	@echo "  quick-start - Complete setup for new developers"
	@echo "  install-tools - Install development tools (golangci-lint)"
	@echo ""
	@echo "Utilities:"
	@echo "  clean      - Clean build artifacts and temporary files"

# Building and Running
build:
	@echo "Building application..."
	@mkdir -p build
	go build -o build/inquiry-bot .
	@echo "Build complete: build/inquiry-bot"

run:
	@echo "Running application with environment from $${ENV_FILE:-.env}..."
	@if [ -f "$${ENV_FILE:-.env}" ]; then \
	  set -a; \
	  source "$${ENV_FILE:-.env}"; \
	  set +a; \
	  go run .; \
	else \
	  echo "Error: Environment file $${ENV_FILE:-.env} not found."; \
	  exit 1; \
	fi

run-dev:
	@echo "Running in development mode..."
	@echo "Running application with environment from $${ENV_FILE:-.env}..."
	@if [ -f "$${ENV_FILE:-.env}" ]; then \
	  set -a; \
	  source "$${ENV_FILE:-.env}"; \
	  set +a; \
	  ENV=development go run .; \
	else \
	  echo "Error: Environment file $${ENV_FILE:-.env} not found."; \
	  exit 1; \
	fi
	ENV=development go run .

# Testing
test:
	@echo "Running Go tests..."
	go test ./...

test-e2e:
	@echo "Running end-to-end tests..."
	@if [ ! -f .env.test ]; then \
		echo "Error: .env.test file not found. Copy .env to .env.test and configure for testing."; \
		echo "Run 'make setup' to create .env.test template."; \
		exit 1; \
	fi
	@echo "Using .env.test for test configuration..."
	@env $(cat .env.test | grep -v '^#' | xargs) ./test/e2e/run_all_tests.sh

# Development Tools
fmt:
	@echo "Formatting Go code..."
	go fmt ./...

vet:
	@echo "Running go vet..."
	go vet ./...

lint:
	@echo "Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install with: make install-tools"; \
		exit 1; \
	fi

install-tools:
	@echo "Installing development tools..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.55.2; \
	else \
		echo "golangci-lint already installed"; \
	fi

# CI Pipeline
ci: fmt vet test build
	@echo "CI pipeline completed successfully"

# Database Management
db-clean:
	@echo "Cleaning database files..."
	@if [ -d data ]; then \
		rm -rf data/; \
		echo "Database files cleaned"; \
	else \
		echo "No database files to clean"; \
	fi

db-backup:
	@echo "Backing up database..."
	@if [ -f data/inquiries.db ]; then \
		timestamp=$$(date +%Y%m%d_%H%M%S); \
		mkdir -p backups; \
		cp data/inquiries.db* backups/inquiries_backup_$$timestamp.db 2>/dev/null || true; \
		echo "Database backed up to backups/inquiries_backup_$$timestamp.db"; \
	else \
		echo "No database file found to backup"; \
	fi

# Setup
setup:
	@echo "Setting up development environment..."
	go mod download
	@mkdir -p data build backups
	@if [ ! -f .env ]; then \
		cp config.env.example .env; \
		echo "Created .env from config.env.example"; \
		echo "Please edit .env with your configuration"; \
	else \
		echo ".env already exists"; \
	fi
	@if [ ! -f .env.test ]; then \
		cp config.env.example .env.test; \
		echo "Created .env.test from config.env.example"; \
		echo "Please edit .env.test with your test configuration"; \
	else \
		echo ".env.test already exists"; \
	fi
	@echo "Setup complete"

quick-start: setup install-tools
	@echo "Quick start setup complete!"
	@echo ""
	@echo "Next steps:"
	@echo "1. Edit .env with your Slack and API credentials"
	@echo "2. Run 'make run-dev' to start the bot in development mode"
	@echo "3. Run 'make test' to verify everything works"

# Utilities
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf build/
	@echo "Clean complete"

# Go module management
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

deps-update:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy