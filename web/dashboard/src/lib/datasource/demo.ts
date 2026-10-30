import type {
  DataSource,
  Sourced,
  EvalRunSummary,
  EvalRunDetail,
  OverviewStats,
  AgentSummary,
  SimulationSummary,
  TraceRef,
  ReleaseView,
  ScenarioOutcome,
} from "./types";
import { demo } from "./types";
import {
  demoEvalRuns,
  demoOverview,
  demoScenarios130,
  demoScenarios131,
  demoAgents,
  demoSimulations,
  demoTraces,
  FIXTURE_VERSION,
} from "./fixtures";
import { decideGate, mapEvalStatus } from "../gate";

const NOTE = `demo fixtures ${FIXTURE_VERSION} · deterministic-fake planner · seeds shown`;

function scenariosFor(candidate: string): ScenarioOutcome[] {
  return candidate === "1.3.0" ? demoScenarios130 : demoScenarios131;
}

function detail(run: EvalRunSummary): EvalRunDetail {
  return {
    ...run,
    datasetName: "refund-regression-suite",
    simulationPairIds:
      run.candidateVersion === "1.3.0"
        ? ["01a18c71-fa92-70c2-93ce-656f886bd179", "01a18c71-faaa-7265-bc3d-5139f51f705c"]
        : ["01a18c73-c653-7245-97bb-3afb9a473943", "01a18c73-c669-72d4-8caf-9b10507df067"],
    perScenario: scenariosFor(run.candidateVersion),
  };
}

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
    return demo<EvalRunSummary | null>(demoEvalRuns.find((r) => r.evalRunId === id) ?? null, NOTE);
  }

  async evalRunDetail(id: string): Promise<Sourced<EvalRunDetail | null>> {
    const run = demoEvalRuns.find((r) => r.evalRunId === id);
    return demo<EvalRunDetail | null>(run ? detail(run) : null, NOTE);
  }

  async agents(): Promise<Sourced<AgentSummary[]>> {
    return demo<AgentSummary[]>(demoAgents, NOTE);
  }

  async simulations(): Promise<Sourced<SimulationSummary[]>> {
    return demo<SimulationSummary[]>(demoSimulations, NOTE);
  }

  async traces(): Promise<Sourced<TraceRef[]>> {
    return demo<TraceRef[]>(demoTraces, NOTE);
  }

  async release(id: string): Promise<Sourced<ReleaseView | null>> {
    const run = demoEvalRuns.find((r) => r.evalRunId === id);
    if (!run) return demo<ReleaseView | null>(null, NOTE);
    const d = detail(run);
    // Demo code verdict: the Phase-0 synthetic scan was clean for these changes.
    const codeVerdict = "PASS" as const;
    const status = mapEvalStatus(run.status, run.counts);
    const gate = decideGate(status, codeVerdict, run.counts);
    const view: ReleaseView = {
      evalRun: d,
      codeVerdict,
      codeFindings: demoOverview.codeFindings,
      gate,
      correlation: {
        repository: "Ozgurisikdamar/demo-co-support-agent",
        commitSha: "aedc1ed4014a684868ef3a339ea1d6d5df8235e0",
        agentId: run.agentId,
        baselineVersion: run.baselineVersion,
        candidateVersion: run.candidateVersion,
        evaluationRunId: run.evalRunId,
      },
    };
    return demo<ReleaseView | null>(view, NOTE);
  }
}
