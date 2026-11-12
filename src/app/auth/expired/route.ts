import { redirect } from "next/navigation";
import { clearSession } from "@/lib/session";

export async function GET(req: Request) {
  const site = req.headers.get("sec-fetch-site");
  if (site !== "same-origin" && site !== "none") redirect("/login?expired=1");
  await clearSession();
  redirect("/login");
}
