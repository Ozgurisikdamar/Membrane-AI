import type { ReactNode } from "react";
import type { Provenance } from "@/lib/datasource/types";
import { SourceBadge } from "./source-badge";

export function Card({ children, className = "" }: { children: ReactNode; className?: string }) {
  return (
    <div className={`rounded-xl border border-ink/10 bg-white/70 p-4 shadow-sm ${className}`}>{children}</div>
  );
}

export function CardHeader({ title, source, note }: { title: string; source?: Provenance; note?: string }) {
  return (
    <div className="mb-2 flex items-center justify-between gap-2">
      <h3 className="text-sm font-semibold text-ink">{title}</h3>
      {source ? <SourceBadge source={source} title={note} /> : null}
    </div>
  );
}

export function Stat({ label, value, tone }: { label: string; value: ReactNode; tone?: "pass" | "warn" | "block" | "default" }) {
  const color =
    tone === "pass" ? "text-pass" : tone === "warn" ? "text-warn" : tone === "block" ? "text-block" : "text-ink";
  return (
    <div className="flex flex-col gap-0.5">
      <span className={`font-mono text-2xl font-semibold ${color}`}>{value}</span>
      <span className="text-xs text-muted">{label}</span>
    </div>
  );
}
