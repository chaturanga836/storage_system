#!/bin/bash

# Build all Go services

set -e

echo "🏗️ Building Storage Engine services..."

# Clean previous builds
rm -rf bin/
mkdir -p bin

# Build ingestion server
echo "📥 Building Ingestion Server..."
go build -o bin/ingestion-server ./cmd/ingestion-server

# Build query server
echo "🔍 Building Query Server..."
go build -o bin/query-server ./cmd/query-server

# Build data processor
echo "⚙️ Building Data Processor..."
go build -o bin/data-processor ./cmd/data-processor

# Build admin CLI
echo "🛠️ Building Admin CLI..."
go build -o bin/storage-admin ./cmd/admin-cli

echo "✅ Build completed successfully!"
echo "📁 Binaries available in: bin/"
echo "  - ingestion-server"
echo "  - query-server" 
echo "  - data-processor"
echo "  - storage-admin"
