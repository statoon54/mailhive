package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	e := NewEcho()
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
			"fields": [{"field": "subject", "message": "Attendu chaîne de caractères, reçu nombre"}],
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
		// Corrigé par la migration : le message indiquait auparavant « Attendu ,
		// reçu … », reflect.Type.Name() étant vide pour un type slice. Le
		// libellé se déduit désormais de la catégorie du type.
		code, body, _ := bindRaw(t, `{"to": "pas-un-tableau"}`, "fr")
		assert.Equal(t, http.StatusBadRequest, code)
		assert.Contains(t, body, `"field":"to"`)
		assert.Contains(t, body, "Attendu tableau, reçu chaîne de caractères")
	})

	t.Run("champ imbriqué : le nom vient du JSON Pointer", func(t *testing.T) {
		// « /to/0/email » : l'indice de tableau est ignoré, seul le nom du
		// champ fautif est remonté.
		code, body, _ := bindRaw(t, `{"to": [{"email": 123}]}`, "fr")
		assert.Equal(t, http.StatusBadRequest, code)
		assert.Contains(t, body, `"field":"email"`)
		assert.Contains(t, body, "Attendu chaîne de caractères, reçu nombre")
	})
}

// TestBindContract_Strictness fige les deux comportements permissifs de v1 face
// au durcissement de json/v2 : l'un est conservé, l'autre volontairement adopté.
func TestBindContract_Strictness(t *testing.T) {
	t.Run("la casse des noms de champs reste tolérée", func(t *testing.T) {
		// Conservé via MatchCaseInsensitiveNames. json/v2 est sensible à la
		// casse et ignore les membres inconnus : sans cette option, la requête
		// serait acceptée avec le champ silencieusement perdu.
		code, _, parsed := bindRaw(t, `{"SUBJECT": "Sujet"}`, "fr")
		assert.Equal(t, http.StatusOK, code, "aucune réponse d'erreur écrite")
		assert.Equal(t, "Sujet", parsed.Subject)
	})

	t.Run("une clé dupliquée est désormais rejetée", func(t *testing.T) {
		// Durcissement assumé : v1 retenait silencieusement la dernière
		// occurrence, ce qui masquait des requêtes ambiguës côté client.
		//
		// À noter : le décodage s'arrête sur la clé dupliquée, après avoir déjà
		// écrit les membres précédents dans la struct (subject vaut ici « a »).
		// La garantie porte sur le rejet, pas sur une struct intacte — les
		// appelants ne doivent pas lire req quand bindRequest a écrit une
		// réponse, ce que la signature impose déjà.
		code, body, _ := bindRaw(t, `{"subject":"a","subject":"b"}`, "fr")
		assert.Equal(t, http.StatusBadRequest, code)
		assert.Contains(t, body, "Format JSON invalide")
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
