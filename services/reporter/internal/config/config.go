// Package config loads the reporter service configuration from the environment
// (prefix MEMBRANE_REPORTER_), with sane local defaults.
package config

import (
	"strings"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/config"
)

// Config holds all runtime settings for the reporter service.
type Config struct {
	HealthAddr    string
	KafkaBrokers  []string
	VerdictTopic  string
	ConsumerGroup string
	// WebhookURL is the Slack-compatible destination; empty disables the
	// webhook notifier (the log notifier is always on).
	WebhookURL string
	// GitHubToken enables the commit-status notifier when set. GitHubAPIURL
	// defaults to api.github.com (override for GHE).
	GitHubToken     string
	GitHubAPIURL    string
	ShutdownTimeout time.Duration
}

// Load reads and validates configuration, failing fast on bad values.
func Load() (Config, error) {
	l := config.NewLoader("MEMBRANE_REPORTER")
	c := Config{
		HealthAddr:      l.String("HEALTH_ADDR", ":8105"),
		KafkaBrokers:    splitCSV(l.String("KAFKA_BROKERS", "localhost:9092")),
		VerdictTopic:    l.String("VERDICT_TOPIC", "code.verdict.v1"),
		ConsumerGroup:   l.String("CONSUMER_GROUP", "membrane-reporter"),
		WebhookURL:      l.String("WEBHOOK_URL", ""),
		GitHubToken:     l.String("GITHUB_TOKEN", ""),
		GitHubAPIURL:    l.String("GITHUB_API_URL", ""),
		ShutdownTimeout: l.Duration("SHUTDOWN_TIMEOUT", 15*time.Second),
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
