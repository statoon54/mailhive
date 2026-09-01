package handler

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"io"

	"github.com/labstack/echo/v5"
)

// JSONSerializer branche Echo sur encoding/json/v2.
//
// Sans ce sérialiseur, tous les c.JSON(...) passeraient par
// echo.DefaultJSONSerializer, câblé en dur sur encoding/json v1 : changer les
// imports des handlers ne suffit donc pas à migrer les réponses HTTP.
type JSONSerializer struct{}

// decodeOptions regroupe les options appliquées au décodage des requêtes.
//
// MatchCaseInsensitiveNames restaure la tolérance à la casse de la v1. La v2
// est sensible à la casse et ignore silencieusement les membres inconnus : sans
// cette option, un client envoyant « Template_ID » recevrait un 200 avec le
// champ perdu, sans erreur ni trace. Tolérance transitoire, à retirer lors d'une
// version majeure une fois l'échéance annoncée aux intégrateurs.
var decodeOptions = json.JoinOptions(
	json.MatchCaseInsensitiveNames(true),
)

// NewEcho crée une instance Echo configurée pour json/v2.
//
// Point de passage unique, utilisé aussi bien par le serveur que par les tests :
// une instance créée directement avec echo.New() retomberait sur le sérialiseur
// v1 par défaut, et les tests valideraient alors un format différent de celui
// que la production émet.
func NewEcho() *echo.Echo {
	e := echo.New()
	e.JSONSerializer = JSONSerializer{}
	return e
}

// Serialize écrit la valeur en JSON dans la réponse.
//
// La v1 utilisait json.Encoder.Encode, qui termine par un saut de ligne, là où
// json.MarshalWrite n'en écrit pas : on le rétablit pour que les corps de
// réponse restent identiques octet pour octet.
func (JSONSerializer) Serialize(c *echo.Context, target any, indent string) error {
	w := c.Response()
	var err error
	if indent != "" {
		err = json.MarshalWrite(w, target, jsontext.WithIndent(indent))
	} else {
		err = json.MarshalWrite(w, target)
	}
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, "\n")
	return err
}

// Deserialize lit le corps de la requête et le décode dans target.
func (JSONSerializer) Deserialize(c *echo.Context, target any) error {
	if err := json.UnmarshalRead(c.Request().Body, target, decodeOptions); err != nil {
		return echo.ErrBadRequest.Wrap(err)
	}
	return nil
}
