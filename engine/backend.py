# engine/backend.py

import os
import shutil
from abc import ABC, abstractmethod
from pathlib import Path
from typing import Union
import json

# Load config
CONFIG_PATH = Path("config/storage.json")
if CONFIG_PATH.exists():
    with open(CONFIG_PATH, "r") as f:
        STORAGE_CONFIG = json.load(f)
else:
    STORAGE_CONFIG = {
        "storage_mode": "local",
        "root_path": "./data/node01",
        "node_id": "node01",
        "cache_size_mb": 128
    }


class StorageBackend(ABC):
    @abstractmethod
    def write_file(self, relative_path: str, data: bytes):
        pass

    @abstractmethod
    def read_file(self, relative_path: str) -> bytes:
        pass

    @abstractmethod
    def delete_file(self, relative_path: str):
        pass

    @abstractmethod
    def get_disk_usage(self) -> int:
        pass

    @abstractmethod
    def get_free_space(self) -> int:
        pass


class LocalFileBackend(StorageBackend):
    def __init__(self, root_path: Union[str, Path]):
        self.root = Path(root_path)
        self.root.mkdir(parents=True, exist_ok=True)

    def _resolve_path(self, relative_path: str) -> Path:
        return self.root / relative_path

    def write_file(self, relative_path: str, data: bytes):
        path = self._resolve_path(relative_path)
        path.parent.mkdir(parents=True, exist_ok=True)
        with open(path, "wb") as f:
            f.write(data)

    def read_file(self, relative_path: str) -> bytes:
        path = self._resolve_path(relative_path)
        with open(path, "rb") as f:
            return f.read()

    def delete_file(self, relative_path: str):
        path = self._resolve_path(relative_path)
        if path.exists():
            path.unlink()

    def get_disk_usage(self) -> int:
        total = 0
        for dirpath, _, filenames in os.walk(self.root):
            for f in filenames:
                fp = os.path.join(dirpath, f)
                total += os.path.getsize(fp)
        return total  # in bytes

    def get_free_space(self) -> int:
        stat = shutil.disk_usage(self.root)
        return stat.free  # in bytes


# Factory function
_backend_instance: StorageBackend = None

def get_storage_backend() -> StorageBackend:
    global _backend_instance
    if _backend_instance is None:
        mode = STORAGE_CONFIG.get("storage_mode", "local")
        if mode == "local":
            _backend_instance = LocalFileBackend(STORAGE_CONFIG["root_path"])
        else:
            raise NotImplementedError(f"Storage mode '{mode}' is not implemented")
    return _backend_instance
