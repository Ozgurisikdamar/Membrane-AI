"use client";

import { useDataSource } from "@/lib/datasource/context";
import { useSourced } from "@/lib/use-sourced";
import { PageHeader } from "@/components/ui/page-header";
import { SourcedView } from "@/components/ui/sourced-view";
import { SourceBadge } from "@/components/ui/source-badge";
import { Card, CardHeader, Stat } from "@/components/ui/card";
import { Empty } from "@/components/ui/states";

function gradeTone(grade: string): "pass" | "warn" | "block" | "default" {
  if (grade === "A" || grade === "B") return "pass";
  if (grade === "C" || grade === "D") return "warn";
  if (grade === "F") return "block";
  return "default";
}

/** Highlights [MASKED:<rule>] placeholders so it's obvious secrets are redacted. */
function MaskedExcerpt({ text }: { text: string }) {
  const parts = text.split(/(\[MASKED:[^\]]+\])/g);
  return (
    <code className="font-mono text-[11px]">
      {parts.map((p, i) =>
        p.startsWith("[MASKED:") ? (
          <span key={i} className="rounded bg-block/15 px-1 text-block">{p}</span>
        ) : (
          <span key={i}>{p}</span>
        ),
      )}
    </code>
  );
}

export default function CodeGovernancePage() {
  const { dataSource, mode } = useDataSource();
  const report = useSourced(() => dataSource.codeGovernance(), [mode]);

  return (
    <div>
      <PageHeader
        title="Code Governance"
        subtitle="Deterministic secret & risky-pattern scan of AI-generated code. Secrets are masked before display and before any LLM stage."
      />
      <SourcedView load={report} loadingLabel="Loading scan…">
        {(data) => {
          const r = data.value;
          if (!r) return <Empty title="No scan" hint="No code-governance scan from this source." />;
          return (
            <div className="flex flex-col gap-4">
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                <Card>
                  <CardHeader title="Health grade" source={data.source} note={data.note} />
                  <div className="flex items-end gap-4">
                    <Stat label={`debt score ${r.debtScore}`} value={r.grade} tone={gradeTone(r.grade)} />
                  </div>
                  <p className="mt-2 text-[11px] text-muted">Severity-weighted findings per 100 files (blocking×5, warning×2, info×1). A=0, B&lt;5, C&lt;15, D&lt;40, F≥40.</p>
                </Card>
                <Card>
                  <CardHeader title="Findings" source={data.source} />
                  <div className="flex gap-6">
                    <Stat label="blocking" value={r.blocking} tone="block" />
                    <Stat label="warning" value={r.warnings} tone="warn" />
                    <Stat label="info" value={r.infos} />
                  </div>
                </Card>
                <Card>
                  <CardHeader title="Scope" source={data.source} />
                  <div className="flex gap-6">
                    <Stat label="files scanned" value={r.filesScanned} />
                    <Stat label="affected" value={r.affectedFiles} />
                  </div>
                  <p className="mt-2 font-mono text-[11px] text-muted">{r.root}</p>
                </Card>
              </div>

              <Card>
                <CardHeader title="Findings by rule" source={data.source} />
                <table className="w-full text-left text-sm">
                  <caption className="sr-only">Finding counts per rule</caption>
                  <thead className="border-b border-ink/10 text-xs uppercase tracking-wide text-muted">
                    <tr><th className="px-3 py-2">Rule</th><th className="px-3 py-2">Severity</th><th className="px-3 py-2">Count</th></tr>
                  </thead>
                  <tbody>
                    {r.byRule.map((ru) => (
                      <tr key={ru.rule} className="border-b border-ink/5 last:border-0">
                        <td className="px-3 py-2 font-mono text-xs">{ru.rule}</td>
                        <td className="px-3 py-2 text-block">{ru.severity}</td>
                        <td className="px-3 py-2">{ru.count}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </Card>

              <Card>
                <CardHeader title={`Detail (${r.findings.length}, secrets masked)`} source={data.source} />
                <div className="overflow-x-auto">
                  <table className="w-full text-left text-sm">
                    <caption className="sr-only">Individual findings with masked source excerpts</caption>
                    <thead className="border-b border-ink/10 text-xs uppercase tracking-wide text-muted">
                      <tr><th className="px-3 py-2">File</th><th className="px-3 py-2">Line</th><th className="px-3 py-2">Rule</th><th className="px-3 py-2">Excerpt</th></tr>
                    </thead>
                    <tbody>
                      {r.findings.map((f, i) => (
                        <tr key={i} className="border-b border-ink/5 last:border-0">
                          <td className="px-3 py-2 font-mono text-[11px]">{f.file}</td>
                          <td className="px-3 py-2">{f.line}</td>
                          <td className="px-3 py-2 font-mono text-[11px]">{f.rule}</td>
                          <td className="px-3 py-2"><MaskedExcerpt text={f.excerpt} /></td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
                <p className="mt-2 text-[11px] text-muted">
                  Detector scope: deterministic secret/risky-pattern scanning + masking (pkg/scan). Not full AST/data-flow; not a generation-time interception gateway.
                </p>
              </Card>
            </div>
          );
        }}
      </SourcedView>
    </div>
  );
}
