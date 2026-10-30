"use client";

import { useDataSource } from "@/lib/datasource/context";
import { PageHeader } from "@/components/ui/page-header";
import { Card, CardHeader } from "@/components/ui/card";
import { SourceBadge } from "@/components/ui/source-badge";

export default function SettingsPage() {
  const { mode, setMode } = useDataSource();
  return (
    <div>
      <PageHeader
        title="Settings / Integrations"
        subtitle="Backend connectivity and data-mode. Secret values are never displayed here or shipped to the browser."
      />

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader title="Data mode" />
          <p className="mb-3 text-sm text-muted">
            <strong className="text-ink">Demo</strong> uses deterministic synthetic fixtures (default, safe to
            deploy statically). <strong className="text-ink">Live</strong> queries a configured backend through
            this app&apos;s server-side BFF.
          </p>
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => setMode("demo")}
              aria-pressed={mode === "demo"}
              className={`rounded-md px-3 py-1.5 text-sm font-semibold ${mode === "demo" ? "bg-accent text-white" : "border border-ink/15 text-ink"}`}
            >
              Demo
            </button>
            <button
              type="button"
              onClick={() => setMode("live")}
              aria-pressed={mode === "live"}
              className={`rounded-md px-3 py-1.5 text-sm font-semibold ${mode === "live" ? "bg-accent text-white" : "border border-ink/15 text-ink"}`}
            >
              Live
            </button>
          </div>
        </Card>

        <Card>
          <CardHeader title="Backend connectivity" source={mode === "live" ? "live" : "demo"} />
          <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
            <dt className="text-muted">Mode</dt>
            <dd className="font-mono">{mode}</dd>
            <dt className="text-muted">BFF base</dt>
            <dd className="font-mono">/api/bff/* (same-origin)</dd>
            <dt className="text-muted">Backend credential</dt>
            <dd>held server-side only — never in the browser bundle <SourceBadge source="unavailable" title="by design" /></dd>
          </dl>
          <p className="mt-3 text-[11px] text-muted">
            Live mode requires a deployed Membrane/AgentTwin backend reachable from the BFF. Without one, Live
            views render an honest &quot;not connected&quot; state. Configure the backend base URL via server
            environment (name only: <code className="font-mono">AGENTTWIN_BASE_URL</code>,{" "}
            <code className="font-mono">AGENTTWIN_API_KEY</code>) — never as a <code className="font-mono">NEXT_PUBLIC_*</code> value.
          </p>
        </Card>
      </div>
    </div>
  );
}
