# app/services/nessie_service.py

from pyiceberg.catalog import Catalog, load_catalog
from pyiceberg.exceptions import NoSuchTableError, TableAlreadyExistsError, NoSuchNamespaceError
import logging

logger = logging.getLogger(__name__)

class NessieService:
    def __init__(self, catalog_name: str, config_file_path: str):
        """
        Initializes the NessieService.
        :param catalog_name: The name of the catalog defined in pyiceberg.yaml.
        :param config_file_path: Path to the pyiceberg.yaml configuration file.
        """
        self.catalog_name = catalog_name
        self.config_file_path = config_file_path
        self._catalog = None

    def get_catalog(self) -> Catalog:
        """
        Loads and returns the PyIceberg Catalog instance connected to Nessie.
        Caches the catalog instance for subsequent calls.
        """
        if self._catalog is None:
            logger.info(f"Loading PyIceberg catalog '{self.catalog_name}' from '{self.config_file_path}'...")
            try:
                # 'conf' parameter is used to pass file path or programmatic config
                self._catalog = load_catalog(name=self.catalog_name, conf={"file": self.config_file_path})
                logger.info(f"Successfully loaded Nessie catalog: {self._catalog.name}")
            except Exception as e:
                logger.error(f"Failed to load Nessie catalog '{self.catalog_name}': {e}")
                raise
        return self._catalog

    def list_namespaces(self) -> list[tuple[str]]:
        """Lists all namespaces in the Nessie catalog."""
        catalog = self.get_catalog()
        namespaces = catalog.list_namespaces()
        logger.debug(f"Found namespaces: {namespaces}")
        return namespaces

    def create_namespace(self, namespace_identifier: str | tuple[str]) -> None:
        """
        Creates a new namespace in the catalog.
        :param namespace_identifier: The name of the namespace (e.g., "my_db" or ("my_db", "staging")).
        """
        catalog = self.get_catalog()
        try:
            catalog.create_namespace(namespace_identifier)
            logger.info(f"Namespace '{namespace_identifier}' created successfully.")
        except TableAlreadyExistsError: # PyIceberg uses this for namespace too sometimes
            logger.warning(f"Namespace '{namespace_identifier}' already exists.")
        except Exception as e:
            logger.error(f"Failed to create namespace '{namespace_identifier}': {e}")
            raise

    def list_tables(self, namespace_identifier: str | tuple[str]) -> list[tuple[str]]:
        """
        Lists all tables within a given namespace.
        :param namespace_identifier: The name of the namespace.
        """
        catalog = self.get_catalog()
        try:
            tables = catalog.list_tables(namespace_identifier)
            logger.debug(f"Tables in namespace '{namespace_identifier}': {tables}")
            return tables
        except NoSuchNamespaceError:
            logger.warning(f"Namespace '{namespace_identifier}' does not exist.")
            return []
        except Exception as e:
            logger.error(f"Failed to list tables in namespace '{namespace_identifier}': {e}")
            raise

    # Note: Nessie branch/tag management (create_branch, merge_branch etc.)
    # is currently not exposed via PyIceberg's SQL-like API directly.
    # You would typically use the Nessie CLI or Nessie API client for these operations.
    # The PyIceberg catalog can connect to a specific branch/tag via the 'uri' or 'ref' property
    # in the pyiceberg.yaml, which you'd update and reload the catalog for.
    # For programmatic branch switching, you would typically re-initialize the catalog
    # with a new `iceberg.nessie-catalog.ref` property.

    # Example of how you *would* implement branch switching, if needed,
    # by reloading the catalog with a different ref.
    # This requires dynamic config updates or re-initialization.
    def switch_branch(self, branch_name: str):
        """
        Switches the catalog's active reference (branch/tag).
        Note: This re-initializes the catalog. Ensure configuration allows this.
        """
        logger.info(f"Attempting to switch Nessie reference to: {branch_name}")
        # This is a conceptual example. In a real app, you might re-read the YAML
        # and modify the 'ref' property, or have a dynamic way to set it.
        # For a simple solution, you'd update pyiceberg.yaml and restart the app
        # or load a new catalog instance with specific properties for that operation.
        #
        # If your pyiceberg.yaml has a 'ref' property:
        # e.g., pyiceberg_nessie: { ..., iceberg.nessie-catalog.ref: main }
        # You'd need to programmatically update that, which is outside a simple service method.
        # More advanced: you could pass specific properties on load_catalog
        # self._catalog = load_catalog(name=self.catalog_name, **current_config_with_new_ref)
        raise NotImplementedError(
            "Programmatic branch switching requires dynamic catalog property updates "
            "or re-initialization with specific ref. Use Nessie CLI for catalog-level "
            "branch management."
        )

# Example usage (would typically be in main.py or another service)
if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    # Ensure you have a pyiceberg.yaml file configured.
    # For testing, you might need to run Nessie and MinIO via docker-compose first.

    # Assuming pyiceberg.yaml is in the same directory as this script for testing
    test_config_path = os.path.join(os.path.dirname(__file__), "../pyiceberg.yaml")

    nessie_service = NessieService("pyiceberg_nessie", test_config_path)

    try:
        # Test listing namespaces
        print("\n--- Listing Namespaces ---")
        namespaces = nessie_service.list_namespaces()
        print(f"Current Namespaces: {namespaces}")

        # Test creating a namespace
        new_namespace = "test_db"
        print(f"\n--- Creating Namespace '{new_namespace}' ---")
        nessie_service.create_namespace(new_namespace)

        print(f"\n--- Listing Namespaces (after creation) ---")
        namespaces = nessie_service.list_namespaces()
        print(f"Current Namespaces: {namespaces}")

        # Test listing tables in a new namespace (should be empty)
        print(f"\n--- Listing Tables in '{new_namespace}' ---")
        tables_in_new_ns = nessie_service.list_tables(new_namespace)
        print(f"Tables: {tables_in_new_ns}")

    except Exception as e:
        print(f"Service test failed: {e}")