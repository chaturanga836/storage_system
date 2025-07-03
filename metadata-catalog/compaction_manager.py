"""
File compaction manager for automatic small file optimization
"""
import asyncio
import logging
import shutil
from pathlib import Path
from typing import List, Dict, Any, Optional
from datetime import datetime, timedelta
import pandas as pd
import pyarrow as pa
import pyarrow.parquet as pq
from dataclasses import dataclass

logger = logging.getLogger(__name__)


@dataclass
class CompactionPolicy:
    """Configuration for file compaction behavior"""
    # File size thresholds
    min_file_size_mb: float = 50.0
    max_file_size_mb: float = 512.0
    target_file_size_mb: float = 256.0
    
    # Compaction triggers
    small_file_threshold: int = 10  # Number of small files to trigger compaction
    total_files_threshold: int = 100  # Total files to trigger compaction
    age_threshold_hours: int = 6  # Age of files to consider for compaction
    
    # Performance settings
    max_concurrent_compactions: int = 3
    compaction_batch_size: int = 20
    
    # Scheduling
    compaction_interval_minutes: int = 60
    maintenance_window_start: int = 2  # 2 AM
    maintenance_window_end: int = 6    # 6 AM


@dataclass
class CompactionJob:
    """Represents a compaction job"""
    job_id: str
    source_id: str
    input_files: List[str]
    output_file: str
    started_at: datetime
    estimated_size_mb: float
    partition_columns: List[str]


