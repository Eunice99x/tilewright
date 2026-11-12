import type { NextRequest } from "next/server";
import { finishOAuth } from "@/lib/oauth-routes";

export function GET(req: NextRequest) {
  return finishOAuth("google", req);
}
