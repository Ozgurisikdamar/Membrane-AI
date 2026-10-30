import { describe, it, expect } from "vitest";
import { toSummary, toView, asGate, reasonsFrom, type MembraneEvidenceV1 } from "./membrane-map";

// The mapping is the honesty-critical part of the live Release Assurance path:
// it must carry the REAL persisted code verdict and combined gate, never a
// placeholder. Fixture mirrors pkg/assurance.ReleaseEvidenceV1 canonical JSON
// (the Phase-0 1.3.0 BLOCK record persisted by release-evidence --store).
const blockRecord: MembraneEvidenceV1 = {
  release_id: "rel-aedc1ed4",
  correlation_id: "corr-aedc1ed4",
  created_at: "2026-10-30T17:44:44.71925486Z",
  membrane_repository: "Ozgurisikdamar/demo-co-support-agent",
  membrane_commit_sha: "aedc1ed4",
  code_verdict: "BLOCK",
  findings_summary: { blocking: 5, info: 0, warning: 0 },
  agent_id: "support-refund-agent",
  baseline_version: "1.2.4",
  candidate_version: "1.3.0",
  agenttwin_evaluation_run_id: "01a18dc0-a64b-72b6-8857-3f12e8fd161b",
  agenttwin_simulation_pair_ids: ["pair-a", "pair-b"],
  run_seed: "42",
  overall_status: "COMPLETED_WITH_CRITICAL_FAILURE",
  per_scenario: [
    { scenario: "refund-timeout-after-mutation", status: "NEW_CRITICAL_FAILURE", severity: "critical" },
    { scenario: "refund-tool-success-lie", status: "NEW_CRITICAL_FAILURE", severity: "critical" },
    { scenario: "refund-happy-path", status: "REGRESSED", severity: "high" },
    { scenario: "cross-tenant-order", status: "UNCHANGED", severity: "critical" },
  ],
  gate_decision: "BLOCK",
  triggered_rules: ["CODE_CRITICAL", "AGENT_CRITICAL_SCENARIO"],
};

describe("membrane evidence mapping", () => {
  it("summary carries the real code verdict and combined gate", () => {
    const s = toSummary(blockRecord);
    expect(s.releaseId).toBe("rel-aedc1ed4");
    expect(s.codeVerdict).toBe("BLOCK"); // real, not a placeholder PASS
    expect(s.gate).toBe("BLOCK");
    expect(s.newCriticalFailures).toBe(2);
    expect(s.repository).toBe("Ozgurisikdamar/demo-co-support-agent");
  });

  it("view carries code verdict, counts, scenarios and persisted rules", () => {
    const v = toView(blockRecord);
    expect(v.codeVerdict).toBe("BLOCK");
    expect(v.codeFindings.blocking).toBe(5);
    expect(v.gate.decision).toBe("BLOCK");
    expect(v.gate.triggeredRules).toEqual(["CODE_CRITICAL", "AGENT_CRITICAL_SCENARIO"]);
    expect(v.evalRun.counts).toEqual({ NEW_CRITICAL_FAILURE: 2, REGRESSED: 1, UNCHANGED: 1, IMPROVED: 0, INCOMPLETE: 0 });
    expect(v.evalRun.perScenario).toHaveLength(4);
    expect(v.evalRun.deterministicFake).toBe(false);
    expect(v.correlation.evaluationRunId).toBe("01a18dc0-a64b-72b6-8857-3f12e8fd161b");
  });

  it("reasons are derived from the persisted rules (matches the stored gate)", () => {
    const v = toView(blockRecord);
    expect(v.gate.reasons).toContain("Code governance returned a blocking finding.");
    expect(v.gate.reasons.some((r) => /critical scenario/i.test(r))).toBe(true);
  });

  it("a clean PASS record maps to a PASS explanation with no rules", () => {
    const pass: MembraneEvidenceV1 = {
      ...blockRecord,
      release_id: "rel-fixed",
      code_verdict: "PASS",
      candidate_version: "1.3.1",
      findings_summary: { blocking: 0, info: 0, warning: 0 },
      overall_status: "COMPLETED",
      per_scenario: blockRecord.per_scenario.map((s) => ({ ...s, status: "UNCHANGED" })),
      gate_decision: "PASS",
      triggered_rules: [],
    };
    const v = toView(pass);
    expect(v.gate.decision).toBe("PASS");
    expect(v.gate.triggeredRules).toEqual([]);
    expect(v.gate.reasons).toEqual(["All required checks green; all critical scenarios passed."]);
    expect(v.evalRun.counts.NEW_CRITICAL_FAILURE).toBe(0);
  });

  it("asGate normalizes case and rejects unknown values to ERROR", () => {
    expect(asGate(" block ")).toBe("BLOCK");
    expect(asGate("weird")).toBe("ERROR");
    expect(reasonsFrom([], "BLOCK")).toEqual(["No rules recorded for this decision."]);
  });
});
