"use client";

import { useActionState } from "react";
import { type PlayState, playNowAction } from "@/app/play/actions";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

export function PlayForm({ signedIn, code }: { signedIn: boolean; code?: string }) {
  const [state, action, pending] = useActionState<PlayState, FormData>(playNowAction, {});
  const askName = !signedIn || state.needName;
  return (
    <form action={action} className="form">
      {code && <input type="hidden" name="code" value={code} />}
      {askName && (
        <label>
          Your name
          <Input name="name" required maxLength={24} autoComplete="nickname" defaultValue={state.name} autoFocus />
          <span className="hint">The other player sees it. No email or password needed.</span>
        </label>
      )}
      {code ? (
        <Button name="intent" value="join" disabled={pending}>{pending ? "Joining…" : "Join the game"}</Button>
      ) : (
        <>
          <Button name="intent" value="versus" disabled={pending}>{pending ? "Starting…" : "Play a friend"}</Button>
          <Button name="intent" value="solo" variant="ghost" disabled={pending}>Practice solo</Button>
        </>
      )}
      {state.error && <p className="err" role="alert">{state.error}</p>}
    </form>
  );
}
