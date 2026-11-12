"use client";

import Link from "next/link";
import { useActionState } from "react";
import { forgotAction, resetAction, verifyAction } from "@/app/auth/email-actions";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

export function ForgotForm() {
  const [state, action, pending] = useActionState(forgotAction, {});
  if (state.done) {
    return (
      <p className="notice" role="status">If that address has an account, a link is on its way. Check your spam folder too.</p>
    );
  }
  return (
    <form className="form" action={action}>
      <label>
        Email
        <Input name="email" type="email" autoComplete="email" required />
      </label>
      {state.error && <p className="err" role="alert">{state.error}</p>}
      <Button disabled={pending}>{pending ? "One moment…" : "Send reset link"}</Button>
    </form>
  );
}

export function ResetForm({ token }: { token: string }) {
  const [state, action, pending] = useActionState(resetAction, {});
  return (
    <form className="form" action={action}>
      <input type="hidden" name="token" value={token} />
      <label>
        New password
        <Input name="password" type="password" autoComplete="new-password" minLength={8} maxLength={72} required />
        <span className="hint">At least 8 characters.</span>
      </label>
      {state.error && <p className="err" role="alert">{state.error} <Link href="/forgot">Get a new link</Link></p>}
      <Button disabled={pending}>{pending ? "One moment…" : "Change password"}</Button>
    </form>
  );
}

export function VerifyForm({ token }: { token: string }) {
  const [state, action, pending] = useActionState(verifyAction, {});
  return (
    <form className="form" action={action}>
      <input type="hidden" name="token" value={token} />
      {state.error && <p className="err" role="alert">{state.error} <Link href="/dashboard">Log in to send a new one</Link></p>}
      <Button disabled={pending}>{pending ? "One moment…" : "Confirm my email"}</Button>
    </form>
  );
}
