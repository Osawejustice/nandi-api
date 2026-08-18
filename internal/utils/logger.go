package utils

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// NewLogger builds a process-wide zerolog logger.
func NewLogger(level, env string) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339Nano

	var writer io.Writer = os.Stdout
	if !strings.EqualFold(env, "production") {
		writer = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	}

	lvl, err := zerolog.ParseLevel(strings.ToLower(level))
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	return zerolog.New(writer).
		Level(lvl).
		With().
		Timestamp().
		Str("service", "nandi-api").
		Str("env", env).
		Logger()
}
