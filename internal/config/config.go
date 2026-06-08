package config

import (
	"os"
	"time"
)

// Config agrega as variáveis de ambiente da aplicação.
type Config struct {
	Port            string
	DatabaseURL     string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

// Load lê a configuração do ambiente, aplicando defaults sensatos para dev.
func Load() Config {
	return Config{
		Port:            getEnv("PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "file:chamados.db?_pragma=foreign_keys(1)"),
		JWTSecret:       getEnv("JWT_SECRET", "dev-secret-change-me"),
		AccessTokenTTL:  15 * time.Minute,   // RNF04
		RefreshTokenTTL: 7 * 24 * time.Hour, // RNF04
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
