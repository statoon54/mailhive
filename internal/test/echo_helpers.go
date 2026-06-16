package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/labstack/echo/v5"
)

// NewTestEchoContext crée un echo.Context de test avec un body JSON optionnel.
func NewTestEchoContext(method, path string, body any) (*echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	var req *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

// SetTenantContext simule le middleware JWT en injectant tenant_id, slug et role dans le contexte.
func SetTenantContext(c *echo.Context, tenantID, role string) {
	c.Set("tenant_id", tenantID)
	c.Set("tenant_slug", "test-tenant")
	c.Set("role", role)
}

// SetPathParams injecte les paramètres de route dans le contexte Echo v5.
func SetPathParams(c *echo.Context, params map[string]string) {
	pv := make(echo.PathValues, 0, len(params))
	for k, v := range params {
		pv = append(pv, echo.PathValue{Name: k, Value: v})
	}
	c.SetPathValues(pv)
}
