// Package frontend embarque le build React (SPA) et fournit un handler HTTP
// pour servir les fichiers statiques avec fallback sur index.html.
package frontend

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
)

//go:embed dist/*
var distFS embed.FS

// RegisterRoutes enregistre le handler SPA sur le routeur Echo.
// Les requêtes vers /api/, /swagger, /monitoring sont ignorées (gérées par l'API).
func RegisterRoutes(e *echo.Echo) error {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return err
	}

	fileServer := http.FileServer(http.FS(sub))

	e.Use(spaMiddleware(sub, fileServer))

	return nil
}

// spaMiddleware sert les fichiers statiques du frontend et redirige
// les routes inconnues vers index.html (comportement SPA).
func spaMiddleware(fsys fs.FS, fileServer http.Handler) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			path := c.Request().URL.Path

			// Laisser passer les routes API et outils
			if strings.HasPrefix(path, "/api/") ||
				strings.HasPrefix(path, "/swagger") ||
				strings.HasPrefix(path, "/monitoring") {
				return next(c)
			}

			// Vérifier si le fichier statique existe
			cleanPath := strings.TrimPrefix(path, "/")
			if cleanPath == "" {
				cleanPath = "index.html"
			}

			if _, err := fs.Stat(fsys, cleanPath); err == nil {
				fileServer.ServeHTTP(c.Response(), c.Request())
				return nil
			}

			// Fallback SPA : servir index.html pour les routes client-side
			c.Request().URL.Path = "/"
			fileServer.ServeHTTP(c.Response(), c.Request())
			return nil
		}
	}
}
