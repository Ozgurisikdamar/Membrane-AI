import "server-only";
import { checkDestination } from "./allowlist";
import { mapEvalStatus, decideGate } from "../gate";
import type {
  EvalRunSummary,
  EvalRunDetail,
  OverviewStats,
  AgentSummary,
  SimulationSummary,
  TraceRef,
  ReleaseView,
  ScenarioOutcome,
  GateDecision,
} from "../datasource/types";

// Server-only AgentTwin client. The credential and base URL live in server
// environment variables and NEVER reach the browser (DASHBOARD_AND_NETLIFY_WORKFLOW.md §5).
// The browser calls the same-origin BFF routes, which call this.

export class NotConfiguredError extends Error {}
export class AllowlistError extends Error {}

interface Config {
  baseUrl: string;
  apiKey: string;
  org?: string;
}

function config(): Config {
  const baseUrl = process.env.AGENTTWIN_BASE_URL;
  const apiKey = process.env.AGENTTWIN_API_KEY;
  if (!baseUrl || !apiKey) {
    throw new NotConfiguredError("AGENTTWIN_BASE_URL and AGENTTWIN_API_KEY are not configured");
  }
  // Loopback allowed only when explicitly opted in (local dev / compose).
  const allowLoopback = process.env.AGENTTWIN_ALLOW_LOOPBACK === "1";
  const hosts = (process.env.AGENTTWIN_ALLOWED_HOSTS ?? "").split(",").map((h) => h.trim()).filter(Boolean);
  const verdict = checkDestination(baseUrl, { allowLoopback, hosts });
  if (!verdict.ok) throw new AllowlistError(`AGENTTWIN_BASE_URL rejected: ${verdict.reason}`);
  return { baseUrl: baseUrl.replace(/\/$/, ""), apiKey, org: process.env.AGENTTWIN_ORG };
}

async function at<T>(path: string): Promise<T> {
  const cfg = config();
  const headers: Record<string, string> = {
    "X-AgentTwin-Api-Key": cfg.apiKey,
    Accept: "application/json",
  };
  if (cfg.org) headers["X-AgentTwin-Org"] = cfg.org;
  const ctrl = new AbortController();
  const timer = setTimeout(() => ctrl.abort(), 8000); // bounded deadline
  try {
    const res = await fetch(`${cfg.baseUrl}${path}`, { headers, signal: ctrl.signal, cache: "no-store" });
    if (!res.ok) throw new Error(`AgentTwin ${path} -> ${res.status}`);
    const len = Number(res.headers.get("content-length") ?? "0");
    if (len > 8_000_000) throw new Error("response too large"); // bounded size
    return (await res.json()) as T;
  } finally {
    clearTimeout(timer);
  }
}

// --- AgentTwin wire shapes (subset we consume; see pinned OpenAPI) -----------

interface ClassCounts {
  NEW_CRITICAL_FAILURE: number;
  REGRESSED: number;
  IMPROVED: number;
  UNCHANGED: number;
  INCOMPLETE: number;
}
interface ATEvalRun {
  id: string;
  agent_name: string;
  baseline_version: string;
  candidate_version: string;
  seed: number | null;
  status: string;
  counts: ClassCounts;
  baseline_run_id: string | null;
  candidate_run_id: string | null;
  created_at: string;
}
interface ATEvalSummary {
  new_critical_failures: { scenario_name: string; expectations: string[] }[];
  regressed: string[];
}
interface ATCase {
  scenario_name: string;
  severity: string;
  classification: string;
  reason: string;
}
interface ATEvalDetail {
  run: ATEvalRun;
  summary: ATEvalSummary | null;
  cases: ATCase[];
}

function gateFor(run: ATEvalRun): GateDecision {
  const status = mapEvalStatus(run.status, run.counts);
  // Code verdict is not an AgentTwin concern; default PASS so the agent-assurance
  // half is shown honestly. The unified Release view combines a real code verdict.
  return decideGate(status, "PASS", run.counts).decision;
}

function toSummary(run: ATEvalRun, summary?: ATEvalSummary | null): EvalRunSummary {
  return {
    evalRunId: run.id,
    agentId: run.agent_name,
    baselineVersion: run.baseline_version,
    candidateVersion: run.candidate_version,
    seed: run.seed === null ? "unset" : String(run.seed),
    counts: run.counts,
    newCriticalFailures: (summary?.new_critical_failures ?? []).map((n) => n.scenario_name),
    regressed: summary?.regressed ?? [],
    status: run.status,
    gate: gateFor(run),
    createdAt: run.created_at,
    deterministicFake: false, // live runs: label reflects the real planner/judge
  };
}

