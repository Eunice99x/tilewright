"use server";

import { redirect } from "next/navigation";
import { ApiError, api } from "@/lib/api";
import { safeNext } from "@/lib/next";
import { clearSession, setSession, visitorIp } from "@/lib/session";

export interface AuthState {
  error?: string;
  email?: string;
}

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/;

async function authenticate(mode: "login" | "signup", fd: FormData): Promise<AuthState> {
  const email = String(fd.get("email") ?? "").trim().toLowerCase();
  const password = String(fd.get("password") ?? "");
  if (!EMAIL_RE.test(email)) return { error: "Enter a valid email address.", email };
  if (!password) return { error: "Enter your password.", email };
  if (mode === "signup" && (password.length < 8 || new TextEncoder().encode(password).length > 72)) {
    return { error: "Password must be 8 to 72 characters.", email };
  }
  try {
    const ip = await visitorIp();
    const { token } = mode === "signup" ? await api.signup(email, password, ip) : await api.login(email, password, ip);
    await setSession(token);
  } catch (e) {
    if (e instanceof ApiError) {
      if (e.code === "email_taken" || e.status === 409) return { error: "An account with this email already exists. Log in, or use Google or GitHub if you signed up with them.", email };
      if (e.code === "weak_password") return { error: "Password must be at least 8 characters.", email };
      if (e.code === "too_many_attempts" || e.status === 429) return { error: "Too many attempts. Wait a while and try again.", email };
      if (e.status === 401) return { error: "Wrong email or password.", email };
      if (e.status === 503) return { error: "The service is unreachable. Try again in a minute.", email };
    }
    return { error: "Something went wrong. Try again.", email };
  }
  redirect(safeNext(fd.get("next")) ?? "/dashboard");
}

export async function signupAction(_prev: AuthState, fd: FormData): Promise<AuthState> {
  return authenticate("signup", fd);
}

export async function loginAction(_prev: AuthState, fd: FormData): Promise<AuthState> {
  return authenticate("login", fd);
}

export async function logoutAction(): Promise<void> {
  try {
    await api.logout();
  } catch {
  }
  await clearSession();
  redirect("/");
}
