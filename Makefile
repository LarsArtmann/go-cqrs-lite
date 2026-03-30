# Go-CQRS-Lite Makefile
# Standard targets for building, testing, and linting

.PHONY: all build test lint fmt clean coverage help

# Default target
all: fmt lint test build

## build: Build the project
build:
	@echo "Building..."
	go build ./...

## test: Run all tests
test:
	@echo "Running tests..."
	go test ./... -v

## test-race: Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	go test ./... -race

## test-short: Run short tests only
test-short:
	@echo "Running short tests..."
	go test ./... -short

## coverage: Generate and display test coverage
coverage:
	@echo "Generating coverage report..."
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

## coverage-html: Generate HTML coverage report
coverage-html: coverage
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## lint: Run golangci-lint
lint:
	@echo "Running linter..."
	golangci-lint run

## fmt: Format all Go files
fmt:
	@echo "Formatting code..."
	gofmt -w .

## imports: Organize imports
imports:
	@echo "Organizing imports..."
	goimports -w .

## vet: Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

## mod-tidy: Tidy go modules
mod-tidy:
	@echo "Tidying modules..."
	go mod tidy

## mod-verify: Verify modules
mod-verify:
	@echo "Verifying modules..."
	go mod verify

## clean: Remove generated files
clean:
	@echo "Cleaning..."
	trash coverage.out coverage.html 2>/dev/null || true
	go clean -testcache

## check: Run all checks (fmt, vet, lint, test)
check: fmt vet lint test

## ci: Run CI pipeline checks
ci: build test-race lint

## help: Display this help
help:
	@echo "Available targets:"
	@grep -E '^##' Makefile | sed 's/## //g' | column -t -s ':'
