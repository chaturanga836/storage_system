# main.py

from fastapi import FastAPI
from api.routes import router as api_router
from engine.storage.replay import replay_all
from engine.catalog import load_catalog
from engine.storage.wal_manager import WALManager  # ✅ NEW
from engine.catalog import load_catalog, migrate_catalog_schema

app = FastAPI()

# ✅ Shared WALManager instance (you may make this global if needed)
wal = WALManager(compress=True, checkpoint_every=100, catalog_loader=load_catalog)

# ✅ Register your API routes
app.include_router(api_router)

# ✅ Run WAL replay and initialize WAL logic
@app.on_event("startup")
def startup_event():
    print("🔁 Replaying WAL entries...")
    replay_all()
    print("✅ WAL replay completed.")

    migrate_catalog_schema()  # ✅ Ensure all entries have 'location'
    # You can also replay directly here if you want:
    # wal.replay(apply_fn)

    print("✅ WAL manager is running.")
