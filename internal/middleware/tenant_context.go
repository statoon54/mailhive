package middleware

import (
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"github.com/statoon54/mailhive/internal/i18n"
	"github.com/statoon54/mailhive/internal/port"
	"github.com/statoon54/mailhive/internal/service"
)

// langFromRequest détermine la langue depuis l'en-tête Accept-Language.
func langFromRequest(c *echo.Context) i18n.Lang {
	return i18n.DetectLang(c.Request().Header.Get("Accept-Language"))
}

// TenantContext extrait le tenant_id du JWT et vérifie que le tenant existe en DB.
func TenantContext(tenantRepo port.TenantRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			l := langFromRequest(c)

			token, ok := c.Get("user").(*jwt.Token)
			if !ok || token == nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": i18n.T(l, "err.invalid_token")})
			}

			claims, ok := token.Claims.(*service.JWTClaims)
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": i18n.T(l, "err.invalid_claims")})
			}

			tenantID, err := uuid.Parse(claims.TenantID)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": i18n.T(l, "err.invalid_tenant_id")})
			}

			if _, err := tenantRepo.GetByID(c.Request().Context(), tenantID); err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": i18n.T(l, "err.tenant_expired")})
			}

			c.Set("tenant_id", claims.TenantID)
			c.Set("tenant_slug", claims.TenantSlug)
			c.Set("role", claims.Role)

			return next(c)
		}
	}
}

// AdminOnly vérifie que l'utilisateur a le rôle admin.
func AdminOnly() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			role, ok := c.Get("role").(string)
			if !ok || role != "admin" {
				l := langFromRequest(c)
				return c.JSON(http.StatusForbidden, map[string]string{"error": i18n.T(l, "err.admin_only")})
			}
			return next(c)
		}
	}
}
