"use client";

import Link from "next/link";
import { type DragEvent, type KeyboardEvent, useEffect, useMemo, useRef, useState, useTransition } from "react";
import { Tile } from "@/components/game/tile";
import { FeedbackBox } from "@/components/feedback-box";
import { Button, buttonClass } from "@/components/ui/button";
import type { Game, GameMove, Placement } from "@/lib/api";
import { ALL_TILES_BONUS, BONUS, CENTRE, MAX_TURNS, RACK_SIZE, SIZE, bonusAt } from "@/lib/rules";
import { endAction, type MoveResult, passAction, playAction, refreshAction, startGameAction, startVersusAction, swapAction } from "./actions";

type RackTile = { id: number; letter: string };
type Square = { row: number; col: number };
type Note = { tone: "good" | "bad"; text: string };

const POLL_MS = 4000;

const key = (r: number, c: number) => `${r},${c}`;
const rackOf = (rack: string): RackTile[] => [...rack].map((letter, id) => ({ id, letter }));
const inside = (n: number) => Math.min(SIZE - 1, Math.max(0, n));
const plural = (n: number, one: string) => `${n} ${n === 1 ? one : `${one}s`}`;

const ERRORS: Record<string, string> = {
  must_cover_start: "The first word has to cover the centre square.",
  word_too_short: "A word needs at least two letters.",
  not_in_line: "Put all your tiles in one row or one column.",
  gap_in_word: "Leave no empty squares between your tiles.",
  not_connected: "Your word has to touch a tile already on the board.",
  square_taken: "That square already has a tile. Reload the page.",
  not_in_rack: "Those letters aren't on your rack. Reload the page.",
  bag_too_small: "The bag has fewer tiles than you want to swap.",
  not_your_turn: "It's not your turn yet.",
  game_over: "This game is over.",
  game_changed: "This game moved on in another tab. Reload the page to catch up.",
  api_unreachable: "The game server isn't answering. Try again in a moment.",
};

function errorText({ error, words }: MoveResult): string {
  if (error === "not_a_word" && words?.length) {
    return `Not in the word list: ${words.join(", ")}.`;
  }
  return ERRORS[error ?? ""] ?? "That didn't work. Try again.";
}

function describe(m: GameMove): string {
  switch (m.kind) {
    case "play": return m.words?.join(", ") ?? "";
    case "swap": return `swapped ${plural(m.tiles ?? 0, "tile")}`;
    case "pass": return "passed";
    case "resign": return "resigned";
    case "end": return "ended the game";
    case "left": return m.points < 0 ? "tiles left on the rack" : "the other rack's tiles";
  }
}

function result(g: Game, them: string): string {
  const mine = g.scores[g.seat];
  const theirs = g.scores[1 - g.seat];
  if (g.resigned === g.seat) return `You resigned. ${them} wins ${theirs} to ${mine}.`;
  if (g.resigned >= 0) return `${them} resigned. You win, ${mine} to ${theirs}.`;
  if (g.winner === -1) return `A draw, ${mine} each.`;
  return g.winner === g.seat ? `You win, ${mine} to ${theirs}.` : `${them} wins, ${theirs} to ${mine}.`;
}

