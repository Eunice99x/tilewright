import type { Metadata } from "next";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { signupAction } from "@/app/auth/actions";
import { AuthForm } from "@/components/auth-form";
import { isInvite, safeNext } from "@/lib/next";
import { enabledProviders } from "@/lib/oauth";
import { APP } from "@/lib/plans";
import { SESSION_COOKIE } from "@/lib/session";

export const metadata: Metadata = {
  title: "Create your account",
  description: `Create your ${APP.name} account.`,
  alternates: { canonical: "/signup" },
};

export default async function Signup({ searchParams }: { searchParams: Promise<{ next?: string }> }) {
  const after = safeNext((await searchParams).next);
  if ((await cookies()).get(SESSION_COOKIE)) redirect(after ?? "/dashboard");
  return (
    <div className="frame auth-wrap">
      <AuthForm mode="signup" action={signupAction} providers={enabledProviders()} next={after} notice={isInvite(after) ? "Create an account to join the game." : undefined} />
    </div>
  );
}
