import { cookies, headers } from "next/headers";

export const SESSION_COOKIE = "session";
export const HINT_COOKIE = "signed_in";
const MAX_AGE = 60 * 60 * 24 * 30;

export async function getSessionToken(): Promise<string | undefined> {
  return (await cookies()).get(SESSION_COOKIE)?.value || undefined;
}

export async function setSession(token: string): Promise<void> {
  const jar = await cookies();
  const base = { sameSite: "lax" as const, secure: process.env.NODE_ENV === "production", path: "/", maxAge: MAX_AGE };
  jar.set(SESSION_COOKIE, token, { ...base, httpOnly: true });
  jar.set(HINT_COOKIE, "1", { ...base, httpOnly: false });
}

export async function clearSession(): Promise<void> {
  const jar = await cookies();
  jar.delete(SESSION_COOKIE);
  jar.delete(HINT_COOKIE);
}

export async function visitorIp(): Promise<string | undefined> {
  const h = await headers();
  return h.get("x-forwarded-for")?.split(",")[0]?.trim() || h.get("x-real-ip")?.trim() || undefined;
}
