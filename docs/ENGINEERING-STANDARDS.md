# ENGINEERING STANDARDS — the binding quality bar

This is the "ultra-premium, professional" bar the user asked for. It is **binding**: every session, any
model, must satisfy it before committing. When in doubt, choose the stricter option.

## 1. Architecture: hexagonal (ports & adapters)

- Each service has four inner rings: **domain** (pure types + rules, zero I/O), **app** (use-cases /
  application services), **ports** (interfaces the app needs/exposes), **adapters** (concrete I/O:
  gRPC, HTTP, Kafka, Redis, Postgres). `cmd/<svc>/main.go` is the **composition root** — the only place
  that constructs concrete adapters and wires them into use-cases.
- **Dependency rule:** imports point inward only. `domain` imports nothing internal. `app` imports
  `domain` + `ports`, never `adapters`. Adapters import `app`/`ports`/`domain`. Enforced by review and
  by `depguard` in `.golangci.yml`.

## 2. SOLID (concrete Go idioms)

- **S** — one reason to change per type; use-cases are single-purpose structs.
- **O** — extend via new adapters/strategies, not by editing existing ones.
- **L** — every implementation of a port fully honors its contract (incl. error semantics); proven by
  a shared port test-suite run against every adapter (real + fake).
- **I** — small, role-specific interfaces defined **in the consumer** (`ports`), e.g. `EventPublisher`,
  `GoldIndex`, `VerdictCache` — never a fat "Repository" of everything.
- **D** — use-cases depend on interfaces; concretes injected via constructors. **No global state, no
  package-level singletons, no `init()` side effects.**

## 3. Dependency injection

- Constructor injection only: `NewEnqueueSubmission(pub ports.EventPublisher, clk Clock) *EnqueueSubmission`.
- Pass `context.Context` as the first argument to every method that does I/O or can block.
- Inject `Clock`/`IDGen` (not `time.Now()`/random) so logic is deterministic and testable.

## 4. Reliability & data integrity (ACID + patterns)

- **ACID:** all multi-row DB mutations run in a single transaction; define isolation explicitly where it
  matters; use `SELECT … FOR UPDATE` for contended rows.
- **Transactional outbox** (D-013): never dual-write DB+Kafka. Write domain row + `outbox` row in one
  tx; a relay publishes and marks `published_at`. Consumers are **idempotent** (dedupe by event id).
- **Design patterns actually used** (don't over-pattern; use where they earn their keep):
  Saga (orchestrator), CQRS-lite (write path vs read/query path), Strategy (pluggable analyzers /
  model routers), Circuit Breaker + timeout + retry-with-jitter (all external LLM/network calls),
  Repository (persistence ports), Factory (adapter construction), Outbox (eventing).
- Every external call has a `context` deadline; no unbounded waits. Graceful shutdown drains in-flight
  work on SIGINT/SIGTERM.

## 5. Errors, logging, config

- **Errors:** typed sentinel/domain errors in `pkg/errs`; wrap with `fmt.Errorf("…: %w", err)`; never
  discard errors; map to transport codes only at the adapter boundary. No `panic` in library/app code.
- **Logging:** `log/slog` JSON handler from `pkg/logging`; always carry `trace_id`; never log secrets or
  raw source code; levels: debug/info/warn/error used deliberately.
- **Config:** `pkg/config` loads from env (12-factor), validates, fails fast with a clear message.

## 6. Testing (gate: must pass before commit)

- **Table-driven** tests are the default. Name cases; assert behavior, not implementation.
- **Coverage floor: ≥80% for `internal/domain` and `internal/app`** of every service (CI-checked).
  Adapters covered by integration tests (bufconn for gRPC, `httptest` for HTTP, testcontainers or fakes
  for Kafka/Redis/Postgres). No network in unit tests.
- Every port has a **fake** in `internal/adapters/<x>/fake` + a shared contract test (Liskov).
- No skipped/`t.Skip` tests on `main`. Race detector on in CI (`go test -race`).
- Bugs get a failing test first, then the fix.

## 7. Go style & lint

- `gofmt`/`goimports` clean; `golangci-lint run` clean with the strict set in `.golangci.yml`
  (govet, staticcheck, errcheck, revive, gocritic, gosec, depguard, bodyclose, contextcheck, errorlint…).
- Exported symbols documented with Go doc comments. No magic numbers — named constants. Keep functions
  small; cyclomatic complexity capped by `gocyclo`.

## 8. Python (semantic service) standards

- Python 3.12+, `ruff` (lint+format) + `mypy --strict` clean; `pydantic v2` for models/validation;
  `async` FastAPI with timeouts and a rate limiter; `pytest` with the same ≥80% domain/app floor;
  same hexagonal split (domain/app/ports/adapters).

## 9. APIs, versioning, commits

- gRPC/proto are the service contracts; package `membrane.<svc>.v1`; **`buf lint` + `buf breaking`**
  pass in CI; breaking changes require a new version package, never an in-place break.
- Commit messages: short, imperative, English, ≤70 chars, conventional-ish scope optional
  (`feat(ingestion): …`, `chore: …`). One logical change per commit. No AI trailers (CLAUDE.md).
- Public HTTP APIs versioned under `/v1/…`.

## 10. Definition of Done (per roadmap item)

A roadmap item is **Done** only when: code matches this standard · unit+integration tests added and
green · coverage floor met · `task lint test build` clean · docs touched if behavior/commands changed ·
`docs/HANDOVER.md` updated · committed (push waits for "pushla").
