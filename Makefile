.PHONY: build run test docker-up docker-down metrics clean help

# Build the Go application
build:
	@echo "Building user service..."
	@go build -o bin/user-service ./cmd/server

# Run the application locally
run:
	@echo "Starting user service..."
	@go run ./cmd/server

# Run tests with coverage
test:
	@echo "Running tests..."
	@go test -v -cover ./internal/... ./cmd/...

# Run only unit tests
test-unit:
	@echo "Running unit tests..."
	@go test -run "^Test[^I]" -v ./internal/...

# Run only integration tests
test-integration:
	@echo "Running integration tests..."
	@go test -v ./test/integration/...

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./internal/...

# Start the complete monitoring stack
docker-up:
	@echo "Starting monitoring stack..."
	@docker-compose -f deployments/docker/docker-compose.yml up -d
	@echo "Services available:"
	@echo "  Application:  http://localhost:8080"
	@echo "  Metrics:      http://localhost:8080/metrics"
	@echo "  Prometheus:   http://localhost:9090"
	@echo "  Grafana:      http://localhost:3000 (admin/admin123)"
	@echo "  AlertManager: http://localhost:9093"

# Stop the monitoring stack
docker-down:
	@echo "Stopping monitoring stack..."
	@docker-compose -f deployments/docker/docker-compose.yml down

# View metrics in terminal
metrics:
	@echo "Fetching current metrics..."
	@curl -s http://localhost:8080/metrics

# Check service health
health:
	@echo "Checking service health..."
	@curl -s http://localhost:8080/health

# Generate load for testing
load-test:
	@echo "Generating load (100 requests)..."
	@for i in {1..100}; do \
		curl -s http://localhost:8080/user?id=1 > /dev/null & \
		curl -s http://localhost:8080/users > /dev/null & \
	done; wait
	@echo "Load test completed"

# Setup project structure (for new projects)
setup:
	@echo "Setting up project structure..."
	@mkdir -p {bin,deployments/{docker,k8s,monitoring},docs,scripts,test/fixtures}
	@echo "Project structure created"

# Clean up everything
clean:
	@echo "Cleaning up..."
	@docker-compose -f deployments/docker/docker-compose.yml down -v 2>/dev/null || true
	@docker system prune -f
	@rm -rf bin/
	@echo "Cleanup completed"

# Show help
help:
	@echo "Available commands:"
	@echo "  build              - Build the Go application"
	@echo "  run                - Run the application locally"
	@echo "  test               - Run all tests with coverage"
	@echo "  test-unit          - Run only unit tests"
	@echo "  test-integration   - Run only integration tests"
	@echo "  bench              - Run benchmarks"
	@echo "  docker-up          - Start monitoring stack with Docker"
	@echo "  docker-down        - Stop monitoring stack"
	@echo "  metrics            - View current metrics"
	@echo "  health             - Check service health"
	@echo "  load-test          - Generate test load"
	@echo "  setup              - Create project structure"
	@echo "  clean              - Clean up everything"
	@echo "  help               - Show this help"
