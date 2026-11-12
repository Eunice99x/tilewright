import { timingSafeEqual } from "node:crypto";

export type OAuthProvider = "google" | "github";

const PROVIDERS: OAuthProvider[] = ["github", "google"];

const CLIENT_ID_ENV: Record<OAuthProvider, string> = { google: "GOOGLE_CLIENT_ID", github: "GITHUB_CLIENT_ID" };

const AUTHORIZE: Record<OAuthProvider, { url: string; params: Record<string, string> }> = {
  github: { url: "https://github.com/login/oauth/authorize", params: { scope: "user:email" } },
  google: {
    url: "https://accounts.google.com/o/oauth2/v2/auth",
    params: { scope: "openid email", response_type: "code", prompt: "select_account" },
  },
};

export const STATE_MAX_AGE = 10 * 60;

export function stateCookie(provider: OAuthProvider): string {
  return `oauth_${provider}`;
}

export function statePath(provider: OAuthProvider): string {
  return `/auth/${provider}`;
}

export function clientId(provider: OAuthProvider): string | undefined {
  return process.env[CLIENT_ID_ENV[provider]] || undefined;
}

export function enabledProviders(): OAuthProvider[] {
  return PROVIDERS.filter((p) => clientId(p));
}

export function siteUrl(): string {
  return (process.env.SITE_URL || "http://localhost:3100").replace(/\/+$/, "");
}

export function redirectUri(provider: OAuthProvider): string {
  return `${siteUrl()}/auth/${provider}/callback`;
}

export function authorizeUrl(provider: OAuthProvider, id: string, state: string): URL {
  const { url, params } = AUTHORIZE[provider];
  const out = new URL(url);
  for (const [k, v] of Object.entries({ ...params, client_id: id, redirect_uri: redirectUri(provider), state })) {
    out.searchParams.set(k, v);
  }
  return out;
}

export function sameState(got: string | null | undefined, saved: string | null | undefined): boolean {
  if (!got || !saved) return false;
  const x = Buffer.from(got);
  const y = Buffer.from(saved);
  return x.length === y.length && timingSafeEqual(x, y);
}
