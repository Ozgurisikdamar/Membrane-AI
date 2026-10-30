import { bff } from "@/lib/server/respond";
import { liveRelease } from "@/lib/server/agenttwin";

export const dynamic = "force-dynamic";

export async function GET(_req: Request, ctx: { params: Promise<{ id: string }> }) {
  const { id } = await ctx.params;
  return bff(() => liveRelease(id));
}
