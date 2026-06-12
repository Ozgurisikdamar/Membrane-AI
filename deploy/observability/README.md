# Observability stack (D-032)

The services export OTLP traces + metrics, gated on `*_OTLP_ENDPOINT`. For the
demo/dev pipeline, `deploy/compose/otel-collector.yaml` receives OTLP and prints
to its logs (`docker compose ... --profile observability up`).

## Production backends

Point the collector at real backends — Tempo (traces), Prometheus (metrics) —
then import `grafana-dashboard.json` into Grafana (uid `membrane-overview`). It
visualizes `membrane_verdicts_total{decision,source}`: verdict rate by decision,
the approve/reject/needs-review mix, cache effectiveness (pipeline vs cache), and
rejected-verdict count.

## SLOs (suggested)

- Detector accuracy: precision ≥ 0.95, recall ≥ 0.90, FP-rate ≤ 0.05 — gated in
  CI by `task eval` (the golden corpus currently scores 1.0 / 1.0 / 0).
- Pipeline: p95 verdict latency within the Saga stage deadline (1200 ms);
  alert on a rising `rejected` rate or a collapse in cache-sourced verdicts.
