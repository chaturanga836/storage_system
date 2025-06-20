package metrics

import (
	"runtime"
	"time"
)

type SystemMetrics struct {
	CPUCount   int    `json:"cpu_count"`
	GoRoutines int    `json:"goroutines"`
	MemoryMB   uint64 `json:"memory_mb"`
	Uptime     string `json:"uptime"`
	Note       string `json:"note"`
}

var startTime = time.Now()

func GetSystemMetrics() SystemMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemMetrics{
		CPUCount:   runtime.NumCPU(),
		GoRoutines: runtime.NumGoroutine(),
		MemoryMB:   m.Alloc / 1024 / 1024,
		Uptime:     time.Since(startTime).String(),
		Note:       "Disk metrics not available on Windows",
	}
}
