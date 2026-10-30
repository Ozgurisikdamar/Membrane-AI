import type { DataSource, Sourced, EvalRunSummary, OverviewStats, GateDecision } from "./types";
import { live, unavailable } from "./types";

// LiveDataSource calls the dashboard's own BFF (same-origin `/api/bff/*`), which
// holds the backend credential server-side and forwards to AgentTwin / Membrane.
// The browser never talks to a backend directly and never holds a token
// (DASHBOARD_AND_NETLIFY_WORKFLOW.md §5). On any failure it returns an
// `unavailable` Sourced value rather than throwing, so a disconnected backend
// renders an honest "not connected" state instead of a fabricated number.
//
// D1 ships this contract; the BFF routes and the full AgentTwin→EvalRunSummary
// mapping are completed in D2. Until a backend is configured, every call here
// resolves to `unavailable`.

async function getJSON<T>(path: string): Promise<T> {
  const res = await fetch(path, { credentials: "same-origin", cache: "no-store" });
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  return (await res.json()) as T;
}

export class LiveDataSource implements DataSource {
  readonly mode = "live" as const;

  async overview(): Promise<Sourced<OverviewStats>> {
    try {
      const data = await getJSON<OverviewStats>("/api/bff/overview");
      return live<OverviewStats>(data, new Date().toISOString());
    } catch (e) {
      return unavailable<OverviewStats>(emptyOverview(), connErr(e));
    }
  }

  async evalRuns(): Promise<Sourced<EvalRunSummary[]>> {
    try {
      const data = await getJSON<EvalRunSummary[]>("/api/bff/eval-runs");
      return live<EvalRunSummary[]>(data, new Date().toISOString());
    } catch (e) {
      return unavailable<EvalRunSummary[]>([], connErr(e));
    }
  }

  async evalRun(id: string): Promise<Sourced<EvalRunSummary | null>> {
    try {
      const data = await getJSON<EvalRunSummary>(`/api/bff/eval-runs/${encodeURIComponent(id)}`);
      return live<EvalRunSummary | null>(data, new Date().toISOString());
    } catch (e) {
      return unavailable<EvalRunSummary | null>(null, connErr(e));
    }
  }
}

function connErr(e: unknown): string {
  const msg = e instanceof Error ? e.message : String(e);
  return `backend not reachable (${msg})`;
}

function emptyOverview(): OverviewStats {
  const mix: Record<GateDecision, number> = { PASS: 0, WARN: 0, BLOCK: 0, INCOMPLETE: 0, ERROR: 0 };
  return {
    codeFindings: { blocking: 0, warning: 0, info: 0 },
    evalTotals: { total: 0, completed: 0, blocked: 0, passed: 0 },
    releaseMix: mix,
  };
}
