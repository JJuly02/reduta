package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/JJuly02/reduta/internal/auth"
	"github.com/JJuly02/reduta/internal/config"
	"github.com/JJuly02/reduta/internal/db"
	"github.com/JJuly02/reduta/internal/httpserver"
	"github.com/JJuly02/reduta/internal/observability"
	"github.com/JJuly02/reduta/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	log := observability.NewLogger(cfg.LogLevel, cfg.Env)
	log.Info().Str("env", cfg.Env).Msg("reduta-server starting")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error().Err(err).Msg("database connect failed")
		os.Exit(1)
	}
	defer pool.Close()

	if err := db.Migrate(cfg.DatabaseURL); err != nil {
		log.Error().Err(err).Msg("migrations failed")
		os.Exit(1)
	}
	log.Info().Msg("migrations applied")

	st := store.New(pool)
	bootstrapAdmin(ctx, st, cfg, log)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpserver.New(cfg, st, log).Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	log.Info().Str("addr", cfg.HTTPAddr).Msg("listening")

	select {
	case err := <-errCh:
		log.Error().Err(err).Msg("http server failed")
		os.Exit(1)
	case <-ctx.Done():
		log.Info().Msg("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Shutdown)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("graceful shutdown failed")
	}
	log.Info().Msg("stopped")
}

// bootstrapAdmin creates an owner account from env on first start (dev/ops convenience).
func bootstrapAdmin(ctx context.Context, st *store.Store, cfg config.Config, log zeroLogger) {
	email := strings.TrimSpace(strings.ToLower(cfg.BootstrapAdminEmail))
	if email == "" || cfg.BootstrapAdminPassword == "" {
		return
	}
	if _, err := st.GetUserByEmail(ctx, email); err == nil {
		return // already exists
	}
	orgID, err := st.DefaultOrgID(ctx)
	if err != nil {
		log.Error().Err(err).Msg("bootstrap admin: no default org")
		return
	}
	hash, err := auth.HashPassword(cfg.BootstrapAdminPassword)
	if err != nil {
		log.Error().Err(err).Msg("bootstrap admin: hash failed")
		return
	}
	if _, err := st.CreateUser(ctx, orgID, email, "Admin", hash, "owner"); err != nil {
		if !errors.Is(err, store.ErrConflict) {
			log.Error().Err(err).Msg("bootstrap admin: create failed")
		}
		return
	}
	log.Info().Str("email", email).Msg("bootstrap owner created")
}
