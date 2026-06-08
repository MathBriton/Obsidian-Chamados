package auth

import "golang.org/x/crypto/bcrypt"

// bcryptCost define o custo do hash. Mínimo 12 conforme RNF03.
const bcryptCost = 12

// HashPassword gera o hash bcrypt de uma senha em texto puro.
func HashPassword(plain string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword compara uma senha em texto puro com um hash bcrypt.
// Retorna nil se corresponderem.
func CheckPassword(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
