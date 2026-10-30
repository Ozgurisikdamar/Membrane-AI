import { bff } from "@/lib/server/respond";
import { liveSimulations } from "@/lib/server/agenttwin";

export const dynamic = "force-dynamic";

export async function GET() {
  return bff(() => liveSimulations());
}
