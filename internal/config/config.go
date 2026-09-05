// Package config loads runtime configuration from the environment.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Env         string
	HTTPAddr    string
	DatabaseURL string
	RedisAddr   string
	LogLevel    string
	Shutdown    time.Duration

	SessionTTL       time.Duration
	SubmitRatePerMin int

	FlagSecret       string        // HMAC secret for per-team dynamic flags
	InstanceTTL      time.Duration
	InstanceMax      int

	// Optional bootstrap owner, created on startup if absent.
	BootstrapAdminEmail    string
	BootstrapAdminPassword string
}

func Load() (Config, error) {
	c := Config{
		Env:              env("REDUTA_ENV", "dev"),
		HTTPAddr:         env("REDUTA_HTTP_ADDR", ":8080"),
		DatabaseURL:      env("REDUTA_DATABASE_URL", "postgres://reduta:reduta@localhost:5432/reduta?sslmode=disable"),
		RedisAddr:        env("REDUTA_REDIS_ADDR", "localhost:6379"),
		LogLevel:         env("REDUTA_LOG_LEVEL", "info"),
		Shutdown:         envDuration("REDUTA_SHUTDOWN_TIMEOUT", 15*time.Second),
		SessionTTL:       envDuration("REDUTA_SESSION_TTL", 168*time.Hour),
		SubmitRatePerMin: envInt("REDUTA_SUBMIT_RATE_PER_MIN", 10),
		FlagSecret:       env("REDUTA_FLAG_SECRET", "dev-flag-secret"),
		InstanceTTL:      envDuration("REDUTA_INSTANCE_TTL", 45*time.Minute),
		InstanceMax:      envInt("REDUTA_INSTANCE_MAX", 100),
		BootstrapAdminEmail:    env("REDUTA_BOOTSTRAP_ADMIN_EMAIL", ""),
		BootstrapAdminPassword: env("REDUTA_BOOTSTRAP_ADMIN_PASSWORD", ""),
	}
	if c.DatabaseURL == "" {
		return Config{}, fmt.Errorf("REDUTA_DATABASE_URL is required")
	}
	return c, nil
}

func env(k, def string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envDuration(k string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
