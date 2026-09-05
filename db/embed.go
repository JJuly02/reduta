// Package migrations embeds the SQL migration files so the server ships as a
// single self-contained binary (ADR-001). SQL lives in db/migrations/.
package migrations

import "embed"

//go:embed migrations/*.sql
var FS embed.FS
