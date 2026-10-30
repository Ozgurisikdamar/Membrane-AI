import { describe, it, expect } from "vitest";
import { mapEvalStatus, decideGate, type EvalCounts } from "./gate";

const clean: EvalCounts = { NEW_CRITICAL_FAILURE: 0, REGRESSED: 0, UNCHANGED: 9, IMPROVED: 0, INCOMPLETE: 0 };
const critical: EvalCounts = { NEW_CRITICAL_FAILURE: 2, REGRESSED: 1, UNCHANGED: 6, IMPROVED: 0, INCOMPLETE: 0 };
const regressionOnly: EvalCounts = { NEW_CRITICAL_FAILURE: 0, REGRESSED: 1, UNCHANGED: 8, IMPROVED: 0, INCOMPLETE: 0 };

describe("mapEvalStatus", () => {
  it("classifies the real 1.3.0 run as a critical failure", () => {
    expect(mapEvalStatus("COMPLETED", critical)).toBe("COMPLETED_WITH_CRITICAL_FAILURE");
  });
  it("classifies the real 1.3.1 run as clean completed", () => {
    expect(mapEvalStatus("COMPLETED", clean)).toBe("COMPLETED");
  });
  it("maps ERROR/unknown to non-passing states", () => {
    expect(mapEvalStatus("ERROR", clean)).toBe("ERROR");
    expect(mapEvalStatus("RUNNING", clean)).toBe("PENDING");
  });
});

describe("decideGate (fail-safe)", () => {
  it("1.3.0 critical failure → BLOCK citing AGENT_CRITICAL_SCENARIO", () => {
    const g = decideGate("COMPLETED_WITH_CRITICAL_FAILURE", "PASS", critical);
    expect(g.decision).toBe("BLOCK");
    expect(g.triggeredRules).toContain("AGENT_CRITICAL_SCENARIO");
    expect(g.reasons.length).toBeGreaterThan(0);
  });
  it("1.3.1 clean + code PASS → PASS", () => {
    expect(decideGate("COMPLETED", "PASS", clean).decision).toBe("PASS");
  });
  it("code BLOCK forces BLOCK even with a clean agent run", () => {
    const g = decideGate("COMPLETED", "BLOCK", clean);
    expect(g.decision).toBe("BLOCK");
    expect(g.triggeredRules).toContain("CODE_CRITICAL");
  });
  it("non-critical regression → WARN by default, BLOCK under policy", () => {
    expect(decideGate("COMPLETED_WITH_REGRESSION", "PASS", regressionOnly).decision).toBe("WARN");
    expect(decideGate("COMPLETED_WITH_REGRESSION", "PASS", regressionOnly, { regressionBlocks: true }).decision).toBe("BLOCK");
  });
  it("ERROR and INCOMPLETE never PASS", () => {
    expect(decideGate("ERROR", "PASS", clean).decision).toBe("ERROR");
    expect(decideGate("INCOMPLETE", "PASS", clean).decision).toBe("INCOMPLETE");
    expect(decideGate("PENDING", "PASS", clean).decision).toBe("INCOMPLETE");
  });
});
