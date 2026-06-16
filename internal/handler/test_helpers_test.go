package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/labstack/echo/v5"
)

func newTestContext(method, path string, body any) (*echo.Context, *httptest.ResponseRecorder) {
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

func setTenantCtx(c *echo.Context, tenantID, role string) {
	c.Set("tenant_id", tenantID)
	c.Set("tenant_slug", "test")
	c.Set("role", role)
}

func setPathParams(c *echo.Context, params map[string]string) {
	pv := make(echo.PathValues, 0, len(params))
	for k, v := range params {
		pv = append(pv, echo.PathValue{Name: k, Value: v})
	}
	c.SetPathValues(pv)
}
