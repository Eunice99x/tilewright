"use client";

import Link from "next/link";
import { useActionState } from "react";
import type { AuthState } from "@/app/auth/actions";
import { OAuthButtons } from "@/components/oauth-buttons";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { OAuthProvider } from "@/lib/oauth";
import { APP } from "@/lib/plans";

export function AuthForm({ mode, action, providers, oauthFailed, usePassword, otherIdentity, oauthRateLimited, notice, next }: {
  mode: "login" | "signup";
  action: (s: AuthState, fd: FormData) => Promise<AuthState>;
  providers: OAuthProvider[];
  oauthFailed?: boolean;
  usePassword?: boolean;
  otherIdentity?: boolean;
  oauthRateLimited?: boolean;
  notice?: string;
  next?: string;
}) {
  const [state, formAction, pending] = useActionState(action, {});
  const signup = mode === "signup";
  return (
    <div className="auth card">
      <h1>{signup ? "Create your account" : "Log in"}</h1>
      <p className="muted">{signup ? "No card needed." : "Welcome back."}</p>
      {notice && <p className="notice" role="status">{notice}</p>}
      <OAuthButtons providers={providers} />
      <form className="form" action={formAction}>
        {next && <input type="hidden" name="next" value={next} />}
        <label>
          Email
          <Input type="email" name="email" required autoComplete="email" defaultValue={state.email} />
        </label>
        <label>
          Password
          <Input type="password" name="password" required minLength={signup ? 8 : undefined} autoComplete={signup ? "new-password" : "current-password"} />
          {signup && <span className="hint">At least 8 characters.</span>}
        </label>
        {(state.error || oauthFailed || usePassword || otherIdentity || oauthRateLimited) && (
          <p className="err" role="alert">
            {state.error || (usePassword ? <>This email has a password. Log in with it, or <Link href="/forgot">reset it</Link>.</> : (otherIdentity ? "This email signed up with a different Google or GitHub account. Use that one. Can't? Email " + APP.support_email : oauthRateLimited ? "Too many attempts. Wait a while and try again." : "That sign-in didn't work. Try again, and check that your Google or GitHub email is verified."))}
          </p>
        )}
        <Button disabled={pending}>{pending ? "One moment…" : signup ? "Create account" : "Log in"}</Button>
        {!signup && <Link className="forgot" href="/forgot">Forgot your password?</Link>}
      </form>
      <p className="alt">
        {signup ? <>Have an account? <Link href={next ? `/login?next=${encodeURIComponent(next)}` : "/login"}>Log in</Link></> : <>New here? <Link href={next ? `/signup?next=${encodeURIComponent(next)}` : "/signup"}>Create an account</Link></>}
      </p>
    </div>
  );
}
