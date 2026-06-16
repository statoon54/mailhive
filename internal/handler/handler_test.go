package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/statoon54/mailhive/internal/domain"
)

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Name", "name"},
		{"FromEmail", "from_email"},
		{"HTMLBody", "h_t_m_l_body"},
		{"simple", "simple"},
		{"", ""},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, toSnakeCase(tt.input))
	}
}

func TestHandleError_TableDriven(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
	}{
		{"TemplateNotFound", domain.ErrTemplateNotFound, 404},
		{"SMTPConfigNotFound", domain.ErrSMTPConfigNotFound, 404},
		{"MailNotFound", domain.ErrMailNotFound, 404},
		{"TenantNotFound", domain.ErrTenantNotFound, 404},
		{"NotFound", domain.ErrNotFound, 404},
		{"Conflict", domain.ErrConflict, 409},
		{"Unauthorized", domain.ErrUnauthorized, 401},
		{"Forbidden", domain.ErrForbidden, 403},
		{"Validation", domain.ErrValidation, 400},
		{"InvalidAPIKey", domain.ErrInvalidAPIKey, 401},
		{"TenantInactive", domain.ErrTenantInactive, 403},
		{"SMTPConfigNotSet", domain.ErrSMTPConfigNotSet, 400},
		{"MailNotPending", domain.ErrMailNotPending, 409},
		{"MailNotFailed", domain.ErrMailNotFailed, 409},
		{"RateLimited", domain.ErrRateLimited, 429},
		{"SpamBlocked", domain.ErrSpamBlocked, 400},
		{"LinkCheckRateLimit", domain.ErrLinkCheckRateLimit, 429},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, rec := newTestContext("GET", "/test", nil)
			_ = handleError(c, tt.err)
			assert.Equal(t, tt.statusCode, rec.Code)
		})
	}
}

func TestHandleError_Unknown(t *testing.T) {
	c, rec := newTestContext("GET", "/test", nil)
	_ = handleError(c, assert.AnError)
	assert.Equal(t, 500, rec.Code)
}

func TestParseUUID_Valid(t *testing.T) {
	c, _ := newTestContext("GET", "/test/123e4567-e89b-12d3-a456-426614174000", nil)
	setPathParams(c, map[string]string{"id": "123e4567-e89b-12d3-a456-426614174000"})
	id, err := parseUUID(c)
	assert.NoError(t, err)
	assert.Equal(t, "123e4567-e89b-12d3-a456-426614174000", id.String())
}

func TestParseUUID_Invalid(t *testing.T) {
	c, _ := newTestContext("GET", "/test/invalid", nil)
	setPathParams(c, map[string]string{"id": "invalid"})
	_, err := parseUUID(c)
	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestPaginationParams_Defaults(t *testing.T) {
	c, _ := newTestContext("GET", "/test", nil)
	page, limit := paginationParams(c)
	assert.Equal(t, 1, page)
	assert.Equal(t, 20, limit)
}

func TestPaginationParams_Custom(t *testing.T) {
	c, _ := newTestContext("GET", "/test?page=3&limit=50", nil)
	page, limit := paginationParams(c)
	assert.Equal(t, 3, page)
	assert.Equal(t, 50, limit)
}

func TestValidateRequest_Valid(t *testing.T) {
	type Req struct {
		Name string `validate:"required"`
	}
	errs := validateRequest(Req{Name: "test"})
	assert.Nil(t, errs)
}

func TestValidateRequest_Invalid(t *testing.T) {
	type Req struct {
		Name string `validate:"required"`
	}
	errs := validateRequest(Req{})
	assert.NotEmpty(t, errs)
	assert.Equal(t, "name", errs[0].Field)
}
