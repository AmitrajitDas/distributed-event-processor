#!/bin/bash

# Script to generate Go code from protobuf definitions
# This script generates both protobuf and gRPC code

set -e

echo "Generating Go code from protobuf definitions..."

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Output directory for generated code
OUT_DIR="$SCRIPT_DIR"

echo "Script directory: $SCRIPT_DIR"
echo "Project root: $PROJECT_ROOT"
echo "Output directory: $OUT_DIR"

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo "Error: protoc is not installed"
    echo "Please install Protocol Buffer Compiler:"
    echo "  macOS: brew install protobuf"
    echo "  Linux: apt-get install -y protobuf-compiler"
    exit 1
fi

# Check protoc version
PROTOC_VERSION=$(protoc --version | awk '{print $2}')
echo "Found protoc version: $PROTOC_VERSION"

# Check if protoc-gen-go is installed
if ! command -v protoc-gen-go &> /dev/null; then
    echo "Installing protoc-gen-go..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

# Check if protoc-gen-go-grpc is installed
if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "Installing protoc-gen-go-grpc..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

echo "All required tools are installed"
echo ""

# Generate code for each proto file
for proto_file in $(find "$SCRIPT_DIR" -name "*.proto"); do
    echo "Generating code for: $(basename $proto_file)"

    protoc \
        --proto_path="$SCRIPT_DIR" \
        --go_out="$OUT_DIR" \
        --go_opt=paths=source_relative \
        --go-grpc_out="$OUT_DIR" \
        --go-grpc_opt=paths=source_relative \
        "$proto_file"

    echo "  Generated Go code"
done

echo ""
echo "Code generation complete!"
echo ""
echo "Generated files:"
find "$OUT_DIR" -name "*.pb.go" -o -name "*_grpc.pb.go" | while read file; do
    echo "  $file"
done