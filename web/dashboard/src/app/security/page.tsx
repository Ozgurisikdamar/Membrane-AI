"use client";

import { useDataSource } from "@/lib/datasource/context";
import { useSourced } from "@/lib/use-sourced";
import { PageHeader } from "@/components/ui/page-header";
import { SourcedView } from "@/components/ui/sourced-view";
import { SourceBadge } from "@/components/ui/source-badge";
import { Empty } from "@/components/ui/states";

const DECISION_CLASS: Record<string, string> = {
  allow: "text-pass",
  deny: "text-block font-semibold",
  review: "text-warn font-semibold",
};

export default function SecurityPage() {
  const { dataSource, mode } = useDataSource();
  const decisions = useSourced(() => dataSource.policyDecisions(), [mode]);
  return (
    <div>
      <PageHeader
        title="Security Center"
        subtitle="MCP/tool-call policy, package-install screening (typosquat), and sanctioned/shadow-AI classification from the Membrane gateway engine."
      />
      <SourcedView load={decisions} loadingLabel="Loading policy decisions…">
        {(data) =>
          data.value.length === 0 ? (
            <Empty title="No decisions" hint="No policy decisions from this source." />
          ) : (
            <div className="overflow-x-auto rounded-xl border border-ink/10 bg-white/70">
              <table className="w-full text-left text-sm">
                <caption className="sr-only">Gateway policy decisions</caption>
                <thead className="border-b border-ink/10 text-xs uppercase tracking-wide text-muted">
                  <tr>
                    <th className="px-3 py-2">Kind</th>
                    <th className="px-3 py-2">Subject</th>
                    <th className="px-3 py-2">Decision</th>
                    <th className="px-3 py-2">Reasons</th>
                    <th className="px-3 py-2">Source</th>
                  </tr>
                </thead>
                <tbody>
                  {data.value.map((d, i) => (
                    <tr key={i} className="border-b border-ink/5 last:border-0">
                      <td className="px-3 py-2 font-mono text-xs">{d.kind}</td>
                      <td className="px-3 py-2 font-mono text-xs">{d.subject}</td>
                      <td className={`px-3 py-2 ${DECISION_CLASS[d.decision] ?? ""}`}>{d.decision}</td>
                      <td className="px-3 py-2 text-xs text-muted">{d.reasons.join("; ")}</td>
                      <td className="px-3 py-2"><SourceBadge source={data.source} /></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )
        }
      </SourcedView>
      <p className="mt-3 text-[11px] text-muted">
        Scope: deny-list + dangerous-argument patterns (tool-call), typosquat distance vs popular-package baselines (package),
        sanctioned-registry check (shadow-AI). Typosquat intelligence is limited to the shipped baselines — not a full registry feed.
      </p>
    </div>
  );
}
