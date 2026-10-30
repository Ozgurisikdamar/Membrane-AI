import { bff } from "@/lib/server/respond";
import { liveRelease } from "@/lib/server/agenttwin";
import { membraneConfigured, membraneRelease, MembraneNotFoundError } from "@/lib/server/membrane";

export const dynamic = "force-dynamic";

// Release evidence by id. When the Membrane evidence read API is configured, the
// id is a Membrane release id and the response carries the REAL persisted
// combined decision (real code verdict). Otherwise it falls back to the
// AgentTwin-only view keyed by eval-run id (code verdict is a labelled
// placeholder — the D0 gap). A Membrane 404 returns null so the page shows an
// honest "not found" rather than a fabricated record.
export async function GET(_req: Request, ctx: { params: Promise<{ id: string }> }) {
  const { id } = await ctx.params;
  if (membraneConfigured()) {
    return bff(async () => {
      try {
        return await membraneRelease(id);
      } catch (e) {
        if (e instanceof MembraneNotFoundError) return null;
        throw e;
      }
    });
  }
  return bff(() => liveRelease(id));
}
