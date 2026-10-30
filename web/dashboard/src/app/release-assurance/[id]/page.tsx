"use client";

import { useParams } from "next/navigation";
import { useDataSource } from "@/lib/datasource/context";
import { useSourced } from "@/lib/use-sourced";
import { PageHeader } from "@/components/ui/page-header";
import { SourcedView } from "@/components/ui/sourced-view";
import { SourceBadge } from "@/components/ui/source-badge";
import { Card, CardHeader } from "@/components/ui/card";
import { Empty } from "@/components/ui/states";
import type { GateDecision, ScenarioOutcome } from "@/lib/datasource/types";

const GATE_BG: Record<GateDecision, string> = {
  PASS: "bg-pass/10 text-pass border-pass/40",
  WARN: "bg-warn/10 text-warn border-warn/40",
  BLOCK: "bg-block/10 text-block border-block/40",
  INCOMPLETE: "bg-unavailable/15 text-muted border-unavailable/40",
  ERROR: "bg-unavailable/15 text-muted border-unavailable/40",
};

function scenarioTone(status: string): string {
  if (status === "NEW_CRITICAL_FAILURE") return "text-block font-semibold";
  if (status === "REGRESSED") return "text-warn font-semibold";
  if (status === "IMPROVED") return "text-pass";
  return "text-muted";
}

export default function ReleaseDetailPage() {
  const params = useParams<{ id: string }>();
  const id = decodeURIComponent(params.id);
  const { dataSource, mode } = useDataSource();
  const rel = useSourced(() => dataSource.release(id), [mode, id]);

  return (
    <div>
      <PageHeader title="Release evidence" subtitle={`Evaluation ${id}`} />
      <SourcedView load={rel} loadingLabel="Loading release evidence…">
        {(data) => {
          const r = data.value;
          if (!r) return <Empty title="Not found" hint="No release evidence for this id from the current source." />;
          return (
            <div className="flex flex-col gap-4">
              <div className={`flex items-center justify-between rounded-xl border p-4 ${GATE_BG[r.gate.decision]}`}>
                <div>
                  <p className="text-xs uppercase tracking-wide">Combined gate decision</p>
                  <p className="font-mono text-3xl font-bold">{r.gate.decision}</p>
                </div>
                <SourceBadge source={data.source} title={data.note} />
              </div>

              <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
                <Card>
                  <CardHeader title="Why this decision" source={data.source} />
                  <ul className="list-disc space-y-1 pl-5 text-sm">
                    {r.gate.reasons.map((reason, i) => (
                      <li key={i}>{reason}</li>
                    ))}
                  </ul>
                  {r.gate.triggeredRules.length > 0 ? (
                    <div className="mt-3 flex flex-wrap gap-1">
                      {r.gate.triggeredRules.map((rule) => (
                        <span key={rule} className="rounded border border-ink/20 bg-white px-1.5 py-0.5 font-mono text-[10px] text-ink">
                          {rule}
                        </span>
                      ))}
                    </div>
                  ) : null}
                  <p className="mt-3 text-[11px] text-muted">
                    Recommended deterministic policy (research §5.3). A critical failure, ERROR or INCOMPLETE can never be PASS.
                  </p>
                </Card>

                <Card>
                  <CardHeader title="Correlation" source={data.source} />
                  <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-xs">
                    <dt className="text-muted">Repository</dt>
                    <dd className="font-mono">{r.correlation.repository}</dd>
                    <dt className="text-muted">Commit</dt>
                    <dd className="font-mono break-all">{r.correlation.commitSha}</dd>
                    <dt className="text-muted">Agent</dt>
                    <dd className="font-mono">{r.correlation.agentId}</dd>
                    <dt className="text-muted">Baseline → Candidate</dt>
                    <dd className="font-mono">{r.correlation.baselineVersion} → {r.correlation.candidateVersion}</dd>
                    <dt className="text-muted">Evaluation run</dt>
                    <dd className="font-mono break-all">{r.correlation.evaluationRunId}</dd>
                    <dt className="text-muted">Code verdict</dt>
                    <dd className="font-mono">{r.codeVerdict} ({r.codeFindings.blocking} blocking)</dd>
                    <dt className="text-muted">Seed</dt>
                    <dd className="font-mono">{r.evalRun.seed}</dd>
                  </dl>
                </Card>
              </div>

              <Card>
                <CardHeader title={`Baseline/candidate scenarios (${r.evalRun.perScenario.length})`} source={data.source} />
                <div className="overflow-x-auto">
                  <table className="w-full text-left text-sm">
                    <caption className="sr-only">Per-scenario classification for the candidate vs baseline</caption>
                    <thead className="border-b border-ink/10 text-xs uppercase tracking-wide text-muted">
                      <tr>
                        <th className="px-3 py-2">Scenario</th>
                        <th className="px-3 py-2">Severity</th>
                        <th className="px-3 py-2">Classification</th>
                        <th className="px-3 py-2">Evidence</th>
                      </tr>
                    </thead>
                    <tbody>
                      {r.evalRun.perScenario.map((s: ScenarioOutcome) => (
                        <tr key={s.scenario} className="border-b border-ink/5 last:border-0">
                          <td className="px-3 py-2 font-mono text-xs">{s.scenario}</td>
                          <td className="px-3 py-2">{s.severity}</td>
                          <td className={`px-3 py-2 ${scenarioTone(s.status)}`}>{s.status}</td>
                          <td className="px-3 py-2">
                            {s.evidenceUrls?.trajectory ? (
                              <a className="text-accent underline" href={s.evidenceUrls.trajectory} target="_blank" rel="noreferrer">
                                trajectory →
                              </a>
                            ) : (
                              <span className="text-muted">—</span>
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
                {data.source === "demo" ? (
                  <p className="mt-2 text-[11px] text-demo">Deterministic-fake planner; evidence links are illustrative in demo mode.</p>
                ) : null}
              </Card>
            </div>
          );
        }}
      </SourcedView>
    </div>
  );
}
