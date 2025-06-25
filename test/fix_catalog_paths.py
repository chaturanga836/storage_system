# fix_catalog_paths.py
import json

catalog_path = "config/catalog.json"
with open(catalog_path, "r") as f:
    catalog = json.load(f)

fixed_catalog = {}
for k, v in catalog.items():
    new_key = k.replace("node0/", "node01/")  # Adjust if needed
    fixed_catalog[new_key] = v

with open(catalog_path, "w") as f:
    json.dump(fixed_catalog, f, indent=2)

print("✅ Catalog paths updated.")
