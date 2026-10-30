"use client";

import { useDataSource } from "@/lib/datasource/context";
import { useSourced } from "@/lib/use-sourced";
import { PageHeader } from "@/components/ui/page-header";
import { SourcedView } from "@/components/ui/sourced-view";
import { SourceBadge } from "@/components/ui/source-badge";
import { Empty } from "@/components/ui/states";

export default function SimulationsPage() {
  const { dataSource, mode } = useDataSource();
  const sims = useSourced(() => dataSource.simulations(), [mode]);
  return (
    <div>
      <PageHeader title="Simulation Center" subtitle="Seeded scenario runs against stateful tool twins. Each run is a version simulated over the pinned suite." />
      <SourcedView load={sims} loadingLabel="Loading simulations…">
        {(data) =>
          data.value.length === 0 ? (
            <Empty title="No simulations" hint="No simulation runs from this source." />
          ) : (
            <div className="overflow-x-auto rounded-xl border border-ink/10 bg-white/70">
              <table className="w-full text-left text-sm">
                <caption className="sr-only">Simulation runs</caption>
                <thead className="border-b border-ink/10 text-xs uppercase tracking-wide text-muted">
                  <tr>
                    <th className="px-3 py-2">Version</th>
                    <th className="px-3 py-2">Cases</th>
                    <th className="px-3 py-2">Passed</th>
                    <th className="px-3 py-2">Failed</th>
                    <th className="px-3 py-2">Critical</th>
                    <th className="px-3 py-2">Status</th>
                    <th className="px-3 py-2">Run</th>
                    <th className="px-3 py-2">Source</th>
                  </tr>
                </thead>
                <tbody>
                  {data.value.map((s) => (
                    <tr key={s.runId} className="border-b border-ink/5 last:border-0">
                      <td className="px-3 py-2 font-mono text-xs">{s.version}</td>
                      <td className="px-3 py-2">{s.cases}</td>
                      <td className="px-3 py-2 text-pass">{s.passed}</td>
                      <td className="px-3 py-2 text-warn">{s.failed}</td>
                      <td className={`px-3 py-2 ${s.criticalFailures > 0 ? "text-block font-semibold" : "text-muted"}`}>{s.criticalFailures}</td>
                      <td className="px-3 py-2 text-xs">{s.status}</td>
                      <td className="px-3 py-2 font-mono text-[11px]">{s.runId.slice(0, 12)}…</td>
                      <td className="px-3 py-2"><SourceBadge source={data.source} /></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )
        }
      </SourcedView>
    </div>
  );
}
