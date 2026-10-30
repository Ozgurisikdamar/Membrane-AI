# Dashboard + Netlify Demo Delivery Work Order

**Status:** Authorized scope extension; implementation not started  
**Owner:** Membrane AI repository owner  
**Primary repo:** https://github.com/Ozgurisikdamar/Membrane-AI  
**Execution branch:** `feature/agenttwin-assurance`  
**AgentTwin reference (read-only):** https://github.com/Ozgurisikdamar/AgentTwin  
**Parent instructions:** [Claude integration work order](CLAUDE.md)  
**Living status:** [PROGRESS.md](PROGRESS.md)  
**Research:** [Combined research report](../research/MembraneAI_AgentTwin_Birlesik_Platform_Arastirma_ve_Demo_Tasarimi.md)

---

## 1. Mission

Extend the existing Agent Assurance integration workstream to deliver a **real, demonstrable Membrane AI web dashboard** and a Netlify-ready frontend deployment package.

The dashboard is a first-class Membrane AI product surface, not a replacement for the backend and not a static marketing landing page. It must bring together Code Governance and Agent Assurance while preserving the service/data ownership boundaries documented in the parent work order.

**Do not start by drawing charts against invented numbers.** First inspect the current repository, finish the baseline revalidation, and establish which backend endpoints and evidence actually exist. Then build the smallest functional vertical slice that can be honestly demonstrated.

The dashboard must make it immediately clear which data is:
- live from Membrane or AgentTwin;
- locally generated deterministic demo data;
- unavailable / not configured / stale.

Never label mock data as live, never fabricate a passing evaluation, and never make a simulated action look like a real refund, payment, deployment, or production side effect.

## 2. Product outcome and navigation

Product identity: **MEMBRANE.AI — AI Governance & Agent Assurance**.

Required navigation (labels may be refined after checking existing design conventions):

1. **Overview** — system status, latest code scans, latest agent evaluations, release decisions, recent activity.
2. **Code Governance** — scan/submission history, severity breakdown, policy findings, links to existing reports.
3. **Agent Twin Lab** — registered agents, versions, scenarios, twins and configuration references supported by actual AgentTwin capabilities.
4. **Simulation Center** — baseline/candidate runs, seeded scenarios, replay links, state diffs and outcome verification.
5. **Trace Explorer** — trace/run list and detail deep links; avoid copying full sensitive span payloads into Membrane.
6. **Release Assurance** — combined PASS/WARN/BLOCK decisions, triggered rules, evidence links, pinned versions and incomplete/error states.
7. **Security Center** — MCP/tool policy events and security findings that existing APIs can safely expose.
8. **Reports** — Markdown/JSON evidence report access and export where implemented.
9. **Settings / Integrations** — backend connectivity, project mapping and environment status. Never display secret values.

Do not render a navigation item as functional if it is only a placeholder. Either implement a useful empty/configuration state or mark the capability clearly as planned/not connected.

## 3. Required screens and information hierarchy

### 3.1 Overview

Show, only when backed by API or clearly labelled demo fixtures:
- Membrane backend health and AgentTwin connectivity separately;
- latest scan/evaluation timestamps and data freshness;
- code findings by severity;
- evaluation totals and scenario outcomes;
- release decisions by PASS/WARN/BLOCK/INCOMPLETE/ERROR;
- recent audit/activity items;
- explicit empty, loading, error, stale and not-configured states.

Avoid vanity metrics. Every aggregate must define its denominator, time window and source. No fabricated percentage or risk probability.

### 3.2 Agent Assurance

A user must be able to:
- inspect registered agent/version references available through supported APIs;
- inspect a baseline/candidate evaluation pair;
- open a scenario result and its trace/trajectory/state-diff evidence;
- distinguish deterministic assertions from semantic judge results;
- see the exact reason for a BLOCK/WARN/INCOMPLETE result.

Do not imply that Membrane owns AgentTwin's twin/scenario database. Use AgentTwin APIs and deep links.

### 3.3 Release Assurance

A release detail must show:
- code commit/submission and Membrane code verdict;
- agent ID and pinned baseline/candidate versions;
- evaluation/run IDs, scenario suite and seed where supplied;
- deterministic gate rules triggered;
- evidence links and timestamps;
- final status and the source of each fact.

A critical deterministic failure must remain visible regardless of any aggregate score.

### 3.4 Demo mode

Provide a clearly labelled **Demo workspace** using synthetic data and deterministic fixtures when real services are unavailable. Reuse the existing AgentTwin Demo Co support/refund story and its documented versions only after Phase 0 has reproduced them.

