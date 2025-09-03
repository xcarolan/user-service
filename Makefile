# =============================================================================
# Makefile - Convenient commands
# =============================================================================
.PHONY: build run test docker-up docker-down metrics clean help

# Build the Go application
build:
	@echo "Building user service..."
	@go build -o bin/user-service .

# Run the application locally
run:
	@echo "Starting user service..."
	@go run .

# Run tests with coverage
test:
	@echo "Running tests..."
	@go test -v -cover ./...

# Run only unit tests
test-unit:
	@echo "Running unit tests..."
	@go test -run "^Test[^I]" -v

# Run only integration tests
test-integration:
	@echo "Running integration tests..."
	@go test -run "^TestIntegration" -v

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem

# Start the complete monitoring stack
docker-up:
	@echo "Starting monitoring stack..."
	@docker-compose up -d
	@echo "Services available:"
	@echo "  Application:  http://localhost:8080"
	@echo "  Metrics:      http://localhost:8080/metrics"
	@echo "  Prometheus:   http://localhost:9090"
	@echo "  Grafana:      http://localhost:3000 (admin/admin123)"
	@echo "  AlertManager: http://localhost:9093"

# Stop the monitoring stack
docker-down:
	@echo "Stopping monitoring stack..."
	@docker-compose down

# View metrics in terminal
metrics:
	@echo "Fetching current metrics..."
	@curl -s http://localhost:8080/metrics

# Check service health
health:
	@echo "Checking service health..."
	@curl -s http://localhost:8080/health | jq

# Generate load for testing
load-test:
	@echo "Generating load (100 requests)..."
	@for i in {1..100}; do \
		curl -s http://localhost:8080/user?id=1 > /dev/null & \
		curl -s http://localhost:8080/users > /dev/null & \
	done; wait
	@echo "Load test completed"

# View Prometheus targets
prometheus-targets:
	@echo "Checking Prometheus targets..."
	@curl -s http://localhost:9090/api/v1/targets | jq

# View logs from all services
logs:
	@docker-compose logs -f

# Setup project structure
setup:
	@echo "Setting up project structure..."
	@mkdir -p grafana/provisioning/datasources
	@mkdir -p grafana/provisioning/dashboards
	@mkdir -p grafana/dashboards
	@mkdir -p bin
	@echo "Project structure created"

# Clean up everything
clean:
	@echo "Cleaning up..."
	@docker-compose down -v
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
	@echo "  prometheus-targets - Check Prometheus targets"
	@echo "  logs               - View all service logs"
	@echo "  setup              - Create project structure"
	@echo "  clean              - Clean up everything"
	@echo "  help               - Show this help"

---