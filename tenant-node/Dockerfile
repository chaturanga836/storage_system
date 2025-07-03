FROM python:3.11-slim

WORKDIR /app

# Copy requirements and install dependencies
COPY tenant-node/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy shared protos
COPY shared-protos ./shared-protos

# Copy service code
COPY tenant-node/ .

# Generate protobuf files
RUN cd shared-protos && python -m grpc_tools.protoc --proto_path=. --python_out=. --grpc_python_out=. storage.proto

# Expose ports
EXPOSE 8000 50051

# Run the service
CMD ["python", "-m", "main"]
