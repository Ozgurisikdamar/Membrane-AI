// Shared data-state views. Every data-backed panel must render one of these
// when it is not showing real content, so missing/partial/stale data is always
// visible and never silently blank (DASHBOARD_AND_NETLIFY_WORKFLOW.md §4.6).

function Frame({ tone, title, children, testid }: {
  tone: "neutral" | "warn" | "error";
  title: string;
  children?: React.ReactNode;
  testid: string;
}) {
  const border =
    tone === "error" ? "border-block/40" : tone === "warn" ? "border-warn/40" : "border-unavailable/40";
  return (
    <div
      role={tone === "error" ? "alert" : "status"}
      data-testid={testid}
      className={`flex flex-col items-start gap-1 rounded-lg border ${border} bg-white/60 p-4 text-sm`}
    >
      <span className="font-semibold text-ink">{title}</span>
      {children ? <span className="text-muted">{children}</span> : null}
    </div>
  );
}

export function Loading({ label = "Loading…" }: { label?: string }) {
  return (
    <div role="status" aria-busy="true" data-testid="state-loading" className="animate-pulse rounded-lg border border-unavailable/30 bg-white/50 p-4 text-sm text-muted">
      {label}
    </div>
  );
}

export function Empty({ title = "Nothing yet", hint }: { title?: string; hint?: string }) {
  return <Frame tone="neutral" title={title} testid="state-empty">{hint}</Frame>;
}

export function ErrorState({ title = "Something went wrong", detail }: { title?: string; detail?: string }) {
  return <Frame tone="error" title={title} testid="state-error">{detail}</Frame>;
}

export function Stale({ asOf }: { asOf?: string }) {
  return <Frame tone="warn" title="Stale data" testid="state-stale">{asOf ? `As of ${asOf}` : undefined}</Frame>;
}

export function Disconnected({ detail }: { detail?: string }) {
  return (
    <Frame tone="warn" title="Backend not connected" testid="state-disconnected">
      {detail ?? "Showing no live data. Switch to Demo mode or configure a backend in Settings."}
    </Frame>
  );
}

export function PermissionDenied({ detail }: { detail?: string }) {
  return <Frame tone="error" title="Permission denied" testid="state-permission">{detail ?? "You are not authorized to view this resource."}</Frame>;
}

export function Planned({ phase, detail }: { phase: string; detail?: string }) {
  return (
    <Frame tone="neutral" title={`Planned — ${phase}`} testid="state-planned">
      {detail ?? "This capability is scoped but not implemented yet. It is intentionally shown as planned, not functional."}
    </Frame>
  );
}