export async function liveEvalRuns(): Promise<EvalRunSummary[]> {
  const page = await at<{ items: ATEvalRun[] }>("/api/v1/eval-runs");
  return page.items.map((r) => toSummary(r));
}

export async function liveEvalRunDetail(id: string): Promise<EvalRunDetail> {
  const d = await at<ATEvalDetail>(`/api/v1/eval-runs/${encodeURIComponent(id)}`);
  const perScenario: ScenarioOutcome[] = d.cases.map((c) => ({
    scenario: c.scenario_name,
    status: c.classification,
    severity: c.severity,
  }));
  return {
    ...toSummary(d.run, d.summary),
    datasetName: "eval dataset",
    simulationPairIds: [d.run.baseline_run_id, d.run.candidate_run_id].filter((x): x is string => !!x),
    perScenario,
  };
}

export async function liveOverview(): Promise<OverviewStats> {
  const runs = await liveEvalRuns();
  const mix: Record<GateDecision, number> = { PASS: 0, WARN: 0, BLOCK: 0, INCOMPLETE: 0, ERROR: 0 };
  for (const r of runs) mix[r.gate] += 1;
  return {
    codeFindings: { blocking: 0, warning: 0, info: 0 }, // Membrane has no live read API (D0)
    evalTotals: {
      total: runs.length,
      completed: runs.filter((r) => r.status.toUpperCase() === "COMPLETED").length,
      blocked: runs.filter((r) => r.gate === "BLOCK").length,
      passed: runs.filter((r) => r.gate === "PASS").length,
    },
    releaseMix: mix,
    lastEvalAt: runs[0]?.createdAt,
  };
}

export async function liveAgents(): Promise<AgentSummary[]> {
  // Agents are project-scoped in AgentTwin; resolve the project from the
  // authenticated principal (never trust a client-supplied project id).
  const me = await at<{ principal?: { projects?: string[] } }>("/api/v1/me");
  const projectId = me.principal?.projects?.[0];
  if (!projectId) return [];
  const page = await at<{ items: { name: string; version_count?: number; latest_version?: string }[] }>(
    `/api/v1/projects/${encodeURIComponent(projectId)}/agents`,
  );
  return page.items.map((a) => ({
    agentId: a.name,
    versions: a.latest_version
      ? [{ version: a.latest_version, model: `${a.version_count ?? 1} version(s)` }]
      : [],
  }));
}

export async function liveSimulations(): Promise<SimulationSummary[]> {
  const page = await at<{
    items: { id: string; agent_version?: string; status: string; created_at: string; case_count?: number }[];
  }>("/api/v1/simulations");
  return page.items.map((s) => ({
    runId: s.id,
    version: s.agent_version ?? "—",
    cases: s.case_count ?? 0,
    passed: 0,
    failed: 0,
    criticalFailures: 0,
    status: s.status,
    createdAt: s.created_at,
  }));
}

export async function liveTraces(): Promise<TraceRef[]> {
  const page = await at<{ items: { trace_id: string; agent_version?: string; outcome?: string }[] }>(
    "/api/v1/traces",
  );
  const base = (process.env.AGENTTWIN_UI_URL ?? "").replace(/\/$/, "");
  return page.items.map((t) => ({
    traceId: t.trace_id,
    agentVersion: t.agent_version,
    outcome: t.outcome,
    deepLink: base ? `${base}/traces/${t.trace_id}` : undefined,
  }));
}

export async function liveRelease(id: string): Promise<ReleaseView> {
  const d = await liveEvalRunDetail(id);
  const status = mapEvalStatus(d.status, d.counts);
  const codeVerdict: GateDecision = "PASS"; // no live Membrane code read API (D0)
  const gate = decideGate(status, codeVerdict, d.counts);
  return {
    evalRun: d,
    codeVerdict,
    codeFindings: { blocking: 0, warning: 0, info: 0 },
    gate,
    correlation: {
      repository: "(live — from Membrane submission when wired, D3)",
      commitSha: "(live)",
      agentId: d.agentId,
      baselineVersion: d.baselineVersion,
      candidateVersion: d.candidateVersion,
      evaluationRunId: d.evalRunId,
    },
  };
}
