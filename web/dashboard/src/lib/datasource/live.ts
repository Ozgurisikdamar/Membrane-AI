import type {
  DataSource,
  Sourced,
  EvalRunSummary,
  EvalRunDetail,
  OverviewStats,
  AgentSummary,
  SimulationSummary,
  TraceRef,
  ReleaseView,
  ReleaseSummary,
  GateDecision,
  CodeGovernanceReport,
  PolicyDecision,
} from "./types";
import { live, unavailable } from "./types";

// LiveDataSource calls the dashboard's own BFF (same-origin `/api/bff/*`), which
// holds the backend credential server-side and forwards to AgentTwin / Membrane.
// The browser never talks to a backend directly and never holds a token
// (DASHBOARD_AND_NETLIFY_WORKFLOW.md §5). On any failure it returns an
// `unavailable` Sourced value rather than throwing, so a disconnected backend
// renders an honest "not connected" state instead of a fabricated number.

async function getJSON<T>(path: string): Promise<T> {
  const res = await fetch(path, { credentials: "same-origin", cache: "no-store" });
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  return (await res.json()) as T;
}

function connErr(e: unknown): string {
  const msg = e instanceof Error ? e.message : String(e);
  return `backend not reachable (${msg})`;
}

export class LiveDataSource implements DataSource {
  readonly mode = "live" as const;

  private async load<T>(path: string, fallback: T): Promise<Sourced<T>> {
    try {
      return live<T>(await getJSON<T>(path), new Date().toISOString());
    } catch (e) {
      return unavailable<T>(fallback, connErr(e));
    }
  }

  overview() {
    return this.load<OverviewStats>("/api/bff/overview", emptyOverview());
  }
  evalRuns() {
    return this.load<EvalRunSummary[]>("/api/bff/eval-runs", []);
  }
  evalRun(id: string) {
    return this.load<EvalRunSummary | null>(`/api/bff/eval-runs/${encodeURIComponent(id)}`, null);
  }
  evalRunDetail(id: string) {
    return this.load<EvalRunDetail | null>(`/api/bff/eval-runs/${encodeURIComponent(id)}?detail=1`, null);
  }
  agents() {
    return this.load<AgentSummary[]>("/api/bff/agents", []);
  }
  simulations() {
    return this.load<SimulationSummary[]>("/api/bff/simulations", []);
  }
  traces() {
    return this.load<TraceRef[]>("/api/bff/traces", []);
  }
  releases() {
    // Real persisted combined evidence from the Membrane evidence read API.
    return this.load<ReleaseSummary[]>("/api/bff/membrane-releases", []);
  }
  release(id: string) {
    return this.load<ReleaseView | null>(`/api/bff/release/${encodeURIComponent(id)}`, null);
  }
  async codeGovernance() {
    // Membrane exposes no read API for scan history today (D0). Live code
    // governance history is therefore unavailable; the honest state is shown.
    return unavailable<CodeGovernanceReport | null>(null, "Membrane has no read API for scan history (D0 gap)");
  }
  async policyDecisions() {
    return unavailable<PolicyDecision[]>([], "Membrane gateway exposes decisions, not a queryable history (D0 gap)");
  }
}

function emptyOverview(): OverviewStats {
  const mix: Record<GateDecision, number> = { PASS: 0, WARN: 0, BLOCK: 0, INCOMPLETE: 0, ERROR: 0 };
  return {
    codeFindings: { blocking: 0, warning: 0, info: 0 },
    evalTotals: { total: 0, completed: 0, blocked: 0, passed: 0 },
    releaseMix: mix,
  };
}
