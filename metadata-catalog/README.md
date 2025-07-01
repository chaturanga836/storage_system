# Metadata Catalog

The Metadata Catalog service manages metadata, schema information, and file compaction across the storage system.

## Features

- Centralized metadata management
- Schema registry and evolution
- File compaction coordination
- Partition metadata tracking
- Index management
- Storage statistics

## Architecture

The Metadata Catalog serves as the central repository for all metadata in the storage system, enabling efficient query planning and data organization.

## Key Components

- `metadata.py` - Core metadata management
- `compaction_manager.py` - File compaction coordination
- `schema_registry.py` - Schema management and evolution
- `partition_manager.py` - Partition metadata tracking

## API

The Metadata Catalog exposes APIs for:
- Metadata CRUD operations
- Schema registration and retrieval
- Compaction scheduling and monitoring
- Statistics collection

## Dependencies

- Database for metadata storage (PostgreSQL, etc.)
- Schema validation libraries
- Distributed coordination tools
