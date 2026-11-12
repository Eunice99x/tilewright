import { startOAuth } from "@/lib/oauth-routes";

export const dynamic = "force-dynamic";

export function GET() {
  return startOAuth("google");
}
