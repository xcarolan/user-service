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