export function Play({ initial, inviteUrl, guest = false }: { initial: Game; inviteUrl?: string; guest?: boolean }) {
  const [game, setGame] = useState(initial);
  const [rack, setRack] = useState<RackTile[]>(() => rackOf(initial.rack));
  const [pending, setPending] = useState<Map<string, RackTile>>(() => new Map());
  const [selected, setSelected] = useState<number | null>(null);
  const [cursor, setCursor] = useState<Square>({ row: CENTRE, col: CENTRE });
  const [dir, setDir] = useState<"across" | "down">("across");
  const [swapping, setSwapping] = useState<Set<number> | null>(null);
  const [confirmEnd, setConfirmEnd] = useState(false);
  const [copied, setCopied] = useState(false);
  const [note, setNote] = useState<Note | null>(null);
  const [busy, startTransition] = useTransition();
  const cells = useRef<(HTMLButtonElement | null)[]>([]);

  const versus = game.mode === "versus";
  const them = game.opponent || "Your friend";
  const myTurn = !game.over && game.to_move === game.seat;
  const placedIds = useMemo(() => new Set([...pending.values()].map((t) => t.id)), [pending]);
  const onRack = rack.filter((t) => !placedIds.has(t.id));
  const locked = game.over || busy;

  useEffect(() => {
    if (!versus || game.over || (myTurn && !game.waiting)) return;
    const timer = setInterval(async () => {
      if (document.visibilityState !== "visible") return;
      const fresh = (await refreshAction(game.id)).game;
      if (!fresh || (fresh.turn === game.turn && fresh.over === game.over && fresh.waiting === game.waiting)) return;
      setGame(fresh);
      if (fresh.turn !== game.turn) {
        setRack(rackOf(fresh.rack));
        setPending(new Map());
        setSwapping(null);
      }
      const theirMoves = fresh.moves.slice(game.moves.length).filter((m) => m.seat !== fresh.seat && m.kind !== "left");
      const last = theirMoves.at(-1);
      if (fresh.over) setNote({ tone: "good", text: result(fresh, fresh.opponent || "Your friend") });
      else if (last) setNote({ tone: "good", text: `${fresh.opponent || "Your friend"} ${last.kind === "play" ? `played ${describe(last)} for ${last.points}` : describe(last)}. Your turn.` });
      else if (game.waiting && !fresh.waiting) setNote({ tone: "good", text: `${fresh.opponent} joined the game.` });
    }, POLL_MS);
    return () => clearInterval(timer);
  }, [versus, game, myTurn]);

  const focusSquare = (s: Square) => {
    setCursor(s);
    cells.current[s.row * SIZE + s.col]?.focus();
  };

  const nextEmpty = (s: Square, taken: (r: number, c: number) => boolean): Square => {
    let { row, col } = s;
    do {
      if (dir === "across") col++;
      else row++;
    } while (row < SIZE && col < SIZE && taken(row, col));
    return row < SIZE && col < SIZE ? { row, col } : s;
  };

  const isTaken = (r: number, c: number, extra?: string) =>
    game.board[r][c] !== "." || pending.has(key(r, c)) || key(r, c) === extra;

  function place(s: Square, tile: RackTile) {
    setPending((p) => new Map(p).set(key(s.row, s.col), tile));
    setNote(null);
  }

  function unplace(s: Square) {
    setPending((p) => {
      const next = new Map(p);
      next.delete(key(s.row, s.col));
      return next;
    });
  }

  function recall() {
    setPending(new Map());
    setSelected(null);
  }

  function shuffle() {
    setRack((r) => {
      const next = [...r];
      for (let i = next.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1));
        [next[i], next[j]] = [next[j], next[i]];
      }
      return next;
    });
  }

  function notYet(): boolean {
    if (myTurn) return false;
    setNote({ tone: "bad", text: game.waiting && game.to_move !== game.seat ? "Wait for your friend to join and play." : `Wait for ${them} to play first.` });
    return true;
  }

  function clickSquare(s: Square) {
    if (locked || swapping) return;
    const wasCursor = cursor.row === s.row && cursor.col === s.col;
    setCursor(s);
    if (pending.has(key(s.row, s.col))) {
      unplace(s);
      return;
    }
    if (game.board[s.row][s.col] !== ".") return;
    const tile = onRack.find((t) => t.id === selected);
    if (tile && notYet()) return;
    if (tile) {
      place(s, tile);
      setSelected(null);
    } else if (wasCursor) {
      setDir((d) => (d === "across" ? "down" : "across"));
    }
  }

  function clickRackTile(t: RackTile) {
    if (locked) return;
    if (swapping) {
      const next = new Set(swapping);
      if (next.has(t.id)) next.delete(t.id);
      else next.add(t.id);
      setSwapping(next);
      return;
    }
    if (notYet()) return;
    setSelected((id) => (id === t.id ? null : t.id));
  }

  function dropOnSquare(e: DragEvent, s: Square) {
    e.preventDefault();
    if (locked || swapping || isTaken(s.row, s.col) || notYet()) return;
    const tile = onRack.find((t) => t.id === Number(e.dataTransfer.getData("text/plain")));
    if (tile) {
      place(s, tile);
      setSelected(null);
      setCursor(s);
    }
  }

  function onBoardKey(e: KeyboardEvent) {
    if (locked || swapping) return;
    const { row, col } = cursor;
    const go = (dr: number, dc: number) => {
      e.preventDefault();
      focusSquare({ row: inside(row + dr), col: inside(col + dc) });
    };
    switch (e.key) {
      case "ArrowUp": return go(-1, 0);
      case "ArrowDown": return go(1, 0);
      case "ArrowLeft": return go(0, -1);
      case "ArrowRight": return go(0, 1);
      case "Enter":
        e.preventDefault();
        return submit();
      case "Escape":
        return recall();
      case " ":
        e.preventDefault();
        return setDir((d) => (d === "across" ? "down" : "across"));
      case "Backspace": {
        e.preventDefault();
        if (pending.has(key(row, col))) return unplace(cursor);
        const back = dir === "across" ? { row, col: inside(col - 1) } : { row: inside(row - 1), col };
        if (pending.has(key(back.row, back.col))) unplace(back);
        return focusSquare(back);
      }
    }
    if (/^[a-z]$/i.test(e.key) && !e.ctrlKey && !e.metaKey && !e.altKey) {
      e.preventDefault();
      if (notYet()) return;
      const letter = e.key.toUpperCase();
      const tile = onRack.find((t) => t.letter === letter);
      if (!tile) {
        setNote({ tone: "bad", text: `There's no ${letter} on your rack.` });
        return;
      }
      const at = isTaken(row, col) ? nextEmpty(cursor, (r, c) => isTaken(r, c)) : cursor;
      if (isTaken(at.row, at.col)) return;
      place(at, tile);
      focusSquare(nextEmpty(at, (r, c) => isTaken(r, c, key(at.row, at.col))));
    }
  }

  function done(res: MoveResult, message: (g: Game) => string) {
    if (!res.game) {
      setNote({ tone: "bad", text: errorText(res) });
      return;
    }
    setGame(res.game);
    setRack(rackOf(res.game.rack));
    setPending(new Map());
    setSelected(null);
    setSwapping(null);
    setConfirmEnd(false);
    setNote({ tone: "good", text: res.game.over && versus ? result(res.game, them) : message(res.game) });
  }

  function submit() {
    if (pending.size === 0 || locked || !myTurn) return;
    const tiles: Placement[] = [...pending].map(([k, t]) => {
      const [row, col] = k.split(",").map(Number);
      return { row, col, letter: t.letter };
    });
    startTransition(async () => {
      const res = await playAction(game.id, tiles);
      done(res, (g) => {
        const bonus = tiles.length === RACK_SIZE ? `, with the ${ALL_TILES_BONUS}-point all-tiles bonus` : "";
        return `${g.last?.words?.join(", ") ?? ""}: ${g.last?.points ?? 0} points${bonus}.`;
      });
    });
  }

  function swap() {
    if (!swapping || swapping.size === 0 || locked) return;
    const letters = rack.filter((t) => swapping.has(t.id)).map((t) => t.letter).join("");
    startTransition(async () => {
      done(await swapAction(game.id, letters), () => `Swapped ${plural(letters.length, "tile")}.`);
    });
  }

  function pass() {
    if (locked || !myTurn) return;
    startTransition(async () => {
      done(await passAction(game.id), () => "You passed.");
    });
  }

  function end() {
    if (locked) return;
    if (!confirmEnd) {
      setConfirmEnd(true);
      return;
    }
    startTransition(async () => {
      done(await endAction(game.id), (g) => `Game over. Final score: ${g.scores[0]}.`);
    });
  }

  async function copyInvite() {
    if (!inviteUrl) return;
    try {
      await navigator.clipboard.writeText(inviteUrl);
      setCopied(true);
    } catch {
      setNote({ tone: "bad", text: "Couldn't copy it. Select the link and copy it yourself." });
    }
  }

  const turnLine = game.over
    ? null
    : !versus
      ? null
      : game.waiting && !myTurn
        ? "Waiting for your friend to join and play."
        : myTurn
          ? "Your turn."
          : `${them}'s turn. Their move shows up here when they make it.`;

  const title = versus ? `You vs ${game.opponent || "a friend"}` : "Solo game";

  return (
    <>
    <div className="app-head">
      <h1>{title}</h1>
    </div>
    <div className={myTurn || game.over ? "play" : "play not-mine"}>
      <div className="play-main">
        {versus && game.waiting && inviteUrl && (
          <section className="invite" aria-label="Invite a friend">
            <p><strong>Send this link to a friend.</strong> They join, and you take turns from there. You can play the first word now.</p>
            <div className="invite-row">
              <input className="input" readOnly value={inviteUrl} aria-label="Invite link" onFocus={(e) => e.currentTarget.select()} />
              <Button variant="ghost" onClick={copyInvite}>{copied ? "Copied" : "Copy link"}</Button>
            </div>
          </section>
        )}

        {versus ? (
          <dl className="stats">
            <div className={myTurn ? "on" : undefined}><dt>You</dt><dd>{game.scores[game.seat]}</dd></div>
            <div className={!game.over && !myTurn ? "on" : undefined}><dt>{game.opponent || "Friend"}</dt><dd>{game.scores[1 - game.seat]}</dd></div>
            <div><dt>Bag</dt><dd>{game.bag_left}</dd></div>
          </dl>
        ) : (
          <dl className="stats">
            <div><dt>Score</dt><dd>{game.scores[0]}</dd></div>
            <div><dt>Turn</dt><dd>{game.over ? "–" : `${Math.min(game.turn + 1, MAX_TURNS)}/${MAX_TURNS}`}</dd></div>
            <div><dt>Bag</dt><dd>{game.bag_left}</dd></div>
          </dl>
        )}
        {turnLine && <p className={myTurn ? "turn-line mine" : "turn-line"}>{turnLine}</p>}

        <div
          className="board"
          role="group"
          aria-label="Board. Arrow keys move, letters place tiles, Space turns typing down or across, Enter plays."
          aria-busy={busy}
          onKeyDown={onBoardKey}
        >
          {game.board.map((line, r) =>
            [...line].map((ch, c) => {
              const s = { row: r, col: c };
              const fresh = pending.get(key(r, c));
              const letter = ch !== "." ? ch : fresh?.letter;
              const b = bonusAt(r, c);
              const here = cursor.row === r && cursor.col === c;
              const what = letter ? `${letter}${fresh ? ", placed this turn" : ""}` : b === "." ? "empty" : BONUS[b].name;
              return (
                <button
                  key={key(r, c)}
                  ref={(el) => {
                    cells.current[r * SIZE + c] = el;
                  }}
                  type="button"
                  className={here && !game.over ? "sq cursor" : "sq"}
                  data-bonus={b}
                  data-dir={here && !letter && !game.over ? dir : undefined}
                  tabIndex={here ? 0 : -1}
                  aria-label={`Row ${r + 1}, column ${c + 1}: ${what}`}
                  onClick={() => clickSquare(s)}
                  onFocus={() => setCursor(s)}
                  onDragOver={(e) => !isTaken(r, c) && e.preventDefault()}
                  onDrop={(e) => dropOnSquare(e, s)}
                >
                  {letter ? <Tile letter={letter} fresh={!!fresh} /> : b !== "." && b !== "*" ? <span className="bonus">{BONUS[b].short}</span> : null}
                </button>
              );
            }),
          )}
        </div>

        {!game.over && (
          <div className="rack" role="group" aria-label={swapping ? "Pick the tiles to swap" : "Your rack"}>
            {onRack.map((t) => (
              <button
                key={t.id}
                type="button"
                className={swapping?.has(t.id) ? "rack-tile swap" : "rack-tile"}
                draggable={!locked && !swapping && myTurn}
                onDragStart={(e) => e.dataTransfer.setData("text/plain", String(t.id))}
                onClick={() => clickRackTile(t)}
                aria-pressed={swapping ? swapping.has(t.id) : selected === t.id}
                aria-label={`${t.letter}${swapping?.has(t.id) ? ", to swap" : ""}`}
              >
                <Tile letter={t.letter} />
              </button>
            ))}
            {Array.from({ length: Math.max(0, RACK_SIZE - onRack.length) }, (_, i) => (
              <span key={`slot-${i}`} className="rack-slot" aria-hidden="true" />
            ))}
          </div>
        )}

        {game.over ? (
          <section className="final" aria-label="Result">
            <p className="final-score">{versus ? `${game.scores[game.seat]}–${game.scores[1 - game.seat]}` : game.scores[0]}</p>
            <p className="muted">{versus ? result(game, them) : `Final score after ${plural(game.turn, "turn")}.`}</p>
            <div className="inline">
              <form action={versus ? startVersusAction : startGameAction}>
                <Button>{versus ? "Play a friend again" : "Start a new game"}</Button>
              </form>
              <Link href="/dashboard" className={buttonClass("ghost")}>All games</Link>
            </div>
            <FeedbackBox page={versus ? "/dashboard/play (versus, over)" : "/dashboard/play (solo, over)"} signedIn={!guest} />
          </section>
        ) : swapping ? (
          <div className="controls">
            <Button onClick={swap} disabled={locked || swapping.size === 0 || !myTurn}>
              {swapping.size === 0 ? "Pick tiles to swap" : `Swap ${plural(swapping.size, "tile")}`}
            </Button>
            <Button variant="ghost" onClick={() => setSwapping(null)} disabled={locked}>Keep my tiles</Button>
          </div>
        ) : (
          <div className="controls">
            <Button onClick={submit} disabled={locked || pending.size === 0 || !myTurn}>{busy ? "Checking…" : "Play word"}</Button>
            <Button variant="ghost" onClick={recall} disabled={locked || pending.size === 0}>Recall tiles</Button>
            <Button variant="ghost" onClick={shuffle} disabled={locked}>Shuffle rack</Button>
            <Button
              variant="ghost"
              onClick={() => {
                recall();
                setSwapping(new Set());
              }}
              disabled={locked || game.bag_left === 0 || !myTurn}
            >
              Swap tiles
            </Button>
            {versus && <Button variant="ghost" onClick={pass} disabled={locked || !myTurn}>Pass the turn</Button>}
          </div>
        )}

        {pending.size > 0 && !game.over && (
          <p className="pending-hint">
            {plural(pending.size, "tile")} placed, not played yet: only you can see {pending.size === 1 ? "it" : "them"}. Press Play word to send {pending.size === 1 ? "it" : "them"}.
          </p>
        )}
        <p className={note ? `play-note ${note.tone}` : "play-note"} role="status">{note?.text}</p>
      </div>

      <aside className="play-side" aria-label="Moves">
        <h2 className="h3">Moves</h2>
        {game.moves.length === 0 ? (
          <p className="muted small">Nothing played yet. The first word covers the centre square.</p>
        ) : (
          <ol className="moves">
            {game.moves.map((m, i) => (
              <li key={i}>
                <span>{versus && <strong>{m.seat === game.seat ? "You" : them}: </strong>}{describe(m)}</span>
                {(m.kind === "play" || m.kind === "left") && <span className="pts">{m.kind === "left" && m.points > 0 ? `+${m.points}` : m.points}</span>}
              </li>
            ))}
          </ol>
        )}

        {!game.over && (
          <>
            <p className="muted small help">
              Tap a tile, then a square, or drag it there. On a keyboard, pick a square and type: Space turns typing down or across, Enter plays.
            </p>
            <Button variant={confirmEnd ? "danger" : "ghost"} onClick={end} disabled={locked}>
              {versus ? (confirmEnd ? "Resign this game" : "Resign") : confirmEnd ? "End it, keep my score" : "End game"}
            </Button>
          </>
        )}
      </aside>
    </div>
    </>
  );
}
