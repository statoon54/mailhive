package domain

import (
	"time"

	"github.com/google/uuid"
)

// Template représente un modèle de mail réutilisable.
type Template struct {
	Variables   map[string]string `json:"variables"`
	CreatedAt   time.Time         `json:"created_at"`
	ID          uuid.UUID         `json:"id"`
	TenantID    uuid.UUID         `json:"tenant_id"`
	UpdatedAt   time.Time         `json:"updated_at"`
	HTMLBody    string            `json:"html_body"`
	Name        string            `json:"name"`
	Slug        string            `json:"slug"`
	SubjectTmpl string            `json:"subject_tmpl"`
	TextBody    string            `json:"text_body"`
	IsActive    bool              `json:"is_active"`
}

// CreateTemplateRequest contient les données pour créer un template.
type CreateTemplateRequest struct {
	Variables   map[string]string `json:"variables,omitempty"`
	HTMLBody    string            `json:"html_body"`
	Name        string            `json:"name"                validate:"required"`
	Slug        string            `json:"slug"`
	SubjectTmpl string            `json:"subject_tmpl"        validate:"required"`
	TextBody    string            `json:"text_body"`
}

// UpdateTemplateRequest contient les données pour modifier un template.
type UpdateTemplateRequest struct {
	Name        *string           `json:"name,omitempty"`
	Slug        *string           `json:"slug,omitempty"`
	SubjectTmpl *string           `json:"subject_tmpl,omitempty"`
	TextBody    *string           `json:"text_body,omitempty"`
	HTMLBody    *string           `json:"html_body,omitempty"`
	Variables   map[string]string `json:"variables,omitempty"`
	IsActive    *bool             `json:"is_active,omitempty"`
}

// PreviewTemplateRequest contient les données pour prévisualiser un template.
type PreviewTemplateRequest struct {
	Data map[string]string `json:"data"`
}

// PreviewTemplateResponse contient le résultat de la prévisualisation.
type PreviewTemplateResponse struct {
	Subject  string `json:"subject"`
	TextBody string `json:"text_body"`
	HTMLBody string `json:"html_body"`
}
