package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	echojwt "github.com/labstack/echo-jwt/v5"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/service"
	"github.com/statoon54/mailhive/internal/test/mocks"
)

const testSecret = "test-secret-key-for-tests!!!!!"

func generateToken(tenantID, slug, role string) string {
	claims := &service.JWTClaims{
		TenantID:   tenantID,
		TenantSlug: slug,
		Role:       role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(testSecret))
	return signed
}

func setupEchoWithMiddleware(tenantRepo *mocks.MockTenantRepo) *echo.Echo {
	e := echo.New()
	jwtMiddleware := echojwt.WithConfig(echojwt.Config{
		SigningKey:    []byte(testSecret),
		NewClaimsFunc: func(c *echo.Context) jwt.Claims { return &service.JWTClaims{} },
		ContextKey:    "user",
	})
	e.Use(jwtMiddleware)
	e.Use(TenantContext(tenantRepo))
	e.GET("/test", func(c *echo.Context) error {
		tid, _ := c.Get("tenant_id").(string)
		role, _ := c.Get("role").(string)
		return c.JSON(http.StatusOK, map[string]string{"tenant_id": tid, "role": role})
	})
	return e
}

func TestTenantContext_ValidJWT(t *testing.T) {
	tenantID := uuid.New()
	repo := mocks.NewMockTenantRepo()
	repo.Tenants[tenantID] = &domain.Tenant{ID: tenantID, IsActive: true}

	e := setupEchoWithMiddleware(repo)
	token := generateToken(tenantID.String(), "test", "tenant")

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTenantContext_NoToken(t *testing.T) {
	repo := mocks.NewMockTenantRepo()
	e := setupEchoWithMiddleware(repo)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestTenantContext_InvalidTenantID(t *testing.T) {
	repo := mocks.NewMockTenantRepo()
	e := setupEchoWithMiddleware(repo)
	token := generateToken("not-a-uuid", "test", "tenant")

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestTenantContext_TenantNotInDB(t *testing.T) {
	repo := mocks.NewMockTenantRepo()
	e := setupEchoWithMiddleware(repo)
	token := generateToken(uuid.New().String(), "test", "tenant")

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAdminOnly_AdminPasses(t *testing.T) {
	e := echo.New()
	e.Use(AdminOnly())
	e.GET("/admin", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("role", "admin")

	handler := AdminOnly()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	_ = handler(c)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAdminOnly_NonAdmin403(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("role", "tenant")

	handler := AdminOnly()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	_ = handler(c)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}
