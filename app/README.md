# App Directory Structure

This directory contains configuration and code for interacting with Iceberg tables using PyIceberg and related tools.

## File Structure

```
app/
├── pyiceberg.yaml         # PyIceberg catalog configuration for Nessie + MinIO
├── requirements.txt       # Python dependencies for data lake operations
├── .env                   # (Optional) Environment variables for local dev
├── main.py                # (Optional) Entrypoint for FastAPI or CLI app
├── notebooks/             # Jupyter notebooks for exploration and analytics
│   └── example.ipynb
├── etl/                   # ETL scripts and data pipelines
│   └── load_user_activity.py
├── api/                   # FastAPI app or API-related code
│   ├── __init__.py
│   ├── routes.py
│   └── models.py
└── utils/                 # Utility modules (helpers, config loaders, etc.)
    └── s3_helpers.py
```

## Description
- **pyiceberg.yaml**: Configures the PyIceberg catalog to use Nessie (REST) and MinIO (S3-compatible) as the warehouse backend. Adjust endpoints and credentials as needed for your environment.
- **requirements.txt**: Lists all Python dependencies for working with Iceberg, S3, Trino, Nessie, and related tools.
- **.env**: (Optional) Store environment variables for local development.
- **main.py**: (Optional) Entrypoint for your FastAPI app or CLI tool.
- **notebooks/**: Place Jupyter notebooks here for data exploration and analytics.
- **etl/**: ETL scripts and data pipeline code.
- **api/**: FastAPI app code, including routes and models.
- **utils/**: Utility/helper modules for S3, config, etc.

## Example Usage
- Use `pyiceberg` CLI or Python API to interact with Iceberg tables:
  ```sh
  pyiceberg -c pyiceberg.yaml list-tables pyiceberg_nessie
  ```
- Develop and run Python scripts for ETL, analytics, or serving APIs using the dependencies in `requirements.txt`.

## Next Steps
- Add your application code (FastAPI, ETL, notebooks) to this directory.
- Adjust `pyiceberg.yaml` and `requirements.txt` as your stack evolves.
