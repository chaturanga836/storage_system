package utils

import (
	"encoding/json"
	"errors"
	"os"
	"sync"

	"github.com/chaturanga836/storage_system/go-control-plane/internal/models"
)

var tenantFile = "data/tenants.json"
var tenantMu sync.Mutex

func LoadTenants() ([]models.Tenant, error) {
	data, err := os.ReadFile(tenantFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []models.Tenant{}, nil
		}
		return nil, err
	}
	var tenants []models.Tenant
	err = json.Unmarshal(data, &tenants)
	return tenants, err
}

func SaveTenants(tenants []models.Tenant) error {
	tenantMu.Lock()
	defer tenantMu.Unlock()

	if err := os.MkdirAll("data", 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(tenants, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(tenantFile, data, 0644)
}
