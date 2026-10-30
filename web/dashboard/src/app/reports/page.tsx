"use client";

import { useDataSource } from "@/lib/datasource/context";
import { useSourced } from "@/lib/use-sourced";
import { PageHeader } from "@/components/ui/page-header";
import { SourcedView } from "@/components/ui/sourced-view";
import { Card, CardHeader } from "@/components/ui/card";
import { Empty } from "@/components/ui/states";
import type { ReleaseView } from "@/lib/datasource/types";

function download(name: string, text: string, type: string) {
  const blob = new Blob([text], { type });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = name;
  a.click();
  URL.revokeObjectURL(url);
}

function toMarkdown(r: ReleaseView): string {
  const lines = [
    `# Release evidence — ${r.correlation.agentId} ${r.correlation.baselineVersion} → ${r.correlation.candidateVersion}`,
    "",
    `- Gate decision: **${r.gate.decision}**`,
    `- Triggered rules: ${r.gate.triggeredRules.join(", ") || "none"}`,
    `- Code verdict: ${r.codeVerdict} (${r.codeFindings.blocking} blocking)`,
    `- Evaluation run: ${r.correlation.evaluationRunId}`,
    `- Commit: ${r.correlation.commitSha}`,
    `- Seed: ${r.evalRun.seed}`,
    "",
    "## Reasons",
    ...r.gate.reasons.map((x) => `- ${x}`),
    "",
    "## Scenarios",
    "| Scenario | Severity | Classification |",
    "| --- | --- | --- |",
    ...r.evalRun.perScenario.map((s) => `| ${s.scenario} | ${s.severity} | ${s.status} |`),
  ];
  return lines.join("\n");
}

export default function ReportsPage() {
  const { dataSource, mode } = useDataSource();
  const runs = useSourced(() => dataSource.evalRuns(), [mode]);

  return (
    <div>
      <PageHeader
        title="Reports"
        subtitle="Export the combined release-evidence record as JSON or Markdown — the same contract reviewers audit (pkg/assurance ReleaseEvidenceV1)."
      />
      <SourcedView load={runs} loadingLabel="Loading…">
        {(data) =>
          data.value.length === 0 ? (
            <Empty title="Nothing to export" hint="No evaluations available from this source." />
          ) : (
            <div className="flex flex-col gap-3">
              {data.value.map((run) => (
                <ReportRow key={run.evalRunId} id={run.evalRunId} label={`${run.agentId} ${run.baselineVersion} → ${run.candidateVersion} (${run.gate})`} />
              ))}
            </div>
          )
        }
      </SourcedView>
    </div>
  );
}

function ReportRow({ id, label }: { id: string; label: string }) {
  const { dataSource, mode } = useDataSource();
  const rel = useSourced(() => dataSource.release(id), [mode, id]);
  return (
    <Card>
      <CardHeader title={label} source={rel.status === "ready" ? rel.data.source : undefined} />
      {rel.status === "ready" && rel.data.value ? (
        <div className="flex gap-2">
          <button
            type="button"
            className="rounded-md bg-accent px-3 py-1.5 text-sm font-semibold text-white"
            onClick={() => download(`release-${id}.json`, JSON.stringify(rel.data.value, null, 2), "application/json")}
          >
            Download JSON
          </button>
          <button
            type="button"
            className="rounded-md border border-ink/20 px-3 py-1.5 text-sm font-semibold text-ink"
            onClick={() => download(`release-${id}.md`, toMarkdown(rel.data.value!), "text/markdown")}
          >
            Download Markdown
          </button>
        </div>
      ) : (
        <p className="text-sm text-muted">Evidence unavailable from this source.</p>
      )}
    </Card>
  );
}
