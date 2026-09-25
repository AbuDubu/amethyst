// Package migrations embeds the versioned SQL schema migrations so the server
// binary can apply them without shipping loose files.
package migrations

import "embed"

// FS holds every migration, named NNNNN_description.sql and applied in order.
//
//go:embed *.sql
var FS embed.FS
