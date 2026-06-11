// Package config loads the analyzer service configuration from the environment
// (prefix MEMBRANE_ANALYZER_), with sane local defaults.
package config

import (
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/config"
)

// Config holds all runtime settings for the analyzer service.
type Config struct {
	GRPCAddr        string
	HealthAddr      string
	ShutdownTimeout time.Duration
}

// Load reads and validates configuration, failing fast on bad values.
func Load() (Config, error) {
	l := config.NewLoader("MEMBRANE_ANALYZER")
	c := Config{
		GRPCAddr:        l.String("GRPC_ADDR", ":9003"),
		HealthAddr:      l.String("HEALTH_ADDR", ":8103"),
		ShutdownTimeout: l.Duration("SHUTDOWN_TIMEOUT", 15*time.Second),
	}
	return c, l.Err()
}
