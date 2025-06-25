#node_agent/node_agent.py
import time
import socket
import uuid
import requests
import psutil

CONTROL_PLANE_URL = "http://localhost:8081"
NODE_ID = f"node-{uuid.uuid4().hex[:6]}"  # or set manually
DISK_MOUNT = "/"  # or "/data" if you're mounting EBS volume

def register_node():
    hostname = socket.gethostname()
    ip_address = socket.gethostbyname(hostname)
    disk = psutil.disk_usage(DISK_MOUNT)

    payload = {
        "id": NODE_ID,
        "hostname": hostname,
        "ip_address": ip_address,
        "status": "active",
        "last_seen": None,
        "storage_used": disk.used,
        "total_storage": disk.total,
        "tags": ["ec2", "storage-node"]
    }

    try:
        res = requests.post(f"{CONTROL_PLANE_URL}/register-node", json=payload)
        res.raise_for_status()
        print(f"[✔] Registered node {NODE_ID}")
    except Exception as e:
        print(f"[!] Registration failed: {e}")

def send_heartbeat():
    try:
        disk = psutil.disk_usage(DISK_MOUNT)
        payload = {
            "id": NODE_ID,
            "storage_used": disk.used
        }
        res = requests.post(f"{CONTROL_PLANE_URL}/heartbeat", json=payload)
        res.raise_for_status()
        print(f"[♥] Heartbeat sent: used={disk.used // (1024*1024)} MB")
    except Exception as e:
        print(f"[!] Heartbeat failed: {e}")

def main():
    register_node()
    while True:
        send_heartbeat()
        time.sleep(30)

if __name__ == "__main__":
    main()
