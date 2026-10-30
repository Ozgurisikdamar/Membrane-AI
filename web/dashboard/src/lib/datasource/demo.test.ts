import { describe, it, expect } from "vitest";
import { DemoDataSource } from "./demo";
import { demoEvalRuns } from "./fixtures";

describe("DemoDataSource", () => {
  const ds = new DemoDataSource();

  it("reports demo provenance on every call", async () => {
    expect((await ds.overview()).source).toBe("demo");
    expect((await ds.evalRuns()).source).toBe("demo");
  });

  it("returns the real-derived eval runs (1.3.0 BLOCK, 1.3.1 PASS)", async () => {
    const { value } = await ds.evalRuns();
    const byCand = Object.fromEntries(value.map((r) => [r.candidateVersion, r]));
    expect(byCand["1.3.0"].gate).toBe("BLOCK");
    expect(byCand["1.3.0"].counts.NEW_CRITICAL_FAILURE).toBe(2);
    expect(byCand["1.3.1"].gate).toBe("PASS");
    expect(byCand["1.3.1"].counts.NEW_CRITICAL_FAILURE).toBe(0);
  });

  it("marks demo runs as deterministic-fake (never shown as a real LLM)", async () => {
    const { value } = await ds.evalRuns();
    expect(value.every((r) => r.deterministicFake)).toBe(true);
  });

  it("looks up a run by id and returns null for an unknown id", async () => {
    const known = demoEvalRuns[0].evalRunId;
    expect((await ds.evalRun(known)).value?.evalRunId).toBe(known);
    expect((await ds.evalRun("does-not-exist")).value).toBeNull();
  });

  it("is deterministic across calls", async () => {
    const a = await ds.overview();
    const b = await ds.overview();
    expect(a.value).toEqual(b.value);
  });
});
