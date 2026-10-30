import "server-only";
import { checkDestination } from "./allowlist";
import { NotConfiguredError, AllowlistError } from "./agenttwin";
import { toSummary, toView, type MembraneEvidenceV1 } from "./membrane-map";
import type { ReleaseView, ReleaseSummary } from "../datasource/types";

// Server-only client for the Membrane evidence read API (pkg/cmd/evidence-api).
// It serves the REAL persisted combined release-evidence records
// (pkg/assurance.ReleaseEvidenceV1) — including the real code verdict — so the
// dashboard's Release Assurance can show genuine combined decisions instead of
// the AgentTwin-only view with a PASS placeholder (the D0 honesty gap).
//
// The credential and base URL live in server env vars and NEVER reach the
// browser. The browser calls same-origin BFF routes, which call this.

/** Raised when the evidence API responds 404 for a release id. */
export class MembraneNotFoundError extends Error {}

interface Config {
  baseUrl: string;
  apiKey: string;
}

/** True when the Membrane evidence API is configured for this deployment. */
export function membraneConfigured(): boolean {
  return !!process.env.MEMBRANE_EVIDENCE_URL && !!process.env.MEMBRANE_EVIDENCE_API_KEY;
}

function config(): Config {
  const baseUrl = process.env.MEMBRANE_EVIDENCE_URL;
  const apiKey = process.env.MEMBRANE_EVIDENCE_API_KEY;
  if (!baseUrl || !apiKey) {
    throw new NotConfiguredError("MEMBRANE_EVIDENCE_URL and MEMBRANE_EVIDENCE_API_KEY are not configured");
  }
  const allowLoopback = process.env.MEMBRANE_EVIDENCE_ALLOW_LOOPBACK === "1";
  const hosts = (process.env.MEMBRANE_EVIDENCE_ALLOWED_HOSTS ?? "").split(",").map((h) => h.trim()).filter(Boolean);
  const verdict = checkDestination(baseUrl, { allowLoopback, hosts });
  if (!verdict.ok) throw new AllowlistError(`MEMBRANE_EVIDENCE_URL rejected: ${verdict.reason}`);
  return { baseUrl: baseUrl.replace(/\/$/, ""), apiKey };
}

async function mem<T>(path: string): Promise<T> {
  const cfg = config();
  const ctrl = new AbortController();
  const timer = setTimeout(() => ctrl.abort(), 8000); // bounded deadline
  try {
    const res = await fetch(`${cfg.baseUrl}${path}`, {
      headers: { "X-Membrane-Api-Key": cfg.apiKey, Accept: "application/json" },
      signal: ctrl.signal,
      cache: "no-store",
    });
    if (res.status === 404) throw new MembraneNotFoundError(`Membrane evidence ${path} -> 404`);
    if (!res.ok) throw new Error(`Membrane evidence ${path} -> ${res.status}`);
    const len = Number(res.headers.get("content-length") ?? "0");
    if (len > 8_000_000) throw new Error("response too large"); // bounded size
    return (await res.json()) as T;
  } finally {
    clearTimeout(timer);
  }
}

/** List persisted combined release-evidence records, newest first. */
export async function membraneReleases(): Promise<ReleaseSummary[]> {
  const items = await mem<MembraneEvidenceV1[]>("/v1/releases");
  return (items ?? []).map(toSummary);
}

/** Read one persisted combined release-evidence record by release id. */
export async function membraneRelease(id: string): Promise<ReleaseView> {
  const ev = await mem<MembraneEvidenceV1>(`/v1/releases/${encodeURIComponent(id)}`);
  return toView(ev);
}
