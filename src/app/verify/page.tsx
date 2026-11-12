import type { Metadata } from "next";
import { cookies } from "next/headers";
import Link from "next/link";
import { redirect } from "next/navigation";
import { VerifyForm } from "@/components/email-forms";
import { SESSION_COOKIE } from "@/lib/session";

export const metadata: Metadata = { title: "Confirm your email", robots: { index: false }, referrer: "no-referrer" };

export default async function Verify({ searchParams }: { searchParams: Promise<{ token?: string }> }) {
  const { token } = await searchParams;
  if (token && !(await cookies()).get(SESSION_COOKIE)) redirect(`/login?next=${encodeURIComponent(`/verify?token=${token}`)}`);
  return (
    <div className="frame auth-wrap">
      <div className="auth card">
        <h1>Confirm your email</h1>
        {token ? (
          <>
            <p className="muted">One click and you&rsquo;re done.</p>
            <VerifyForm token={token} />
          </>
        ) : (
          <p className="muted">This link is incomplete. <Link href="/dashboard">Log in to send a new one</Link></p>
        )}
      </div>
    </div>
  );
}
