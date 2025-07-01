# Shared Protocol Buffers

This directory contains the shared protobuf definitions used across all microservices in the storage system.

## Files

- `storage.proto` - Main storage service definitions
- Generated gRPC files will be placed here after compilation

## Usage

To generate Python gRPC files:

```bash
python -m grpc_tools.protoc --proto_path=. --python_out=. --grpc_python_out=. storage.proto
```

## Dependencies

Each microservice that uses these protos should include them as a dependency.
