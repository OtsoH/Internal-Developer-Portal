// Package migrations embeds the SQL migration files so the binary can run
// them at startup without a separate migrate CLI.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
