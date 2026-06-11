package config_test

import (
	"testing"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	c, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.GRPCAddr != ":9001" || c.HTTPAddr != ":8001" || c.HealthAddr != ":8101" {
		t.Fatalf("addr defaults wrong: %+v", c)
	}
	if c.KafkaTopic != "code.submission.v1" {
		t.Fatalf("topic default = %q", c.KafkaTopic)
	}
	if len(c.KafkaBrokers) != 1 || c.KafkaBrokers[0] != "localhost:9092" {
		t.Fatalf("brokers default = %v", c.KafkaBrokers)
	}
	if c.ShutdownTimeout != 15*time.Second {
		t.Fatalf("shutdown default = %v", c.ShutdownTimeout)
	}
}

func TestLoad_EnvOverrideAndCSV(t *testing.T) {
	t.Setenv("MEMBRANE_INGESTION_KAFKA_BROKERS", "a:9092, b:9092 ,c:9092")
	t.Setenv("MEMBRANE_INGESTION_USE_IN_MEMORY", "true")
	t.Setenv("MEMBRANE_INGESTION_SHUTDOWN_TIMEOUT", "5s")
	c, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(c.KafkaBrokers) != 3 || c.KafkaBrokers[1] != "b:9092" {
		t.Fatalf("CSV parse = %v", c.KafkaBrokers)
	}
	if !c.UseInMemory {
		t.Fatal("UseInMemory should be true")
	}
	if c.ShutdownTimeout != 5*time.Second {
		t.Fatalf("shutdown = %v", c.ShutdownTimeout)
	}
}

func TestLoad_InvalidDurationErrors(t *testing.T) {
	t.Setenv("MEMBRANE_INGESTION_SHUTDOWN_TIMEOUT", "not-a-duration")
	if _, err := config.Load(); err == nil {
		t.Fatal("expected error for invalid duration")
	}
}
