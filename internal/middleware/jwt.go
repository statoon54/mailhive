package middleware

import (
	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v5"
	"github.com/labstack/echo/v5"

	"github.com/statoon54/mailhive/internal/service"
)

// JWTMiddleware crée le middleware JWT pour Echo v5.
func JWTMiddleware(secret string) echo.MiddlewareFunc {
	return echojwt.WithConfig(echojwt.Config{
		SigningKey: []byte(secret),
		ContextKey: "user",
		NewClaimsFunc: func(c *echo.Context) jwt.Claims {
			return &service.JWTClaims{}
		},
	})
}
