#!/bin/bash
apt update -y
apt install -y golang python3 python3-venv git unzip curl

# Optional: clone your Go control plane repo (example)
cd /home/ubuntu
git clone https://github.com/chaturanga836/storage_control_plane.git

# Optional: clone your Python data engine repo
git clone https://github.com/chaturanga836/storage_system.git

chown -R ubuntu:ubuntu /home/ubuntu/
