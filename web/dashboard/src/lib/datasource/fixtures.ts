import type { EvalRunSummary, OverviewStats } from "./types";

// Deterministic demo fixtures DERIVED FROM THE REAL Phase-0 reproduced runs
// (see docs/agenttwin-integration/PROGRESS.md §7 and pkg/assurance/contracts/
// fixtures). These are synthetic for display but faithful to what the live
// AgentTwin stack actually produced: 1.2.4 = 9/9; 1.3.0 = 6/9 (2 critical, 1
// regressed) → BLOCK; 1.3.1 = 9/9 → PASS. Never presented as live.

export const FIXTURE_VERSION = "phase0-2026-10-30";

export const demoEvalRuns: EvalRunSummary[] = [
  {
    evalRunId: "01a18c73-c690-73bd-8d2c-711c059ba059",
    agentId: "support-refund-agent",
    baselineVersion: "1.2.4",
    candidateVersion: "1.3.1",
    seed: "42",
    counts: { NEW_CRITICAL_FAILURE: 0, REGRESSED: 0, UNCHANGED: 9, IMPROVED: 0, INCOMPLETE: 0 },
    newCriticalFailures: [],
    regressed: [],
    status: "COMPLETED",
    gate: "PASS",
    createdAt: "2026-10-30T10:56:00Z",
    deterministicFake: true,
  },
  {
    evalRunId: "01a18c71-faf5-70b5-a459-7052a0b23f37",
    agentId: "support-refund-agent",
    baselineVersion: "1.2.4",
    candidateVersion: "1.3.0",
    seed: "default",
    counts: { NEW_CRITICAL_FAILURE: 2, REGRESSED: 1, UNCHANGED: 6, IMPROVED: 0, INCOMPLETE: 0 },
    newCriticalFailures: ["refund-timeout-after-mutation", "refund-tool-success-lie"],
    regressed: ["refund-happy-path"],
    status: "COMPLETED",
    gate: "BLOCK",
    createdAt: "2026-10-30T10:54:02Z",
    deterministicFake: true,
  },
];

export const demoOverview: OverviewStats = {
  // Code findings mirror the Phase-0 synthetic scan sample (3 blocking secrets).
  codeFindings: { blocking: 3, warning: 0, info: 0 },
  evalTotals: { total: 2, completed: 2, blocked: 1, passed: 1 },
  releaseMix: { PASS: 1, WARN: 0, BLOCK: 1, INCOMPLETE: 0, ERROR: 0 },
  lastScanAt: "2026-10-30T10:36:00Z",
  lastEvalAt: "2026-10-30T10:56:00Z",
};
