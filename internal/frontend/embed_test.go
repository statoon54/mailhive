package frontend

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestServer monte le handler SPA sur un système de fichiers en mémoire :
// le dist embarqué est vide hors build du frontend, notamment en CI.
func newTestServer(t *testing.T) *echo.Echo {
	t.Helper()

	files := fstest.MapFS{
		"index.html":               {Data: []byte("<!doctype html><title>MailHive</title>")},
		"logo.svg":                 {Data: []byte("<svg/>")},
		"assets/index-V7zwEu2V.js": {Data: []byte("console.log(1)")},
	}

	e := echo.New()
	e.GET("/api/v1/health", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	require.NoError(t, registerFS(e, files))
	return e
}

func get(t *testing.T, e *echo.Echo, path string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestCacheHeaders(t *testing.T) {
	const immutable = "public, max-age=31536000, immutable"

	tests := []struct {
		name         string
		path         string
		wantCache    string
		wantETag     bool
		wantBodyPart string
	}{
		{
			name:         "un asset au nom haché se garde indéfiniment",
			path:         "/assets/index-V7zwEu2V.js",
			wantCache:    immutable,
			wantETag:     true,
			wantBodyPart: "console.log(1)",
		},
		{
			name:         "la racine sert index.html et doit être revalidée",
			path:         "/",
			wantCache:    "no-cache",
			wantETag:     true,
			wantBodyPart: "MailHive",
		},
		{
			name:         "une route SPA retombe sur index.html et doit être revalidée",
			path:         "/templates",
			wantCache:    "no-cache",
			wantETag:     true,
			wantBodyPart: "MailHive",
		},
		{
			name:         "un fichier au nom stable doit être revalidé",
			path:         "/logo.svg",
			wantCache:    "no-cache",
			wantETag:     true,
			wantBodyPart: "<svg/>",
		},
		{
			name:         "l'API n'est pas touchée",
			path:         "/api/v1/health",
			wantCache:    "",
			wantETag:     false,
			wantBodyPart: "ok",
		},
	}

	e := newTestServer(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := get(t, e, tt.path, nil)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Contains(t, rec.Body.String(), tt.wantBodyPart)
			assert.Equal(t, tt.wantCache, rec.Header().Get("Cache-Control"))
			if tt.wantETag {
				assert.NotEmpty(t, rec.Header().Get("ETag"))
			} else {
				assert.Empty(t, rec.Header().Get("ETag"))
			}
		})
	}
}

func TestCacheHeaders_RevalidationRepond304(t *testing.T) {
	// C'est ce test qui prouve l'intérêt de l'ETag : sans lui, « no-cache »
	// forcerait un téléchargement complet à chaque navigation.
	e := newTestServer(t)

	first := get(t, e, "/", nil)
	etag := first.Header().Get("ETag")
	require.NotEmpty(t, etag)

	second := get(t, e, "/", map[string]string{"If-None-Match": etag})

	assert.Equal(t, http.StatusNotModified, second.Code)
	assert.Empty(t, second.Body.String())
}

func TestCacheHeaders_ETagDistinctParFichier(t *testing.T) {
	e := newTestServer(t)

	index := get(t, e, "/", nil).Header().Get("ETag")
	logo := get(t, e, "/logo.svg", nil).Header().Get("ETag")

	assert.NotEqual(t, index, logo, "deux contenus différents doivent avoir des ETag différents")
}

func TestCacheHeaders_FallbackSPAPartageLETagDIndex(t *testing.T) {
	e := newTestServer(t)

	index := get(t, e, "/", nil).Header().Get("ETag")
	spa := get(t, e, "/templates", nil).Header().Get("ETag")

	assert.Equal(t, index, spa, "le fallback sert index.html, son ETag doit être celui d'index.html")
}
