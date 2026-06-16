package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/statoon54/mailhive/internal/i18n"
)

func validSettings() TenantSettings {
	return TenantSettings{
		RateLimit:        10,
		RateBurst:        10,
		MaxDestinataires: 100,
		DefaultPriority:  MailPriorityDefault,
	}
}

func TestTenantSettings_Validate_Valid(t *testing.T) {
	require.NoError(t, validSettings().Validate())
}

func TestTenantSettings_Validate_RateLimitZero(t *testing.T) {
	s := validSettings()
	s.RateLimit = 0
	err := s.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrValidation))
}

func TestTenantSettings_Validate_RateLimitNegative(t *testing.T) {
	s := validSettings()
	s.RateLimit = -1
	err := s.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrValidation))
}

func TestTenantSettings_Validate_BurstLessThanRate(t *testing.T) {
	s := validSettings()
	s.RateLimit = 20
	s.RateBurst = 10
	err := s.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrValidation))
	assert.Contains(t, err.Error(), "rate_burst")
}

func TestTenantSettings_Validate_MaxDestZero(t *testing.T) {
	s := validSettings()
	s.MaxDestinataires = 0
	err := s.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrValidation))
}

func TestTenantSettings_Validate_InvalidPriority(t *testing.T) {
	s := validSettings()
	s.DefaultPriority = "urgent"
	err := s.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrValidation))
	assert.Contains(t, err.Error(), "default_priority")
}

func TestTenantSettings_Validate_InvalidLanguage(t *testing.T) {
	s := validSettings()
	s.Language = "de"
	err := s.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrValidation))
	assert.Contains(t, err.Error(), "language")
}

func TestTenantSettings_Validate_ValidLanguage(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.FR, i18n.EN, ""} {
		s := validSettings()
		s.Language = lang
		assert.NoError(t, s.Validate(), "language %q should be valid", lang)
	}
}

func TestTenantSettings_Lang_DefaultFR(t *testing.T) {
	s := TenantSettings{}
	assert.Equal(t, i18n.FR, s.Lang())
}

func TestTenantSettings_Lang_ReturnsEN(t *testing.T) {
	s := TenantSettings{Language: i18n.EN}
	assert.Equal(t, i18n.EN, s.Lang())
}
