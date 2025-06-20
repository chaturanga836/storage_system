package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/chaturanga836/storage_system/go-control-plane/internal/models"
)

var (
	tenantList []models.Tenant
	mu         sync.RWMutex
	dataFile   = "data/tenants.json"
)

func LoadTenantRegistry() error {
	mu.Lock()
	defer mu.Unlock()

	file, err := os.Open(dataFile)
	if err != nil {
		if os.IsNotExist(err) {
			tenantList = []models.Tenant{}
			return nil
		}
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(&tenantList)
}

func SaveTenantRegistry() error {
	mu.RLock()
	defer mu.RUnlock()

	file, err := os.Create(dataFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(tenantList)
}

func RegisterTenant(t models.Tenant) error {
	mu.Lock()
	defer mu.Unlock()

	for _, tenant := range tenantList {
		if tenant.TenantID  == t.TenantID  {
			return fmt.Errorf("tenant TenantID  already exists")
		}
	}

	tenantList = append(tenantList, t)
	return SaveTenantRegistry()
}

func AssignNodeToTenant(tenantID, nodeID string) error {
	mu.Lock()
	defer mu.Unlock()

	for i, t := range tenantList {
		if t.TenantID  == tenantID {
			tenantList[i].NodeID = nodeID
			return SaveTenantRegistry()
		}
	}
	return fmt.Errorf("tenant not found")
}

func GetTenantByID(id string) (models.Tenant, bool) {
	mu.RLock()
	defer mu.RUnlock()

	for _, t := range tenantList {
		if t.TenantID  == id {
			return t, true
		}
	}
	return models.Tenant{}, false
}
