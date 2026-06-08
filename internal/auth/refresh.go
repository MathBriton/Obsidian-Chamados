package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// refreshTokenBytes é a entropia do refresh token opaco. 32 bytes = 256 bits.
const refreshTokenBytes = 32

// GenerateRefreshToken cria um refresh token opaco aleatório, devolvido em
// texto puro (entregue ao cliente) e seu hash SHA-256 (persistido). O valor
// em texto puro nunca é armazenado (RNF03).
func GenerateRefreshToken() (token, hash string, err error) {
	buf := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	token = base64.RawURLEncoding.EncodeToString(buf)
	return token, HashRefreshToken(token), nil
}

// HashRefreshToken calcula o SHA-256 (hex) de um refresh token. Usado tanto
// na emissão quanto na validação para localizar o registro pelo hash.
func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
