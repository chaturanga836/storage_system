# App Directory Structure

This directory contains configuration and code for interacting with Iceberg tables using PyIceberg and related tools.

## File Structure

```
app/
├── README.md                # This file
├── config.py                # Configuration for the app (env, settings, etc.)
├── main.py                  # Entrypoint for FastAPI or CLI app
├── pyiceberg.yaml           # PyIceberg catalog configuration for Nessie + MinIO
├── __init__.py              # Package marker
├── models/                  # Data models
│   ├── __init__.py
│   └── table_model.py
├── services/                # Service layer for Iceberg, MinIO, Nessie
│   ├── __init__.py
│   ├── iceberg_service.py
│   ├── minio_service.py
│   └── nessie_service.py
└── utils/                   # Utility modules (helpers, etc.)
    ├── __init__.py
    └── helpers.py
```

## Description
- **config.py**: App configuration (environment variables, settings, etc.)
- **main.py**: Entrypoint for your FastAPI app or CLI tool.
- **pyiceberg.yaml**: Configures the PyIceberg catalog to use Nessie (REST) and MinIO (S3-compatible) as the warehouse backend.
- **models/**: Data models for the application (e.g., Iceberg table schemas).
- **services/**: Service layer for interacting with Iceberg, MinIO, and Nessie.
- **utils/**: Utility/helper modules for common functions.
- **README.md**: Documentation for this directory.

## Next Steps
- Add your application logic to the appropriate modules.
- Adjust configuration and dependencies as your stack evolves.
