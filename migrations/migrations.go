// Package migrations содержит SQL миграции базы данных и функцию для их применения.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
