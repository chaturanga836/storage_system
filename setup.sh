#!/bin/bash
echo "[*] Creating virtual environment..."
python3 -m venv venv

echo "[*] Activating virtual environment..."
source venv/bin/activate

echo "[*] Installing runtime requirements..."
pip install -r requirements.txt

echo "[*] Installing development tools..."
pip install -r dev-requirements.txt

echo "[✓] Setup complete!"
