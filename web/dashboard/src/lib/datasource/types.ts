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
}