Demo mode requirements:
- persistent banner: "DEMO DATA — synthetic / not production";
- fixture source and fixture version visible;
- deterministic results across reloads;
- no calls to real payment, refund, order, email or customer systems;
- no hidden mutation of live AgentTwin or Membrane data;
- controls that trigger scenarios must be local fixture simulation unless an authenticated, safe sandbox API has been verified;
- show a regressed candidate and a fixed candidate only when the source evidence supports those claims.

A "Run demo" button must perform the advertised deterministic local/sandbox action, not be decorative. If no safe runnable backend exists, show a documented read-only walkthrough instead.

## 4. Frontend implementation rules

1. Inspect the entire repository before choosing framework/build tool. Reuse an existing frontend if one exists and is suitable. Do not assume a package manager, framework or build command.
2. The existing `docs/landing/index.html` is a marketing landing page, not a dashboard. Do not simply relabel it as the dashboard.
3. If no suitable dashboard app exists, create a separately scoped frontend (preferred path: `web/dashboard/`, subject to repository conventions) with a documented package manager, pinned lockfile and reproducible build.
4. Use TypeScript and accessible semantic components if consistent with the repository; keep dependencies minimal and maintained.
5. Responsive layout: desktop first for dense operational views, but usable at tablet and mobile widths.
6. Include loading, empty, error, stale, disconnected and permission-denied states for every data-backed view.
7. Add accessible keyboard navigation, visible focus, semantic headings, sufficient contrast and chart/table alternatives.
8. Charts must show units, time range, denominator and source. Provide tabular equivalents for critical findings.
9. Avoid fake live animations, random counters and auto-changing metrics.
10. Add component/unit tests and browser-level smoke/E2E tests for primary navigation, overview, release detail, demo mode and failure states.
11. Add a screenshot capture/runbook only after a real local build is available. Do not invent screenshots or claim browser tests were run if not executed.

## 5. Backend integration and security

The browser must not call AgentTwin directly with a privileged service credential. The preferred topology is:

```
Browser (Netlify static frontend)
        |
        | HTTPS, user/session-scoped auth
        v
Membrane API / BFF / existing authenticated API boundary
        |
        | service-to-service credential, allowlisted destination
        v
AgentTwin API
```

- Reuse the existing Membrane auth/session/API patterns after verifying them.
- If a safe user authentication flow does not exist, do not invent insecure localStorage bearer-token handling. Provide a clearly scoped demo-only mode and record production auth as a deployment blocker.
- No secrets in `NEXT_PUBLIC_*`, `VITE_*`, frontend bundles, Netlify build logs, query strings or client-side storage.
- Netlify environment variables may contain only values safe for the browser if prefixed for public exposure. Privileged secrets belong in backend secret management, never Netlify frontend build variables.
- Enforce tenant/project authorization on the backend for every data request. Hiding a UI element is not authorization.
- Use explicit CORS allowlists, HTTPS, CSRF protection where cookie sessions are used, CSP/security headers and safe error messages.
- Never accept an arbitrary AgentTwin base URL from the browser. Backend destination must be configured and allowlisted.
- Redact prompts, customer data, tool inputs/results, tokens and secrets from frontend logs and analytics.
- Use synthetic demo data only.
- Do not expose raw traces by default; show summaries and links. Apply retention/redaction rules from the source service.

If a required backend-for-frontend/API endpoint does not exist, document the contract and implement the smallest safe backend route in Membrane only after inspecting the current architecture. Do not bypass backend authorization by calling AgentTwin from the browser.

## 6. Netlify deployment package

Prepare the frontend for Netlify only after its local production build and tests pass.

Required:
- correct Netlify base directory and build command for the actual frontend;
- publish directory generated by the verified build;
- SPA fallback/redirect only if the chosen router requires it;
- security headers and cache policy appropriate to immutable assets vs HTML;
- environment variable inventory with public/non-public classification;
- documented API base URL and CORS origin configuration;
- preview deploy instructions and production deploy checklist;
- no backend services or databases assumed to run on Netlify.

Do not create a `netlify.toml` with guessed paths. Verify the exact framework output directory from the real build. If the repository's existing `deploy/` conventions are appropriate, document how Netlify should point to the frontend subdirectory.

### Deployment modes

**Mode A — Demo frontend:** Netlify hosts a static dashboard with explicitly labelled deterministic synthetic fixtures. This can be deployed independently, provided no privileged credentials are required and all interactive actions are local/read-only.

**Mode B — Connected product dashboard:** Netlify frontend calls a deployed Membrane API over HTTPS; Membrane backend calls AgentTwin. This mode is not deploy-ready until auth, tenant isolation, CORS, TLS, API availability, secrets and end-to-end tests are verified.

State clearly which mode has actually been validated. A successful static deploy is not proof that the integrated product backend is production-ready.

