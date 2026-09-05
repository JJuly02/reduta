// Package observability wires structured logging (and, later, OTel traces
// and Prometheus metrics per the spec section 3).
package observability

import (
	"os"

	"github.com/rs/zerolog"
)

func NewLogger(level, env string) zerolog.Logger {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)
	if env == "dev" {
		return zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	}
	return zerolog.New(os.Stderr).With().Timestamp().Str("service", "reduta").Logger()
}
