# app/main.py

from fastapi import FastAPI, Depends, HTTPException
import logging
import os
from contextlib import asynccontextmanager # <--- Import this!

# Assuming you'll have a config.py to manage settings
from .config import load_app_config, AppSettings # Import the loading function

# Import your services
from .services import NessieService, IcebergService, MinioService

# Define your example schema and table identifier (as in previous example)
from pyiceberg.schema import Schema, StructType, NestedField
from pyiceberg.types import StringType, TimestampType, DoubleType
import pandas as pd
import datetime

EXAMPLE_SCHEMA = Schema(
    StructType(
        NestedField(1, "id", StringType(), required=True),
        NestedField(2, "value", DoubleType(), required=False),
        NestedField(3, "timestamp", TimestampType(), required=True)
    )
)
EXAMPLE_TABLE_IDENTIFIER = "default.my_fastapi_table"


logger = logging.getLogger(__name__)

# --- Application Setup with Lifespan ---

# Global variables for service instances
# These will be initialized within the lifespan context
app_config: AppSettings = None
nessie_service_instance: NessieService = None
minio_service_instance: MinioService = None
iceberg_service_instance: IcebergService = None

@asynccontextmanager
async def lifespan(app: FastAPI):
    """
    Context manager for application startup and shutdown events.
    Code before 'yield' runs on startup.
    Code after 'yield' runs on shutdown.
    """
    global app_config, nessie_service_instance, minio_service_instance, iceberg_service_instance

    # --- Startup Logic ---
    logger.info("Application startup event triggered. Initializing services...")

    # 1. Load configuration
    app_config = load_app_config()
    logger.info("Application configuration loaded.")

    # 2. Initialize MinIO Service
    minio_service_instance = MinioService(
        endpoint_url=app_config.MINIO_ENDPOINT,
        access_key=app_config.MINIO_ACCESS_KEY,
        secret_key=app_config.MINIO_SECRET_KEY,
        region_name=app_config.MINIO_REGION
    )
    # Ensure MinIO warehouse bucket exists
    try:
        minio_service_instance.create_bucket_if_not_exists(app_config.ICEBERG_WAREHOUSE_BUCKET)
        logger.info(f"MinIO warehouse bucket '{app_config.ICEBERG_WAREHOUSE_BUCKET}' ensured.")
    except Exception as e:
        logger.error(f"Failed to ensure Minio bucket on startup: {e}")
        # It's critical to have the bucket, so raise to prevent app start
        raise RuntimeError("Failed to connect to MinIO on startup. Check MinIO service and credentials.") from e

    # 3. Initialize Nessie Service
    nessie_service_instance = NessieService(
        catalog_name=app_config.NESSIE_CATALOG_NAME,
        config_file_path=app_config.PYICEBERG_CONFIG_PATH
    )
    logger.info(f"Nessie Service initialized for catalog '{app_config.NESSIE_CATALOG_NAME}'.")

    # 4. Initialize Iceberg Service
    iceberg_service_instance = IcebergService(
        nessie_service=nessie_service_instance
    )
    logger.info("Iceberg Service initialized.")

    # You can also store these instances on app.state for access in dependencies if preferred
    app.state.minio_service = minio_service_instance
    app.state.nessie_service = nessie_service_instance
    app.state.iceberg_service = iceberg_service_instance

    logger.info("Application startup complete. Ready to serve requests.")

    yield # This is where the application starts receiving requests

    # --- Shutdown Logic (code after yield) ---
    logger.info("Application shutdown event triggered. Cleaning up resources...")
    # Add any cleanup logic here if necessary
    # For services like Minio/Nessie/Iceberg, there might not be explicit 'close' methods
    # unless you establish persistent connections that need closing.
    # In this specific case, our services are mostly stateless or handle connections internally
    # on demand, so explicit cleanup here might not be strictly necessary,
    # but for DB connections, etc., this is where you'd close them.
    logger.info("Application shutdown complete.")

# --- FastAPI Application ---
# Pass the lifespan context manager to the FastAPI app
app = FastAPI(title=app_config.APP_NAME if 'app_config' in locals() else "Iceberg Data Lakehouse App", lifespan=lifespan)


@app.get("/")
async def read_root():
    return {"message": "Welcome to the Iceberg Data Lakehouse API!"}

@app.get("/health")
async def health_check():
    # Simple health check, could be extended to check connections
    # Access services via global instances for simple use cases
    # For more complex apps, use FastAPI's Depends with a callable that returns the service
    try:
        # Example: Try to get the catalog to check Nessie connection
        catalog = iceberg_service_instance.get_catalog()
        # You could also try listing namespaces or buckets for a deeper check
        return {"status": "ok", "catalog_name": catalog.name}
    except Exception as e:
        logger.error(f"Health check failed: {e}")
        raise HTTPException(status_code=503, detail=f"Service Unavailable: {e}")


@app.post("/data/{item_id}")
async def write_data_to_iceberg(item_id: str, value: float):
    """
    Example endpoint to write data to an Iceberg table.
    """
    try:
        # 1. Ensure the namespace exists
        # Access services via global instances
        await nessie_service_instance.create_namespace("default")

        # 2. Ensure the table exists (or create it)
        table = await iceberg_service_instance.create_iceberg_table(
            identifier=EXAMPLE_TABLE_IDENTIFIER,
            schema=EXAMPLE_SCHEMA,
            overwrite=False
        )

        # 3. Prepare data
        data_to_append = pd.DataFrame([{
            "id": item_id,
            "value": value,
            "timestamp": datetime.datetime.now(datetime.timezone.utc)
        }])

        # 4. Append data
        await iceberg_service_instance.append_data(table, data_to_append)

        return {"status": "success", "message": f"Data appended to {EXAMPLE_TABLE_IDENTIFIER}"}
    except Exception as e:
        logger.exception(f"Error writing data to Iceberg: {e}")
        raise HTTPException(status_code=500, detail=f"Failed to write data: {e}")

@app.get("/data")
async def read_data_from_iceberg():
    """
    Example endpoint to read data from an Iceberg table.
    """
    try:
        table = await iceberg_service_instance.load_iceberg_table(EXAMPLE_TABLE_IDENTIFIER)
        df = await iceberg_service_instance.read_data(table)
        return {"data": df.to_dict(orient="records")}
    except Exception as e:
        logger.exception(f"Error reading data from Iceberg: {e}")
        raise HTTPException(status_code=500, detail=f"Failed to read data: {e}")

# To run this file: uvicorn app.main:app --reload --host 0.0.0.0 --port 8000
# Make sure your docker-compose services (Nessie, MinIO) are running.