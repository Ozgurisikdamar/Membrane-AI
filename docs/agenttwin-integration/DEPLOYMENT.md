# Deployment — Netlify dashboard + Fly.io evidence API (end-to-end)

> **Goal:** deploy the Membrane evidence read API (`pkg/cmd/evidence-api`) on **Fly.io**
> and the product dashboard (`web/dashboard`) on **Netlify**, wired so the dashboard's
> **Release Assurance shows the real persisted combined evidence** (real code verdict +
> combined gate) in production — not demo fixtures.
>
> This is the production counterpart of the local flow verified in
> [PROGRESS.md](PROGRESS.md) §9 (loopback end-to-end: 1.3.0 BLOCK / 1.3.1 PASS).

## 0. What runs where, and why

```text
            Netlify (serverless)                         Fly.io (stateful VM + volume)
┌───────────────────────────────────────┐        ┌──────────────────────────────────────┐
│  web/dashboard (Next.js)               │        │  evidence-api (Go, distroless)         │
│  browser → same-origin BFF             │ https  │  GET /healthz        (unauth)          │
│  /api/bff/membrane-releases[/:id] ─────┼───────▶│  GET /v1/releases[/{id}]  (X-Membrane- │
│  (server-side key, SSRF allowlist)     │  key   │      Api-Key, constant-time)           │
│                                        │        │  reads /data (persistent volume)       │
│  /api/bff/* (AgentTwin, optional) ─────┼──────▶ │  … AgentTwin control-plane (separate)  │
└───────────────────────────────────────┘        └──────────────────────────────────────┘
        credential NEVER in the browser                 records WRITTEN by release-evidence
```

- **Netlify is serverless** — it cannot host a long-lived server with a writable filesystem,
  so the evidence API lives on Fly.io with a **persistent volume** (its `FileStore` is a
  directory of one JSON file per release id).
- **evidence-api only reads.** Records are produced by the `release-evidence` command
  (manual or CI) and written into the store/volume — see **Part D**.
- The dashboard's BFF holds every credential **server-side**; the browser only ever calls the
  same-origin `/api/bff/*` routes. Both backends are independent: wire the evidence API,
  AgentTwin, both, or neither (demo-only).

## 1. Prerequisites

- **flyctl** installed and authenticated: `fly auth login` (https://fly.io/docs/flyctl/install/).
- A **Netlify** account with access to import this GitHub repo.
- Local toolchain to build/seed (same as the repo): Go 1.25 (`GOWORK=off`), Docker (optional,
  for the image), Node 22 for the dashboard.
- A strong random **evidence API key**, e.g. `openssl rand -hex 32`. Keep it out of git.

---

## Part A — Deploy the evidence API to Fly.io

The Fly config is checked in at **`deploy/fly/evidence-api/fly.toml`**. It builds the image
from the shared `deploy/docker/Dockerfile.go` with `SERVICE=pkg CMD_PATH=cmd/evidence-api`
(because `pkg` has more than one `cmd`), mounts a volume at `/data`, and health-checks
`/healthz`. **All `fly` commands run from the repo root** so the Docker build context is the
repo root.

### A.1 Pick an app name and region

Edit `deploy/fly/evidence-api/fly.toml` and set a globally-unique `app` name and your
`primary_region` (e.g. `iad`, `fra`, `lhr`). Then create the app:

```bash
fly apps create <your-evidence-app> --machines
```

### A.2 Create the persistent volume

```bash
fly volumes create evidence_data --size 1 --region <your-region> \
  --config deploy/fly/evidence-api/fly.toml --yes
```

(The `source` in `[[mounts]]` is `evidence_data`; keep the names matching.)

### A.3 Set the API key as a secret (never in the TOML)

```bash
fly secrets set MEMBRANE_EVIDENCE_API_KEY="$(openssl rand -hex 32)" \
  --config deploy/fly/evidence-api/fly.toml
```

Print it once to copy into Netlify later (or re-generate and set it in both places):

```bash
# store the value you generated somewhere safe; you'll paste it into Netlify in Part B.
```

### A.4 Deploy

```bash
fly deploy --config deploy/fly/evidence-api/fly.toml .
```

The trailing `.` sets the Docker build context to the repo root. First build compiles the Go
binary on `golang:1.25` and ships a distroless image.

### A.5 Seed the volume with evidence records

The volume starts empty, so `/v1/releases` returns `[]` until you put records in it. Produce
records locally with `release-evidence`, then upload them over Fly's SFTP (which works on the
distroless image — note `fly ssh console` does **not**, there is no shell in the image).

