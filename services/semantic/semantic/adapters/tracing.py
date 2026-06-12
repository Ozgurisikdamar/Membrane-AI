"""OpenTelemetry tracing for the semantic service (D-032).

Mirrors the Go `pkg/observability` contract: tracing is gated on an OTLP
endpoint (empty = no-op) and the W3C propagator is installed so the service
joins the orchestrator's distributed trace. FastAPI auto-instrumentation opens
the inbound server span; the global propagator extracts the `traceparent`
header the orchestrator's otelhttp client sends.
"""

from __future__ import annotations

from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from fastapi import FastAPI

    from semantic.config import Config


def configure(app: FastAPI, cfg: Config) -> None:
    """Wire OTLP tracing onto the app when an endpoint is configured; otherwise
    do nothing (production-safe default — no exporter, no overhead)."""
    if not cfg.otlp_endpoint:
        return

    # Imported lazily so the dependency is only touched when tracing is enabled.
    from opentelemetry import trace
    from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
    from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
    from opentelemetry.propagate import set_global_textmap
    from opentelemetry.sdk.resources import Resource
    from opentelemetry.sdk.trace import TracerProvider
    from opentelemetry.sdk.trace.export import BatchSpanProcessor
    from opentelemetry.trace.propagation.tracecontext import TraceContextTextMapPropagator

    set_global_textmap(TraceContextTextMapPropagator())

    resource = Resource.create({"service.name": "semantic", "service.version": "0.1.0"})
    provider = TracerProvider(resource=resource)
    exporter = OTLPSpanExporter(endpoint=cfg.otlp_endpoint, insecure=cfg.otlp_insecure)
    provider.add_span_processor(BatchSpanProcessor(exporter))
    trace.set_tracer_provider(provider)

    FastAPIInstrumentor.instrument_app(app)
