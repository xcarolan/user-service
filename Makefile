.PHONY: build run test test-unit test-integration bench docker-up docker-down metrics health load-test setup clean help doctor info _engine-check

# Choose container engine: docker | podman | auto
ENGINE ?= auto

COMPOSE_FILE ?= docker-compose.yml

# Detect availability
HAVE_DOCKER := $(shell command -v docker >/dev/null 2>&1 && echo yes || echo no)
HAVE_PODMAN := $(shell command -v podman >/dev/null 2>&1 && echo yes || echo no)
HAVE_PODMAN_COMPOSE := $(shell command -v podman >/dev/null 2>&1 && podman compose version >/dev/null 2>&1 && echo yes || echo no)
HAVE_PODMAN_COMPOSE_PY := $(shell command -v podman-compose >/dev/null 2>&1 && echo yes || echo no)

# Select engine (prefer Docker; fall back if requested engine missing)
ifeq ($(ENGINE),docker)
  ENGINE_SELECTED := docker
else ifeq ($(ENGINE),podman)
  ifeq ($(HAVE_PODMAN),yes)
    ENGINE_SELECTED := podman
  else ifeq ($(HAVE_DOCKER),yes)
    ENGINE_SELECTED := docker
    ENGINE_FALLBACK := yes
  else
    ENGINE_SELECTED := none
  endif
else
  ifeq ($(HAVE_DOCKER),yes)
    ENGINE_SELECTED := docker
  else ifeq ($(HAVE_PODMAN),yes)
    ENGINE_SELECTED := podman
  else
    ENGINE_SELECTED := none
  endif
endif

# Compose command for the selected engine
ifeq ($(ENGINE_SELECTED),docker)
  ENGINE_BIN := docker
  COMPOSE_CMD := docker compose
else ifeq ($(ENGINE_SELECTED),podman)
  ENGINE_BIN := podman
  ifeq ($(HAVE_PODMAN_COMPOSE),yes)
    COMPOSE_CMD := podman compose
  else ifeq ($(HAVE_PODMAN_COMPOSE_PY),yes)
    COMPOSE_CMD := podman-compose
  else
    COMPOSE_CMD := podman compose
  endif
else
  ENGINE_BIN := echo
  COMPOSE_CMD := echo
endif

# Host ports (match docker-compose.yml)
APP_PORT ?= 8082
PROM_PORT ?= 9090
GRAFANA_PORT ?= 3001
ALERT_PORT ?= 9093

# Docker variables
DOCKER_IMAGE ?= user-service
DOCKER_TAG ?= latest
DOCKER_REGISTRY ?= ghcr.io/$(shell git config --get remote.origin.url | sed 's/.*github.com[:/]\([^.]*\).*/\1/')

# Docker targets
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run:
	@echo "Running Docker container..."
	@docker run --rm -p 8082:8082 $(DOCKER_IMAGE):$(DOCKER_TAG)

docker-push:
	@echo "Pushing Docker image..."
	@docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)
	@docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)

docker-clean:
	@echo "Cleaning Docker images..."
	@docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG) 2>/dev/null || true
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

# Internal preflight
_engine-check:
	@if [ "$(ENGINE_SELECTED)" = "none" ]; then \
		echo "Error: No container engine found (install Docker or Podman)."; \
		echo "Hint: run with ENGINE=docker or ENGINE=podman once installed."; \
		exit 1; \
	fi
	@if [ "$(ENGINE_FALLBACK)" = "yes" ]; then \
		echo "Notice: Requested ENGINE=$(ENGINE) not found; falling back to $(ENGINE_SELECTED)."; \
	fi
	@if ! sh -c '$(COMPOSE_CMD) version >/dev/null 2>&1'; then \
		echo "Error: Compose command '$(COMPOSE_CMD)' not available."; \
		if [ "$(ENGINE_SELECTED)" = "docker" ]; then \
			echo "Ensure Docker is installed and 'docker compose' works."; \
		else \
			echo "Install podman-compose or a Podman version that supports 'podman compose'."; \
		fi; \
		exit 1; \
	fi

# Start the complete monitoring stack
docker-up: _engine-check
	@echo "Building images (ENGINE=$(ENGINE_SELECTED), COMPOSE=$(COMPOSE_CMD))..."
	@$(COMPOSE_CMD) -f $(COMPOSE_FILE) build --no-cache
	@echo "Starting monitoring stack..."
	@$(COMPOSE_CMD) -f $(COMPOSE_FILE) up -d
	@echo "Services available:"
	@echo "  Application:  http://localhost:$(APP_PORT)"
	@echo "  Metrics:      http://localhost:$(APP_PORT)/metrics"
	@echo "  Prometheus:   http://localhost:$(PROM_PORT)"
	@echo "  Grafana:      http://localhost:$(GRAFANA_PORT)"
	@echo "  AlertManager: http://localhost:$(ALERT_PORT)"

# Stop the monitoring stack
docker-down: _engine-check
	@echo "Stopping monitoring stack..."
	@$(COMPOSE_CMD) -f $(COMPOSE_FILE) down

# View metrics in terminal
metrics:
	@echo "Fetching current metrics..."
	@curl -s http://localhost:$(APP_PORT)/metrics

# Check service health
health:
	@echo "Checking service health..."
	@curl -s http://localhost:$(APP_PORT)/health

# Generate load for testing
load-test:
	@echo "Generating load (100 requests)..."
	@for i in {1..100}; do \
		curl -s http://localhost:$(APP_PORT)/user?id=1 > /dev/null & \
		curl -s http://localhost:$(APP_PORT)/users > /dev/null & \
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
	-@$(COMPOSE_CMD) -f $(COMPOSE_FILE) down -v 2>/dev/null || true
	@$(ENGINE_BIN) system prune -f
	@rm -rf bin/
	@echo "Cleanup completed"

# Diagnostics
doctor:
	@echo "Requested ENGINE:   $(ENGINE)"
	@echo "Selected ENGINE:    $(ENGINE_SELECTED)"
	@echo "Compose Command:    $(COMPOSE_CMD)"
	@echo "Compose File:       $(COMPOSE_FILE)"
	@echo "DOCKER_HOST:        $${DOCKER_HOST:-<not set>}"
	@sh -c '$(COMPOSE_CMD) version >/dev/null 2>&1' && echo "$(COMPOSE_CMD): available" || echo "$(COMPOSE_CMD): not available"

info: doctor

# Show help
help:
	@echo "Available commands:"
	@echo "  build              - Build the Go application"
	@echo "  run                - Run the application locally"
	@echo "  test               - Run all tests with coverage"
	@echo "  test-unit          - Run only unit tests"
	@echo "  test-integration   - Run only integration tests"
	@echo "  bench              - Run benchmarks"
	@echo "  docker-up          - Start monitoring stack (auto-detects engine)"
	@echo "  docker-down        - Stop monitoring stack"
	@echo "  metrics            - View current metrics"
	@echo "  health             - Check service health"
	@echo "  load-test          - Generate test load"
	@echo "  setup              - Create project structure"
	@echo "  clean              - Clean up everything"
	@echo "  doctor|info        - Show diagnostics"
	@echo "  help               - Show this help"