import type { Metadata } from "next";
import Link from "next/link";
import { ResetForm } from "@/components/email-forms";

export const metadata: Metadata = { title: "Choose a new password", robots: { index: false }, referrer: "no-referrer" };

export default async function Reset({ searchParams }: { searchParams: Promise<{ token?: string }> }) {
  const { token } = await searchParams;
  return (
    <div className="frame auth-wrap">
      <div className="auth card">
        <h1>Choose a new password</h1>
        {token ? (
          <>
            <p className="muted">This logs you out everywhere.</p>
            <ResetForm token={token} />
          </>
        ) : (
          <p className="muted">This link is incomplete. <Link href="/forgot">Get a new one</Link></p>
        )}
      </div>
    </div>
  );
}
