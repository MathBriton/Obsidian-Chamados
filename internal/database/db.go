package database

import (
	"database/sql"

	_ "modernc.org/sqlite" // driver SQLite pure-Go (ADR-004)
)

// Open abre uma conexão com o SQLite usando o driver pure-Go.
// O dsn segue o formato modernc, ex: "file:chamados.db?_pragma=foreign_keys(1)".
func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// SQLite não lida bem com múltiplas conexões de escrita concorrentes.
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
