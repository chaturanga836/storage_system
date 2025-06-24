# engine/writer.py

from engine.storage.wal_manager import WALManager
import atexit
import json
from typing import List, Dict
from engine.backend import get_storage_backend
from engine.catalog import register_partition
from datetime import datetime, UTC
from pathlib import Path
import os

AUDIT_LOG_PATH = Path("logs/audit.jsonl")
AUDIT_LOG_PATH.parent.mkdir(parents=True, exist_ok=True)

wal = WALManager(compress=True)
atexit.register(wal.shutdown)

def write_jsonl(dataset: str, tenant_id: str, records: List[Dict]):
    """Write a list of dictionaries as JSONL."""
    storage = get_storage_backend()
    timestamp = datetime.now(UTC).strftime("%Y%m%dT%H%M%S")
    filename = f"{dataset}/{tenant_id}/{timestamp}.jsonl"
    data = "\n".join(json.dumps(record) for record in records).encode("utf-8")

    # --- WAL BEFORE actual write ---
    wal.append({
        "op": "write_jsonl",
        "dataset": dataset,
        "tenant_id": tenant_id,
        "filename": filename,
        "timestamp": timestamp,
        "num_records": len(records)
    })

    # --- Write file ---
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

    # --- WAL BEFORE actual write ---
    wal.append({
        "op": "write_parquet",
        "dataset": dataset,
        "tenant_id": tenant_id,
        "filename": filename,
        "timestamp": timestamp,
        "num_records": len(records)
    })

    # --- Write file ---
    storage.write_file(filename, data)

    # Catalog registration
    size = len(data)
    rows = len(records)
    register_partition(filename, size, rows, records[0].get("timestamp", timestamp), records[-1].get("timestamp", timestamp))
    return filename


def log_audit_event(action: str, performed_by: str, target_user: str, details: str):
    """Append an audit event to the audit log."""
    event = {
        "timestamp": datetime.utcnow().isoformat(),
        "action": action,
        "performed_by": performed_by,
        "target_user": target_user,
        "details": details,
    }

    with open(AUDIT_LOG_PATH, "a", encoding="utf-8") as f:
        f.write(json.dumps(event) + "\n")