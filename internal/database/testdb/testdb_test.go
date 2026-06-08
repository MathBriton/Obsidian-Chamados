package testdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Valida que o helper aplica as migrations e cria as 8 tabelas de negócio.
func TestNew_AplicaMigrations(t *testing.T) {
	db := New(t)

	wantTables := []string{
		"tenants", "users", "teams", "team_members",
		"categories", "tickets", "comments", "refresh_tokens",
	}
	for _, name := range wantTables {
		var got string
		err := db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, name,
		).Scan(&got)
		require.NoError(t, err, "tabela %q deveria existir", name)
		assert.Equal(t, name, got)
	}
}

// Garante isolamento: cada chamada a New retorna um banco vazio e independente.
func TestNew_BancoIsolado(t *testing.T) {
	db := New(t)
	_, err := db.Exec(`INSERT INTO tenants (name, slug) VALUES ('A', 'a')`)
	require.NoError(t, err)

	db2 := New(t)
	var count int
	require.NoError(t, db2.QueryRow(`SELECT COUNT(*) FROM tenants`).Scan(&count))
	assert.Equal(t, 0, count, "banco novo deve estar vazio")
}