class FileCompactionManager:
    """Manages automatic file compaction for optimized storage"""
    
    def __init__(self, tenant_config, compaction_policy: Optional[CompactionPolicy] = None):
        self.tenant_config = tenant_config
        self.policy = compaction_policy or CompactionPolicy()
        
        # Job tracking
        self.active_jobs: Dict[str, CompactionJob] = {}
        self.completed_jobs: List[CompactionJob] = []
        self.failed_jobs: List[CompactionJob] = []
        
        # Metrics
        self.total_compactions = 0
        self.total_size_reduced = 0
        self.total_files_reduced = 0
        
        logger.info("File compaction manager initialized")
    
    async def start_compaction_scheduler(self):
        """Start the automatic compaction scheduler"""
        logger.info("Starting file compaction scheduler")
        
        while True:
            try:
                # Check if we're in maintenance window
                if self._is_maintenance_window():
                    await self._run_compaction_cycle()
                else:
                    # Still check for urgent compactions outside maintenance window
                    await self._check_urgent_compactions()
                
                # Wait for next check
                await asyncio.sleep(self.policy.compaction_interval_minutes * 60)
                
            except Exception as e:
                logger.error(f"Error in compaction scheduler: {e}")
                await asyncio.sleep(300)  # 5 minute back-off on error
    
    def _is_maintenance_window(self) -> bool:
        """Check if current time is within maintenance window"""
        current_hour = datetime.now().hour
        return self.policy.maintenance_window_start <= current_hour < self.policy.maintenance_window_end
    
    async def _run_compaction_cycle(self):
        """Run a full compaction cycle for all sources"""
        logger.info("Starting compaction cycle")
        
        # from .data_source import SourceManager  # Not used currently
        
        # This would be injected in real implementation
        # For now, we'll work with the tenant config
        sources_path = Path(self.tenant_config.base_data_path) / "sources"
        
        if not sources_path.exists():
            return
        
        # Find all source directories
        source_dirs = [d for d in sources_path.iterdir() if d.is_dir()]
        
        # Process each source
        compaction_tasks = []
        for source_dir in source_dirs:
            if len(self.active_jobs) >= self.policy.max_concurrent_compactions:
                break
            
            task = asyncio.create_task(
                self._compact_source(source_dir.name, source_dir)
            )
            compaction_tasks.append(task)
        
        # Wait for all compaction tasks
        if compaction_tasks:
            await asyncio.gather(*compaction_tasks, return_exceptions=True)
        
        logger.info("Compaction cycle completed")
    
    async def _check_urgent_compactions(self):
        """Check for urgent compactions needed outside maintenance window"""
        sources_path = Path(self.tenant_config.base_data_path) / "sources"
        
        if not sources_path.exists():
            return
        
        for source_dir in sources_path.iterdir():
            if not source_dir.is_dir():
                continue
            
            parquet_dir = source_dir / "parquet"
            if not parquet_dir.exists():
                continue
            
            # Check if urgent compaction is needed
            file_stats = await self._analyze_files(parquet_dir)
            
            if self._is_urgent_compaction_needed(file_stats):
                logger.warning(f"Urgent compaction needed for source: {source_dir.name}")
                await self._compact_source(source_dir.name, source_dir)
    
    def _is_urgent_compaction_needed(self, file_stats: Dict[str, Any]) -> bool:
        """Determine if urgent compaction is needed"""
        return (
            file_stats["small_files"] > self.policy.small_file_threshold * 2 or
            file_stats["total_files"] > self.policy.total_files_threshold * 2
        )
    
    async def _compact_source(self, source_id: str, source_path: Path):
        """Compact files for a specific source"""
        parquet_dir = source_path / "parquet"
        
        if not parquet_dir.exists():
            return
        
        try:
            logger.info(f"Analyzing source for compaction: {source_id}")
            
            # Analyze files to determine compaction strategy
            file_stats = await self._analyze_files(parquet_dir)
            
            if not self._should_compact(file_stats):
                logger.debug(f"No compaction needed for source: {source_id}")
                return
            
            # Find compaction candidates
            compaction_groups = await self._find_compaction_candidates(parquet_dir)
            
            # Execute compaction jobs
            for group in compaction_groups:
                if len(self.active_jobs) >= self.policy.max_concurrent_compactions:
                    break
                
                await self._execute_compaction_job(source_id, group)
        
        except Exception as e:
            logger.error(f"Error compacting source {source_id}: {e}")
    
    async def _analyze_files(self, parquet_dir: Path) -> Dict[str, Any]:
        """Analyze parquet files for compaction decision"""
        files = list(parquet_dir.rglob("*.parquet"))
        
        small_files = []
        large_files = []
        total_size = 0
        old_files = []
        
        cutoff_time = datetime.now() - timedelta(hours=self.policy.age_threshold_hours)
        
        for file_path in files:
            try:
                stat = file_path.stat()
                size_mb = stat.st_size / (1024 * 1024)
                modified_time = datetime.fromtimestamp(stat.st_mtime)
                
                total_size += size_mb
                
                if size_mb < self.policy.min_file_size_mb:
                    small_files.append(file_path)
                elif size_mb > self.policy.max_file_size_mb:
                    large_files.append(file_path)
                
                if modified_time < cutoff_time:
                    old_files.append(file_path)
                    
            except Exception as e:
                logger.warning(f"Error analyzing file {file_path}: {e}")
        
        return {
            "total_files": len(files),
            "small_files": len(small_files),
            "large_files": len(large_files),
            "old_files": len(old_files),
            "total_size_mb": total_size,
            "small_file_paths": small_files,
            "large_file_paths": large_files,
            "old_file_paths": old_files
        }
    
    def _should_compact(self, file_stats: Dict[str, Any]) -> bool:
        """Determine if compaction should be performed"""
        return (
            file_stats["small_files"] >= self.policy.small_file_threshold or
            file_stats["total_files"] >= self.policy.total_files_threshold or
            file_stats["old_files"] >= self.policy.small_file_threshold
        )
    
    async def _find_compaction_candidates(self, parquet_dir: Path) -> List[List[Path]]:
        """Find groups of files that should be compacted together"""
        file_stats = await self._analyze_files(parquet_dir)
        
        compaction_groups = []
        
        # Group small files by partition if applicable
        small_files = file_stats["small_file_paths"]
        
        # Simple grouping by directory (partition)
        partition_groups = {}
        for file_path in small_files:
            partition_key = str(file_path.parent)
            if partition_key not in partition_groups:
                partition_groups[partition_key] = []
            partition_groups[partition_key].append(file_path)
        
        # Create compaction groups
        for partition_key, files in partition_groups.items():
            if len(files) >= 2:  # Need at least 2 files to compact
                # Split into batches
                for i in range(0, len(files), self.policy.compaction_batch_size):
                    batch = files[i:i + self.policy.compaction_batch_size]
                    if len(batch) >= 2:
                        compaction_groups.append(batch)
        
        return compaction_groups
    
    async def _execute_compaction_job(self, source_id: str, file_group: List[Path]):
        """Execute a compaction job for a group of files"""
        import uuid
        
        job_id = str(uuid.uuid4())
        
        try:
            # Estimate output size
            total_size = sum(f.stat().st_size for f in file_group) / (1024 * 1024)
            
            # Create compaction job
            job = CompactionJob(
                job_id=job_id,
                source_id=source_id,
                input_files=[str(f) for f in file_group],
                output_file="",  # Will be set during execution
                started_at=datetime.now(),
                estimated_size_mb=total_size,
                partition_columns=[]  # Will be detected from schema
            )
            
            self.active_jobs[job_id] = job
            
            logger.info(f"Starting compaction job {job_id} for {len(file_group)} files "
                       f"(estimated {total_size:.1f} MB)")
            
            # Execute the actual compaction
            await self._compact_files(job, file_group)
            
            # Mark job as completed
            self.completed_jobs.append(job)
            del self.active_jobs[job_id]
            
            self.total_compactions += 1
            self.total_files_reduced += len(file_group) - 1  # -1 because we create 1 output file
            
            logger.info(f"Compaction job {job_id} completed successfully")
            
        except Exception as e:
            logger.error(f"Compaction job {job_id} failed: {e}")
            
            if job_id in self.active_jobs:
                job = self.active_jobs[job_id]
                self.failed_jobs.append(job)
                del self.active_jobs[job_id]
    
    async def _compact_files(self, job: CompactionJob, file_group: List[Path]):
        """Perform the actual file compaction"""
        try:
            # Read all input files
            dataframes = []
            schemas = []
            
            for file_path in file_group:
                try:
                    # Read parquet file
                    table = pq.read_table(str(file_path))
                    df = table.to_pandas()
                    dataframes.append(df)
                    schemas.append(table.schema)
                    
                except Exception as e:
                    logger.warning(f"Error reading file {file_path}: {e}")
                    continue
            
            if not dataframes:
                raise ValueError("No valid input files to compact")
            
            # Combine all dataframes
            combined_df = pd.concat(dataframes, ignore_index=True)
            
            # Detect partition columns from schema
            if schemas:
                # Use the first schema as reference
                schema = schemas[0]
                # For simplicity, assume no partitioning for now
                # In real implementation, detect from file paths or metadata
                job.partition_columns = []
            
            # Generate output filename
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            output_dir = file_group[0].parent
            output_filename = f"compacted_{timestamp}_{job.job_id[:8]}.parquet"
            output_path = output_dir / output_filename
            
            job.output_file = str(output_path)
            
            # Write compacted file
            combined_df.to_parquet(
                str(output_path),
                compression="snappy",
                index=False
            )
            
            # Verify output file
            output_size = output_path.stat().st_size / (1024 * 1024)
            self.total_size_reduced += job.estimated_size_mb - output_size
            
            logger.info(f"Compacted {len(file_group)} files into {output_filename} "
                       f"({output_size:.1f} MB)")
            
            # Remove original files
            await self._cleanup_original_files(file_group, output_path)
            
        except Exception as e:
            logger.error(f"Error during file compaction: {e}")
            raise
    
    async def _cleanup_original_files(self, original_files: List[Path], output_file: Path):
        """Safely remove original files after successful compaction"""
        try:
            # Create backup directory for safety
            backup_dir = output_file.parent / "compaction_backup" / datetime.now().strftime("%Y%m%d")
            backup_dir.mkdir(parents=True, exist_ok=True)
            
            # Move original files to backup
            for file_path in original_files:
                backup_path = backup_dir / file_path.name
                shutil.move(str(file_path), str(backup_path))
            
            logger.info(f"Moved {len(original_files)} original files to backup")
            
            # Schedule cleanup of old backups (after 7 days)
            asyncio.create_task(self._cleanup_old_backups(backup_dir.parent))
            
        except Exception as e:
            logger.error(f"Error cleaning up original files: {e}")
            # Don't fail the compaction job for cleanup errors
    
    async def _cleanup_old_backups(self, backup_root: Path):
        """Clean up old backup files"""
        try:
            cutoff_date = datetime.now() - timedelta(days=7)
            
            for backup_dir in backup_root.iterdir():
                if not backup_dir.is_dir():
                    continue
                
                try:
                    dir_date = datetime.strptime(backup_dir.name, "%Y%m%d")
                    if dir_date < cutoff_date:
                        shutil.rmtree(str(backup_dir))
                        logger.info(f"Cleaned up old backup directory: {backup_dir.name}")
                except ValueError:
                    # Skip directories that don't match date format
                    continue
                    
        except Exception as e:
            logger.error(f"Error cleaning up old backups: {e}")
    
    def get_compaction_status(self) -> Dict[str, Any]:
        """Get current compaction status"""
        return {
            "active_jobs": len(self.active_jobs),
            "completed_jobs": len(self.completed_jobs),
            "failed_jobs": len(self.failed_jobs),
            "total_compactions": self.total_compactions,
            "total_size_reduced_mb": self.total_size_reduced,
            "total_files_reduced": self.total_files_reduced,
            "current_jobs": [
                {
                    "job_id": job.job_id,
                    "source_id": job.source_id,
                    "input_files_count": len(job.input_files),
                    "estimated_size_mb": job.estimated_size_mb,
                    "started_at": job.started_at.isoformat()
                }
                for job in self.active_jobs.values()
            ]
        }
    
    async def trigger_manual_compaction(self, source_id: str) -> bool:
        """Manually trigger compaction for a specific source"""
        try:
            source_path = Path(self.tenant_config.base_data_path) / "sources" / source_id
            
            if not source_path.exists():
                logger.error(f"Source not found: {source_id}")
                return False
            
            logger.info(f"Manual compaction triggered for source: {source_id}")
            await self._compact_source(source_id, source_path)
            return True
            
        except Exception as e:
            logger.error(f"Error in manual compaction for {source_id}: {e}")
            return False
