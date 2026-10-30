import { bff } from "@/lib/server/respond";
import { liveOverview } from "@/lib/server/agenttwin";

export const dynamic = "force-dynamic";

export async function GET() {
  return bff(() => liveOverview());
}
