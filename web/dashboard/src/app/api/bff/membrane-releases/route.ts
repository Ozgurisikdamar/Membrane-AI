import { bff } from "@/lib/server/respond";
import { membraneReleases } from "@/lib/server/membrane";

export const dynamic = "force-dynamic";

// Lists persisted combined release-evidence records from the Membrane evidence
// read API (pkg/cmd/evidence-api). 503 when not configured — the browser then
// shows an honest "not connected" state.
export async function GET() {
  return bff(() => membraneReleases());
}
