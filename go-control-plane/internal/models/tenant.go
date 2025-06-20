package models

type Tenant struct {
	TenantID    string `json:"tenant_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	NodeID      string `json:"node_id"` // NEW FIELD
}