# api/routes/admin.py

from fastapi import APIRouter
from engine.storage.migrate import migrate_to_cold_storage

router = APIRouter()

@router.post("/migrate-cold")
def run_migration(days: int = 7):
    """
    Move files from hot to cold tier if older than `days` (default 7).
    """
    migrated = migrate_to_cold_storage(cutoff_days=days)
    return {"migrated": migrated}
