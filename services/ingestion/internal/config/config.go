// Package config loads the ingestion service configuration from the environment
// (prefix MEMBRANE_INGESTION_), with sane local defaults.
package config

import (
	"strings"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/config"
)

// Config holds all runtime settings for the ingestion service.
type Config struct {
	GRPCAddr        string
	HTTPAddr        string
	HealthAddr      string
	KafkaBrokers    []string
	KafkaTopic      string
	UseInMemory     bool // when true, publish in-memory (no Kafka) — local/dev only
	WebhookSecret   string
	ShutdownTimeout time.Duration
	// OTLPEndpoint is the OTLP/gRPC collector address (host:port); empty disables
	// tracing (D-032). OTLPInsecure sends over plaintext gRPC for dev collectors.
	OTLPEndpoint string
	OTLPInsecure bool
}

// Load reads and validates configuration, failing fast on bad values.
func Load() (Config, error) {
	l := config.NewLoader("MEMBRANE_INGESTION")
	c := Config{
		GRPCAddr:        l.String("GRPC_ADDR", ":9001"),
		HTTPAddr:        l.String("HTTP_ADDR", ":8001"),
		HealthAddr:      l.String("HEALTH_ADDR", ":8101"),
		KafkaBrokers:    splitCSV(l.String("KAFKA_BROKERS", "localhost:9092")),
		KafkaTopic:      l.String("KAFKA_TOPIC", "code.submission.v1"),
		UseInMemory:     l.Bool("USE_IN_MEMORY", false),
		WebhookSecret:   l.String("WEBHOOK_SECRET", ""),
		ShutdownTimeout: l.Duration("SHUTDOWN_TIMEOUT", 15*time.Second),
		OTLPEndpoint:    l.String("OTLP_ENDPOINT", ""),
		OTLPInsecure:    l.Bool("OTLP_INSECURE", false), // secure by default; dev opts in
	}
	return c, l.Err()
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
