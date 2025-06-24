# test/test_wal_manager.py

import time
from pathlib import Path
from engine.storage.wal_manager import WALManager

def test_wal_append_and_replay(tmp_path):
    wal_dir = tmp_path / "wal"
    wal = WALManager(compress=False, wal_dir=wal_dir)

    sample_entries = [
        {"op": "write", "id": 1},
        {"op": "write", "id": 2},
    ]

    for entry in sample_entries:
        wal.append(entry)

    time.sleep(1)
    wal.shutdown()

    replayed = []
    wal.replay(lambda e: replayed.append(e))

    assert len(replayed) == 2
    assert replayed[0]["id"] == 1
    assert replayed[1]["id"] == 2

