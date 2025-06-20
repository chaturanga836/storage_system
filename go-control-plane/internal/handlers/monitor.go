package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/chaturanga836/storage_system/go-control-plane/internal/metrics"
)

func Monitor(w http.ResponseWriter, r *http.Request) {
	sysMetrics := metrics.GetSystemMetrics()
	json.NewEncoder(w).Encode(sysMetrics)
}
