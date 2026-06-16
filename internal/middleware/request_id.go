package middleware

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

// RequestID ajoute un identifiant unique à chaque requête.
func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			reqID := c.Request().Header.Get("X-Request-ID")
			if reqID == "" {
				reqID = uuid.New().String()
			}
			c.Set("request_id", reqID)
			c.Response().Header().Set("X-Request-ID", reqID)
			return next(c)
		}
	}
}
