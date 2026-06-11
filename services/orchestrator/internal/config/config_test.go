package config_test

import (
	"testing"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	c, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.HealthAddr != ":8102" {
		t.Fatalf("health addr default = %q", c.HealthAddr)
	}
	if c.SubmissionTopic != "code.submission.v1" || c.VerdictTopic != "code.verdict.v1" {
		t.Fatalf("topic defaults wrong: %+v", c)
	}
	if c.StageDeadline != 1200*time.Millisecond {
		t.Fatalf("stage deadline default = %v", c.StageDeadline)
	}
	if c.CacheTTL != 72*time.Hour {
		t.Fatalf("ttl default = %v", c.CacheTTL)
	}
	if c.RulesetVersion != "v1" {
		t.Fatalf("ruleset default = %q", c.RulesetVersion)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("MEMBRANE_ORCHESTRATOR_KAFKA_BROKERS", "k1:9092, k2:9092")
	t.Setenv("MEMBRANE_ORCHESTRATOR_STAGE_DEADLINE", "500ms")
	t.Setenv("MEMBRANE_ORCHESTRATOR_USE_IN_MEMORY", "yes")
	c, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(c.KafkaBrokers) != 2 || c.KafkaBrokers[1] != "k2:9092" {
		t.Fatalf("brokers = %v", c.KafkaBrokers)
	}
	if c.StageDeadline != 500*time.Millisecond {
		t.Fatalf("stage deadline = %v", c.StageDeadline)
	}
	if !c.UseInMemory {
		t.Fatal("UseInMemory should be true")
	}
}

func TestLoad_InvalidValueErrors(t *testing.T) {
	t.Setenv("MEMBRANE_ORCHESTRATOR_CACHE_TTL", "banana")
	if _, err := config.Load(); err == nil {
		t.Fatal("expected error for invalid duration")
	}
}
