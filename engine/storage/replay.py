import os
from engine.storage.wal_manager import WALManager
from engine.catalog import register_partition
from engine.backend import get_storage_backend
import pyarrow.parquet as pq
import io

wal = WALManager(compress=True)  # or False depending on usage

def apply_fn(entry: dict):
    op = entry.get("op")
    if op in ("write_jsonl", "write_parquet"):
        # No actual file rewrite — just re-register catalog
        filename = entry["filename"]
        size = entry.get("size", 0)
        rows = entry.get("num_records", 0)
        ts = entry.get("timestamp", "")
        register_partition(filename, size, rows, ts, ts)
    else:
        print(f"[⟳] Unknown WAL entry: {entry}")
        
def apply_parquet_entry(entry: dict):
    if entry.get("op") != "write_parquet":
        return

    filename = entry.get("filename")
    tenant_id = entry.get("tenant_id")

    try:
        data = get_storage_backend().read_file(filename)
        table = pq.read_table(io.BytesIO(data))
        rows = table.num_rows
        size = len(data)

        # Use timestamps from WAL or fallback
        min_ts = entry.get("min_ts") or entry.get("timestamp")
        max_ts = entry.get("max_ts") or entry.get("timestamp")

        # Register again in catalog
        register_partition(filename, size, rows, min_ts, max_ts)
        print(f"[✓] Re-registered {filename} with {rows} rows")

    except Exception as e:
        print(f"[!] Failed to replay parquet WAL entry for {filename}: {e}")

def replay_all():
    print("[⟳] Replaying WAL...")

    def apply(entry: dict):
        if entry.get("type") in ("write_parquet", "write_jsonl"):
            path = entry["path"]
            if not os.path.exists(path):
                print(f"[✗] Skipped missing file: {path}")
                return
            register_partition(
                path=path,
                size=entry["size"],
                rows=entry["rows"],
                min_ts=entry["min_ts"],
                max_ts=entry["max_ts"]
            )
            print(f"[✓] Re-registered {path}")

    
    wal.replay(apply_fn)
    print("[✓] WAL replay complete.")
