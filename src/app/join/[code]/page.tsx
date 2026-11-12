import type { Metadata } from "next";
import { cookies } from "next/headers";
import Link from "next/link";
import { notFound } from "next/navigation";
import { PlayForm } from "@/components/play-form";
import { APP } from "@/lib/plans";
import { SESSION_COOKIE } from "@/lib/session";

export const metadata: Metadata = {
  title: "You're invited to a game",
  description: `A friend wants to play ${APP.name}, a word-tile game, with you.`,
  robots: { index: false },
};

export default async function Join({ params }: { params: Promise<{ code: string }> }) {
  const { code } = await params;
  if (!/^[a-z0-9]{1,32}$/i.test(code)) notFound();
  const signedIn = !!(await cookies()).get(SESSION_COOKIE);
  return (
    <div className="frame auth-wrap">
      <div className="auth card">
        <h1>A friend wants to play</h1>
        <p className="muted">
          {APP.name} is a word-tile game for two. You take turns building words on the same board, and the higher score when the tiles run out wins.
        </p>
        <PlayForm signedIn={signedIn} code={code} />
        {!signedIn && <p className="alt">Have an account? <Link href={`/login?next=${encodeURIComponent(`/join/${code}`)}`}>Log in</Link></p>}
      </div>
    </div>
  );
}
