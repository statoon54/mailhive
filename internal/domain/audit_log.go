package domain

import (
	"time"

	"github.com/google/uuid"
)

// AuditLog représente une entrée du journal d'audit.
type AuditLog struct {
	CreatedAt    time.Time `json:"created_at"`
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	Action       string    `json:"action"`
	Details      string    `json:"details,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	Method       string    `json:"method"`
	Path         string    `json:"path"`
	ResourceID   string    `json:"resource_id,omitempty"`
	ResourceType string    `json:"resource_type"`
	Status       string    `json:"status"`
	TenantName   string    `json:"tenant_name,omitempty"`
	StatusCode   int       `json:"status_code"`
}

// AuditLogFilter contient les critères de filtrage pour le journal d'audit.
type AuditLogFilter struct {
	TenantID     *uuid.UUID `json:"-"`
	Status       *string    `json:"status,omitempty"`
	ResourceType *string    `json:"resource_type,omitempty"`
	Page         int        `json:"page"`
	Limit        int        `json:"limit"`
}
