"use server";

import { redirect } from "next/navigation";
import { ApiError, api } from "@/lib/api";
import { APP } from "@/lib/plans";

function expiredIf401(e: unknown): void {
  if (e instanceof ApiError && e.status === 401) redirect("/auth/expired");
}

export interface ResendState {
  sent?: boolean;
  error?: string;
}

export async function resendVerificationAction(): Promise<ResendState> {
  try {
    await api.resendVerification();
    return { sent: true };
  } catch (e) {
    expiredIf401(e);
    if (e instanceof ApiError && e.status === 429) return { error: "Too many emails. Wait a while and try again." };
    return { error: `Couldn't send it. Try again, or email ${APP.support_email}.` };
  }
}

export async function portalAction(fd: FormData): Promise<void> {
  const to = String(fd.get("to") ?? "");
  let target: string;
  try {
    const links = await api.billingPortal();
    target = to === "cancel" ? links.cancel : to === "update_payment" ? links.update_payment : links.overview;
    target ||= links.overview;
  } catch (e) {
    expiredIf401(e);
    redirect("/dashboard/billing?portal=failed");
  }
  redirect(target);
}
