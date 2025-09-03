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
