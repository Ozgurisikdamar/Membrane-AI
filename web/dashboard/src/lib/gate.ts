import type { GateDecision } from "./datasource/types";

// TypeScript mirror of pkg/assurance (status.go MapEvalStatus/Decide + the
// CONTRACTS.md §5 rule catalog). Kept deliberately small and pure so the
// Release Assurance view explains a decision with the SAME deterministic,
// fail-safe logic the Go contract uses — a critical failure dominates, and
// nothing non-clean can be PASS. This is the "recommended gate policy" shown in
// the UI (research §5.3), not a live enforcement engine.

export type IntegrationStatus =
  | "COMPLETED"
  | "COMPLETED_WITH_CRITICAL_FAILURE"
  | "COMPLETED_WITH_REGRESSION"
  | "ERROR"
  | "INCOMPLETE"
  | "PENDING";

export interface EvalCounts {
  NEW_CRITICAL_FAILURE: number;
  REGRESSED: number;
  UNCHANGED: number;
  IMPROVED: number;
  INCOMPLETE: number;
}

export interface GatePolicy {
  regressionBlocks?: boolean;
}

export function mapEvalStatus(rawStatus: string, counts: EvalCounts, errored = 0): IntegrationStatus {
  switch (rawStatus.trim().toUpperCase()) {
    case "COMPLETED":
      if (errored > 0) return "ERROR";
      if (counts.INCOMPLETE > 0) return "INCOMPLETE";
      if (counts.NEW_CRITICAL_FAILURE > 0) return "COMPLETED_WITH_CRITICAL_FAILURE";
      if (counts.REGRESSED > 0) return "COMPLETED_WITH_REGRESSION";
      return "COMPLETED";
    case "ERROR":
    case "FAILED":
      return "ERROR";
    case "CANCELLED":
      return "INCOMPLETE";
    default:
      return "PENDING";
  }
}

export interface GateResult {
  decision: GateDecision;
  triggeredRules: string[];
  reasons: string[];
}

/**
 * Combined gate over the agent-assurance status plus the code-governance
 * verdict. Fail-safe: a critical failure, ERROR, INCOMPLETE or PENDING can
 * never be PASS, and every BLOCK/WARN cites a rule.
 */
export function decideGate(
  status: IntegrationStatus,
  codeVerdict: GateDecision,
  counts: EvalCounts,
  policy: GatePolicy = {},
): GateResult {
  const rules: string[] = [];
  const reasons: string[] = [];

  if (codeVerdict === "BLOCK") {
    rules.push("CODE_CRITICAL");
    reasons.push("Code governance returned a blocking finding.");
  }

  switch (status) {
    case "COMPLETED_WITH_CRITICAL_FAILURE":
      rules.push("AGENT_CRITICAL_SCENARIO");
      reasons.push(`${counts.NEW_CRITICAL_FAILURE} new critical scenario failure(s) in the candidate.`);
      break;
    case "COMPLETED_WITH_REGRESSION":
      rules.push("NONCRITICAL_REGRESSION");
      reasons.push(`${counts.REGRESSED} non-critical regression(s).`);
      break;
    case "ERROR":
      rules.push("AGENT_EVAL_INCOMPLETE");
      reasons.push("Agent evaluation errored — never treated as PASS.");
      break;
    case "INCOMPLETE":
    case "PENDING":
      rules.push("AGENT_EVAL_INCOMPLETE");
      reasons.push("Agent evaluation is incomplete/pending — never treated as PASS.");
      break;
    case "COMPLETED":
      break;
  }

  // Decision precedence: BLOCK > ERROR/INCOMPLETE > WARN > PASS.
  if (rules.includes("CODE_CRITICAL") || rules.includes("AGENT_CRITICAL_SCENARIO")) {
    return { decision: "BLOCK", triggeredRules: rules, reasons };
  }
  if (status === "ERROR") return { decision: "ERROR", triggeredRules: rules, reasons };
  if (status === "INCOMPLETE" || status === "PENDING")
    return { decision: "INCOMPLETE", triggeredRules: rules, reasons };
  if (status === "COMPLETED_WITH_REGRESSION") {
    if (policy.regressionBlocks) {
      rules.push("NONCRITICAL_REGRESSION");
      return { decision: "BLOCK", triggeredRules: rules, reasons };
    }
    return { decision: "WARN", triggeredRules: rules, reasons };
  }
  if (codeVerdict === "WARN") {
    rules.push("COVERAGE_GAP");
    reasons.push("Code governance returned a non-blocking warning.");
    return { decision: "WARN", triggeredRules: rules, reasons };
  }
  reasons.push("All required checks green; all critical scenarios passed.");
  return { decision: "PASS", triggeredRules: rules, reasons };
}
