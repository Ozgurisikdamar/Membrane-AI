# Membrane AI — Dashboard (`web/dashboard`)

A real Membrane AI product dashboard unifying **Code Governance** and **Agent
Assurance** (evaluations, traces, simulations, baseline/candidate comparisons,
security findings, release decisions). Next.js 15 (App Router, TypeScript,
Tailwind). Part of the AgentTwin integration workstream
(`docs/agenttwin-integration/`).

**Honesty contract:** every panel is badged `LIVE` / `DEMO` / `N/A`. Demo data
is deterministic synthetic, derived from the real Phase-0 reproduced runs
(deterministic-fake planner, seeds shown) and never presented as live. The
combined release gate is shown as the *recommended deterministic policy*, not a
live enforcement engine (research report §5.3).

## Clean-clone setup

```bash
git clone https://github.com/Ozgurisikdamar/Membrane-AI
cd Membrane-AI/web/dashboard
npm ci              # Node 22 (see .nvmrc)
npm run build
npm start           # http://localhost:3000  (Demo mode by default)
```

Or from the repo root with go-task: `task dashboard:install && task dashboard:ci`.

## Scripts / local gate

| Command | What |
|---|---|
| `npm run dev` | dev server |
| `npm run build` | production build (12 routes) |
| `npm run typecheck` | `tsc --noEmit` |
| `npm run test` | Vitest unit/component (33) |
| `npm run e2e` | Playwright smoke (7) — set `PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH` to a Chromium binary |
| `task dashboard:ci` | typecheck + test + build |

## Data modes

- **Demo** (default): local deterministic fixtures. No backend, safe to deploy statically.
- **Live**: the app's server-side BFF (`/api/bff/*`) calls a configured backend.
  The browser never holds a credential. Set **server-side only** (never `NEXT_PUBLIC_*`):

  ```bash
  AGENTTWIN_BASE_URL=https://…         # allowlisted AgentTwin base
  AGENTTWIN_API_KEY=…                  # secret, header-injected by the BFF
  AGENTTWIN_ALLOWED_HOSTS=host[,host]  # SSRF allowlist
  AGENTTWIN_ALLOW_LOOPBACK=1           # local/dev only
  AGENTTWIN_UI_URL=https://…           # trace deep links
  ```

  Local live demo against the AgentTwin compose stack:

  ```bash
  AGENTTWIN_BASE_URL=http://localhost:8080 \
  AGENTTWIN_API_KEY=<demo key> \
  AGENTTWIN_ALLOW_LOOPBACK=1 AGENTTWIN_ALLOWED_HOSTS=localhost \
  npm start
  ```

  Without a reachable backend, Live views render an honest "not connected" state.

## Deploy

Netlify via `@netlify/plugin-nextjs` — see `NETLIFY.md` (Mode A demo is
deploy-ready; Mode B connected needs a verified backend). Static demo readiness
is **not** connected production readiness.

## Architecture

- `src/lib/datasource/` — the `DataSource` boundary; `DemoDataSource` (fixtures) +
  `LiveDataSource` (BFF). Every value is a `Sourced<T>` carrying provenance.
- `src/lib/server/` — server-only BFF client (`agenttwin.ts`), SSRF `allowlist.ts`,
  error→HTTP `respond.ts`. Credentials live here, never in the browser.
- `src/lib/gate.ts` — TS mirror of `pkg/assurance` (status mapping + fail-safe gate).
- `src/app/api/bff/*` — BFF route handlers (server functions).
- `src/app/*` — the nine product sections.
```
