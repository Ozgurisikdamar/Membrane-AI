// The dashboard never shows a number without saying where it came from. Every
// datum is wrapped in `Sourced<T>` carrying its provenance, and every data call
// goes through the `DataSource` interface so a screen is identical whether it is
// backed by a live backend, deterministic demo fixtures, or nothing at all.

/** Where a value came from. Rendered as a badge on every panel. */
export type Provenance = "live" | "demo" | "unavailable";

/** A value plus its provenance and freshness. */
export interface Sourced<T> {
  value: T;
  source: Provenance;
  /** ISO timestamp the underlying data was produced/fetched, if known. */
  asOf?: string;
  /** Short human note, e.g. "deterministic-fake planner, seed 42" or an error. */
  note?: string;
}

export function live<T>(value: T, asOf?: string, note?: string): Sourced<T> {
  return { value, source: "live", asOf, note };
}
export function demo<T>(value: T, note?: string): Sourced<T> {
  return { value, source: "demo", note };
}
export function unavailable<T>(value: T, note?: string): Sourced<T> {
  return { value, source: "unavailable", note };
}

/** Classification of an AgentTwin paired evaluation, mirrors pkg/assurance. */
export type GateDecision = "PASS" | "WARN" | "BLOCK" | "INCOMPLETE" | "ERROR";

export interface EvalRunSummary {
  evalRunId: string;
  agentId: string;
  baselineVersion: string;
  candidateVersion: string;
  seed: string;
  /** Per-classification counts from the baseline/candidate comparison. */
  counts: {
    NEW_CRITICAL_FAILURE: number;
    REGRESSED: number;
    UNCHANGED: number;
    IMPROVED: number;
    INCOMPLETE: number;
  };
  newCriticalFailures: string[];
  regressed: string[];
  /** AgentTwin run status, e.g. COMPLETED. */
  status: string;
  /** The Membrane-side combined gate decision for this evaluation. */
  gate: GateDecision;
  createdAt: string;
  /** True when produced by the deterministic-fake planner (not a real LLM). */
  deterministicFake: boolean;
}

export interface OverviewStats {
  codeFindings: { blocking: number; warning: number; info: number };
  evalTotals: { total: number; completed: number; blocked: number; passed: number };
  releaseMix: Record<GateDecision, number>;
  lastScanAt?: string;
  lastEvalAt?: string;
}

export interface ScenarioOutcome {
  scenario: string;
  status: string;
  severity: string;
  evidenceUrls?: Record<string, string>;
}

export interface EvalRunDetail extends EvalRunSummary {
  datasetName: string;
  simulationPairIds: string[];
  perScenario: ScenarioOutcome[];
}

export interface AgentVersion {
  version: string;
  model: string;
  manifestSha?: string;
  promptSha?: string;
  createdAt?: string;
}

export interface AgentSummary {
  agentId: string;
  versions: AgentVersion[];
}

export interface SimulationSummary {
  runId: string;
  version: string;
  cases: number;
  passed: number;
  failed: number;
  criticalFailures: number;
  status: string;
  createdAt: string;
}

export interface TraceRef {
  traceId: string;
  agentVersion?: string;
  outcome?: string;
  verified?: boolean;
  /** Canonical deep link into the AgentTwin trace explorer. */
  deepLink?: string;
}

export interface ReleaseView {
  evalRun: EvalRunDetail;
  codeVerdict: GateDecision;
  codeFindings: { blocking: number; warning: number; info: number };
  gate: { decision: GateDecision; triggeredRules: string[]; reasons: string[] };
  correlation: {
    repository: string;
    commitSha: string;
    agentId: string;
    baselineVersion: string;
    candidateVersion: string;
    evaluationRunId: string;
  };
}

/**
 * One row of the Release Assurance list: a persisted combined release-evidence
 * record (pkg/assurance.ReleaseEvidenceV1). Unlike EvalRunSummary this carries
 * the REAL code verdict and the combined gate, keyed by the Membrane release id.
 */
export interface ReleaseSummary {
  releaseId: string;
  agentId: string;
  baselineVersion: string;
  candidateVersion: string;
  newCriticalFailures: number;
  codeVerdict: GateDecision;
  gate: GateDecision;
  repository: string;
  commitSha: string;
  createdAt: string;
}

export interface CodeFinding {
  file: string;
  line: number;
  rule: string;
  severity: "blocking" | "warning" | "info";
  message: string;
  /** Source line AFTER secret masking (secrets appear only as [MASKED:<rule>]). */
  excerpt: string;
}

export interface CodeRuleStat {
  rule: string;
  severity: string;
  count: number;
}

export interface CodeGovernanceReport {
  root: string;
  filesScanned: number;
  filesSkipped: number;
  total: number;
  blocking: number;
  warnings: number;
  infos: number;
  affectedFiles: number;
  grade: string;
  debtScore: number;
  byRule: CodeRuleStat[];
  findings: CodeFinding[];
}

export interface PolicyDecision {
  kind: "tool-call" | "package" | "shadow-ai";
  subject: string;
  decision: "allow" | "deny" | "review";
  reasons: string[];
}

/**
 * The single data boundary every screen uses. Implementations: DemoDataSource
 * (local deterministic fixtures) and LiveDataSource (BFF → backend). A method
 * returns `unavailable` rather than throwing when a backend is not reachable.
 */
export interface DataSource {
  readonly mode: Provenance;
  overview(): Promise<Sourced<OverviewStats>>;
  evalRuns(): Promise<Sourced<EvalRunSummary[]>>;
  evalRun(id: string): Promise<Sourced<EvalRunSummary | null>>;
  evalRunDetail(id: string): Promise<Sourced<EvalRunDetail | null>>;
  agents(): Promise<Sourced<AgentSummary[]>>;
  simulations(): Promise<Sourced<SimulationSummary[]>>;
  traces(): Promise<Sourced<TraceRef[]>>;
  /** Persisted combined release-evidence records (real code verdict + gate). */
  releases(): Promise<Sourced<ReleaseSummary[]>>;
  release(id: string): Promise<Sourced<ReleaseView | null>>;
  codeGovernance(): Promise<Sourced<CodeGovernanceReport | null>>;
  policyDecisions(): Promise<Sourced<PolicyDecision[]>>;
}
