package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAndParse_Roundtrip(t *testing.T) {
	m := NewTokenManager("test-secret", 15*time.Minute)

	tokenStr, err := m.GenerateAccessToken(5, 1, "agent")
	require.NoError(t, err)

	claims, err := m.ParseAccessToken(tokenStr)
	require.NoError(t, err)
	assert.Equal(t, int64(5), claims.UserID)
	assert.Equal(t, int64(1), claims.TenantID)
	assert.Equal(t, "agent", claims.Role)
}

func TestParse_SecretErrado(t *testing.T) {
	signer := NewTokenManager("secret-A", 15*time.Minute)
	verifier := NewTokenManager("secret-B", 15*time.Minute)

	tokenStr, err := signer.GenerateAccessToken(5, 1, "agent")
	require.NoError(t, err)

	_, err = verifier.ParseAccessToken(tokenStr)
	assert.Error(t, err, "token assinado com outro secret deve ser rejeitado")
}

func TestParse_TokenExpirado(t *testing.T) {
	m := NewTokenManager("test-secret", -1*time.Minute) // já expirado

	tokenStr, err := m.GenerateAccessToken(5, 1, "agent")
	require.NoError(t, err)

	_, err = m.ParseAccessToken(tokenStr)
	assert.Error(t, err, "token expirado deve ser rejeitado")
}
