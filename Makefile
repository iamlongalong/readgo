# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOVET=$(GOCMD) vet
GOMOD=$(GOCMD) mod
BINARY_NAME=readgo

# Linting
GOLINT=golangci-lint

# Build flags
BUILD_FLAGS=-v

# Test flags
TEST_FLAGS=-v -race -count=1

.PHONY: all build test clean vet lint fmt mod-tidy check pre-commit help

all: build

build:
	$(GOBUILD) $(BUILD_FLAGS) ./...

# Run tests
test:
	$(GOTEST) $(TEST_FLAGS) ./...

# Run go vet
vet:
	$(GOVET) ./...

# Run golangci-lint
lint:
	$(GOLINT) run

# Format code
fmt:
	$(GOCMD) fmt ./...

# Tidy modules
mod-tidy:
	$(GOMOD) tidy

# Clean build files
clean:
	$(GOCMD) clean
	rm -f $(BINARY_NAME)

# Run all checks (pre-commit)
check: fmt mod-tidy vet lint test
	@echo "All checks passed!"

# Pre-commit hook
pre-commit: check
	@echo "Ready to commit!"

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Show help
help:
	@echo "Available commands:"
	@echo "  make build       - Build the project"
	@echo "  make test        - Run tests"
	@echo "  make vet        - Run go vet"
	@echo "  make lint       - Run golangci-lint"
	@echo "  make fmt        - Format code"
	@echo "  make mod-tidy   - Tidy go modules"
	@echo "  make clean      - Clean build files"
	@echo "  make check      - Run all checks (fmt, mod-tidy, vet, lint, test)"
	@echo "  make pre-commit - Run all checks before commit"
	@echo "  make install-tools - Install development tools"
	@echo "  make help       - Show this help message"

# Default target
.DEFAULT_GOAL := help 