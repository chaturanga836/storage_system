import json
from typing import List, Dict
from engine.backend import get_storage_backend
from engine.catalog import register_partition
from datetime import datetime
import os

def write_jsonl(dataset: str, tenant_id: str, records: List[Dict]):
    """Write a list of dictionaries as JSONL."""
    storage = get_storage_backend()
    timestamp = datetime.utcnow().strftime("%Y%m%dT%H%M%S")
    filename = f"{dataset}/{tenant_id}/{timestamp}.jsonl"
    data = "\n".join(json.dumps(record) for record in records).encode("utf-8")
    storage.write_file(filename, data)
    # Catalog registration
    size = len(data)
    rows = len(records)
    register_partition(filename, size, rows, records[0].get("timestamp", timestamp), records[-1].get("timestamp", timestamp))
    return filename

def write_parquet(dataset: str, tenant_id: str, records: List[Dict]):
    import pyarrow as pa
    import pyarrow.parquet as pq
    import io

    table = pa.Table.from_pylist(records)
    buf = io.BytesIO()
    pq.write_table(table, buf)
    buf.seek(0)

    timestamp = datetime.utcnow().strftime("%Y%m%dT%H%M%S")
    filename = f"{dataset}/{tenant_id}/{timestamp}.parquet"
    storage = get_storage_backend()
    data = buf.read()
    storage.write_file(filename, data)
    # Catalog registration
    size = len(data)
    rows = len(records)
    register_partition(filename, size, rows, records[0].get("timestamp", timestamp), records[-1].get("timestamp", timestamp))
    return filename