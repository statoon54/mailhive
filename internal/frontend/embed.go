// Package frontend embarque le build React (SPA) et fournit un handler HTTP
// pour servir les fichiers statiques avec fallback sur index.html.
package frontend

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
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
	return registerFS(e, sub)
}

// registerFS monte le handler SPA sur un système de fichiers quelconque, ce qui
// permet de le tester sans dépendre du dist embarqué — vide hors build du
// frontend, notamment en CI.
func registerFS(e *echo.Echo, fsys fs.FS) error {
	// Sans build du frontend, dist ne contient que le marqueur de répertoire.
	// Servir un message explicite vaut mieux qu'un 404 sans explication.
	if _, err := fs.Stat(fsys, "index.html"); err != nil {
		e.Use(notBuiltMiddleware())
		return nil
	}

	etags, err := computeETags(fsys)
	if err != nil {
		return err
	}

	e.Use(spaMiddleware(fsys, http.FileServer(http.FS(fsys)), etags))

	return nil
}

// assetsPrefix désigne les fichiers dont Vite hache le nom. Leur contenu ne
// change jamais sous un même nom : le navigateur peut les garder indéfiniment.
const assetsPrefix = "assets/"

// immutableCache autorise la conservation d'un fichier haché pendant un an.
const immutableCache = "public, max-age=31536000, immutable"

// computeETags empreinte chaque fichier servi.
//
// Un embed.FS rapporte une date de modification nulle : sans ETag, ni
// http.FileServer ni http.ServeContent n'émettent de validateur, le navigateur
// applique sa mise en cache heuristique et continue de servir l'ancien
// index.html — donc l'ancien bundle — après un déploiement.
func computeETags(fsys fs.FS) (map[string]string, error) {
	etags := make(map[string]string)

	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		content, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(content)
		etags[path] = `"` + hex.EncodeToString(sum[:]) + `"`
		return nil
	})
	if err != nil {
		return nil, err
	}

	return etags, nil
}

// setCacheHeaders pose le régime de cache correspondant au fichier servi.
// http.ServeContent lit l'ETag déjà présent sur la réponse et répond 304 de
// lui-même aux requêtes conditionnelles.
func setCacheHeaders(h http.Header, path string, etags map[string]string) {
	if etag, ok := etags[path]; ok {
		h.Set("ETag", etag)
	}

	if strings.HasPrefix(path, assetsPrefix) {
		h.Set("Cache-Control", immutableCache)
		return
	}

	// index.html et les fichiers au nom stable changent d'un déploiement à
	// l'autre : le navigateur doit revalider avant de les réutiliser.
	// « no-cache » n'interdit pas de les conserver, il impose cette question.
	h.Set("Cache-Control", "no-cache")
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
			c.Response().Header().Set("Cache-Control", "no-cache")
			return c.HTML(http.StatusOK, notBuiltPage)
		}
	}
}

// spaMiddleware sert les fichiers statiques du frontend et redirige
// les routes inconnues vers index.html (comportement SPA).
func spaMiddleware(fsys fs.FS, fileServer http.Handler, etags map[string]string) echo.MiddlewareFunc {
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
				setCacheHeaders(c.Response().Header(), cleanPath, etags)
				fileServer.ServeHTTP(c.Response(), c.Request())
				return nil
			}

			// Fallback SPA : servir index.html pour les routes client-side
			setCacheHeaders(c.Response().Header(), "index.html", etags)
			c.Request().URL.Path = "/"
			fileServer.ServeHTTP(c.Response(), c.Request())
			return nil
		}
	}
}
