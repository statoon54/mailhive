package domain

import (
	"time"

	"github.com/google/uuid"
)

// AuditLog représente une entrée du journal d'audit.
type AuditLog struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	TenantName   string    `json:"tenant_name,omitempty"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id,omitempty"`
	Status       string    `json:"status"`
	StatusCode   int       `json:"status_code"`
	ErrorMessage string    `json:"error_message,omitempty"`
	Details      string    `json:"details,omitempty"`
	Method       string    `json:"method"`
	Path         string    `json:"path"`
	CreatedAt    time.Time `json:"created_at"`
}

// AuditLogFilter contient les critères de filtrage pour le journal d'audit.
type AuditLogFilter struct {
	TenantID     *uuid.UUID `json:"-"`
	Status       *string    `json:"status,omitempty"`
	ResourceType *string    `json:"resource_type,omitempty"`
	Page         int        `json:"page"`
	Limit        int        `json:"limit"`
}
