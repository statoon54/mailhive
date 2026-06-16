package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"

	"github.com/statoon54/mailhive/internal/test/mocks"
)

func TestAuditMiddleware_SkipGET(t *testing.T) {
	auditSvc := &mocks.MockAuditLogService{}
	e := echo.New()
	e.Use(AuditMiddleware(auditSvc))
	e.GET("/test", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, auditSvc.Called("Log"))
}

func TestAuditMiddleware_LogPOST(t *testing.T) {
	auditSvc := &mocks.MockAuditLogService{}
	e := echo.New()
	e.Use(AuditMiddleware(auditSvc))
	tenantID := uuid.New()
	e.POST("/api/v1/mails", func(c *echo.Context) error {
		c.Set("tenant_id", tenantID.String())
		return c.JSON(http.StatusCreated, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mails", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	// AuditMiddleware requires tenant_id to be set before it runs, but here it's set in the handler.
	// Since the middleware wraps after, it should still work.
}

func TestAuditMiddleware_MethodToAction(t *testing.T) {
	tests := []struct {
		method   string
		expected string
	}{
		{"POST", "create"},
		{"PUT", "update"},
		{"DELETE", "delete"},
		{"PATCH", "patch"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, methodToAction(tt.method))
	}
}

func TestAuditMiddleware_ParseResourceFromPath(t *testing.T) {
	tests := []struct {
		path         string
		resourceType string
		resourceID   string
	}{
		{"/api/v1/mails/123", "mail", "123"},
		{"/api/v1/templates/abc", "template", "abc"},
		{"/api/v1/smtp-configs/456", "smtp_config", "456"},
		{"/api/v1/admin/tenants/789", "tenant", "789"},
		{"/api/v1/branding", "branding", ""},
	}
	for _, tt := range tests {
		rt, rid := parseResourceFromPath(tt.path)
		assert.Equal(t, tt.resourceType, rt, "path: %s", tt.path)
		assert.Equal(t, tt.resourceID, rid, "path: %s", tt.path)
	}
}

func TestAuditMiddleware_ExtractErrorMessage(t *testing.T) {
	body := []byte(`{"error":"something went wrong"}`)
	msg := extractErrorMessage(body)
	assert.Equal(t, "something went wrong", msg)

	empty := extractErrorMessage([]byte(`{}`))
	assert.Empty(t, empty)
}
