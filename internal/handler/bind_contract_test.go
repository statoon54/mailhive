package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/statoon54/mailhive/internal/domain"
)

// Ces tests figent le contrat des erreurs de binding JSON. bindRequest traduit
// les erreurs du moteur JSON en messages i18n : c'est le point du code le plus
// couplé à l'implémentation d'encoding/json, et donc celui qui casse le plus
// silencieusement lors de la migration vers json/v2 (les types d'erreur changent,
// et un mapping non mis à jour retomberait sur le message générique sans que rien
// n'échoue).

// bindRaw exécute bindRequest sur un corps brut et retourne le code et le corps
// de la réponse. Le corps est passé tel quel — contrairement aux autres
// helpers de test, on a besoin de JSON volontairement invalide.
func bindRaw(t *testing.T, body, acceptLanguage string) (int, string, domain.CreateMailRequest) {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/mails", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", acceptLanguage)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var parsed domain.CreateMailRequest
	require.NoError(t, bindRequest(c, &parsed))
	return rec.Code, strings.TrimSpace(rec.Body.String()), parsed
}

// TestBindContract_SyntaxError fige la réponse pour un JSON syntaxiquement invalide.
func TestBindContract_SyntaxError(t *testing.T) {
	t.Run("français", func(t *testing.T) {
		code, body, _ := bindRaw(t, `{bad}`, "fr")
		assert.Equal(t, http.StatusBadRequest, code)
		assert.JSONEq(t, `{
			"error": "Format JSON invalide",
			"fields": [{"field": "json", "message": "Erreur de syntaxe près de : ...{bad}..."}],
			"success": false
		}`, body)
	})

	t.Run("anglais", func(t *testing.T) {
		code, body, _ := bindRaw(t, `{bad}`, "en")
		assert.Equal(t, http.StatusBadRequest, code)
		assert.JSONEq(t, `{
			"error": "Invalid JSON format",
			"fields": [{"field": "json", "message": "Syntax error near: ...{bad}..."}],
			"success": false
		}`, body)
	})

	t.Run("JSON tronqué", func(t *testing.T) {
		code, body, _ := bindRaw(t, `{"subject": "a"`, "fr")
		assert.Equal(t, http.StatusBadRequest, code)
		assert.Contains(t, body, "Erreur de syntaxe près de")
	})
}

// TestBindContract_TypeError fige la réponse pour un type de champ incorrect.
func TestBindContract_TypeError(t *testing.T) {
	t.Run("chaîne attendue, nombre reçu", func(t *testing.T) {
		code, body, _ := bindRaw(t, `{"subject": 123}`, "fr")
		assert.Equal(t, http.StatusBadRequest, code)
		assert.JSONEq(t, `{
			"error": "Format JSON invalide",
			"fields": [{"field": "subject", "message": "Attendu chaîne de caractères, reçu number"}],
			"success": false
		}`, body)
	})

	t.Run("anglais", func(t *testing.T) {
		code, body, _ := bindRaw(t, `{"subject": 123}`, "en")
		assert.Equal(t, http.StatusBadRequest, code)
		assert.JSONEq(t, `{
			"error": "Invalid JSON format",
			"fields": [{"field": "subject", "message": "Expected string, got number"}],
			"success": false
		}`, body)
	})

	t.Run("tableau attendu, chaîne reçue", func(t *testing.T) {
		// Contrat actuel, volontairement figé tel quel : le type attendu est
		// vide (« Attendu , reçu … ») parce que reflect.Type.Name() ne retourne
		// rien pour un type slice. C'est un défaut préexistant, corrigé lors du
		// passage à json/v2 — ce cas documente l'état d'avant.
		code, body, _ := bindRaw(t, `{"to": "pas-un-tableau"}`, "fr")
		assert.Equal(t, http.StatusBadRequest, code)
		assert.Contains(t, body, `"field":"to"`)
		assert.Contains(t, body, "Attendu , reçu chaîne de caractères")
	})
}

// TestBindContract_Strictness fige les deux comportements permissifs de v1 que
// json/v2 durcit : la tolérance à la casse et l'acceptation des clés dupliquées.
func TestBindContract_Strictness(t *testing.T) {
	t.Run("la casse des noms de champs est tolérée", func(t *testing.T) {
		code, _, parsed := bindRaw(t, `{"SUBJECT": "Sujet"}`, "fr")
		assert.Equal(t, http.StatusOK, code, "aucune réponse d'erreur écrite")
		assert.Equal(t, "Sujet", parsed.Subject,
			"v1 associe SUBJECT au champ subject ; json/v2 ne le fait qu'avec MatchCaseInsensitiveNames")
	})

	t.Run("une clé dupliquée est acceptée, la dernière gagne", func(t *testing.T) {
		code, _, parsed := bindRaw(t, `{"subject":"a","subject":"b"}`, "fr")
		assert.Equal(t, http.StatusOK, code, "aucune réponse d'erreur écrite")
		assert.Equal(t, "b", parsed.Subject,
			"v1 retient la dernière occurrence ; json/v2 rejette le document")
	})
}

// TestBindContract_Valid vérifie qu'un corps valide ne produit aucune réponse.
func TestBindContract_Valid(t *testing.T) {
	code, body, parsed := bindRaw(t,
		`{"subject":"Sujet","to":[{"email":"destinataire@exemple.test"}]}`, "fr")
	assert.Equal(t, http.StatusOK, code)
	assert.Empty(t, body)
	assert.Equal(t, "Sujet", parsed.Subject)
	require.Len(t, parsed.To, 1)
}
