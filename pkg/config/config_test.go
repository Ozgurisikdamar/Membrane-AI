package config_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/config"
)

func TestLoader_Defaults(t *testing.T) {
	l := config.NewLoader("MEMBRANE_TEST")
	if got := l.String("HTTP_ADDR", ":8001"); got != ":8001" {
		t.Errorf("String default = %q", got)
	}
	if got := l.Int("PORT", 9001); got != 9001 {
		t.Errorf("Int default = %d", got)
	}
	if got := l.Duration("TIMEOUT", 1200*time.Millisecond); got != 1200*time.Millisecond {
		t.Errorf("Duration default = %v", got)
	}
	if got := l.Bool("DEBUG", false); got != false {
		t.Errorf("Bool default = %v", got)
	}
	if err := l.Err(); err != nil {
		t.Fatalf("Err on all-defaults = %v", err)
	}
}

func TestLoader_ReadsAndParses(t *testing.T) {
	t.Setenv("MEMBRANE_TEST_HTTP_ADDR", ":8080")
	t.Setenv("MEMBRANE_TEST_PORT", "9100")
	t.Setenv("MEMBRANE_TEST_TIMEOUT", "2s")
	t.Setenv("MEMBRANE_TEST_DEBUG", "true")

	l := config.NewLoader("MEMBRANE_TEST")
	if l.String("HTTP_ADDR", ":8001") != ":8080" {
		t.Error("String not read from env")
	}
	if l.Int("PORT", 1) != 9100 {
		t.Error("Int not read from env")
	}
	if l.Duration("TIMEOUT", 0) != 2*time.Second {
		t.Error("Duration not read from env")
	}
	if !l.Bool("DEBUG", false) {
		t.Error("Bool not read from env")
	}
	if err := l.Err(); err != nil {
		t.Fatalf("Err = %v", err)
	}
}

func TestLoader_MissingRequiredAndBadParse(t *testing.T) {
	t.Setenv("MEMBRANE_TEST_PORT", "not-a-number")
	l := config.NewLoader("MEMBRANE_TEST")
	_ = l.Required("DB_URL") // missing
	_ = l.Int("PORT", 0)     // bad parse
	err := l.Err()
	if err == nil {
		t.Fatal("expected aggregated error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "MEMBRANE_TEST_DB_URL") || !strings.Contains(msg, "MEMBRANE_TEST_PORT") {
		t.Fatalf("error missing details: %v", msg)
	}
}

func TestLoader_NoPrefix(t *testing.T) {
	t.Setenv("RAW_KEY", "v")
	l := config.NewLoader("")
	if l.String("RAW_KEY", "") != "v" {
		t.Error("empty prefix should read bare key")
	}
}
