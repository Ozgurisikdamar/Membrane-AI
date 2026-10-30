import { bff } from "@/lib/server/respond";
import { liveAgents } from "@/lib/server/agenttwin";

export const dynamic = "force-dynamic";

export async function GET() {
  return bff(() => liveAgents());
}
