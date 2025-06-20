package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/chaturanga836/storage_system/go-control-plane/internal/models"
	"github.com/chaturanga836/storage_system/go-control-plane/internal/utils"
)

type TenantRequest struct {
	Name    string `json:"name"`
	NodeID string `json:"node_id"`
}

func RegisterTenant(w http.ResponseWriter, r *http.Request) {
	var req TenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.NodeID == "" {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	tenants, _ := utils.LoadTenants()
	for _, t := range tenants {
		if t.Name == req.Name {
			http.Error(w, "Tenant already exists", http.StatusConflict)
			return
		}
	}

	newTenant := models.Tenant{
		TenantID: uuid.New().String(),
		Name:     req.Name,
		NodeID:   req.NodeID,
	}

	tenants = append(tenants, newTenant)

	if err := utils.SaveTenants(tenants); err != nil {
		http.Error(w, "Failed to save tenant", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newTenant)
}

type AssignRequest struct {
	TenantID string `json:"tenant_id"`
	NodeID  string `json:"node_id"`
}

func AssignNode(w http.ResponseWriter, r *http.Request) {
	var req AssignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TenantID == "" || req.NodeID == "" {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	tenants, err := utils.LoadTenants()
	if err != nil {
		http.Error(w, "Failed to load tenants", http.StatusInternalServerError)
		return
	}

	updated := false
	for i, tenant := range tenants {
		if tenant.TenantID == req.TenantID {
			tenants[i].NodeID = req.NodeID
			updated = true
			break
		}
	}

	if !updated {
		http.Error(w, "Tenant not found", http.StatusNotFound)
		return
	}

	if err := utils.SaveTenants(tenants); err != nil {
		http.Error(w, "Failed to update tenant", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "node assigned"})
}
