import type { DataSource, Sourced, EvalRunSummary, OverviewStats } from "./types";
import { demo } from "./types";
import { demoEvalRuns, demoOverview, FIXTURE_VERSION } from "./fixtures";

const NOTE = `demo fixtures ${FIXTURE_VERSION} · deterministic-fake planner · seeds shown`;

/** Deterministic, local, read-only. Identical across reloads. No backend calls. */
export class DemoDataSource implements DataSource {
  readonly mode = "demo" as const;

  async overview(): Promise<Sourced<OverviewStats>> {
    return demo<OverviewStats>(demoOverview, NOTE);
  }

  async evalRuns(): Promise<Sourced<EvalRunSummary[]>> {
    return demo<EvalRunSummary[]>(demoEvalRuns, NOTE);
  }

  async evalRun(id: string): Promise<Sourced<EvalRunSummary | null>> {
    const run = demoEvalRuns.find((r) => r.evalRunId === id) ?? null;
    return demo<EvalRunSummary | null>(run, NOTE);
  }
}
