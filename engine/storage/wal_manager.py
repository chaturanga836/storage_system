# engine/storage/wal_manager.py

import json
import os
import threading
import time
from pathlib import Path
from datetime import datetime
from typing import Callable, Optional
import zlib

WAL_DIR = Path("logs/wal/")
WAL_DIR.mkdir(parents=True, exist_ok=True)

MAX_WAL_SIZE_BYTES = 5 * 1024 * 1024  # 5MB per WAL file
FLUSH_INTERVAL_SECS = 0.5
ENTRIES_BEFORE_FLUSH = 50

class WALManager:
    def __init__(self, compress: bool = False, wal_dir: Optional[Path] = None, checkpoint_every: Optional[int] = None, catalog_loader: Optional[Callable[[], dict]] = None):
        self.checkpoint_every = checkpoint_every
        self.catalog_loader = catalog_loader
        self.write_count = 0

        self.lock = threading.Lock()
        self.buffer = []
        self.compress = compress

        self.wal_dir = wal_dir or WAL_DIR
        self.wal_dir.mkdir(parents=True, exist_ok=True)
        self.current_wal_path = self._new_wal_path()

        self.last_flush_time = time.time()
        self.running = True

        self.flush_thread = threading.Thread(target=self._flush_loop, daemon=True)
        self.flush_thread.start()

    def _new_wal_path(self) -> Path:
        timestamp = datetime.utcnow().strftime("%Y%m%dT%H%M%S")
        return self.wal_dir / f"wal_{timestamp}.log"

    def append(self, entry: dict):
        with self.lock:
            self.buffer.append(entry)
            if len(self.buffer) >= ENTRIES_BEFORE_FLUSH:
                self._flush_locked()

    def _flush_loop(self):
        while self.running:
            time.sleep(FLUSH_INTERVAL_SECS)
            with self.lock:
                if self.buffer:
                    self._flush_locked()

    def _flush_locked(self):
        data_to_write = self.buffer
        self.buffer = []

        if self.current_wal_path.exists() and self.current_wal_path.stat().st_size > MAX_WAL_SIZE_BYTES:
            self.current_wal_path = self._new_wal_path()

        binary_mode = self.compress
        mode = "ab" if binary_mode else "a"
        with open(self.current_wal_path, mode) as f:
            for entry in data_to_write:
                if binary_mode:
                    encoded = zlib.compress(json.dumps(entry).encode("utf-8"))
                    f.write(encoded)
                else:
                    f.write(json.dumps(entry) + "\n")
        self.last_flush_time = time.time()

            # ✅ Update write counter
        self.write_count += len(data_to_write)

        # ✅ Perform checkpoint if needed
        if (
            self.catalog_loader
            and self.checkpoint_every
            and self.write_count >= self.checkpoint_every
        ):
            self._save_checkpoint()
            self.write_count = 0

    def shutdown(self):
        self.running = False
        self.flush_thread.join()
        with self.lock:
            if self.buffer:
                self._flush_locked()

    def replay(self, apply_fn: Callable[[dict], None]):
        """Replay all WAL entries using a callback function."""
        for file in sorted(self.wal_dir.glob("wal_*.log")):
            if self.compress:
                with open(file, "rb") as f:
                    for line in f:
                        try:
                            decoded = zlib.decompress(line)
                            entry = json.loads(decoded.decode("utf-8"))
                            apply_fn(entry)
                        except Exception as e:
                            print(f"[WAL] Error decoding compressed line in {file}: {e}")
            else:
                with open(file, "r", encoding="utf-8") as f:
                    for line in f:
                        try:
                            entry = json.loads(line.strip())
                            apply_fn(entry)
                        except Exception as e:
                            print(f"[WAL] Error decoding plain line in {file}: {e}")

    def _save_checkpoint(self):
        try:
            catalog = self.catalog_loader()
            checkpoint_path = Path("config/catalog.checkpoint.json")
            checkpoint_path.parent.mkdir(parents=True, exist_ok=True)

            with open(checkpoint_path, "w", encoding="utf-8") as f:
                json.dump(catalog, f, indent=2)

            print(f"[📌] Checkpoint saved: {checkpoint_path}")
        except Exception as e:
            print(f"[⚠️] Failed to save checkpoint: {e}")