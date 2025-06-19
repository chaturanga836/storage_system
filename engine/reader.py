from engine.backend import get_storage_backend
from engine.cache import LRUCache
from config.storage import STORAGE_CONFIG
import json
import pyarrow.parquet as pq
import io

# Initialize shared cache based on configured size (convert MB to bytes)
CACHE = LRUCache(max_bytes=STORAGE_CONFIG.get("cache_size_mb", 128) * 1024 * 1024)

def read_jsonl(path: str):
    cached = CACHE.get(path)
    if cached:
        lines = cached.decode("utf-8").splitlines()
    else:
        data = get_storage_backend().read_file(path)
        CACHE.put(path, data)
        lines = data.decode("utf-8").splitlines()
    return [json.loads(line) for line in lines]

def read_parquet(path: str):
    cached = CACHE.get(path)
    if cached:
        buf = io.BytesIO(cached)
    else:
        data = get_storage_backend().read_file(path)
        CACHE.put(path, data)
        buf = io.BytesIO(data)
    table = pq.read_table(buf)
    return table.to_pylist()