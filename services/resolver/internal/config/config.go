// Package config loads the resolver service configuration from the environment
// (prefix MEMBRANE_RESOLVER_), with sane local defaults.
package config

import (
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/config"
)

// Config holds all runtime settings for the resolver service.
type Config struct {
	GRPCAddr        string
	HealthAddr      string
	DatabaseURL     string
	EmbedDim        int
	ShutdownTimeout time.Duration
}

// Load reads and validates configuration, failing fast on bad values.
func Load() (Config, error) {
	l := config.NewLoader("MEMBRANE_RESOLVER")
	c := Config{
		GRPCAddr:        l.String("GRPC_ADDR", ":9004"),
		HealthAddr:      l.String("HEALTH_ADDR", ":8104"),
		DatabaseURL:     l.String("DATABASE_URL", "postgres://membrane:membrane@localhost:5432/membrane"),
		EmbedDim:        l.Int("EMBED_DIM", 3072), // must match gold_codebase_index.embedding
		ShutdownTimeout: l.Duration("SHUTDOWN_TIMEOUT", 15*time.Second),
	}
	return c, l.Err()
}
