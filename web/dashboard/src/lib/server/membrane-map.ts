import type {
  ReleaseView,
  ReleaseSummary,
  EvalRunDetail,
  ScenarioOutcome,
  GateDecision,
} from "../datasource/types";

// Pure, I/O-free mapping from the persisted canonical evidence JSON
// (pkg/assurance.ReleaseEvidenceV1) to the dashboard view models. Kept separate
// from membrane.ts (which is server-only and does the HTTP) so this transform —
// the honesty-critical part, carrying the REAL code verdict and gate — is unit
// tested directly.

export interface EvScenario {
  scenario: string;
  status: string;
  severity: string;
}

export interface MembraneEvidenceV1 {
  release_id: string;
  correlation_id: string;
  created_at: string;
  membrane_repository: string;
  membrane_commit_sha: string;
  code_verdict: string;
  findings_summary: Record<string, number>;
  agent_id: string;
  baseline_version: string;
  candidate_version: string;
  agenttwin_evaluation_run_id: string;
  agenttwin_simulation_pair_ids?: string[];
  run_seed: string;
  overall_status: string;
  per_scenario: EvScenario[];
  gate_decision: string;
  triggered_rules?: string[];
}

export function asGate(s: string): GateDecision {
  const up = s.trim().toUpperCase();
  return (["PASS", "WARN", "BLOCK", "INCOMPLETE", "ERROR"].includes(up) ? up : "ERROR") as GateDecision;
}

function countsFrom(scenarios: EvScenario[]) {
  const c = { NEW_CRITICAL_FAILURE: 0, REGRESSED: 0, UNCHANGED: 0, IMPROVED: 0, INCOMPLETE: 0 };
  for (const s of scenarios) {
    const k = s.status.trim().toUpperCase() as keyof typeof c;
    if (k in c) c[k] += 1;
  }
  return c;
}

// Deterministic reasons derived from the persisted triggered rules, so the
// "Why this decision" panel matches the real stored gate rather than recomputing
// and risking drift from the authoritative record.
const RULE_REASON: Record<string, string> = {
  CODE_CRITICAL: "Code governance returned a blocking finding.",
  AGENT_CRITICAL_SCENARIO: "New critical scenario failure(s) in the candidate.",
  NONCRITICAL_REGRESSION: "Non-critical regression(s) versus baseline.",
  AGENT_EVAL_INCOMPLETE: "Agent evaluation is incomplete or errored — never treated as PASS.",
  OPTIONAL_ASSURANCE_UNAVAILABLE: "Optional agent assurance was unavailable.",
  COVERAGE_GAP: "Code governance returned a non-blocking warning.",
};

export function reasonsFrom(rules: string[], gate: GateDecision): string[] {
  if (rules.length === 0) {
    return gate === "PASS"
      ? ["All required checks green; all critical scenarios passed."]
      : ["No rules recorded for this decision."];
  }
  return rules.map((r) => RULE_REASON[r] ?? r);
}

export function toSummary(ev: MembraneEvidenceV1): ReleaseSummary {
  const counts = countsFrom(ev.per_scenario);
  return {
    releaseId: ev.release_id,
    agentId: ev.agent_id,
    baselineVersion: ev.baseline_version,
    candidateVersion: ev.candidate_version,
    newCriticalFailures: counts.NEW_CRITICAL_FAILURE,
    codeVerdict: asGate(ev.code_verdict),
    gate: asGate(ev.gate_decision),
    repository: ev.membrane_repository,
    commitSha: ev.membrane_commit_sha,
    createdAt: ev.created_at,
  };
}

export function toView(ev: MembraneEvidenceV1): ReleaseView {
  const counts = countsFrom(ev.per_scenario);
  const gate = asGate(ev.gate_decision);
  const rules = ev.triggered_rules ?? [];
  const perScenario: ScenarioOutcome[] = ev.per_scenario.map((s) => ({
    scenario: s.scenario,
    status: s.status,
    severity: s.severity,
  }));
  const evalRun: EvalRunDetail = {
    evalRunId: ev.agenttwin_evaluation_run_id,
    agentId: ev.agent_id,
    baselineVersion: ev.baseline_version,
    candidateVersion: ev.candidate_version,
    seed: ev.run_seed,
    counts,
    newCriticalFailures: ev.per_scenario.filter((s) => s.status.toUpperCase() === "NEW_CRITICAL_FAILURE").map((s) => s.scenario),
    regressed: ev.per_scenario.filter((s) => s.status.toUpperCase() === "REGRESSED").map((s) => s.scenario),
    status: ev.overall_status,
    gate,
    createdAt: ev.created_at,
    deterministicFake: false, // persisted from a real reproduced run
    datasetName: "refund-regression-suite",
    simulationPairIds: ev.agenttwin_simulation_pair_ids ?? [],
    perScenario,
  };
  return {
    evalRun,
    codeVerdict: asGate(ev.code_verdict),
    codeFindings: {
      blocking: ev.findings_summary.blocking ?? 0,
      warning: ev.findings_summary.warning ?? 0,
      info: ev.findings_summary.info ?? 0,
    },
    gate: { decision: gate, triggeredRules: rules, reasons: reasonsFrom(rules, gate) },
    correlation: {
      repository: ev.membrane_repository,
      commitSha: ev.membrane_commit_sha,
      agentId: ev.agent_id,
      baselineVersion: ev.baseline_version,
      candidateVersion: ev.candidate_version,
      evaluationRunId: ev.agenttwin_evaluation_run_id,
    },
  };
}
