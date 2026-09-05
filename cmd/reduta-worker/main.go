package main

import (
	"os"

	"github.com/JJuly02/reduta/internal/config"
	"github.com/JJuly02/reduta/internal/observability"
	"github.com/hibiken/asynq"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	log := observability.NewLogger(cfg.LogLevel, cfg.Env)
	log.Info().Str("redis", cfg.RedisAddr).Msg("reduta-worker starting")

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.RedisAddr},
		asynq.Config{Concurrency: 10},
	)
	mux := asynq.NewServeMux()
	// Task handlers (bulk actions, imports, provisioning) register from M2 on.
	if err := srv.Run(mux); err != nil {
		log.Error().Err(err).Msg("worker failed")
		os.Exit(1)
	}
}
