#!/bin/bash

# Generate Protocol Buffer files for Go and Python

set -e

echo "🔧 Generating Protocol Buffer files..."

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo "❌ protoc is not installed. Please install Protocol Buffers compiler."
    exit 1
fi

# Create output directories
mkdir -p proto/generated/go
mkdir -p proto/generated/python

# Generate Go files
echo "📦 Generating Go protobuf files..."
protoc --go_out=proto/generated/go \
       --go-grpc_out=proto/generated/go \
       --proto_path=proto \
       proto/storage/*.proto

# Generate Python files (optional, for client integration)
echo "🐍 Generating Python protobuf files..."
protoc --python_out=proto/generated/python \
       --grpc_python_out=proto/generated/python \
       --proto_path=proto \
       proto/storage/*.proto

echo "✅ Protocol Buffer generation completed!"
echo "📁 Go files: proto/generated/go/"
echo "📁 Python files: proto/generated/python/"
