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
	// OTLPEndpoint is the OTLP/gRPC collector address (host:port); empty disables
	// tracing (D-032). OTLPInsecure sends over plaintext gRPC for dev collectors.
	OTLPEndpoint string
	OTLPInsecure bool
}

// Load reads and validates configuration, failing fast on bad values.
func Load() (Config, error) {
	l := config.NewLoader("MEMBRANE_ANALYZER")
	c := Config{
		GRPCAddr:        l.String("GRPC_ADDR", ":9003"),
		HealthAddr:      l.String("HEALTH_ADDR", ":8103"),
		ShutdownTimeout: l.Duration("SHUTDOWN_TIMEOUT", 15*time.Second),
		OTLPEndpoint:    l.String("OTLP_ENDPOINT", ""),
		OTLPInsecure:    l.Bool("OTLP_INSECURE", false), // secure by default; dev opts in
	}
	return c, l.Err()
}
