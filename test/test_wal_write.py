# test/test_wal_write.py


from engine.storage.wal import append_wal_entry

append_wal_entry({
    "type": "test",
    "timestamp": "2025-06-23T18:00:00Z",
    "message": "This is a test WAL entry"
})
print("✅ WAL entry written.")
