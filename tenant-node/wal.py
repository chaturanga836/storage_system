"""
Write-Ahead Log (WAL) Manager for ensuring data consistency
"""
import asyncio
import json
import uuid
from datetime import datetime, timezone, timedelta
from pathlib import Path
from typing import Dict, List, Any, Optional
import logging

logger = logging.getLogger(__name__)


class WALEntry:
    """Represents a single WAL entry"""
    
    def __init__(self, operation_id: str, operation_type: str, data: Any, timestamp: datetime):
        self.operation_id = operation_id
        self.operation_type = operation_type
        self.data = data
        self.timestamp = timestamp
        self.status = "pending"  # pending, completed, failed
        self.error_message = None
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization"""
        return {
            "operation_id": self.operation_id,
            "operation_type": self.operation_type,
            "data": self.data,
            "timestamp": self.timestamp.isoformat(),
            "status": self.status,
            "error_message": self.error_message
        }
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "WALEntry":
        """Create from dictionary"""
        entry = cls(
            operation_id=data["operation_id"],
            operation_type=data["operation_type"],
            data=data["data"],
            timestamp=datetime.fromisoformat(data["timestamp"])
        )
        entry.status = data.get("status", "pending")
        entry.error_message = data.get("error_message")
        return entry


class WALManager:
    """Manages Write-Ahead Logging for data consistency"""
    
    def __init__(self, wal_path: Path, retention_hours: int = 24):
        self.wal_path = wal_path
        self.retention_hours = retention_hours
        self.current_log_file = None
        self.log_rotation_size = 100 * 1024 * 1024  # 100MB
        self._lock = asyncio.Lock()
        
    async def initialize(self):
        """Initialize the WAL manager"""
        self.wal_path.mkdir(parents=True, exist_ok=True)
        
        # Create or open current log file
        await self._create_new_log_file()
        
        # Start background tasks
        asyncio.create_task(self._cleanup_old_logs())
        
        logger.info(f"WAL Manager initialized at {self.wal_path}")
    
    async def _create_new_log_file(self):
        """Create a new log file"""
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        self.current_log_file = self.wal_path / f"wal_{timestamp}.jsonl"
        
        # Touch the file to create it
        with open(self.current_log_file, 'a') as f:
            pass
    
    async def log_write_operation(self, operation_id: str, data: List[Dict[str, Any]]) -> WALEntry:
        """Log a write operation to WAL"""
        entry = WALEntry(
            operation_id=operation_id,
            operation_type="write",
            data=data,
            timestamp=datetime.now(timezone.utc)
        )
        
        await self._write_wal_entry(entry)
        return entry
    
    async def log_delete_operation(self, operation_id: str, filters: Dict[str, Any]) -> WALEntry:
        """Log a delete operation to WAL"""
        entry = WALEntry(
            operation_id=operation_id,
            operation_type="delete",
            data=filters,
            timestamp=datetime.now(timezone.utc)
        )
        
        await self._write_wal_entry(entry)
        return entry
    
    async def mark_operation_complete(self, operation_id: str):
        """Mark an operation as completed"""
        await self._update_operation_status(operation_id, "completed")
    
    async def mark_operation_failed(self, operation_id: str, error_message: str):
        """Mark an operation as failed"""
        await self._update_operation_status(operation_id, "failed", error_message)
    
    async def _write_wal_entry(self, entry: WALEntry):
        """Write a WAL entry to the log file"""
        async with self._lock:
            # Check if we need to rotate the log file
            if self.current_log_file.exists() and self.current_log_file.stat().st_size > self.log_rotation_size:
                await self._create_new_log_file()
            
            # Write the entry
            with open(self.current_log_file, 'a') as f:
                f.write(json.dumps(entry.to_dict()) + '\n')
                f.flush()
    
    async def _update_operation_status(self, operation_id: str, status: str, error_message: str = None):
        """Update the status of an operation"""
        # For simplicity, we'll append a status update entry
        # In a more sophisticated implementation, we might update in-place or use a separate status file
        status_entry = {
            "operation_id": operation_id,
            "status_update": status,
            "error_message": error_message,
            "timestamp": datetime.now(timezone.utc).isoformat()
        }
        
        async with self._lock:
            with open(self.current_log_file, 'a') as f:
                f.write(json.dumps(status_entry) + '\n')
                f.flush()
    
    async def get_pending_operations(self) -> List[WALEntry]:
        """Get all pending operations for recovery"""
        pending_ops = []
        
        # Read all log files
        for log_file in sorted(self.wal_path.glob("wal_*.jsonl")):
            with open(log_file, 'r') as f:
                for line in f:
                    try:
                        data = json.loads(line.strip())
                        
                        # Skip status updates
                        if "status_update" in data:
                            continue
                        
                        entry = WALEntry.from_dict(data)
                        if entry.status == "pending":
                            pending_ops.append(entry)
                    
                    except (json.JSONDecodeError, KeyError) as e:
                        logger.error(f"Error parsing WAL entry: {e}")
                        continue
        
        return pending_ops
    
    async def replay_operations(self) -> List[str]:
        """Replay pending operations for recovery"""
        pending_ops = await self.get_pending_operations()
        replayed_ops = []
        
        for entry in pending_ops:
            try:
                # This would be implemented based on specific recovery logic
                # For now, we'll just mark them as failed with a recovery message
                await self.mark_operation_failed(
                    entry.operation_id, 
                    "Operation replayed during recovery"
                )
                replayed_ops.append(entry.operation_id)
                
            except Exception as e:
                logger.error(f"Error replaying operation {entry.operation_id}: {e}")
        
        return replayed_ops
    
    async def _cleanup_old_logs(self):
        """Background task to cleanup old log files"""
        while True:
            try:
                cutoff_time = datetime.now() - timedelta(hours=self.retention_hours)
                
                for log_file in self.wal_path.glob("wal_*.jsonl"):
                    # Parse timestamp from filename
                    try:
                        timestamp_str = log_file.stem.split("_", 1)[1]
                        file_time = datetime.strptime(timestamp_str, "%Y%m%d_%H%M%S")
                        
                        if file_time < cutoff_time and log_file != self.current_log_file:
                            log_file.unlink()
                            logger.info(f"Cleaned up old WAL file: {log_file}")
                    
                    except (ValueError, IndexError) as e:
                        logger.warning(f"Could not parse timestamp from WAL file {log_file}: {e}")
                
                # Sleep for 1 hour before next cleanup
                await asyncio.sleep(3600)
                
            except Exception as e:
                logger.error(f"Error in WAL cleanup: {e}")
                await asyncio.sleep(3600)
    
    async def get_status(self) -> Dict[str, Any]:
        """Get WAL status information"""
        log_files = list(self.wal_path.glob("wal_*.jsonl"))
        total_size = sum(f.stat().st_size for f in log_files)
        
        pending_ops = await self.get_pending_operations()
        
        return {
            "total_log_files": len(log_files),
            "total_size_bytes": total_size,
            "current_log_file": str(self.current_log_file) if self.current_log_file else None,
            "pending_operations": len(pending_ops),
            "retention_hours": self.retention_hours
        }
    
    async def cleanup(self):
        """Cleanup WAL manager resources"""
        # Cancel any background tasks if needed
        logger.info("WAL Manager cleanup complete")
