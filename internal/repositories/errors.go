package repositories

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/MathBriton/Obsidian-Chamados/internal/models"
)

// notFound traduz sql.ErrNoRows para o erro de domínio ErrNotFound,
// preservando qualquer outro erro inalterado.
func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return models.ErrNotFound
	}
	return err
}

// isUniqueViolation reporta se err é uma violação de UNIQUE no SQLite
// envolvendo todas as colunas informadas (ex: "users.email").
func isUniqueViolation(err error, columns ...string) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	if !strings.Contains(msg, "UNIQUE constraint failed") {
		return false
	}
	for _, col := range columns {
		if !strings.Contains(msg, col) {
			return false
		}
	}
	return true
}
