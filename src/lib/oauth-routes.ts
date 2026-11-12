import { randomBytes } from "node:crypto";
import { cookies } from "next/headers";
import { type NextRequest, NextResponse } from "next/server";
import { ApiError, api } from "@/lib/api";
import { type OAuthProvider, STATE_MAX_AGE, authorizeUrl, clientId, redirectUri, sameState, siteUrl, stateCookie, statePath } from "@/lib/oauth";
import { setSession, visitorIp } from "@/lib/session";

function toLoginWithError(code = "oauth"): NextResponse {
  return NextResponse.redirect(new URL(`/login?error=${code}`, siteUrl()));
}

export function startOAuth(provider: OAuthProvider): NextResponse {
  const id = clientId(provider);
  if (!id) return toLoginWithError();
  const state = randomBytes(32).toString("base64url");
  const res = NextResponse.redirect(authorizeUrl(provider, id, state));
  res.cookies.set(stateCookie(provider), state, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: statePath(provider),
    maxAge: STATE_MAX_AGE,
  });
  return res;
}

export async function finishOAuth(provider: OAuthProvider, req: NextRequest): Promise<NextResponse> {
  const jar = await cookies();
  const saved = jar.get(stateCookie(provider))?.value;
  jar.delete({ name: stateCookie(provider), path: statePath(provider) });

  const sp = req.nextUrl.searchParams;
  if (!sameState(sp.get("state"), saved)) return toLoginWithError();
  const code = sp.get("code");
  if (sp.get("error") || !code) return toLoginWithError();

  try {
    const { token } = await api.oauth(provider, code, redirectUri(provider), await visitorIp());
    await setSession(token);
  } catch (e) {
    if (e instanceof ApiError && e.code === "use_password_login") return toLoginWithError("use_password");
    if (e instanceof ApiError && e.code === "oauth_other_identity") return toLoginWithError("other_identity");
    if (e instanceof ApiError && e.code === "too_many_attempts") return toLoginWithError("too_many");
    console.error(`oauth ${provider}:`, e instanceof Error ? e.message : "failed");
    return toLoginWithError();
  }
  return NextResponse.redirect(new URL("/dashboard", siteUrl()));
}
