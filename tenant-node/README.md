# Tenant Node

The Tenant Node is the core data processing service that handles data storage, retrieval, and basic operations for individual tenants.

## Features

- Multi-format data storage (Parquet, JSON)
- RESTful and gRPC APIs
- Write-Ahead Logging (WAL)
- Indexing and search capabilities
- Data source management
- Real-time data ingestion

## Architecture

Each Tenant Node is a self-contained service that manages data for a specific tenant. Multiple tenant nodes can run independently and scale based on demand.

## Key Components

- `tenant_node.py` - Main tenant node implementation
- `data_source.py` - Data source management
- `grpc_service.py` - gRPC API implementation
- `rest_api.py` - REST API implementation
- `config.py` - Configuration management
- `wal.py` - Write-Ahead Logging
- `index.py` - Indexing functionality

## API

The Tenant Node exposes both REST and gRPC APIs for:
- Data ingestion (WriteData)
- Data querying (SearchData, AggregateData)
- Source management (AddSource, RemoveSource)
- Statistics and health monitoring

## Dependencies

- Pandas for data processing
- Parquet for columnar storage
- FastAPI for REST endpoints
- gRPC for high-performance communication
