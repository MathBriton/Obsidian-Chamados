// Package testdb fornece um banco SQLite em memória com as migrations
// já aplicadas, para uso em testes de integração (RNF10).
package testdb

import (
	"database/sql"
	"testing"

	"github.com/MathBriton/Obsidian-Chamados/internal/database"
)

// New abre um SQLite :memory: isolado, aplica todas as migrations e
// registra o cleanup automático. Cada chamada retorna um banco limpo.
func New(t *testing.T) *sql.DB {
	t.Helper()

	// _pragma=foreign_keys(1) garante o enforcement de FKs no SQLite.
	db, err := database.Open("file::memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("abrir sqlite em memória: %v", err)
	}

	if err := database.Migrate(db); err != nil {
		t.Fatalf("aplicar migrations: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })
	return db
}
