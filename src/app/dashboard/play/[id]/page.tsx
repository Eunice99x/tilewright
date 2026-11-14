import type { Metadata } from "next";
import { notFound, redirect } from "next/navigation";
import { ApiError, api, type Game, type Me } from "@/lib/api";
import { siteUrl } from "@/lib/oauth";
import { Play } from "../play";

export const metadata: Metadata = { title: "Play", robots: { index: false } };

export default async function PlayPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  let game: Game, me: Me;
  try {
    [game, me] = await Promise.all([api.game(id), api.me()]);
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) redirect("/auth/expired");
    if (e instanceof ApiError && e.status === 404) notFound();
    throw e;
  }
  return <Play key={game.id} initial={game} guest={me.user.guest} inviteUrl={game.invite_code ? `${siteUrl()}/join/${game.invite_code}` : undefined} />;
}
