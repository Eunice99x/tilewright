"use client";

import { useActionState } from "react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { sendFeedbackAction, type FeedbackState } from "@/lib/feedback-action";

/** Asks what's missing. Signed-in people aren't asked for an email: the API uses the account's. */
export function FeedbackBox({ page, signedIn }: { page: string; signedIn: boolean }) {
  const [state, action, pending] = useActionState<FeedbackState, FormData>(sendFeedbackAction, {});
  if (state.sent) {
    return (
      <Card aria-label="Feedback">
        <p className="notice" role="status">Thanks, got it. Every message is read.</p>
      </Card>
    );
  }
  return (
    <Card aria-label="Feedback">
      <form action={action} className="form">
        <input type="hidden" name="page" value={page} />
        <label>What would make this worth paying for? Anything broken or missing?
          <textarea className="input area" name="message" rows={3} required minLength={3} maxLength={2000} />
        </label>
        {!signedIn && (
          <label>Email
            <span className="hint">Optional, if you&rsquo;d like a reply.</span>
            <Input name="email" type="email" maxLength={254} autoComplete="email" />
          </label>
        )}
        <div><Button variant="ghost" disabled={pending}>{pending ? "Sending" : "Send"}</Button></div>
      </form>
      {state.error && <p className="err" role="alert">{state.error}</p>}
    </Card>
  );
}