## 7. Delivery phases and acceptance gates

These phases extend, not replace, the parent integration plan.

### D0 — Baseline and scope verification
- inspect current branch and all existing frontend/landing/deploy assets;
- complete or continue integration Phase 0; record exact Membrane and AgentTwin SHAs;
- inventory current APIs, auth, health endpoints, data schemas and demo fixture availability;
- choose dashboard framework/build path based on repository evidence;
- document the live/demo/unavailable data contract.

**Exit:** verified UI/backend capability matrix and architecture decision. No dashboard implementation assumptions remain.

### D1 — Dashboard foundation
- app shell, navigation, design tokens, responsive layout;
- typed API client and source/status metadata model;
- loading/error/empty/stale/disconnected states;
- deterministic demo fixture provider, visibly separated from live provider;
- unit/component tests.

**Exit:** production build succeeds and core shell tests pass.

### D2 — Functional Agent Assurance dashboard
- overview with real or explicitly synthetic evidence;
- agent/version and baseline/candidate evaluation views;
- scenario/trace/state-diff evidence views or verified AgentTwin deep links;
- release detail with deterministic gate reasons;
- demo workspace using verified refund-agent fixtures;
- browser tests for key user journeys.

**Exit:** demo journey is repeatable and every displayed datum has a known source.

### D3 — Code Governance and unified product view
- connect existing Membrane scan/report/security data;
- unify code and agent evidence under release/correlation identity;
- show disconnected and partial integration states honestly;
- test tenant/project isolation and authorization at the backend boundary.

**Exit:** one coherent product journey from code finding to agent evidence to release decision.

### D4 — Netlify demo deployment readiness
- verified production build, lint, tests and E2E smoke;
- exact Netlify build/publish settings and config;
- environment variable classification, CORS and headers;
- preview deploy verification and rollback notes;
- deployment mode A or B explicitly selected and evidenced.

**Exit:** Netlify demo URL can be opened and its advertised features work. Connected mode additionally requires backend end-to-end verification.

### D5 — Final product/demo acceptance
- clean-clone setup and runbook;
- desktop/mobile screenshots from actual build;
- no console errors or broken navigation in supported browsers;
- no secrets or sensitive payloads in bundle/repository;
- accessibility and responsive checks;
- all dashboard data provenance labels verified;
- PROGRESS.md, HANDOVER.md and relevant roadmap updated;
- milestone commit pushed to authorized branch.

**Exit:** owner can present the dashboard and reproduce the demo from documented instructions.

## 8. Definition of done

The dashboard workstream is complete only when:
- the UI is a real built application, not just a static design mockup;
- navigation and every advertised primary interaction work;
- Demo mode is deterministic, synthetic and visibly labelled;
- live data is shown only when a verified API supplies it;
- errors, missing integrations and stale data are visible;
- the combined Agent Assurance journey is backed by actual evidence;
- frontend build, tests and E2E smoke pass and exact results are recorded;
- Netlify config uses verified paths and a successful production build;
- no privileged secret is shipped to the browser;
- the selected deployment mode is documented;
- PROGRESS.md includes commit SHAs, commands, outcomes and remaining limitations.

**Important:** Netlify static demo readiness and connected production readiness are separate statuses. Do not mark the whole platform production-ready because a static frontend deployed successfully.

## 9. Mandatory reporting additions

For every D-phase, update `PROGRESS.md` with:
- frontend framework/version and package manager;
- build command and exact output directory;
- files/screens delivered;
- data source per screen (live/demo/unavailable);
- API endpoints actually verified (method/path/auth/response schema);
- tests, test counts, browser/device coverage and failures;
- accessibility/responsive checks;
- Netlify config path and deployment mode;
- environment variable names only (never values);
- preview URL only if actually deployed and verified;
- known limitations, security blockers and next action.

Keep the original integration phase statuses intact. Add a separate Dashboard & Deployment status table so dashboard progress is not confused with AgentTwin integration completion.

## 10. Immediate next action for Claude

1. Read root `CLAUDE.md`, parent integration `CLAUDE.md`, this file, `IMPLEMENTATION_PLAN.md`, `PROGRESS.md`, research report and Membrane engineering docs.
2. Inspect current branch/working tree and remote SHAs; do not overwrite user work.
3. Complete D0 inventory and report evidence before coding.
4. Implement D1–D5 in order, while continuing the parent AgentTwin integration phases where dependencies allow.
5. Commit and push each completed milestone to `feature/agenttwin-assurance`; update `PROGRESS.md` in the same milestone commit.
6. Do not merge to main, change AgentTwin, deploy to production, or claim connected production readiness without separate approval.
