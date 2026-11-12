"use client";

import { useActionState } from "react";
import { Button } from "@/components/ui/button";
import { resendVerificationAction } from "./actions";

export function ConfirmEmail({ email }: { email: string }) {
  const [state, action, pending] = useActionState(resendVerificationAction, {});
  return (
    <section className="notice" aria-label="Confirm your email">
      <p><strong>Confirm your email address.</strong> We sent a link to {email}.</p>
      <form action={action} className="inline">
        <Button variant="ghost" disabled={pending || state.sent}>{pending ? "One moment…" : state.sent ? "Sent" : "Send it again"}</Button>
        {state.error && <span role="alert">{state.error}</span>}
      </form>
    </section>
  );
}
