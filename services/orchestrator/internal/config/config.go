// Package config loads the orchestrator service configuration from the
// environment (prefix MEMBRANE_ORCHESTRATOR_), with sane local defaults.
package config

import (
	"strings"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/config"
)

// Config holds all runtime settings for the orchestrator service.
type Config struct {
	HealthAddr      string
	KafkaBrokers    []string
	SubmissionTopic string
	VerdictTopic    string
	ConsumerGroup   string
	RedisAddr       string
	// AnalyzerAddr is the gRPC address of the analyzer service; empty means
	// "use the in-process secret-scan stage only".
	AnalyzerAddr     string
	CacheTTL         time.Duration
	StageDeadline    time.Duration
	FallbackDeadline time.Duration
	RulesetVersion   string
	UseInMemory      bool // when true, run without Kafka/Redis — dev only
	ShutdownTimeout  time.Duration
}

// Load reads and validates configuration, failing fast on bad values.
func Load() (Config, error) {
	l := config.NewLoader("MEMBRANE_ORCHESTRATOR")
	c := Config{
		HealthAddr:       l.String("HEALTH_ADDR", ":8102"),
		KafkaBrokers:     splitCSV(l.String("KAFKA_BROKERS", "localhost:9092")),
		SubmissionTopic:  l.String("SUBMISSION_TOPIC", "code.submission.v1"),
		VerdictTopic:     l.String("VERDICT_TOPIC", "code.verdict.v1"),
		ConsumerGroup:    l.String("CONSUMER_GROUP", "orchestrator"),
		RedisAddr:        l.String("REDIS_ADDR", "localhost:6379"),
		AnalyzerAddr:     l.String("ANALYZER_ADDR", ""),
		CacheTTL:         l.Duration("CACHE_TTL", 72*time.Hour),
		StageDeadline:    l.Duration("STAGE_DEADLINE", 1200*time.Millisecond),
		FallbackDeadline: l.Duration("FALLBACK_DEADLINE", 200*time.Millisecond),
		RulesetVersion:   l.String("RULESET_VERSION", "v1"),
		UseInMemory:      l.Bool("USE_IN_MEMORY", false),
		ShutdownTimeout:  l.Duration("SHUTDOWN_TIMEOUT", 15*time.Second),
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
