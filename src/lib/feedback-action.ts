"use server";

import { ApiError, api } from "@/lib/api";
import { APP } from "@/lib/plans";
import { visitorIp } from "@/lib/session";

export interface FeedbackState {
  sent?: boolean;
  error?: string;
}

/** The feedback box's send. The session goes along when there is one, so the API can attach the account. */
export async function sendFeedbackAction(_prev: FeedbackState, fd: FormData): Promise<FeedbackState> {
  const message = String(fd.get("message") ?? "").trim();
  if (message.length < 3) return { error: "Write a few words first." };
  try {
    await api.sendFeedback(message.slice(0, 2000), String(fd.get("email") ?? "").trim().slice(0, 254), String(fd.get("page") ?? "").slice(0, 200), await visitorIp());
    return { sent: true };
  } catch (e) {
    if (e instanceof ApiError && e.status === 429) return { error: "That's a lot of messages from here. Try again in an hour." };
    if (e instanceof ApiError && e.status === 400) return { error: "That didn't go through. Check your message and email." };
    return { error: `Couldn't send it. Try again, or email ${APP.support_email}.` };
  }
}
