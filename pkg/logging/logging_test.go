package logging_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/logging"
)

func TestNew_EmitsJSONAtLevel(t *testing.T) {
	var buf bytes.Buffer
	log := logging.New("warn", &buf)

	log.Info("ignored") // below warn → no output
	if buf.Len() != 0 {
		t.Fatalf("info should be filtered at warn level, got %q", buf.String())
	}

	log.Warn("kept", "k", "v")
	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("output is not JSON: %v (%q)", err, buf.String())
	}
	if rec["msg"] != "kept" || rec["k"] != "v" || rec["level"] != "WARN" {
		t.Fatalf("unexpected record: %v", rec)
	}
}

func TestTraceID_RoundTrip(t *testing.T) {
	ctx := logging.WithTraceID(context.Background(), "abc-123")
	if got := logging.TraceID(ctx); got != "abc-123" {
		t.Fatalf("TraceID = %q, want abc-123", got)
	}
	if got := logging.TraceID(context.Background()); got != "" {
		t.Fatalf("TraceID on empty ctx = %q, want empty", got)
	}
}

func TestNew_LevelParsing(t *testing.T) {
	cases := map[string]bool{ // level string -> whether a debug line is emitted
		"debug":   true,
		"error":   false,
		"warning": false,
		"bogus":   false, // unknown -> info, so debug is filtered
	}
	for level, debugVisible := range cases {
		var buf bytes.Buffer
		logging.New(level, &buf).Debug("d")
		if got := buf.Len() > 0; got != debugVisible {
			t.Errorf("level %q: debug visible = %v, want %v", level, got, debugVisible)
		}
	}
}

func TestFromContext_NoTraceID_ReturnsBase(t *testing.T) {
	var buf bytes.Buffer
	base := logging.New("info", &buf)
	if logging.FromContext(context.Background(), base) != base {
		t.Fatal("FromContext without trace id should return the base logger unchanged")
	}
}

func TestFromContext_AddsTraceID(t *testing.T) {
	var buf bytes.Buffer
	base := logging.New("info", &buf)
	ctx := logging.WithTraceID(context.Background(), "tid-9")

	logging.FromContext(ctx, base).Info("hello")

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	if rec["trace_id"] != "tid-9" {
		t.Fatalf("trace_id = %v, want tid-9", rec["trace_id"])
	}
}
