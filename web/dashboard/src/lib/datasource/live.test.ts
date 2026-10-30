import { describe, it, expect, vi, afterEach } from "vitest";
import { LiveDataSource } from "./live";

// LiveDataSource must never fabricate data: an unreachable backend, or a surface
// Membrane does not expose, resolves to an `unavailable` Sourced value — never a
// thrown error and never a made-up number (honesty guardrails, doc §1, §3.1).

afterEach(() => vi.restoreAllMocks());

describe("LiveDataSource honesty", () => {
  const ds = new LiveDataSource();

  it("reports code governance as unavailable (no Membrane read API, D0)", async () => {
    const r = await ds.codeGovernance();
    expect(r.source).toBe("unavailable");
    expect(r.value).toBeNull();
    expect(r.note).toMatch(/no read API/i);
  });

  it("reports policy decision history as unavailable", async () => {
    const r = await ds.policyDecisions();
    expect(r.source).toBe("unavailable");
    expect(r.value).toEqual([]);
  });

  it("maps a failed BFF fetch to unavailable, not an exception", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: false, status: 503, statusText: "Service Unavailable" }));
    const r = await ds.evalRuns();
    expect(r.source).toBe("unavailable");
    expect(r.value).toEqual([]);
    expect(r.note).toMatch(/not reachable/i);
  });

  it("returns live-sourced data when the BFF responds OK", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: true, json: async () => [] }));
    const r = await ds.evalRuns();
    expect(r.source).toBe("live");
  });
});
