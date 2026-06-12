// Package observability is the shared OpenTelemetry foundation for every
// MEMBRANE.AI service: one Setup call wires a resource, an OTLP/gRPC trace
// exporter (gated on an endpoint so dev/test stay zero-dependency) and the W3C
// propagator. Helpers carry trace context across Kafka records and expose the
// active trace ID for log correlation.
//
// Design: the W3C propagator is ALWAYS installed so context flows even when
// tracing is disabled; only the exporter + SDK TracerProvider are conditional.
// With no endpoint the global provider stays the no-op, so spans cost nothing.
package observability

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// Config configures the trace pipeline for one service.
type Config struct {
	ServiceName    string
	ServiceVersion string
	// OTLPEndpoint is host:port of an OTLP/gRPC collector. Empty disables the
	// exporter entirely (no-op tracing) — the production-safe default.
	OTLPEndpoint string
	// Insecure sends over plaintext gRPC (dev collectors); production uses TLS.
	Insecure bool
}

// Shutdown flushes and stops the trace pipeline; safe to call on a disabled setup.
type Shutdown func(context.Context) error

// Setup installs the W3C propagator and, when an endpoint is configured, an OTLP
// trace exporter + SDK TracerProvider. The returned Shutdown is ALWAYS non-nil —
// even on error — so callers can safely `defer Stop(shutdown)` and treat a tracing
// setup failure as non-fatal (tracing is a non-critical subsystem; a misconfigured
// collector must never crash the data plane).
func Setup(ctx context.Context, cfg Config) (Shutdown, error) {
	noop := func(context.Context) error { return nil }

	// Always propagate W3C trace context + baggage, even with tracing off, so a
	// later-enabled service still sees inbound context.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	if cfg.OTLPEndpoint == "" {
		return noop, nil
	}

	// Merge with resource.Default() so spans carry the standard telemetry.sdk.*
	// identity attributes alongside our service.name/version.
	res, err := resource.Merge(resource.Default(), resource.NewWithAttributes(
		"", // no schema URL → never conflicts with Default()'s schema
		attribute.String("service.name", cfg.ServiceName),
		attribute.String("service.version", cfg.ServiceVersion),
	))
	if err != nil {
		return noop, err
	}

	opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint)}
	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}
	exporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return noop, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	// Metrics share the same endpoint/resource (D-032 increment 3).
	metricOpts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(cfg.OTLPEndpoint)}
	if cfg.Insecure {
		metricOpts = append(metricOpts, otlpmetricgrpc.WithInsecure())
	}
	metricExp, err := otlpmetricgrpc.New(ctx, metricOpts...)
	if err != nil {
		return tp.Shutdown, err // tracing is up; surface the metric error
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	// Combined shutdown flushes both pipelines.
	return func(c context.Context) error {
		tErr := tp.Shutdown(c)
		mErr := mp.Shutdown(c)
		if tErr != nil {
			return tErr
		}
		return mErr
	}, nil
}

// Stop runs a Shutdown under a bounded timeout — for `defer observability.Stop(sh)`
// in a composition root, so trace flushing never hangs process exit.
func Stop(shutdown Shutdown) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = shutdown(ctx)
}

// Tracer returns a named tracer from the global provider.
func Tracer(name string) trace.Tracer { return otel.Tracer(name) }

// Meter returns a named meter from the global provider (no-op until Setup wires
// an exporter, so instrument creation is always safe).
func Meter(name string) metric.Meter { return otel.Meter(name) }

// Start opens a span on the named tracer and returns the child context plus an
// end func — so adapters can instrument a boundary without importing the otel
// trace packages directly (keeps the otel surface inside this package). With
// tracing disabled the span is a no-op and end() is cheap.
func Start(ctx context.Context, tracer, span string) (context.Context, func()) {
	ctx, s := otel.Tracer(tracer).Start(ctx, span)
	return ctx, func() { s.End() }
}

// InjectHeaders serializes the active trace context into a string map suitable
// for attaching to a transport that has no native carrier — e.g. Kafka record
// headers. Returns an empty map when there is no active context.
func InjectHeaders(ctx context.Context) map[string]string {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return carrier
}

// ExtractContext returns ctx enriched with the trace context carried in headers
// (the inverse of InjectHeaders). Unknown/empty headers leave ctx unchanged.
func ExtractContext(ctx context.Context, headers map[string]string) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(headers))
}

// TraceID returns the active span's trace ID, or "" when there is none — the
// bridge to logging.WithTraceID for correlating logs with traces.
func TraceID(ctx context.Context) string {
	if sc := trace.SpanContextFromContext(ctx); sc.HasTraceID() {
		return sc.TraceID().String()
	}
	return ""
}
