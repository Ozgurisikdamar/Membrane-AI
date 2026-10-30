"use client";

import { useDataSource } from "@/lib/datasource/context";
import { useSourced } from "@/lib/use-sourced";
import { PageHeader } from "@/components/ui/page-header";
import { SourcedView } from "@/components/ui/sourced-view";
import { SourceBadge } from "@/components/ui/source-badge";
import { Card, CardHeader } from "@/components/ui/card";
import { Empty } from "@/components/ui/states";

export default function AgentsPage() {
  const { dataSource, mode } = useDataSource();
  const agents = useSourced(() => dataSource.agents(), [mode]);
  return (
    <div>
      <PageHeader title="Agent Twin Lab" subtitle="Registered agents and their immutable versions (manifest / prompt / model references)." />
      <SourcedView load={agents} loadingLabel="Loading agents…">
        {(data) =>
          data.value.length === 0 ? (
            <Empty title="No agents" hint="No registered agents from this source." />
          ) : (
            <div className="flex flex-col gap-4">
              {data.value.map((a) => (
                <Card key={a.agentId}>
                  <CardHeader title={a.agentId} source={data.source} note={data.note} />
                  <div className="overflow-x-auto">
                    <table className="w-full text-left text-sm">
                      <caption className="sr-only">Versions of {a.agentId}</caption>
                      <thead className="border-b border-ink/10 text-xs uppercase tracking-wide text-muted">
                        <tr>
                          <th className="px-3 py-2">Version</th>
                          <th className="px-3 py-2">Model</th>
                          <th className="px-3 py-2">Manifest</th>
                          <th className="px-3 py-2">Prompt</th>
                          <th className="px-3 py-2">Created</th>
                        </tr>
                      </thead>
                      <tbody>
                        {a.versions.map((v) => (
                          <tr key={v.version} className="border-b border-ink/5 last:border-0">
                            <td className="px-3 py-2 font-mono text-xs">{v.version}</td>
                            <td className="px-3 py-2 font-mono text-xs">{v.model}</td>
                            <td className="px-3 py-2 font-mono text-[11px]">{v.manifestSha ?? "—"}</td>
                            <td className="px-3 py-2 font-mono text-[11px]">{v.promptSha ?? "—"}</td>
                            <td className="px-3 py-2 text-xs">{v.createdAt ?? "—"}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </Card>
              ))}
            </div>
          )
        }
      </SourcedView>
    </div>
  );
}
