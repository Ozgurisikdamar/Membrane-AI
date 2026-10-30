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

| Variable | Public? | Purpose |
|---|---|---|
| `AGENTTWIN_BASE_URL` | server-only | https, allowlisted AgentTwin base |
| `AGENTTWIN_API_KEY` | **secret, server-only** | AgentTwin API key (header-injected by the BFF) |
| `AGENTTWIN_ALLOWED_HOSTS` | server-only | comma-separated host allowlist (SSRF guard) |
| `AGENTTWIN_ALLOW_LOOPBACK` | server-only | `1` only for local/dev loopback |
| `AGENTTWIN_UI_URL` | server-only | base for trace deep links |

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
