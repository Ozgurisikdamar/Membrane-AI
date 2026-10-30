import { NotConfiguredError, AllowlistError } from "./agenttwin";

/** Run a BFF handler and map backend conditions to honest HTTP statuses. The
 *  browser-side LiveDataSource turns any non-200 into an `unavailable` state. */
export async function bff<T>(fn: () => Promise<T>): Promise<Response> {
  try {
    const data = await fn();
    return Response.json(data, { headers: { "Cache-Control": "no-store" } });
  } catch (e) {
    if (e instanceof NotConfiguredError) {
      return Response.json({ error: "backend_not_configured", message: e.message }, { status: 503 });
    }
    if (e instanceof AllowlistError) {
      return Response.json({ error: "destination_not_allowed", message: e.message }, { status: 500 });
    }
    const msg = e instanceof Error ? e.message : String(e);
    return Response.json({ error: "upstream_error", message: msg }, { status: 502 });
  }
}
