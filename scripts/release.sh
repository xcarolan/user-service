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
