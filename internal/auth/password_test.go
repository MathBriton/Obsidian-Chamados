package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword_NaoRetornaPlaintext(t *testing.T) {
	hash, err := HashPassword("secret123")
	require.NoError(t, err)
	assert.NotEqual(t, "secret123", hash)
	assert.True(t, strings.HasPrefix(hash, "$2"), "deve ser um hash bcrypt")
}

func TestCheckPassword_SenhaCorreta(t *testing.T) {
	hash, err := HashPassword("secret123")
	require.NoError(t, err)
	assert.NoError(t, CheckPassword(hash, "secret123"))
}

func TestCheckPassword_SenhaErrada(t *testing.T) {
	hash, err := HashPassword("secret123")
	require.NoError(t, err)
	assert.Error(t, CheckPassword(hash, "senhaerrada"))
}
