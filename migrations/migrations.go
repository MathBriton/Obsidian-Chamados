// Package migrations embute os arquivos .sql de migração no binário,
// permitindo que sejam aplicados no boot sem depender do filesystem.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
