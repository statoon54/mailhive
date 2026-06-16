package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestID_GeneratesID(t *testing.T) {
	e := echo.New()
	e.Use(RequestID())
	e.GET("/test", func(c *echo.Context) error {
		reqID, _ := c.Get("request_id").(string)
		assert.NotEmpty(t, reqID)
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("X-Request-ID"))
}

func TestRequestID_UsesExistingID(t *testing.T) {
	e := echo.New()
	e.Use(RequestID())
	e.GET("/test", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "my-custom-id")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, "my-custom-id", rec.Header().Get("X-Request-ID"))
}

func TestRequestID_UniquePerRequest(t *testing.T) {
	e := echo.New()
	e.Use(RequestID())
	e.GET("/test", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec1 := httptest.NewRecorder()
	e.ServeHTTP(rec1, req1)

	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)

	id1 := rec1.Header().Get("X-Request-ID")
	id2 := rec2.Header().Get("X-Request-ID")
	require.NotEmpty(t, id1)
	require.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
}
