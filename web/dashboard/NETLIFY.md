# Netlify deployment — Membrane AI dashboard

The dashboard (`web/dashboard`, Next.js 15) deploys on Netlify via the official
Next.js runtime (`@netlify/plugin-nextjs`). Config lives at the repo-root
`netlify.toml` (`base = "web/dashboard"`). **Two separate readiness levels** —
do not conflate them (`DASHBOARD_AND_NETLIFY_WORKFLOW.md` §6, §8).

## Mode A — Demo frontend (deployable now)

Deterministic synthetic fixtures (derived from the real Phase-0 runs), clearly
labelled DEMO, all interactions local/read-only. **No privileged credentials.**

- Build command: `npm run build` (in `web/dashboard`), Node 22.
- Output: the Next.js runtime adapter sets the publish output (`.next`); do not
  hard-code a publish dir.
- Env vars: **none required.** Live mode simply renders "not connected" states.

## Mode B — Connected product dashboard (NOT production-ready yet)

The Netlify frontend's server-side BFF calls a reachable Membrane/AgentTwin
backend. Requires, set in **Netlify UI → Site settings → Environment variables**
as **non-public, server-side only** (never `NEXT_PUBLIC_*`, never in `netlify.toml`):

**Membrane evidence read API** — real persisted release decisions (Release Assurance):

| Variable | Public? | Purpose |
|---|---|---|
| `MEMBRANE_EVIDENCE_URL` | server-only | https base of the evidence API (e.g. `https://<app>.fly.dev`) |
| `MEMBRANE_EVIDENCE_API_KEY` | **secret, server-only** | evidence API key (sent as `X-Membrane-Api-Key`) |
| `MEMBRANE_EVIDENCE_ALLOWED_HOSTS` | server-only | bare hostname(s) allowlist, e.g. `<app>.fly.dev` |
| `MEMBRANE_EVIDENCE_ALLOW_LOOPBACK` | server-only | `1` only for local/dev loopback |

**AgentTwin control-plane** — live Agent Assurance (eval/trace/simulation):

| Variable | Public? | Purpose |
|---|---|---|
| `AGENTTWIN_BASE_URL` | server-only | https, allowlisted AgentTwin base |
| `AGENTTWIN_API_KEY` | **secret, server-only** | AgentTwin API key (header-injected by the BFF) |
| `AGENTTWIN_ALLOWED_HOSTS` | server-only | bare hostname(s) allowlist (SSRF guard) |
| `AGENTTWIN_ALLOW_LOOPBACK` | server-only | `1` only for local/dev loopback |
| `AGENTTWIN_ORG` | server-only | optional org id → `X-AgentTwin-Org` |
| `AGENTTWIN_UI_URL` | server-only | base for trace deep links |

Each backend is independent: set the `MEMBRANE_EVIDENCE_*` group to light up
Release Assurance from real persisted evidence, the `AGENTTWIN_*` group for live
eval/trace/simulation — either, both, or neither (demo-only). **Allowlist rule**
(`src/lib/server/allowlist.ts`): a non-loopback destination **must be https** and
its **bare hostname** must appear in the matching `*_ALLOWED_HOSTS` (no scheme, no
path, no wildcards; `host:port` only when the URL carries a non-default port).

The end-to-end deploy runbook (evidence API on Fly.io + this dashboard on
Netlify) is **[DEPLOYMENT.md](../../docs/agenttwin-integration/DEPLOYMENT.md)**.

Mode B is **not** deploy-ready until auth, tenant isolation, CORS/TLS, backend
availability, secret handling and end-to-end tests are separately verified. A
successful Mode A deploy is **not** proof of Mode B production readiness.

## Deploy steps (owner)

1. In Netlify: **Add new site → Import from Git** → this repo; set **Base
   directory** `web/dashboard` (or rely on the repo-root `netlify.toml`).
2. Netlify auto-detects Next.js and installs `@netlify/plugin-nextjs`.
3. For a demo (Mode A): no env vars. Deploy → open the `*.netlify.app` URL.
4. For Mode B: add the server-side vars above, then deploy; verify the live
   views against the backend before claiming connected readiness.

`netlify deploy` from a local machine also works once the site is linked
(`netlify link`). This repository has **not** been deployed to a Netlify URL
from this environment (no Netlify account/credentials here) — the config and
production build are verified; the deploy is the owner's step.

## Verified in this environment

- `npm run build` (production) succeeds; `npm run test` (33) + `npm run e2e` (7) pass.
- **No secret leak:** the client bundle contains only the env var *name*
  `AGENTTWIN_API_KEY` as documentation text — no secret value (`atk_*`,
  `sk_live`, `AKIA*`) and not the real demo key; the server-only client module
  (`src/lib/server/*`) is excluded from the browser bundle.
- Security headers set via `next.config.mjs` + `netlify.toml`; hashed static
  assets marked immutable, HTML revalidated.

## Selected deployment mode

**Mode A (demo)** is the selected, evidenced target. Mode B is documented but
gated on a reachable, authenticated backend and separate E2E verification.
