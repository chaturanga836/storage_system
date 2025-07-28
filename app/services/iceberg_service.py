# app/services/iceberg_service.py

from pyiceberg.catalog import Catalog
from .nessie_service import NessieService
from pyiceberg.exceptions import TableAlreadyExistsError, NoSuchTableError
from pyiceberg.schema import Schema, StructType, NestedField
from pyiceberg.types import (
    LongType, StringType, TimestampType, DoubleType,
    FloatType, IntegerType, BooleanType, DateType, DecimalType,
    UUIDType
)
from pyiceberg.expressions import transforms
from pyiceberg.table import Table
import pyarrow as pa
import pyarrow.parquet as pq
import pandas as pd # For convenience in data preparation
import logging

# Assuming you have a config.py to load your pyiceberg.yaml configuration
from ..config import AppConfig # Assuming AppConfig has nessie_catalog_name and pyiceberg_config_path

logger = logging.getLogger(__name__)

class IcebergService:
    def __init__(self, nessie_service: NessieService):
        """
        Initializes the IcebergService.
        :param nessie_service: An instance of NessieService to get the catalog.
        """
        self.nessie_service = nessie_service
        self._catalog: Catalog = None

    def get_catalog(self) -> Catalog:
        """Helper to get the PyIceberg catalog instance."""
        if self._catalog is None:
            self._catalog = self.nessie_service.get_catalog()
        return self._catalog

    def create_iceberg_table(
        self,
        identifier: str | tuple[str],
        schema: Schema,
        partition_spec: transforms.PartitionSpec | None = None,
        properties: dict | None = None,
        location: str | None = None, # Optional: specifies exact location, otherwise uses warehouse + identifier
        overwrite: bool = False # If true, drops and recreates table if exists
    ) -> Table:
        """
        Creates a new Iceberg table or loads an existing one.
        :param identifier: Table identifier (e.g., "default.my_table").
        :param schema: PyIceberg Schema object.
        :param partition_spec: Optional PyIceberg PartitionSpec.
        :param properties: Optional dictionary of table properties.
        :param location: Optional explicit table location.
        :param overwrite: If True, drops table if exists before creating.
        :return: The created or loaded Iceberg Table instance.
        """
        catalog = self.get_catalog()
        full_properties = properties or {}
        # Set format version 2 for modern features if not specified
        if "format-version" not in full_properties:
            full_properties["format-version"] = "2"

        try:
            if overwrite:
                try:
                    catalog.drop_table(identifier)
                    logger.warning(f"Existing table '{identifier}' dropped for overwrite.")
                except NoSuchTableError:
                    logger.info(f"Table '{identifier}' did not exist for overwrite.")
                except Exception as e:
                    logger.error(f"Failed to drop table '{identifier}' during overwrite: {e}")
                    raise

            table = catalog.create_table(
                identifier=identifier,
                schema=schema,
                partition_spec=partition_spec,
                properties=full_properties,
                location=location
            )
            logger.info(f"Table '{identifier}' created successfully at location: {table.location}")
            return table
        except TableAlreadyExistsError:
            logger.warning(f"Table '{identifier}' already exists. Loading existing table.")
            return catalog.load_table(identifier)
        except Exception as e:
            logger.error(f"Failed to create or load table '{identifier}': {e}")
            raise

    def load_iceberg_table(self, identifier: str | tuple[str]) -> Table:
        """
        Loads an existing Iceberg table.
        :param identifier: Table identifier (e.g., "default.my_table").
        :return: The loaded Iceberg Table instance.
        """
        catalog = self.get_catalog()
        try:
            table = catalog.load_table(identifier)
            logger.info(f"Table '{identifier}' loaded successfully.")
            return table
        except NoSuchTableError:
            logger.error(f"Table '{identifier}' does not exist.")
            raise
        except Exception as e:
            logger.error(f"Failed to load table '{identifier}': {e}")
            raise

    def append_data(self, table: Table, data: pd.DataFrame):
        """
        Appends data to an Iceberg table.
        :param table: The PyIceberg Table object to append to.
        :param data: A Pandas DataFrame containing the data to append.
                     The DataFrame schema should be compatible with the table's schema.
        """
        # Convert Pandas DataFrame to PyArrow Table
        arrow_table = pa.Table.from_pandas(data, schema=table.schema().as_arrow(), preserve_index=False)
        # Note: PyIceberg's append currently works reliably for unpartitioned tables.
        # For partitioned tables, you might need specific versions or other tools.
        # Ensure your PyIceberg version (e.g., 0.6.0+) supports writes.
        try:
            table.append(arrow_table)
            logger.info(f"Appended {len(data)} rows to table '{table.name}'.")
        except Exception as e:
            logger.error(f"Failed to append data to table '{table.name}': {e}")
            raise

    def read_data(self, table: Table) -> pd.DataFrame:
        """
        Reads all data from an Iceberg table into a Pandas DataFrame.
        Note: For large tables, this might be memory intensive.
              Consider using DuckDB or other query engines for production reads.
        :param table: The PyIceberg Table object to read from.
        :return: A Pandas DataFrame containing the table data.
        """
        try:
            # PyIceberg can integrate with DuckDB for efficient reads.
            # If DuckDB is installed, table.to_pandas() uses it.
            # Otherwise, it uses PyArrow's default readers.
            df = table.to_pandas()
            logger.info(f"Read {len(df)} rows from table '{table.name}'.")
            return df
        except Exception as e:
            logger.error(f"Failed to read data from table '{table.name}': {e}")
            raise

    def evolve_table_schema(self, table: Table, new_schema_fields: dict, method: str = "add_columns"):
        """
        Evolves the schema of an Iceberg table.
        :param table: The PyIceberg Table object.
        :param new_schema_fields: A dictionary mapping column names to pyiceberg.types.Type instances.
                                  e.g., {"new_column": StringType(), "another_col": LongType()}
        :param method: "add_columns" (default) or other future methods (rename, drop, etc.).
        """
        try:
            with table.update_schema() as update:
                if method == "add_columns":
                    for name, field_type in new_schema_fields.items():
                        # Using NestedField.required=False for new columns unless specified
                        update.add_column(name, field_type, required=False)
                else:
                    raise NotImplementedError(f"Schema evolution method '{method}' not yet implemented.")
            logger.info(f"Schema for table '{table.name}' evolved successfully using method '{method}'.")
        except Exception as e:
            logger.error(f"Failed to evolve schema for table '{table.name}': {e}")
            raise

    def drop_table(self, identifier: str | tuple[str]):
        """
        Drops an Iceberg table from the catalog.
        :param identifier: Table identifier (e.g., "default.my_table").
        """
        catalog = self.get_catalog()
        try:
            catalog.drop_table(identifier)
            logger.info(f"Table '{identifier}' dropped successfully.")
        except NoSuchTableError:
            logger.warning(f"Table '{identifier}' does not exist, skipping drop.")
        except Exception as e:
            logger.error(f"Failed to drop table '{identifier}': {e}")
            raise


