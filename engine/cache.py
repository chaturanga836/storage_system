from typing import Any, Dict, List
class LRUCache:
    def __init__(self, max_bytes: int = 128 * 1024 * 1024):
        self.max_size = max_bytes
        self.cache: Dict[str, Any] = {}
        self.current_size = 0
        self.access_order: List[str] = []
        self.hits = 0
        self.misses = 0

    def get(self, key: str):
        if key in self.cache:
            self.access_order.remove(key)
            self.access_order.append(key)
            self.hits += 1      # ✅ Track hit
            return self.cache[key]
        self.misses += 1        # ✅ Track miss
        return None

    def set(self, key: str, value: Any, size_bytes: int):
        if key in self.cache:
            self.current_size -= self.cache[key]["_size"]

        while self.current_size + size_bytes > self.max_size and self.access_order:
            oldest = self.access_order.pop(0)
            self.current_size -= self.cache[oldest]["_size"]
            del self.cache[oldest]

        self.cache[key] = value
        self.cache[key]["_size"] = size_bytes
        self.access_order.append(key)
        self.current_size += size_bytes

    def stats(self) -> Dict[str, int]:
        return {
            "entries": len(self.cache),
            "total_size_bytes": self.current_size,
            "hits": self.hits,
            "misses": self.misses
        }
