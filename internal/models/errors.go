package models

import "errors"

// Erros de domínio reutilizados pelas camadas service e handler.
// Cada erro mapeia para um código de API estável (RNF13).
var (
	ErrNotFound           = errors.New("recurso não encontrado")
	ErrInvalidCredentials = errors.New("credenciais inválidas")
	ErrInvalidToken       = errors.New("token inválido")
	ErrExpiredToken       = errors.New("token expirado")
	ErrSlugTaken          = errors.New("slug já em uso")
	ErrEmailTaken         = errors.New("email já em uso")
	ErrInactiveUser       = errors.New("usuário inativo")
	ErrForbidden          = errors.New("acesso negado")
	ErrCategoryTaken      = errors.New("categoria já existe")
	ErrInvalidCategory    = errors.New("categoria inválida")
	ErrInvalidAssignee    = errors.New("responsável inválido")
)
