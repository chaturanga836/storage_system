# engine/encryptor.py

import json
from cryptography.fernet import Fernet
from pathlib import Path

# Config location for tenant keys
KEY_FILE = Path("config/keys.json")


def load_keys() -> dict:
    if not KEY_FILE.exists():
        raise FileNotFoundError("Missing config/keys.json")
    with open(KEY_FILE, "r") as f:
        return json.load(f)


def get_fernet(tenant_id: str) -> Fernet:
    keys = load_keys()
    key = keys.get(tenant_id)
    if not key:
        raise ValueError(f"No encryption key for tenant: {tenant_id}")
    return Fernet(key.encode())


def encrypt_file(input_path: str, output_path: str, tenant_id: str):
    fernet = get_fernet(tenant_id)
    with open(input_path, "rb") as infile:
        plaintext = infile.read()
    encrypted = fernet.encrypt(plaintext)
    with open(output_path, "wb") as outfile:
        outfile.write(encrypted)
    print(f"[✓] Encrypted: {input_path} → {output_path}")


def decrypt_file(input_path: str, output_path: str, tenant_id: str):
    fernet = get_fernet(tenant_id)
    with open(input_path, "rb") as infile:
        encrypted = infile.read()
    decrypted = fernet.decrypt(encrypted)
    with open(output_path, "wb") as outfile:
        outfile.write(decrypted)
    print(f"[✓] Decrypted: {input_path} → {output_path}")
