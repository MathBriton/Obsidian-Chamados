package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/MathBriton/Obsidian-Chamados/internal/db"
)

// Store é o ponto de entrada da camada de dados: expõe os repositórios sobre
// o pool de conexões e oferece execução transacional via ExecTx.
type Store struct {
	*Repositories
	db *sql.DB
}

// NewStore monta um Store sobre um *sql.DB já aberto e migrado.
func NewStore(database *sql.DB) *Store {
	return &Store{
		Repositories: New(db.New(database)),
		db:           database,
	}
}

// ExecTx executa fn dentro de uma transação. Os repositórios entregues a fn
// operam sobre a transação; um erro retornado dispara rollback, caso
// contrário a transação é commitada. Usado em operações atômicas como o
// registro (cria tenant + usuário admin juntos).
func (s *Store) ExecTx(ctx context.Context, fn func(*Repositories) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if err := fn(New(db.New(tx))); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx falhou: %v; rollback falhou: %w", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}
