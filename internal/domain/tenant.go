package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/statoon54/mailhive/internal/i18n"
)

// TenantSettings contient les paramètres personnalisables d'un tenant.
type TenantSettings struct {
	RateLimit          float64          `json:"rate_limit"`
	RateBurst          int              `json:"rate_burst"`
	MaxDestinataires   int              `json:"max_destinataires"`
	DefaultPriority    MailPriority     `json:"default_priority"`
	SpamScoreThreshold *float32         `json:"spam_score_threshold,omitempty"`
	SpamScoreAction    *SpamScoreAction `json:"spam_score_action,omitempty"`
	Language           i18n.Lang        `json:"language,omitempty"`
	StoreBody          bool             `json:"store_body"`
}

// Lang retourne la langue du tenant, avec fallback sur FR.
func (s TenantSettings) Lang() i18n.Lang {
	if s.Language == "" {
		return i18n.FR
	}
	return s.Language
}

// Validate vérifie la cohérence des paramètres du tenant.
func (s TenantSettings) Validate() error {
	if s.RateLimit <= 0 {
		return fmt.Errorf("%w : rate_limit doit être > 0", ErrValidation)
	}
	if s.RateBurst < int(s.RateLimit) {
		return fmt.Errorf(
			"%w : rate_burst (%d) doit être >= rate_limit (%d)",
			ErrValidation,
			s.RateBurst,
			int(s.RateLimit),
		)
	}
	if s.MaxDestinataires <= 0 {
		return fmt.Errorf("%w : max_destinataires doit être > 0", ErrValidation)
	}
	if s.DefaultPriority != "" && !ValidMailPriorities[s.DefaultPriority] {
		return fmt.Errorf(
			"%w : default_priority invalide : %s (valeurs : critical, default, low)",
			ErrValidation,
			s.DefaultPriority,
		)
	}
	if s.Language != "" && s.Language != i18n.FR && s.Language != i18n.EN {
		return fmt.Errorf("%w : language invalide : %s (valeurs : fr, en)", ErrValidation, s.Language)
	}
	return nil
}

// Tenant représente un locataire de l'application.
type Tenant struct {
	ID        uuid.UUID      `json:"id"`
	Name      string         `json:"name"`
	Slug      string         `json:"slug"`
	APIKey    string         `json:"api_key,omitempty"`
	IsActive  bool           `json:"is_active"`
	Settings  TenantSettings `json:"settings"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// CreateTenantRequest contient les données pour créer un tenant.
type CreateTenantRequest struct {
	Settings *TenantSettings `json:"settings,omitempty"`
	Name     string          `json:"name"               validate:"required"`
	Slug     string          `json:"slug"`
}

// UpdateTenantRequest contient les données pour modifier un tenant.
type UpdateTenantRequest struct {
	Name     *string         `json:"name,omitempty"`
	Slug     *string         `json:"slug,omitempty"`
	IsActive *bool           `json:"is_active,omitempty"`
	Settings *TenantSettings `json:"settings,omitempty"`
}
