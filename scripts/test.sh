#!/bin/bash
set -euo pipefail

echo "Running comprehensive tests..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    local status=$1
    local message=$2
    case $status in
        "INFO") echo -e "${YELLOW}[INFO]${NC} $message" ;;
        "SUCCESS") echo -e "${GREEN}[SUCCESS]${NC} $message" ;;
        "ERROR") echo -e "${RED}[ERROR]${NC} $message" ;;
    esac
}

# Check Go installation
if ! command -v go >/dev/null 2>&1; then
    print_status "ERROR" "Go is not installed"
    exit 1
fi

print_status "INFO" "Go version: $(go version)"

# Download dependencies
print_status "INFO" "Downloading dependencies..."
go mod download

# Run unit tests
print_status "INFO" "Running unit tests..."
if go test -run "^Test[^I]" -v -coverprofile=coverage-unit.out ./...; then
    print_status "SUCCESS" "Unit tests passed"
else
    print_status "ERROR" "Unit tests failed"
    exit 1
fi

# Run integration tests
print_status "INFO" "Running integration tests..."
if go test -run "^TestIntegration" -v -coverprofile=coverage-integration.out ./...; then
    print_status "SUCCESS" "Integration tests passed"
else
    print_status "ERROR" "Integration tests failed"
    exit 1
fi

# Generate combined coverage
print_status "INFO" "Generating coverage report..."
go test -v -coverprofile=coverage-combined.out ./...
go tool cover -html=coverage-combined.out -o coverage.html

# Check coverage threshold
COVERAGE=$(go tool cover -func=coverage-combined.out | grep total | awk '{print $3}' | sed 's/%//')
print_status "INFO" "Coverage: ${COVERAGE}%"

if (( $(echo "$COVERAGE >= 70" | bc -l) )); then
    print_status "SUCCESS" "Coverage meets 70% threshold"
else
    print_status "ERROR" "Coverage ${COVERAGE}% is below 70% threshold"
    exit 1
fi

# Run linting (if golangci-lint is installed)
if command -v golangci-lint >/dev/null 2>&1; then
    print_status "INFO" "Running linter..."
    if golangci-lint run; then
        print_status "SUCCESS" "Linting passed"
    else
        print_status "ERROR" "Linting failed"
        exit 1
    fi
else
    print_status "INFO" "golangci-lint not installed, skipping linting"
fi

print_status "SUCCESS" "All tests completed successfully!"
print_status "INFO" "Coverage report generated: coverage.html"
