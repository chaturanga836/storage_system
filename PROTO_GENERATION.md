# Manual Proto Generation for Windows

Since protoc is not installed, let's create a simplified approach for development.

## Option 1: Install Protoc (Recommended)

Download protoc from: https://github.com/protocolbuffers/protobuf/releases
1. Download protoc-VERSION-win64.zip
2. Extract to C:\protoc
3. Add C:\protoc\bin to PATH
4. Run: protoc --go_out=proto/generated/go --go-grpc_out=proto/generated/go --proto_path=proto proto/storage/*.proto

## Option 2: Use Go-based generation (Current approach)

We'll create simplified implementations that match the proto definitions.

## Files that need to be generated:
- proto/generated/go/storage/common.pb.go
- proto/generated/go/storage/ingestion.pb.go
- proto/generated/go/storage/ingestion_grpc.pb.go
- proto/generated/go/storage/query.pb.go  
- proto/generated/go/storage/query_grpc.pb.go
