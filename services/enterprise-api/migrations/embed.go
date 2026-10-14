// Package migrations embeds the SQL files executed by golang-migrate.
package migrations

import "embed"

// FS contains the ordered migration files.
//
//go:embed *.sql
var FS embed.FS
