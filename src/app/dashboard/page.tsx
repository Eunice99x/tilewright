import type { Metadata } from "next";
import Link from "next/link";
import { redirect } from "next/navigation";
import { FeedbackBox } from "@/components/feedback-box";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Table } from "@/components/ui/table";
import { ApiError, api, type GameListItem, type GameSummary, type Me } from "@/lib/api";
import { MAX_TURNS } from "@/lib/rules";
import { startGameAction, startVersusAction } from "./play/actions";
import { ConfirmEmail } from "./widgets";

export const metadata: Metadata = { title: "Your games", robots: { index: false } };

const day = new Intl.DateTimeFormat("en", { day: "numeric", month: "short", year: "numeric" });

function who(g: GameListItem): string {
  return g.mode === "solo" ? "Solo" : g.opponent ? `vs ${g.opponent}` : "Waiting for a friend";
}

function status(g: GameListItem): string {
  const mine = g.scores[g.seat] ?? 0;
  if (g.mode === "solo") return g.over ? `${mine} points` : `Turn ${Math.min(g.turn + 1, MAX_TURNS)} of ${MAX_TURNS}, ${mine} points`;
  const theirs = g.scores[1 - g.seat] ?? 0;
  const score = `${mine} to ${theirs}`;
  if (!g.over) return score;
  if (g.winner === -1) return `Draw, ${score}`;
  return g.winner === g.seat ? `Won ${score}` : `Lost ${score}`;
}

const yourTurn = (g: GameListItem) => !g.over && g.to_move === g.seat;

function GameRows({ games }: { games: GameListItem[] }) {
  return (
    <ul className="game-list">
      {games.map((g) => (
        <li key={g.id}>
          <Link href={`/dashboard/play/${g.id}`} className="game-row">
            <span className="game-who">{who(g)}</span>
            <span className="muted">{status(g)}</span>
          </Link>
        </li>
      ))}
    </ul>
  );
}

export default async function Dashboard({ searchParams }: { searchParams: Promise<{ confirmed?: string }> }) {
  const { confirmed } = await searchParams;
  let me: Me, games: GameListItem[], best: GameSummary[];
  try {
    [me, games, best] = await Promise.all([api.me(), api.games(), api.bestGames()]);
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) redirect("/auth/expired");
    throw e;
  }
  const mine = games.filter(yourTurn);
  const theirs = games.filter((g) => !g.over && !yourTurn(g));
  const done = games.filter((g) => g.over).slice(0, 10);
  const soloGoing = games.some((g) => g.mode === "solo" && !g.over);

  return (
    <>
      <div className="app-head">
        <h1>Your games</h1>
        <div className="inline">
          <form action={startVersusAction}><Button>Play a friend</Button></form>
          <form action={startGameAction}><Button variant="ghost">{soloGoing ? "Continue solo game" : "Practice solo"}</Button></form>
        </div>
      </div>
      {!me.user.email_verified && !me.user.guest && <ConfirmEmail email={me.user.email} />}
      {confirmed && me.user.email_verified && <p className="notice" role="status">Email confirmed.</p>}

      {games.length === 0 && (
        <Card title="No games yet">
          <p className="muted">Play a friend: you get a link to send them, and you take turns whenever you like. Or warm up with a solo game of {MAX_TURNS} turns.</p>
        </Card>
      )}
      {mine.length > 0 && <Card title="Your turn"><GameRows games={mine} /></Card>}
      {theirs.length > 0 && <Card title="Their turn"><GameRows games={theirs} /></Card>}
      {done.length > 0 && <Card title="Finished"><GameRows games={done} /></Card>}

      {best.length > 0 && (
        <Card title="Best solo scores">
          <Table>
            <thead>
              <tr><th>Score</th><th>Turns</th><th>Finished</th></tr>
            </thead>
            <tbody>
              {best.map((g) => (
                <tr key={g.id}>
                  <td><strong>{g.score}</strong></td>
                  <td>{g.turns}</td>
                  <td className="muted">{day.format(new Date(g.finished_at))}</td>
                </tr>
              ))}
            </tbody>
          </Table>
        </Card>
      )}
      <FeedbackBox page="/dashboard" signedIn={!me.user.guest} />
    </>
  );
}
