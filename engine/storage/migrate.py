# engine/storage/migrate.py

from datetime import datetime, timedelta, timezone
from pathlib import Path
from engine.catalog import load_catalog, save_catalog
from engine.backend import STORAGE_CONFIG

ROOT_PATH = Path(STORAGE_CONFIG["root_path"])
HOT_DIR = ROOT_PATH / "hot"
COLD_DIR = ROOT_PATH / "cold"

def migrate_to_cold_storage(cutoff_days: int = 7) -> int:
    catalog = load_catalog()
    cutoff = datetime.now(timezone.utc) - timedelta(days=cutoff_days)
    moved = 0

    print(f"\n🧊 Cold Storage Migration")
    print(f"📅 Cutoff: {cutoff.isoformat()} (older than {cutoff_days} days)")
    print(f"🔍 Catalog entries: {len(catalog)}")

    for path, meta in catalog.items():
        location = meta.get("location")
        max_ts_raw = meta.get("max_ts")

        print(f"— Checking {path}")
        print(f"  Location: {location}, Max TS: {max_ts_raw}")

        if location != "hot":
            print("  ⏩ Skipped (not in hot tier)")
            continue

        try:
            max_ts = datetime.fromisoformat(max_ts_raw.replace("Z", "+00:00"))
        except Exception as e:
            print(f"  ⚠️ Skipping due to bad timestamp: {e}")
            continue

        if max_ts < cutoff:
            hot_path = HOT_DIR / path
            cold_path = COLD_DIR / path
            print(f"  ✅ Eligible for migration")

            if hot_path.exists():
                try:
                    cold_path.parent.mkdir(parents=True, exist_ok=True)
                    hot_path.rename(cold_path)
                    meta["location"] = "cold"
                    moved += 1
                    print(f"  📦 Moved to: {cold_path.resolve()}")
                except Exception as e:
                    print(f"  ❌ Rename failed: {e}")
            else:
                print(f"  🚫 File missing in hot: {hot_path.resolve()}")
        else:
            print(f"  ⏩ Not old enough")

    if moved:
        save_catalog(catalog)
        print(f"\n[✅] Migrated {moved} file(s) to cold storage.")
    else:
        print("\n[ℹ️] No eligible files to migrate.")

    return moved

