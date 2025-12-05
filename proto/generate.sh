#!/bin/bash
set -e

echo "Generating protobuf files..."

# Create output directories
mkdir -p ../pkg/protocol
mkdir -p ../internal/proto

# Generate Go code
protoc --go_out=../pkg/protocol \
       --go-grpc_out=../pkg/protocol \
       --go_opt=paths=source_relative \
       --go-grpc_opt=paths=source_relative \
       monitor.proto

# Generate Python code (optional)
if command -v protoc-gen-python &> /dev/null; then
    mkdir -p python
    protoc --python_out=python monitor.proto
    echo "Python files generated in proto/python/"
fi

# Generate TypeScript for dashboard
if command -v protoc-gen-ts &> /dev/null; then
    mkdir -p ../web/dashboard/src/proto
    protoc --plugin=protoc-gen-ts=./node_modules/.bin/protoc-gen-ts \
           --ts_out=service=grpc-web:../web/dashboard/src/proto \
           monitor.proto
    echo "TypeScript files generated in web/dashboard/src/proto/"
fi

echo "Protobuf generation complete!"