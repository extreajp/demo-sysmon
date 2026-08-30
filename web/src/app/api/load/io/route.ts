import { syncIO } from "@/lib/load";
import { denyIfDisabled, timed } from "@/lib/guard";

export const dynamic = "force-dynamic";

export async function GET(req: Request) {
  const denied = denyIfDisabled();
  if (denied) return denied;
  const mb = Math.min(64, Math.max(1, Number(new URL(req.url).searchParams.get("mb") || "32")));
  return timed(() => syncIO(mb));
}
