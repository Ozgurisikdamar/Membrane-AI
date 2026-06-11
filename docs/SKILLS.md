# SKILLS — Engineering Playbook

Everything an agent needs to *operate* this repo: toolchain, commands, local dev, codegen, and the
checklist for adding a service. Pair with `docs/ENGINEERING-STANDARDS.md` (the "how well") and
`docs/ARCHITECTURE.md` (the "where").

## 1. Toolchain (versions & install)

| Tool | Min version | Install |
| --- | --- | --- |
| Go | 1.25 | already installed; else `winget install GoLang.Go` |
| go-task | v3 | `go install github.com/go-task/task/v3/cmd/task@latest` |
| buf | latest | `go install github.com/bufbuild/buf/cmd/buf@latest` |
| protoc-gen-go | latest | `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest` |
| protoc-gen-go-grpc | latest | `go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest` |
| golangci-lint | v2 | `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest` |
| Docker (+ compose) | any | already installed (dev stack only) |
| Python | 3.12+ | for the semantic service; 3.11 available locally for scripts |

All `go install` binaries land in `%USERPROFILE%\go\bin` (already on PATH). If a command is "not found",
re-run its install line and re-open the shell.

> **buf gotcha:** `go install …/buf` currently fails on this setup (an upstream build bug needing a git
> tree object). Install the **prebuilt binary** instead:
> `gh api repos/bufbuild/buf/releases/latest -q .tag_name` → download
> `https://github.com/bufbuild/buf/releases/download/<tag>/buf-Windows-x86_64.exe` to `%USERPROFILE%\go\bin\buf.exe`.

> **`GOWORK` gotcha:** this machine has a stray user-level `GOWORK` pointing at an unrelated project, which
> hijacks `go` resolution. Always run Go through `task` (it sets `GOWORK=off` inline) or prefix bare commands
> with `GOWORK=off`. The in-repo `replace` directives wire the modules together without the workspace.

> **`-race`** needs cgo + a C compiler; not available locally on Windows. Run race in CI (Linux) only.

## 2. `task` commands (the daily interface)

| Command | Does |
| --- | --- |
| `task` / `task --list` | List all tasks. |
| `task lint` | `golangci-lint run ./...` across every module + `buf lint`. |
| `task test` | `go test -race -cover ./...` per module; enforces the coverage floor. |
| `task build` | `go build ./...` per module (and static client builds). |
| `task proto` | `buf lint` + `buf generate` (regenerate gRPC stubs into `proto/gen`). |
| `task dev-up` | `docker compose -f deploy/compose/docker-compose.dev.yml up -d` (Redpanda, Redis, pgvector). |
| `task dev-down` | Tear the dev stack down. |
| `task run:<svc>` | Run a service locally (e.g. `task run:ingestion`, `task run:semantic`). |
| `task setup:py` | One-time: create the semantic venv + install dev deps. |
| `task lint:py` / `task test:py` | ruff + mypy --strict / pytest on the semantic service. |
| `task ci` | What CI runs: proto + lint + test + build + Python gates. Run this before "pushla". |

> Quality gate before any commit: `task lint && task test` must be green. Before pushing: `task ci`.

## 3. Local development loop

1. `task dev-up` — start infra (Redpanda :9092, Redis :6379, Postgres+pgvector :5432).
2. `task run:ingestion` — starts gRPC `:9001`, HTTP webhook `:8001`, health `:8101`.
3. Health check: `curl http://localhost:8101/readyz`.
4. Iterate; `task test` often. `task dev-down` when done.

Config is via env vars `MEMBRANE_<SVC>_…` (see each service's `internal/config`). Local defaults live in
`deploy/compose/.env.example` (copy to `.env`, which is gitignored).

## 4. Codegen (protobuf / gRPC)

- Contracts live in `proto/membrane/<svc>/v1/*.proto`, package `membrane.<svc>.v1`.
- `task proto` runs `buf lint` then `buf generate` (config `proto/buf.gen.yaml`) → generated Go into
  `proto/gen/...`, imported by services. **Commit generated stubs** (reviewable, no codegen needed to build).
- Never hand-edit generated files. Change the `.proto`, re-run `task proto`. `buf breaking` guards compat.

## 5. Add a new service (checklist)

1. `services/<svc>/` with its own `go.mod` (`github.com/Ozgurisikdamar/Membrane-AI/services/<svc>`);
   add it to `go.work`.
2. Hexagonal dirs: `cmd/<svc>/main.go`, `internal/{domain,app,ports,adapters,config}`.
3. Contract (if it has an API): `proto/membrane/<svc>/v1/…`; `task proto`.
4. Wire `pkg/{config,logging,errs,health}`; expose health on `:81xx`; pick ports per ARCHITECTURE §6.
5. Domain + use-cases first (TDD), then adapters; add fakes + contract tests.
6. `task lint test build` green (≥80% domain/app). Add a `run:<svc>` task.
7. Update `docs/ARCHITECTURE.md` (component table) + `docs/ROADMAP.md` + `docs/HANDOVER.md`.

## 6. Release / packaging (later phases)

- Clients: `CGO_ENABLED=0` static builds, matrix `windows|darwin|linux × amd64|arm64`; package via
  winget/Scoop/MSI, Homebrew/pkg, deb/rpm/apk/tarball, `curl|sh`.
- Services: multi-arch distroless images (`linux/amd64,arm64`) → GHCR; Helm chart in `deploy/helm`.
- Versioning: SemVer tags `vX.Y.Z`; CI builds artifacts on tag. (Details added when P3 begins.)

## 7. GitHub / board

- Mirror roadmap status to Project #5 (IDs in `CLAUDE.md`) when `gh` is authed; `docs/ROADMAP.md` is truth.
- Push only on "pushla". `gh` may need re-auth with a classic `repo`+`project` PAT (see CLAUDE.md).
