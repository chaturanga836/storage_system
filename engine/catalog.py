# engine/catalog.py

import json
from pathlib import Path
from typing import Dict, List, Optional
import threading

CATALOG_PATH = Path("config/catalog.json")

_catalog_lock = threading.Lock()

# Internal structure:
# {
#   "dataset/tenant/file.parquet": {
#       "size": 123456,
#       "rows": 1000,
#       "min_ts": "2025-01-01T00:00:00Z",
#       "max_ts": "2025-01-01T01:00:00Z"
#       "location": "hot"
#   },
#   ...
# }

def load_catalog() -> Dict[str, Dict]:
    if not CATALOG_PATH.exists():
        return {}
    with open(CATALOG_PATH, "r") as f:
        return json.load(f)

def save_catalog(catalog: Dict[str, Dict]):
    with open(CATALOG_PATH, "w") as f:
        json.dump(catalog, f, indent=2)

def get_catalog_stats() -> Dict:
    with _catalog_lock:
        catalog = load_catalog()
        total_size = sum(entry.get("size", 0) for entry in catalog.values())
        total_rows = sum(entry.get("rows", 0) for entry in catalog.values())
        return {
            "total_files": len(catalog),
            "total_size_bytes": total_size,
            "total_rows": total_rows
        }


def register_partition(path: str, size: int, rows: int, min_ts: str, max_ts: str):
    with _catalog_lock:
        catalog = load_catalog()
        catalog[path] = {
            "size": size,
            "rows": rows,
            "min_ts": min_ts,
            "max_ts": max_ts,
            "location": "hot"
        }
        save_catalog(catalog)

def get_partition_meta(path: str) -> Optional[Dict]:
    with _catalog_lock:
        return load_catalog().get(path)

def list_partitions(prefix: str = "") -> List[str]:
    with _catalog_lock:
        catalog = load_catalog()
        return [k for k in catalog if k.startswith(prefix)]
        
def migrate_catalog_schema():
    """
    Ensures all entries in the catalog have a 'location' field.
    Defaults to 'hot' if missing.
    """
    with _catalog_lock:
        catalog = load_catalog()
        updated = False

        for entry in catalog.values():
            if "location" not in entry:
                entry["location"] = "hot"
                updated = True

        if updated:
            save_catalog(catalog)
            print("✅ Catalog schema migrated: 'location' field added where missing.")
        else:
            print("ℹ️ Catalog schema already up-to-date.")
