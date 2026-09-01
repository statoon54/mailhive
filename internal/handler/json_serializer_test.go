package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// respond exécute c.JSON à travers l'instance Echo configurée et retourne le
// corps brut, sans normalisation : c'est bien la sortie octet pour octet qui
// nous intéresse ici.
func respond(t *testing.T, value any) string {
	t.Helper()
	e := NewEcho()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	require.NoError(t, c.JSON(http.StatusOK, value))
	return rec.Body.String()
}

// TestJSONSerializer_UsesV2 vérifie que c.JSON passe bien par json/v2.
//
// Sans JSONSerializer enregistré, Echo utilise son sérialiseur par défaut câblé
// sur encoding/json v1 : ce test échouerait alors, ce qui est précisément son
// rôle — la bascule est invisible autrement.
func TestJSONSerializer_UsesV2(t *testing.T) {
	t.Run("les slices et maps nulles ne sont plus null", func(t *testing.T) {
		body := respond(t, struct {
			Items []string          `json:"items"`
			Meta  map[string]string `json:"meta"`
		}{})
		assert.Equal(t, `{"items":[],"meta":{}}`+"\n", body)
	})

	t.Run("le HTML n'est plus échappé", func(t *testing.T) {
		// v1 écrivait <b> ; v2 laisse les caractères tels quels.
		body := respond(t, struct {
			HTML string `json:"html"`
		}{HTML: "<b>&</b>"})
		assert.Equal(t, `{"html":"<b>&</b>"}`+"\n", body)
	})

	t.Run("le saut de ligne final de la v1 est conservé", func(t *testing.T) {
		// json.Encoder.Encode terminait par \n, pas json.MarshalWrite : on le
		// rétablit pour que les corps restent identiques octet pour octet.
		body := respond(t, struct {
			A int `json:"a"`
		}{A: 1})
		assert.Equal(t, "{\"a\":1}\n", body)
		assert.True(t, strings.HasSuffix(body, "\n"))
	})
}

// TestJSONSerializer_Deserialize vérifie le décodage des requêtes via Echo.
func TestJSONSerializer_Deserialize(t *testing.T) {
	type payload struct {
		Subject string `json:"subject"`
		Count   int    `json:"count"`
	}

	bind := func(t *testing.T, raw string) (payload, error) {
		t.Helper()
		e := NewEcho()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		var p payload
		return p, e.JSONSerializer.Deserialize(c, &p)
	}

	t.Run("corps valide", func(t *testing.T) {
		p, err := bind(t, `{"subject":"Sujet","count":3}`)
		require.NoError(t, err)
		assert.Equal(t, "Sujet", p.Subject)
		assert.Equal(t, 3, p.Count)
	})

	t.Run("la casse reste tolérée", func(t *testing.T) {
		p, err := bind(t, `{"Subject":"Sujet"}`)
		require.NoError(t, err)
		assert.Equal(t, "Sujet", p.Subject)
	})

	t.Run("clé dupliquée rejetée", func(t *testing.T) {
		_, err := bind(t, `{"subject":"a","subject":"b"}`)
		require.Error(t, err)
	})
}
