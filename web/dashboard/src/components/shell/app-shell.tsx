"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { NAV, type NavItem } from "./nav";
import { useDataSource } from "@/lib/datasource/context";

function ShieldLogo() {
  // The Membrane shield-check mark (inline SVG, reused from the landing page).
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path d="M12 2l8 3v6c0 5-3.4 8.5-8 11-4.6-2.5-8-6-8-11V5l8-3z" fill="#0f766e" />
      <path d="M8.5 12.2l2.4 2.4 4.6-4.9" stroke="#f6f3ec" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

function ModeToggle() {
  const { mode, setMode } = useDataSource();
  return (
    <div className="flex items-center gap-1 rounded-lg border border-ink/15 bg-white/70 p-0.5 text-xs font-semibold">
      {(["demo", "live"] as const).map((m) => (
        <button
          key={m}
          type="button"
          onClick={() => setMode(m)}
          aria-pressed={mode === m}
          className={`rounded-md px-2 py-1 ${mode === m ? "bg-accent text-white" : "text-muted hover:text-ink"}`}
        >
          {m === "demo" ? "Demo" : "Live"}
        </button>
      ))}
    </div>
  );
}

function groupItems(items: NavItem[]) {
  const groups: Record<string, NavItem[]> = {};
  for (const it of items) (groups[it.group] ??= []).push(it);
  return groups;
}

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const { mode } = useDataSource();
  const groups = groupItems(NAV);

  return (
    <div className="flex min-h-screen bg-paper text-ink">
      <aside className="hidden w-60 shrink-0 flex-col border-r border-ink/10 bg-white/50 p-4 md:flex">
        <Link href="/" className="mb-6 flex items-center gap-2">
          <ShieldLogo />
          <span className="font-semibold leading-tight">
            MEMBRANE<span className="text-accent">.AI</span>
            <span className="block text-[10px] font-normal text-muted">Governance &amp; Agent Assurance</span>
          </span>
        </Link>
        <nav className="flex flex-col gap-4" aria-label="Primary">
          {Object.entries(groups).map(([group, items]) => (
            <div key={group}>
              <p className="mb-1 px-2 text-[10px] font-semibold uppercase tracking-wider text-muted">{group}</p>
              <ul className="flex flex-col gap-0.5">
                {items.map((it) => {
                  const active = pathname === it.href;
                  return (
                    <li key={it.href}>
                      <Link
                        href={it.href}
                        aria-current={active ? "page" : undefined}
                        className={`block rounded-md px-2 py-1.5 text-sm ${active ? "bg-accent/10 font-semibold text-accent" : "text-ink/80 hover:bg-ink/5"}`}
                      >
                        {it.label}
                      </Link>
                    </li>
                  );
                })}
              </ul>
            </div>
          ))}
        </nav>
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex items-center justify-between gap-3 border-b border-ink/10 bg-white/60 px-4 py-3">
          <div className="flex items-center gap-2 md:hidden">
            <ShieldLogo />
            <span className="font-semibold">MEMBRANE<span className="text-accent">.AI</span></span>
          </div>
          <div className="ml-auto flex items-center gap-3">
            <ModeToggle />
          </div>
        </header>

        {mode === "demo" ? (
          <div
            role="note"
            data-testid="demo-banner"
            className="border-b border-demo/30 bg-demo/10 px-4 py-1.5 text-center text-xs font-semibold text-demo"
          >
            DEMO DATA — synthetic / deterministic-fake planner / not production. Switch to Live to query a configured backend.
          </div>
        ) : null}

        <main className="min-w-0 flex-1 p-4 md:p-6">{children}</main>
      </div>
    </div>
  );
}
