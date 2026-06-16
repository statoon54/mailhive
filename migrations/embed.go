package migrations

import "embed"

// FS contient les fichiers de migration SQL embarqués.
//
//go:embed *.sql
var FS embed.FS
