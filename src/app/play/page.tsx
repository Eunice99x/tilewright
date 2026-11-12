import type { Metadata } from "next";
import { cookies } from "next/headers";
import Link from "next/link";
import { PlayForm } from "@/components/play-form";
import { APP } from "@/lib/plans";
import { SESSION_COOKIE } from "@/lib/session";

export const metadata: Metadata = {
  title: "Play",
  description: `Start a game of ${APP.name} and send the link to a friend. No account needed.`,
  alternates: { canonical: "/play" },
};

export default async function PlayNow() {
  const signedIn = !!(await cookies()).get(SESSION_COOKIE);
  return (
    <div className="frame auth-wrap">
      <div className="auth card">
        <h1>Start a game</h1>
        <p className="muted">Play a friend: you get a link to send them, and you take turns whenever you like. Or practice on your own.</p>
        <PlayForm signedIn={signedIn} />
        {!signedIn && <p className="alt">Have an account? <Link href="/login">Log in</Link></p>}
      </div>
    </div>
  );
}
