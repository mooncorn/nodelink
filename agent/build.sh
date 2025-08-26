#!/bin/bash

# Nodelink Agent Build Script

set -e

# Get version from git tag or default to "dev"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Building Nodelink Agent (version: ${VERSION})${NC}"

# Download and verify dependencies
echo -e "${YELLOW}Updating dependencies...${NC}"
go mod tidy
go mod verify

# Create bin directory
mkdir -p bin

# Build agent
echo -e "${YELLOW}Building agent...${NC}"
go build -ldflags "-X main.Version=${VERSION}" -o bin/nodelink-agent ./cmd/agent

# Build updater
echo -e "${YELLOW}Building updater...${NC}"
go build -ldflags "-X main.Version=${VERSION}" -o bin/nodelink-updater ./cmd/updater

# Make binaries executable
chmod +x bin/nodelink-agent bin/nodelink-updater

echo -e "${GREEN}Build complete!${NC}"
echo "Binaries created:"
echo "  - bin/nodelink-agent"
echo "  - bin/nodelink-updater"
echo
echo "Test version:"
./bin/nodelink-agent -version
echo
echo "To install locally:"
echo "  sudo cp bin/nodelink-agent /usr/local/bin/"
echo "  sudo cp bin/nodelink-updater /usr/local/bin/"
