# app/config.py

import os
from pathlib import Path
import yaml
from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict
from dotenv import load_dotenv

# Load environment variables from .env file at the project root
# This ensures that variables in .env are available before Pydantic reads them.
# find_dotenv() searches upwards for the .env file.
load_dotenv(dotenv_path=Path(__file__).parent.parent / ".env")

# Determine the path to pyiceberg.yaml
# Assuming pyiceberg.yaml is in the same directory as config.py or one level up (app/)
PYICEBERG_CONFIG_FILE = Path(__file__).parent.parent / "pyiceberg.yaml"

class AppSettings(BaseSettings):
    """
    Application settings, loaded from environment variables and default values.
    Pydantic's BaseSettings automatically reads environment variables
    that match the field names (case-insensitive by default).
    """

    # --- Application General Settings ---
    APP_NAME: str = "Iceberg Data Lakehouse API"
    ENVIRONMENT: str = Field("development", env="APP_ENV") # Can be 'development', 'production', etc.

    # --- Nessie Catalog Configuration ---
    # The name of the catalog to use from pyiceberg.yaml
    NESSIE_CATALOG_NAME: str = Field("dev_nessie_catalog", env="NESSIE_CATALOG_NAME")
    PYICEBERG_CONFIG_PATH: Path = PYICEBERG_CONFIG_FILE

    # --- MinIO Connection Details ---
    # These will typically come from your .env file
    MINIO_ENDPOINT: str = Field(..., env="MINIO_ENDPOINT") # '...' means required
    MINIO_ACCESS_KEY: str = Field(..., env="MINIO_ACCESS_KEY")
    MINIO_SECRET_KEY: str = Field(..., env="MINIO_SECRET_KEY")
    MINIO_REGION: str = Field("us-east-1", env="MINIO_REGION") # Default region

    # --- Iceberg Warehouse Settings ---
    # This is the bucket name where Iceberg tables will store their data files
    ICEBERG_WAREHOUSE_BUCKET: str = Field("iceberg-warehouse", env="ICEBERG_WAREHOUSE_BUCKET")

    # --- Uvicorn Server Settings (if needed in config, otherwise in uvicorn command) ---
    UVICORN_HOST: str = Field("0.0.0.0", env="UVICORN_HOST")
    UVICORN_PORT: int = Field(8000, env="UVICORN_PORT")
    UVICORN_RELOAD: bool = Field(True, env="UVICORN_RELOAD") # Good for development

    # Pydantic Settings configuration
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        # Optionally, define a prefix for environment variables if you want to avoid clashes
        # env_prefix="APP_" # If set, MINIO_ENDPOINT would be APP_MINIO_ENDPOINT
    )

    # You could add methods here if you need to load specific parts
    # or perform validation based on multiple fields.
    @property
    def is_production(self) -> bool:
        return self.ENVIRONMENT == "production"

# Create a singleton instance of the configuration to be imported across the app
# Use a method to load it, so you can call it explicitly from main.py
# This prevents it from loading prematurely if config is imported by other files
# during module initialization before .env is ready.
_app_config: AppSettings = None

def load_app_config() -> AppSettings:
    global _app_config
    if _app_config is None:
        _app_config = AppSettings()
    return _app_config

# You can also add a function to get pyiceberg catalog conf directly
# if you need to pass specific dicts, but `load_catalog` usually handles it
def get_pyiceberg_catalog_conf(catalog_name: str, config_file_path: Path) -> dict:
    """
    Loads a specific catalog's configuration from pyiceberg.yaml.
    """
    if not config_file_path.exists():
        raise FileNotFoundError(f"PyIceberg config file not found at: {config_file_path}")

    with open(config_file_path, 'r') as f:
        full_config = yaml.safe_load(f)

    catalog_conf = full_config.get(catalog_name)
    if not catalog_conf:
        raise ValueError(f"Catalog '{catalog_name}' not found in {config_file_path}")

    return catalog_conf