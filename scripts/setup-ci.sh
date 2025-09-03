# =============================================================================
# scripts/setup-ci.sh - Setup CI/CD environment
# =============================================================================
#!/bin/bash
set -euo pipefail

echo "Setting up CI/CD pipeline..."

# Create directory structure
mkdir -p .github/workflows
mkdir -p k8s
mkdir -p scripts

# Create GitHub Actions secrets setup script
cat > scripts/setup-github-secrets.sh << 'EOF'
#!/bin/bash
# This script helps you set up GitHub secrets
echo "You need to add these secrets to your GitHub repository:"
echo "1. Go to Settings > Secrets and variables > Actions"
echo "2. Add the following secrets:"
echo ""
echo "DOCKER_REGISTRY_URL=ghcr.io"
echo "DOCKER_USERNAME=<your-github-username>"
echo "DOCKER_PASSWORD=<your-github-token>"
echo "KUBE_CONFIG=<base64-encoded-kubeconfig>"
echo "SLACK_WEBHOOK_URL=<your-slack-webhook-url>"
echo ""
echo "To get your GitHub token:"
echo "1. Go to Settings > Developer settings > Personal access tokens"
echo "2. Generate a new token with 'write:packages' permission"
EOF

# Create local development script
cat > scripts/dev.sh << 'EOF'
#!/bin/bash
set -euo pipefail

echo "Starting development environment..."

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check prerequisites
if ! command_exists docker; then
    echo "Error: Docker is not installed"
    exit 1
fi

if ! command_exists docker-compose; then
    echo "Error: Docker Compose is not installed"
    exit 1
fi

# Start monitoring stack
echo "Starting monitoring stack..."
docker-compose up -d prometheus grafana alertmanager

# Wait for services to be ready
echo "Waiting for services to start..."
sleep 10

# Start the Go application
echo "Starting user service..."
go run . &
APP_PID=$!

echo "Development environment ready!"
echo "Application: http://localhost:8080"
echo "Metrics: http://localhost:8080/metrics"
echo "Health: http://localhost:8080/health"
echo "Prometheus: http://localhost:9090"
echo "Grafana: http://localhost:3000 (admin/admin123)"
echo ""
echo "Press Ctrl+C to stop..."

# Trap to cleanup on exit
trap 'echo "Stopping services..."; kill $APP_PID 2>/dev/null; docker-compose down' EXIT

# Wait for interrupt
wait $APP_PID
EOF

# Create test script
cat > scripts/test.sh << 'EOF'
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
EOF

# Create build script
cat > scripts/build.sh << 'EOF'
#!/bin/bash
set -euo pipefail

VERSION=${1:-latest}
DOCKER_REGISTRY=${DOCKER_REGISTRY:-ghcr.io}
IMAGE_NAME=${DOCKER_REGISTRY}/$(echo $GITHUB_REPOSITORY | tr '[:upper:]' '[:lower:]' || echo "user-service")

echo "Building user-service version: $VERSION"

# Build Go binary
echo "Building Go binary..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags "-s -w" -o bin/user-service .

# Build Docker image
echo "Building Docker image: ${IMAGE_NAME}:${VERSION}"
docker build -t "${IMAGE_NAME}:${VERSION}" .

# Also tag as latest if this is a version tag
if [[ $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    docker tag "${IMAGE_NAME}:${VERSION}" "${IMAGE_NAME}:latest"
    echo "Tagged as latest"
fi

echo "Build completed successfully!"
echo "Image: ${IMAGE_NAME}:${VERSION}"
EOF

# Create release script
cat > scripts/release.sh << 'EOF'
#!/bin/bash
set -euo pipefail

VERSION=${1?"Usage: $0 <version> (e.g., v1.0.0)"}

# Validate version format
if [[ ! $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: Version must be in format v1.0.0"
    exit 1
fi

echo "Preparing release $VERSION..."

# Check if we're on main branch
CURRENT_BRANCH=$(git branch --show-current)
if [[ "$CURRENT_BRANCH" != "main" ]]; then
    echo "Error: Must be on main branch for release"
    exit 1
fi

# Check if working directory is clean
if [[ -n $(git status --porcelain) ]]; then
    echo "Error: Working directory must be clean"
    exit 1
fi

# Run tests
echo "Running tests..."
./scripts/test.sh

# Update version in files (if you have version files)
echo "Updating version..."
# Add version update commands here

# Create git tag
echo "Creating git tag..."
git tag -a "$VERSION" -m "Release $VERSION"

# Push tag
echo "Pushing tag to origin..."
git push origin "$VERSION"

echo "Release $VERSION created successfully!"
echo "GitHub Actions will automatically build and deploy this release."
EOF

# Make scripts executable
chmod +x scripts/*.sh

echo "CI/CD setup completed!"
echo ""
echo "Next steps:"
echo "1. Create GitHub repository secrets (run scripts/setup-github-secrets.sh for guidance)"
echo "2. Customize the deployment targets in the workflow files"
echo "3. Update the Kubernetes manifests with your specific configuration"
echo "4. Test locally with: scripts/dev.sh"
echo "5. Create your first release with: scripts/release.sh v1.0.0"