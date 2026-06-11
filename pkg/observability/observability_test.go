package observability_test

import (
	"context"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/observability"
)

func TestSetup_DisabledWhenNoEndpoint(t *testing.T) {
	shutdown, err := observability.Setup(context.Background(), observability.Config{
		ServiceName: "test", ServiceVersion: "0.0.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if shutdown == nil {
		t.Fatal("shutdown must never be nil")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("disabled shutdown must be a no-op, got %v", err)
	}
}

// startSpanContext returns a context carrying a real, sampled span context so
// the propagation round-trip has something to carry (the no-op tracer would not).
func startSpanContext(t *testing.T) context.Context {
	t.Helper()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	t.Cleanup(func() { span.End() })
	return ctx
}

func TestInjectExtract_RoundTripsTraceContext(t *testing.T) {
	// Propagator must be installed for inject/extract to do anything.
	if _, err := observability.Setup(context.Background(), observability.Config{ServiceName: "t"}); err != nil {
		t.Fatal(err)
	}
	src := startSpanContext(t)
	want := trace.SpanContextFromContext(src).TraceID()

	headers := observability.InjectHeaders(src)
	if _, ok := headers["traceparent"]; !ok {
		t.Fatalf("traceparent not injected: %v", headers)
	}

	// Extract into a FRESH context (simulates the consuming service).
	got := observability.ExtractContext(context.Background(), headers)
	remote := trace.SpanContextFromContext(got)
	if remote.TraceID() != want {
		t.Fatalf("trace id not propagated: got %s want %s", remote.TraceID(), want)
	}
	if !remote.IsRemote() {
		t.Fatal("extracted span context should be marked remote")
	}
}

func TestTraceID(t *testing.T) {
	if got := observability.TraceID(context.Background()); got != "" {
		t.Fatalf("no span ⇒ empty trace id, got %q", got)
	}
	ctx := startSpanContext(t)
	if got := observability.TraceID(ctx); len(got) != 32 {
		t.Fatalf("active span ⇒ 32-hex trace id, got %q", got)
	}
}

func TestInjectHeaders_EmptyWithoutContext(t *testing.T) {
	if _, err := observability.Setup(context.Background(), observability.Config{ServiceName: "t"}); err != nil {
		t.Fatal(err)
	}
	if h := observability.InjectHeaders(context.Background()); len(h) != 0 {
		t.Fatalf("no active span ⇒ no headers, got %v", h)
	}
}
