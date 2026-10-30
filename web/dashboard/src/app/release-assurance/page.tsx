"use client";

import Link from "next/link";
import { useDataSource } from "@/lib/datasource/context";
import { useSourced } from "@/lib/use-sourced";
import { PageHeader } from "@/components/ui/page-header";
import { SourcedView } from "@/components/ui/sourced-view";
import { SourceBadge } from "@/components/ui/source-badge";
import { Empty } from "@/components/ui/states";
import type { GateDecision } from "@/lib/datasource/types";

const GATE_CLASS: Record<GateDecision, string> = {
  PASS: "text-pass",
  WARN: "text-warn",
  BLOCK: "text-block",
  INCOMPLETE: "text-muted",
  ERROR: "text-muted",
};

export default function ReleaseAssurancePage() {
  const { dataSource, mode } = useDataSource();
  const releases = useSourced(() => dataSource.releases(), [mode]);

  return (
    <div>
      <PageHeader
        title="Release Assurance"
        subtitle="Combined PASS/WARN/BLOCK decisions over persisted code + agent evidence. The gate is the deterministic, fail-safe policy (research §5.3) — a critical failure can never be hidden by a score."
      />
      <SourcedView load={releases} loadingLabel="Loading release evidence…">
        {(data) =>
          data.value.length === 0 ? (
            <Empty
              title="No release evidence"
              hint={
                data.source === "unavailable"
                  ? "The Membrane evidence read API is not connected for this deployment. Persisted combined release records appear here once it is configured (MEMBRANE_EVIDENCE_URL)."
                  : "No persisted combined release-evidence records are available from this source yet."
              }
            />
          ) : (
            <div className="overflow-x-auto rounded-xl border border-ink/10 bg-white/70">
              <table className="w-full text-left text-sm">
                <caption className="sr-only">Combined release decisions from persisted evidence</caption>
                <thead className="border-b border-ink/10 text-xs uppercase tracking-wide text-muted">
                  <tr>
                    <th className="px-3 py-2">Agent</th>
                    <th className="px-3 py-2">Baseline → Candidate</th>
                    <th className="px-3 py-2">Code verdict</th>
                    <th className="px-3 py-2">New critical</th>
                    <th className="px-3 py-2">Gate</th>
                    <th className="px-3 py-2">Source</th>
                    <th className="px-3 py-2"></th>
                  </tr>
                </thead>
                <tbody>
                  {data.value.map((r) => (
                    <tr key={r.releaseId} className="border-b border-ink/5 last:border-0">
                      <td className="px-3 py-2 font-mono text-xs">{r.agentId}</td>
                      <td className="px-3 py-2 font-mono text-xs">{r.baselineVersion} → {r.candidateVersion}</td>
                      <td className={`px-3 py-2 font-semibold ${GATE_CLASS[r.codeVerdict]}`}>{r.codeVerdict}</td>
                      <td className="px-3 py-2">{r.newCriticalFailures}</td>
                      <td className={`px-3 py-2 font-semibold ${GATE_CLASS[r.gate]}`}>{r.gate}</td>
                      <td className="px-3 py-2"><SourceBadge source={data.source} /></td>
                      <td className="px-3 py-2">
                        <Link className="text-accent underline" href={`/release-assurance/${encodeURIComponent(r.releaseId)}`}>
                          Evidence →
                        </Link>
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