# Example usage (would typically be in main.py)
if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)

    # You'd need to mock or properly import your config.py here for standalone testing
    # For actual execution, ensure AppConfig is loaded
    class MockAppConfig:
        nessie_catalog_name = "pyiceberg_nessie"
        # Assuming pyiceberg.yaml is in the parent directory of 'services'
        # For this standalone test, adjust path if needed:
        pyiceberg_config_path = os.path.join(os.path.dirname(__file__), "../pyiceberg.yaml")
        minio_endpoint = "http://localhost:9000"
        minio_access_key = "minioadmin"
        minio_secret_key = "minioadmin"

    # Initialize services
    nessie_svc = NessieService(MockAppConfig.nessie_catalog_name, MockAppConfig.pyiceberg_config_path)
    iceberg_svc = IcebergService(nessie_svc)

    test_table_name = "default.user_events_py"
    test_namespace = "default"

    # Define a simple schema
    event_schema = Schema(
        StructType(
            NestedField(1, "event_id", StringType(), required=True),
            NestedField(2, "user_id", StringType(), required=True),
            NestedField(3, "event_timestamp", TimestampType(), required=True),
            NestedField(4, "event_type", StringType(), required=False),
            NestedField(5, "value", DoubleType(), required=False)
        )
    )

    try:
        # 1. Ensure namespace exists
        print(f"\n--- Ensuring namespace '{test_namespace}' exists ---")
        nessie_svc.create_namespace(test_namespace)

        # 2. Create the table
        print(f"\n--- Creating Iceberg Table '{test_table_name}' ---")
        table = iceberg_svc.create_iceberg_table(
            identifier=test_table_name,
            schema=event_schema,
            # For partitioning: partition_spec=transforms.LocalDayTransform(field_id=3, source_id=3),
            overwrite=True # Good for testing to start fresh
        )
        print(f"Table schema: {table.schema()}")
        # print(f"Table properties: {table.properties}")


        # 3. Append some data
        print(f"\n--- Appending data to '{test_table_name}' ---")
        data_to_append = pd.DataFrame([
            {"event_id": "e001", "user_id": "u001", "event_timestamp": "2023-01-01T10:00:00Z", "event_type": "login", "value": 1.0},
            {"event_id": "e002", "user_id": "u002", "event_timestamp": "2023-01-01T10:05:00Z", "event_type": "view_product", "value": 2.5},
        ])
        iceberg_svc.append_data(table, data_to_append)

        # 4. Read data
        print(f"\n--- Reading data from '{test_table_name}' ---")
        read_df = iceberg_svc.read_data(table)
        print(read_df)

        # 5. Evolve schema (add a column)
        print(f"\n--- Evolving schema: Adding 'source_ip' column ---")
        iceberg_svc.evolve_table_schema(table, {"source_ip": StringType()})
        # Reload table to get updated schema
        table_reloaded = iceberg_svc.load_iceberg_table(test_table_name)
        print(f"Updated table schema: {table_reloaded.schema()}")

        # 6. Append more data with the new column (and nulls for old rows implicitly)
        print(f"\n--- Appending more data with new schema ---")
        more_data = pd.DataFrame([
            {"event_id": "e003", "user_id": "u003", "event_timestamp": "2023-01-01T10:10:00Z", "event_type": "logout", "value": 0.0, "source_ip": "192.168.1.10"},
            {"event_id": "e004", "user_id": "u001", "event_timestamp": "2023-01-01T10:15:00Z", "event_type": "add_to_cart", "value": 5.0, "source_ip": "192.168.1.11"},
        ])
        iceberg_svc.append_data(table_reloaded, more_data) # Use reloaded table

        # 7. Read all data again
        print(f"\n--- Reading all data (after schema evolution and more appends) ---")
        final_df = iceberg_svc.read_data(table_reloaded)
        print(final_df)


    except Exception as e:
        print(f"IcebergService test failed: {e}")
    finally:
        # Optional: Clean up the table after testing
        # print(f"\n--- Cleaning up: Dropping table '{test_table_name}' ---")
        # iceberg_svc.drop_table(test_table_name)
        pass