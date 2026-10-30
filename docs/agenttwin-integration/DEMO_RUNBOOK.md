# Demo runbook — Membrane AI × AgentTwin Agent Assurance

A cold-start engineer can start the stack, run the combined code→agent→release
demo, understand the gate, inspect the evidence, and reproduce the failure — all
from this file. Every command and output below was executed for real (not
invented). The deterministic planner is labelled **deterministic-fake**; it is
never a real LLM. Synthetic data only — no real payment/refund/customer side
effects.

## 0. Prerequisites (this cloud container)

- **Go 1.25 full SDK** at `/opt/go1.25.0` with `GOTOOLCHAIN=local` (the
  auto-downloaded toolchain lacks `covdata`, which breaks `go test -cover`).
- **Docker** with a `mirror.gcr.io` registry mirror (`/etc/docker/daemon.json`)
  to dodge Docker Hub anonymous-pull 429s. `dockerd` must be running.
- **Node 22** (for the dashboard). Playwright uses `/opt/pw-browsers/chromium`.
- `task`, `buf`, `golangci-lint` on `PATH` (`$(go env GOPATH)/bin`).
- AgentTwin is **read-only** — never modify it.

## 1. Membrane quality gates (always green before a demo)

```bash
cd Membrane-AI
export GOROOT=/opt/go1.25.0 GOTOOLCHAIN=local
task build && task test && task lint            # Go: 10 modules, pkg/assurance ~90% cov
task lint:py && task test:py                    # Python semantic (45 pytest)
task build:cli                                  # → bin/membrane
cd web/dashboard && npm ci && task dashboard:ci # dashboard: typecheck+test(33)+build
```

## 2. Bring up the AgentTwin stack (Agent Assurance backend)

```bash
cd AgentTwin
make env
# Subset without the web UI / observability trio (enough for the eval demo):
docker compose -f docker-compose.yml -f <noweb-override> up -d \
  postgres rabbitmq control-plane trace-service simulation-service simulation-worker \
  evaluation-service evaluation-worker demo-agent demo-tools
# Seed the regressed and fixed candidates (deterministic, seed 42):
docker compose run --rm seed seed --count 5 --seed 42 --simulate 1.2.4,1.3.0 --evaluate 1.2.4:1.3.0
docker compose run --rm seed seed --count 3 --seed 42 --simulate 1.2.4,1.3.1 --evaluate 1.2.4:1.3.1
```

The seed prints the evaluation run IDs. Reproduced counts (seed 42):
**1.2.4 = 9/9**; **1.3.0 = 6/9 (2 critical: refund-timeout-after-mutation,
refund-tool-success-lie; 1 regressed: refund-happy-path)**; **1.3.1 = 9/9 fixed**.

> Clean reset: `make reset` (or `docker compose down -v --remove-orphans` then re-seed).

## 3. The combined release-evidence golden path (verified live)

`release-evidence` scans code (Membrane `pkg/scan`), fetches the AgentTwin paired
evaluation over HTTP, builds the fail-safe combined gate (`pkg/assurance`), and
publishes an SCM status in shadow mode. Credentials come from the environment and
are never logged.

### 3a. Regressed candidate 1.3.0 + a secret-laden change → BLOCK

```bash
./bin/membrane scan <repo-with-secrets> --json > scan.json      # e.g. 5 blocking secrets
cd pkg && go run ./cmd/release-evidence \
  --scan-json scan.json --repo owner/repo --commit <sha> \
  --agenttwin-url http://localhost:8080 --agenttwin-key "$AGENTTWIN_API_KEY" \
  --eval-run-id <1.3.0 run id> --allow-loopback --allowed-hosts localhost \
  --publish-shadow --store ./evstore --out rel-130
```

Real output:

