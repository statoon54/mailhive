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

// Le préfixe « all: » inclut les fichiers commençant par un point, ce qui
// permet au répertoire de n'être peuplé que de .gitkeep dans un dépôt frais :
// la sortie de build n'est plus versionnée, seul le marqueur l'est.
//
//go:embed all:dist
var distFS embed.FS

// RegisterRoutes enregistre le handler SPA sur le routeur Echo.
// Les requêtes vers /api/, /swagger, /monitoring sont ignorées (gérées par l'API).
func RegisterRoutes(e *echo.Echo) error {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return err
	}

	// Sans build du frontend, dist ne contient que le marqueur de répertoire.
	// Servir un message explicite vaut mieux qu'un 404 sans explication.
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		e.Use(notBuiltMiddleware())
		return nil
	}

	fileServer := http.FileServer(http.FS(sub))

	e.Use(spaMiddleware(sub, fileServer))

	return nil
}

// notBuiltPage est servie quand le binaire a été compilé sans frontend.
const notBuiltPage = `<!doctype html>
<html lang="fr">
<head><meta charset="utf-8"><title>MailHive</title></head>
<body>
  <h1>Frontend non construit</h1>
  <p>Ce binaire a été compilé sans l'interface web. Pour la construire :</p>
  <pre>make build-frontend &amp;&amp; make build-go</pre>
  <p>L'API reste disponible sous <code>/api/</code>.</p>
</body>
</html>`

// notBuiltMiddleware sert notBuiltPage pour toute route non applicative.
func notBuiltMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			path := c.Request().URL.Path
			if strings.HasPrefix(path, "/api/") ||
				strings.HasPrefix(path, "/swagger") ||
				strings.HasPrefix(path, "/monitoring") {
				return next(c)
			}
			return c.HTML(http.StatusOK, notBuiltPage)
		}
	}
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
