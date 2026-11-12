"use server";

import { redirect } from "next/navigation";
import { ApiError, api } from "@/lib/api";
import { clearSession, visitorIp } from "@/lib/session";

export interface FormState {
  error?: string;
  done?: boolean;
}

const INVALID_LINK = "This link has expired or was already used.";

export async function forgotAction(_prev: FormState, fd: FormData): Promise<FormState> {
  const email = String(fd.get("email") ?? "").trim().toLowerCase();
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/.test(email)) return { error: "Enter a valid email address." };
  try {
    await api.forgot(email, await visitorIp());
    return { done: true };
  } catch (e) {
    if (e instanceof ApiError && e.status === 429) return { error: "Too many attempts. Wait a while and try again." };
    if (e instanceof ApiError && e.code === "invalid_email") return { error: "Enter a valid email address." };
    return { error: "Something went wrong. Try again." };
  }
}

export async function resetAction(_prev: FormState, fd: FormData): Promise<FormState> {
  const password = String(fd.get("password") ?? "");
  const bytes = new TextEncoder().encode(password).length;
  if (password.length < 8 || bytes > 72) return { error: "Password must be 8 to 72 characters." };
  try {
    await api.resetPassword(String(fd.get("token") ?? ""), password, await visitorIp());
  } catch (e) {
    if (e instanceof ApiError && e.code === "invalid_token") return { error: INVALID_LINK };
    return { error: "Something went wrong. Try again." };
  }
  redirect("/login?reset=1");
}

export async function verifyAction(_prev: FormState, fd: FormData): Promise<FormState> {
  const token = String(fd.get("token") ?? "");
  try {
    await api.verifyEmail(token);
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) {
      await clearSession();
      redirect(`/login?next=${encodeURIComponent(`/verify?token=${token}`)}`);
    }
    if (e instanceof ApiError && e.code === "invalid_token") {
      return { error: "This link has expired, was already used, or is for another account." };
    }
    return { error: "Something went wrong. Try again." };
  }
  redirect("/dashboard?confirmed=1");
}
