import type { Metadata } from "next";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { loginAction } from "@/app/auth/actions";
import { AuthForm } from "@/components/auth-form";
import { isInvite, safeNext } from "@/lib/next";
import { enabledProviders } from "@/lib/oauth";
import { SESSION_COOKIE } from "@/lib/session";

export const metadata: Metadata = { title: "Log in", robots: { index: false }, referrer: "no-referrer" };

export default async function Login({ searchParams }: { searchParams: Promise<{ expired?: string; error?: string; reset?: string; next?: string }> }) {
  const { expired, error, reset, next } = await searchParams;
  const after = safeNext(next);
  if (!expired && !error && !reset && (await cookies()).get(SESSION_COOKIE)) redirect(after ?? "/dashboard");
  return (
    <div className="frame auth-wrap">
      <AuthForm mode="login" action={loginAction} providers={enabledProviders()} oauthFailed={error === "oauth"} usePassword={error === "use_password"} otherIdentity={error === "other_identity"} oauthRateLimited={error === "too_many"} notice={reset ? "Password changed. Log in with the new one." : isInvite(after) ? "Log in to join the game." : after ? "Log in to confirm your email." : undefined} next={after} />
    </div>
  );
}
