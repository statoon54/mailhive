package domain

import "time"

// AppBranding représente la personnalisation de l'application.
type AppBranding struct {
	UpdatedAt   time.Time `json:"updated_at"`
	AppSubtitle string    `json:"app_subtitle"`
	AppTitle    string    `json:"app_title"`
	LogoURL     string    `json:"logo_url"`
	Timezone    string    `json:"timezone"`
}

// LogoData représente les données brutes du logo.
type LogoData struct {
	ContentType string
	Data        []byte
}

// UpdateBrandingRequest représente une demande de modification du branding.
type UpdateBrandingRequest struct {
	AppTitle    *string `json:"app_title,omitempty"`
	AppSubtitle *string `json:"app_subtitle,omitempty"`
	Timezone    *string `json:"timezone,omitempty"`
}
