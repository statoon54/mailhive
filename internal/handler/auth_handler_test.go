package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/test/mocks"
)

func TestAuthHandler_GenerateToken_Success(t *testing.T) {
	svc := &mocks.MockAuthService{Token: "jwt-token-123"}
	h := NewAuthHandler(svc)

	c, rec := newTestContext(http.MethodPost, "/api/v1/auth/token", map[string]string{"api_key": "test-key"})
	err := h.GenerateToken(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "jwt-token-123")
}

func TestAuthHandler_GenerateToken_EmptyKey(t *testing.T) {
	svc := &mocks.MockAuthService{}
	h := NewAuthHandler(svc)

	c, rec := newTestContext(http.MethodPost, "/api/v1/auth/token", map[string]string{"api_key": ""})
	err := h.GenerateToken(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuthHandler_GenerateToken_InvalidKey(t *testing.T) {
	svc := &mocks.MockAuthService{Err: domain.ErrInvalidAPIKey}
	h := NewAuthHandler(svc)

	c, rec := newTestContext(http.MethodPost, "/api/v1/auth/token", map[string]string{"api_key": "bad-key"})
	err := h.GenerateToken(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthHandler_RefreshToken_Success(t *testing.T) {
	svc := &mocks.MockAuthService{Token: "new-jwt-token"}
	h := NewAuthHandler(svc)

	c, rec := newTestContext(http.MethodPost, "/api/v1/auth/refresh", map[string]string{"token": "old-token"})
	err := h.RefreshToken(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "new-jwt-token")
}

func TestAuthHandler_RefreshToken_EmptyToken(t *testing.T) {
	svc := &mocks.MockAuthService{}
	h := NewAuthHandler(svc)

	c, rec := newTestContext(http.MethodPost, "/api/v1/auth/refresh", map[string]string{"token": ""})
	err := h.RefreshToken(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuthHandler_RefreshToken_Invalid(t *testing.T) {
	svc := &mocks.MockAuthService{Err: domain.ErrUnauthorized}
	h := NewAuthHandler(svc)

	c, rec := newTestContext(http.MethodPost, "/api/v1/auth/refresh", map[string]string{"token": "bad-token"})
	err := h.RefreshToken(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