```
persisted evidence for release rel-… to ./evstore
scm publish (shadow, fake): outcome=neutral state=pending delivered=true ·
  MEMBRANE.AI ⚠ shadow mode — would BLOCK in enforce mode (CODE_CRITICAL, AGENT_CRITICAL_SCENARIO)
wrote rel-130.json and rel-130.md · gate=BLOCK (live AgentTwin)
```

The gate BLOCKs on **two** independent rules: the code secret (`CODE_CRITICAL`)
and the agent critical scenario (`AGENT_CRITICAL_SCENARIO`). A critical failure
can never be hidden by a score.

### 3b. Fixed candidate 1.3.1 + a clean change → PASS

```bash
./bin/membrane scan <clean-repo> --json > clean.json            # 0 blocking
cd pkg && go run ./cmd/release-evidence \
  --scan-json clean.json --repo owner/repo --commit <sha> \
  --agenttwin-url http://localhost:8080 --agenttwin-key "$AGENTTWIN_API_KEY" \
  --eval-run-id <1.3.1 run id> --allow-loopback --allowed-hosts localhost \
  --publish-shadow --store ./evstore --out rel-131
```

Real output:

```
scm publish (shadow, fake): outcome=success state=success delivered=true ·
  MEMBRANE.AI ✓ release checks passed
wrote rel-131.json and rel-131.md · gate=PASS (live AgentTwin)
```

### 3c. Deterministic regeneration from the immutable store

```bash
cd pkg && go run ./cmd/release-evidence --store ./evstore --regenerate rel-<sha> --out rel-regen
diff rel-130.json rel-regen.json     # byte-identical ✓
```

Re-storing a **different** record under the same release id is refused:
`release evidence is immutable: … already stored with different content`.

### Offline variant (no backend, no credentials)

Replace the AgentTwin flags with `--fake` (deterministic 1.3.0 result) and
`--github-token ""` for a fake SCM publish. Useful for CI and the dashboard demo.

## 4. The dashboard

- **Demo mode** (default, no backend): `cd web/dashboard && npm start` → open
  `http://localhost:3000`. Deterministic fixtures, DEMO banner, every panel badged.
- **Live mode**: start with the AgentTwin stack reachable:
  ```bash
  AGENTTWIN_BASE_URL=http://localhost:8080 AGENTTWIN_API_KEY=<demo key> \
  AGENTTWIN_ALLOW_LOOPBACK=1 AGENTTWIN_ALLOWED_HOSTS=localhost npm start
  ```
  Overview / Release Assurance / Agents / Simulations / Traces then render real
  AgentTwin data; Code Governance history stays demo (no Membrane read API).
- Netlify: see `web/dashboard/NETLIFY.md` (Mode A demo deploy-ready; Mode B needs
  a reachable, authenticated backend).

## 5. Fault injection (why 1.3.0 fails)

The tool twin injects `refund_payment:timeout_after_mutation` (~8%): the payment
mutation is applied, then the call times out. The 1.3.0 manifest tells the agent
to "retry the refund right away" and to "issue refunds without waiting for policy
lookups" — so it **double-applies an irreversible refund** and **claims success
the verified state contradicts**. AgentTwin's deterministic checks catch both as
critical failures; 1.3.1 restores the policy lookup and the verify-before-confirm
step, so the same seeded suite passes. Re-run with the same `--seed 42` to
reproduce exactly.

## 6. Limitations (honest)

- `release-evidence` is a **documented command**, not yet auto-fired by the
  orchestrator Saga on every submission, nor persisted to a DB (the filesystem
  `EvidenceStore` is sufficient here; a DB adapter can swap in behind the port).
- Membrane has **no read API** for code/verdict history → the dashboard's Code
  Governance history and any release history are demo/fixture.
- The combined gate is Membrane's **recommended deterministic policy**; AgentTwin
  has no release-gate service of its own (research §5.3).
- Netlify Mode A (demo) is deploy-ready; **Mode B (connected) and an actual
  Netlify URL are the owner's step** (no Netlify account in this environment).
- Static demo readiness ≠ connected production readiness.
```
