// Package config loads the gateway service configuration from the environment
// (prefix MEMBRANE_GATEWAY_), with sane local defaults.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/config"
	"github.com/Ozgurisikdamar/Membrane-AI/services/gateway/internal/domain"
)

// Config holds all runtime settings for the gateway service.
type Config struct {
	HTTPAddr        string
	HealthAddr      string
	ShutdownTimeout time.Duration
	// Policy is the governance policy: loaded from PolicyFile when set, else
	// the built-in DefaultPolicy.
	Policy domain.Policy
	// OTLPEndpoint enables tracing; empty disables it (D-032).
	OTLPEndpoint string
	OTLPInsecure bool
}

// Load reads and validates configuration, failing fast on bad values.
func Load() (Config, error) {
	l := config.NewLoader("MEMBRANE_GATEWAY")
	policyFile := l.String("POLICY_FILE", "")
	c := Config{
		HTTPAddr:        l.String("HTTP_ADDR", ":8006"),
		HealthAddr:      l.String("HEALTH_ADDR", ":8106"),
		ShutdownTimeout: l.Duration("SHUTDOWN_TIMEOUT", 15*time.Second),
		OTLPEndpoint:    l.String("OTLP_ENDPOINT", ""),
		OTLPInsecure:    l.Bool("OTLP_INSECURE", false),
		Policy:          domain.DefaultPolicy(),
	}
	if err := l.Err(); err != nil {
		return Config{}, err
	}
	if policyFile != "" {
		p, err := loadPolicy(policyFile)
		if err != nil {
			return Config{}, err
		}
		c.Policy = p
	}
	return c, nil
}

func loadPolicy(path string) (domain.Policy, error) {
	data, err := os.ReadFile(path) //nolint:gosec // operator-supplied policy path
	if err != nil {
		return domain.Policy{}, fmt.Errorf("read policy file: %w", err)
	}
	var p domain.Policy
	if err := json.Unmarshal(data, &p); err != nil {
		return domain.Policy{}, fmt.Errorf("parse policy file: %w", err)
	}
	return p, nil
}