Produce records locally into `pkg/evstore/` with the `release-evidence` command. The exact
golden-path invocations that yield `rel-aedc1ed4` (BLOCK) and `rel-b1f1c3d1e5a7b9` (PASS) are in
**[DEMO_RUNBOOK.md](DEMO_RUNBOOK.md) §3** — run those, or use `task release-evidence` as a start.
The generic shape (real flags only):

```bash
cd pkg
# offline, deterministic-fake (labelled as such); --commit must be hex; --github-token "" keeps it offline
GOWORK=off go run ./cmd/release-evidence \
  --scan-json <path-to-membrane-scan.json> \
  --repo Ozgurisikdamar/demo-co-support-agent \
  --commit <hex-commit> \
  --store ./evstore --fake --github-token ""
# a real record instead of --fake: pass --agenttwin-url --agenttwin-key --eval-run-id
```

This writes one `pkg/evstore/<release-id>.json` per release (default id `rel-<commit>`).

> **Make the records readable first.** The image runs as **nonroot (uid 65532)**, but
> `release-evidence` writes records mode `0600` (owner-only). Fix the mode **locally before
> uploading** (SFTP uploads the file with its local mode), or the server gets a read error and
> `/v1/releases` returns 500. Validated: with `0600` the nonroot server cannot read; `0644` serves.

```bash
chmod 644 pkg/evstore/*.json
```

Upload the records into the volume over SFTP (SFTP works on the distroless image; `fly ssh
console` does **not** — there is no shell to exec):

```bash
fly ssh sftp shell --config deploy/fly/evidence-api/fly.toml
# at the sftp prompt, for each record:
#   put pkg/evstore/rel-aedc1ed4.json /data/rel-aedc1ed4.json
#   put pkg/evstore/rel-b1f1c3d1e5a7b9.json /data/rel-b1f1c3d1e5a7b9.json
#   quit
```

If A.6's `/v1/releases` returns 500 instead of the records, the volume files are not readable
by uid 65532 — re-upload after `chmod 644` locally (the Fly volume dir itself is `0755` by default).

### A.6 Verify the API is live

```bash
APP=<your-evidence-app>; KEY=<the-key-from-A.3>
curl -s https://$APP.fly.dev/healthz                       # -> ok
curl -s -o /dev/null -w '%{http_code}\n' https://$APP.fly.dev/v1/releases   # -> 401 (no key)
curl -s -H "X-Membrane-Api-Key: $KEY" https://$APP.fly.dev/v1/releases | jq '.[].release_id'
curl -s -H "X-Membrane-Api-Key: $KEY" https://$APP.fly.dev/v1/releases/rel-aedc1ed4 \
  | jq '{gate_decision, code_verdict, triggered_rules}'     # -> BLOCK / BLOCK / [CODE_CRITICAL, AGENT_CRITICAL_SCENARIO]
```

Record the hostname `<your-evidence-app>.fly.dev` — it goes into Netlify next.

---

## Part B — Deploy the dashboard to Netlify

The Netlify config is checked in at the **repo-root `netlify.toml`** (`base = "web/dashboard"`,
Node 22, `@netlify/plugin-nextjs`). Deploy **Mode A (demo) first**, then add env to make it live.

### B.1 Import and deploy Mode A (demo)

