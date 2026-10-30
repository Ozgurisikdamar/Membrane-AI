"use client";

import { useDataSource } from "@/lib/datasource/context";
import { useSourced } from "@/lib/use-sourced";
import { PageHeader } from "@/components/ui/page-header";
import { SourcedView } from "@/components/ui/sourced-view";
import { SourceBadge } from "@/components/ui/source-badge";
import { Empty } from "@/components/ui/states";

export default function TracesPage() {
  const { dataSource, mode } = useDataSource();
  const traces = useSourced(() => dataSource.traces(), [mode]);
  return (
    <div>
      <PageHeader
        title="Trace Explorer"
        subtitle="Trace list with deep links into AgentTwin. Membrane shows summaries + links only — it does not mirror raw sensitive span payloads (doc §2.5)."
      />
      <SourcedView load={traces} loadingLabel="Loading traces…">
        {(data) =>
          data.value.length === 0 ? (
            <Empty title="No traces" hint="No traces from this source." />
          ) : (
            <div className="overflow-x-auto rounded-xl border border-ink/10 bg-white/70">
              <table className="w-full text-left text-sm">
                <caption className="sr-only">Traces</caption>
                <thead className="border-b border-ink/10 text-xs uppercase tracking-wide text-muted">
                  <tr>
                    <th className="px-3 py-2">Trace</th>
                    <th className="px-3 py-2">Agent version</th>
                    <th className="px-3 py-2">Outcome</th>
                    <th className="px-3 py-2">Verified</th>
                    <th className="px-3 py-2">Source</th>
                    <th className="px-3 py-2"></th>
                  </tr>
                </thead>
                <tbody>
                  {data.value.map((t) => (
                    <tr key={t.traceId} className="border-b border-ink/5 last:border-0">
                      <td className="px-3 py-2 font-mono text-[11px]">{t.traceId.slice(0, 16)}…</td>
                      <td className="px-3 py-2 font-mono text-xs">{t.agentVersion ?? "—"}</td>
                      <td className="px-3 py-2 text-xs">{t.outcome ?? "—"}</td>
                      <td className="px-3 py-2">{t.verified === undefined ? "—" : t.verified ? "✓" : "✗"}</td>
                      <td className="px-3 py-2"><SourceBadge source={data.source} /></td>
                      <td className="px-3 py-2">
                        {t.deepLink ? (
                          <a className="text-accent underline" href={t.deepLink} target="_blank" rel="noreferrer">
                            open in AgentTwin →
                          </a>
                        ) : (
                          <span className="text-muted text-xs">no link</span>
                        )}
                      </td>
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
