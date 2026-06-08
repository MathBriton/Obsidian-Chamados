// Command server é o entrypoint HTTP do Obsidian Chamados.
// Abre o banco, aplica migrations, monta as dependências e sobe o servidor
// Gin com desligamento gracioso.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/MathBriton/Obsidian-Chamados/internal/auth"
	"github.com/MathBriton/Obsidian-Chamados/internal/config"
	"github.com/MathBriton/Obsidian-Chamados/internal/database"
	"github.com/MathBriton/Obsidian-Chamados/internal/handlers"
	"github.com/MathBriton/Obsidian-Chamados/internal/repositories"
	"github.com/MathBriton/Obsidian-Chamados/internal/services"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func run() error {
	cfg := config.Load()

	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		return err
	}

	store := repositories.NewStore(db)
	tokens := auth.NewTokenManager(cfg.JWTSecret, cfg.AccessTokenTTL)
	authService := services.NewAuthService(store, tokens, cfg.RefreshTokenTTL)
	router := handlers.New(authService, tokens).Router()

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Sobe o servidor em background e espera por sinal de término.
	serverErr := make(chan error, 1)
	go func() {
		log.Printf("ouvindo em %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		log.Println("desligando...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
