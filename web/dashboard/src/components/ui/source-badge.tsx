import type { Provenance } from "@/lib/datasource/types";

const LABEL: Record<Provenance, string> = {
  live: "LIVE",
  demo: "DEMO",
  unavailable: "N/A",
};

const CLASS: Record<Provenance, string> = {
  live: "bg-live/15 text-live border-live/40",
  demo: "bg-demo/15 text-demo border-demo/40",
  unavailable: "bg-unavailable/20 text-muted border-unavailable/50",
};

/** Every data panel renders this so provenance is never ambiguous. */
export function SourceBadge({ source, title }: { source: Provenance; title?: string }) {
  return (
    <span
      className={`inline-flex items-center rounded border px-1.5 py-0.5 font-mono text-[10px] font-semibold tracking-wide ${CLASS[source]}`}
      title={title}
      data-testid="source-badge"
      data-source={source}
    >
      {LABEL[source]}
    </span>
  );
}
