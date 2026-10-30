"use client";

import { useDataSource } from "@/lib/datasource/context";
import { useSourced } from "@/lib/use-sourced";
import { PageHeader } from "@/components/ui/page-header";
import { Card, CardHeader, Stat } from "@/components/ui/card";
import { Loading, ErrorState, Disconnected } from "@/components/ui/states";
import { SourceBadge } from "@/components/ui/source-badge";
import type { GateDecision } from "@/lib/datasource/types";

const GATE_TONE: Record<GateDecision, "pass" | "warn" | "block" | "default"> = {
  PASS: "pass",
  WARN: "warn",
  BLOCK: "block",
  INCOMPLETE: "default",
  ERROR: "default",
};

export default function OverviewPage() {
  const { dataSource, mode } = useDataSource();
  const overview = useSourced((s = dataSource) => s.overview(), [mode]);
  const runs = useSourced((s = dataSource) => s.evalRuns(), [mode]);

  return (
    <div>
      <PageHeader
        title="Overview"
        subtitle="System status across Code Governance and Agent Assurance. Every panel is badged with its data source."
      />

      {overview.status === "loading" ? (
        <Loading label="Loading overview…" />
      ) : overview.status === "error" ? (
        <ErrorState detail={overview.error} />
      ) : overview.data.source === "unavailable" ? (
        <Disconnected detail={overview.data.note} />
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <Card>
            <CardHeader title="Code findings" source={overview.data.source} note={overview.data.note} />
            <div className="flex gap-6">
              <Stat label="blocking" value={overview.data.value.codeFindings.blocking} tone="block" />
              <Stat label="warning" value={overview.data.value.codeFindings.warning} tone="warn" />
              <Stat label="info" value={overview.data.value.codeFindings.info} />
            </div>
          </Card>

          <Card>
            <CardHeader title="Agent evaluations" source={overview.data.source} note={overview.data.note} />
            <div className="flex gap-6">
              <Stat label="total" value={overview.data.value.evalTotals.total} />
              <Stat label="passed" value={overview.data.value.evalTotals.passed} tone="pass" />
              <Stat label="blocked" value={overview.data.value.evalTotals.blocked} tone="block" />
            </div>
          </Card>

          <Card>
            <CardHeader title="Release decisions" source={overview.data.source} note={overview.data.note} />
            <div className="flex flex-wrap gap-4">
              {(Object.keys(overview.data.value.releaseMix) as GateDecision[]).map((g) => (
                <Stat key={g} label={g} value={overview.data.value.releaseMix[g]} tone={GATE_TONE[g]} />
              ))}
            </div>
            <p className="mt-2 text-[11px] text-muted">
              Combined gate is the recommended policy over code + agent evidence (research §5.3) — not a live enforcement engine.
            </p>
          </Card>
        </div>
      )}

      <section className="mt-6">
        <h2 className="mb-2 text-sm font-semibold text-ink">Recent evaluations</h2>
        {runs.status === "loading" ? (
          <Loading label="Loading evaluations…" />
        ) : runs.status === "error" ? (
          <ErrorState detail={runs.error} />
        ) : runs.data.source === "unavailable" ? (
          <Disconnected detail={runs.data.note} />
        ) : (
          <div className="overflow-x-auto rounded-xl border border-ink/10 bg-white/70">
            <table className="w-full text-left text-sm">
              <caption className="sr-only">Recent baseline/candidate evaluations with gate decisions</caption>
              <thead className="border-b border-ink/10 text-xs uppercase tracking-wide text-muted">
                <tr>
                  <th className="px-3 py-2">Agent</th>
                  <th className="px-3 py-2">Baseline → Candidate</th>
                  <th className="px-3 py-2">New critical</th>
                  <th className="px-3 py-2">Regressed</th>
                  <th className="px-3 py-2">Seed</th>
                  <th className="px-3 py-2">Gate</th>
                  <th className="px-3 py-2">Source</th>
                </tr>
              </thead>
              <tbody>
                {runs.data.value.map((r) => (
                  <tr key={r.evalRunId} className="border-b border-ink/5 last:border-0">
                    <td className="px-3 py-2 font-mono text-xs">{r.agentId}</td>
                    <td className="px-3 py-2 font-mono text-xs">{r.baselineVersion} → {r.candidateVersion}</td>
                    <td className="px-3 py-2">{r.counts.NEW_CRITICAL_FAILURE}</td>
                    <td className="px-3 py-2">{r.counts.REGRESSED}</td>
                    <td className="px-3 py-2 font-mono text-xs">{r.seed}</td>
                    <td className="px-3 py-2">
                      <span className={r.gate === "PASS" ? "text-pass" : r.gate === "BLOCK" ? "text-block" : "text-warn"}>
                        {r.gate}
                      </span>
                    </td>
                    <td className="px-3 py-2"><SourceBadge source={runs.data.source} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}
