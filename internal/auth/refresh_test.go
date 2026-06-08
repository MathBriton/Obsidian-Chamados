package auth

import "testing"

func TestGenerateRefreshToken_UniqueAndHashed(t *testing.T) {
	token1, hash1, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}
	token2, hash2, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}

	if token1 == "" || hash1 == "" {
		t.Fatal("token e hash não podem ser vazios")
	}
	if token1 == token2 {
		t.Fatal("dois tokens gerados não deveriam colidir")
	}
	if hash1 == hash2 {
		t.Fatal("hashes de tokens distintos não deveriam colidir")
	}
	if token1 == hash1 {
		t.Fatal("o hash não pode ser igual ao token em texto puro")
	}
}

func TestHashRefreshToken_Deterministic(t *testing.T) {
	token, hash, _ := GenerateRefreshToken()
	if HashRefreshToken(token) != hash {
		t.Fatal("HashRefreshToken deveria ser determinístico e bater com o hash de emissão")
	}
	// SHA-256 em hex tem 64 caracteres.
	if len(hash) != 64 {
		t.Fatalf("hash com %d caracteres, esperado 64", len(hash))
	}
}
