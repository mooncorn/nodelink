#!/bin/bash

#!/bin/bash

# generate.sh - Script to regenerate protobuf code for the nodelink project

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔧 Setting up protobuf code generation...${NC}"

# Check if we're in the right directory
if [ ! -f "README.md" ] || [ ! -d "proto" ]; then
    echo -e "${RED}❌ Error: Please run this script from the nodelink project root directory${NC}"
    exit 1
fi

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo -e "${RED}❌ protoc is not installed. Please install Protocol Buffers compiler.${NC}"
    echo "   On Ubuntu/Debian: sudo apt-get install protobuf-compiler"
    echo "   On macOS: brew install protobuf"
    echo "   On other systems: https://grpc.io/docs/protoc-installation/"
    exit 1
fi

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed. Please install Go first.${NC}"
    exit 1
fi

echo -e "${YELLOW}📦 Installing Go protobuf dependencies...${NC}"

# Install required protobuf tools (globally)
echo "Installing protoc-gen-go..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

echo "Installing protoc-gen-go-grpc..."
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Ensure tools are in PATH
export PATH="$PATH:$(go env GOPATH)/bin"

# Verify tools are available
if ! command -v protoc-gen-go &> /dev/null; then
    echo -e "${RED}❌ protoc-gen-go not found in PATH. Make sure $(go env GOPATH)/bin is in your PATH${NC}"
    exit 1
fi

if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo -e "${RED}❌ protoc-gen-go-grpc not found in PATH. Make sure $(go env GOPATH)/bin is in your PATH${NC}"
    exit 1
fi

echo -e "${YELLOW}🔨 Generating Go code from protobuf files...${NC}"

# Create proto directories if they don't exist
mkdir -p server/internal/proto
mkdir -p agent/internal/proto

# Remove old generated files
echo "Cleaning up old generated files..."
rm -f server/internal/proto/*.pb.go
rm -f agent/internal/proto/*.pb.go
rm -f proto/*.pb.go  # Clean up old central location

# Generate Go code for server
echo "Generating protobuf files for server..."
cd proto
protoc --go_out=../server/internal/proto --go_opt=paths=source_relative \
       --go-grpc_out=../server/internal/proto --go-grpc_opt=paths=source_relative \
       *.proto
cd ..

# Generate Go code for agent  
echo "Generating protobuf files for agent..."
cd proto
protoc --go_out=../agent/internal/proto --go_opt=paths=source_relative \
       --go-grpc_out=../agent/internal/proto --go-grpc_opt=paths=source_relative \
       *.proto
cd ..

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Generated protobuf Go files:${NC}"
    echo "Server files:"
    ls -la server/internal/proto/*.pb.go
    echo "Agent files:"
    ls -la agent/internal/proto/*.pb.go
else
    echo -e "${RED}❌ Failed to generate protobuf files${NC}"
    exit 1
fi

echo -e "${YELLOW}🔄 Updating module dependencies...${NC}"

# Update server dependencies
echo "Updating server dependencies..."
cd server
if ! go mod tidy; then
    echo -e "${RED}❌ Failed to update server dependencies${NC}"
    exit 1
fi
cd ..

# Update agent dependencies  
echo "Updating agent dependencies..."
cd agent
if ! go mod tidy; then
    echo -e "${RED}❌ Failed to update agent dependencies${NC}"
    exit 1
fi
cd ..

echo -e "${GREEN}🎉 Protobuf code generation complete!${NC}"
echo ""
echo -e "${BLUE}📋 Summary:${NC}"
echo "   - Generated protobuf Go files in server/internal/proto/ and agent/internal/proto/"
echo "   - Updated module dependencies"
echo "   - Ready to build and run server and agent"
echo ""
echo -e "${BLUE}🚀 Next steps:${NC}"
echo "   - Run server: cd server && go run cmd/server/main.go"
echo "   - Run agent:  cd agent && go run cmd/agent/main.go"