1. Netlify → **Add new site → Import an existing project** → pick this GitHub repo.
   (Direct: https://app.netlify.com/start)
2. Build settings auto-fill from `netlify.toml`: **Base directory** `web/dashboard`,
   **Build command** `npm run build`, plugin `@netlify/plugin-nextjs`. Leave publish dir to the plugin.
3. Deploy → open the `*.netlify.app` URL. It works out of the box in **demo** mode (no env).

### B.2 Add Mode B env (server-side, non-public)

In **Site settings → Environment variables** add, as **non-public / server-side only**
(never `NEXT_PUBLIC_*`, never in `netlify.toml`):

| Variable | Value |
|---|---|
| `MEMBRANE_EVIDENCE_URL` | `https://<your-evidence-app>.fly.dev` |
| `MEMBRANE_EVIDENCE_API_KEY` | the key from A.3 (same value) |
| `MEMBRANE_EVIDENCE_ALLOWED_HOSTS` | `<your-evidence-app>.fly.dev` (bare hostname) |

> **Allowlist rule** (`src/lib/server/allowlist.ts`): a non-loopback destination **must be
> https** and its **bare hostname** must be in `*_ALLOWED_HOSTS` — no scheme, no path, no
> wildcards. Do **not** set `MEMBRANE_EVIDENCE_ALLOW_LOOPBACK` in production.

(Optional — live Agent Assurance too: also set `AGENTTWIN_BASE_URL`, `AGENTTWIN_API_KEY`,
`AGENTTWIN_ALLOWED_HOSTS`, optionally `AGENTTWIN_ORG` and `AGENTTWIN_UI_URL`. Same https +
bare-hostname rule. See [NETLIFY.md](../../web/dashboard/NETLIFY.md).)

### B.3 Redeploy

Trigger a redeploy (env changes need a new build): **Deploys → Trigger deploy → Deploy site**.

---

## Part C — Wire and verify end-to-end

1. Open the Netlify site. In the top-right header, click **Live** (the toggle persists per
   browser in `localStorage`; it is convenience only, never an authorization boundary).
2. Go to **Release Assurance**. With the evidence API reachable you should see the real
   persisted rows, newest first — e.g. **1.3.1 PASS** (code PASS) and **1.3.0 BLOCK**
   (code BLOCK, 2 new critical), each badged **live**.
3. Open a release (e.g. `rel-aedc1ed4`): **Combined gate BLOCK**, **Code verdict BLOCK
   (5 blocking)**, rules `CODE_CRITICAL` + `AGENT_CRITICAL_SCENARIO`, the 9 real scenarios.

### Security checks (do these before calling it done)

- **No secret in the browser bundle:** open DevTools → Sources, or
  `curl -s https://<site>.netlify.app/_next/static/... | grep -i membrane_evidence_api_key`
  → must find **nothing**. The key is only read in `src/lib/server/*` (server-side).
- **https only / allowlist:** an `http://` or non-allowlisted `MEMBRANE_EVIDENCE_URL` is
  rejected by `checkDestination` → the panel shows an honest **unavailable** state, never a crash.
- **Auth:** a wrong/empty key → evidence API returns 401 → dashboard shows **unavailable**,
  not fabricated data.
- **Backend down:** stop the Fly app → Release Assurance shows "not connected", not stale numbers.

---

## Part D — Keeping the store populated (ongoing data flow)

**evidence-api only serves; it never writes.** Real records come from the `release-evidence`
command, which runs the code scan + AgentTwin eval, applies the fail-safe gate, and writes an
immutable record with `--store`. To keep the Fly volume current, pick one:

- **CI job (recommended for real releases):** on a release/tag, run
  `release-evidence --scan-json … --repo … --commit <sha> --agenttwin-url … --eval-run-id … --store ./out`,
  then `chmod 644 ./out/*.json` and `fly ssh sftp shell` → `put ./out/<release>.json /data/<release>.json`
  (the `chmod` is needed so the nonroot server can read them — see A.5). Records are **immutable**
  — a differing re-put under the same release id is a conflict, so re-running is safe.
- **Manual/demo:** repeat **A.5** whenever you want to add a release.
- **Co-locate (later):** run `release-evidence` on a box that mounts the same volume and writes
  directly to `/data`. (Not needed for the first deploy.)

> Honesty: the dashboard shows exactly what is in the store. If no release-evidence run has
> happened for a change, there is no record — the UI shows that, it does not invent one.

---

## Limitations & rollback

- **Single-node FileStore.** The evidence API serves a directory of JSON files on one volume.
  A DB-backed, multi-tenant store with per-key org scoping is a later step; the `?project=`
  filter is a convenience, **not** an authorization boundary (production tenant scope must
  derive from the authenticated key).
- **Helm chart does not yet support evidence-api.** `deploy/helm/membrane` probes `/livez`+`/readyz`
  (evidence-api serves only `/healthz`) and has no volume support. Kubernetes deployment needs
  chart changes first — out of scope for this Fly-based guide.
- **Mode A ≠ connected readiness.** A green demo deploy is not proof of a secure connected
  deploy; verify the Part C security checks.
- **Rollback.** Fly: `fly releases --config deploy/fly/evidence-api/fly.toml` then
  `fly deploy --image <previous>` or `fly releases rollback`. Netlify: **Deploys →** pick a
  previous deploy **→ Publish deploy**. To fully disable the live path, remove the
  `MEMBRANE_EVIDENCE_*` env on Netlify and redeploy — the dashboard falls back to demo/unavailable.

## Verified locally (what was tested before this guide shipped)

- `task build:evidence-api` builds `bin/evidence-api`; the image builds via
  `docker build -f deploy/docker/Dockerfile.go --build-arg SERVICE=pkg --build-arg CMD_PATH=cmd/evidence-api`
  and `docker run` serves `/healthz` + authed `/v1/releases` against a seeded store.
- The container → dashboard loopback flow renders real 1.3.0 BLOCK / 1.3.1 PASS (PROGRESS.md §9).
- The actual `fly deploy` / Netlify site creation are owner steps (no Fly/Netlify credentials in
  the build environment); every command above is otherwise validated locally.
