package migrations

import "embed"

// FS содержит встроенные SQL-миграции.
//
//go:embed *.sql
var FS embed.FS
