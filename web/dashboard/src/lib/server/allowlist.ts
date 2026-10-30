// Server-only destination allowlist for the BFF (TS mirror of
// pkg/assurance/allowlist.go). The AgentTwin base URL is operator configuration
// read from the environment — never from the browser — and is checked here
// before any outbound call (SSRF defense, DASHBOARD_AND_NETLIFY_WORKFLOW.md §5).

export interface AllowlistConfig {
  allowLoopback: boolean;
  hosts: string[];
}

function isLoopback(host: string): boolean {
  return host === "localhost" || host === "127.0.0.1" || host === "::1" || host.startsWith("127.");
}

export function checkDestination(raw: string, cfg: AllowlistConfig): { ok: true } | { ok: false; reason: string } {
  let u: URL;
  try {
    u = new URL(raw);
  } catch {
    return { ok: false, reason: "not a valid URL" };
  }
  const scheme = u.protocol.replace(":", "").toLowerCase();
  if (scheme !== "http" && scheme !== "https") return { ok: false, reason: `scheme must be http/https, got ${scheme}` };
  if (u.username || u.password) return { ok: false, reason: "must not embed credentials" };
  const host = u.hostname.toLowerCase();
  if (!host) return { ok: false, reason: "no host" };

  if (isLoopback(host)) {
    return cfg.allowLoopback ? { ok: true } : { ok: false, reason: "loopback not allowed in this configuration" };
  }
  if (scheme !== "https") return { ok: false, reason: "non-loopback destination must use https" };

  const allowed = cfg.hosts.map((h) => h.toLowerCase());
  const hostPort = u.port ? `${host}:${u.port}` : host;
  if (allowed.includes(hostPort) || allowed.includes(host)) return { ok: true };
  return { ok: false, reason: `host not in allowlist: ${host}` };
}
