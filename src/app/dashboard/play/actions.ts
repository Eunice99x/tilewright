"use server";

import { redirect } from "next/navigation";
import { ApiError, api, type Game, type Placement } from "@/lib/api";
import { RACK_SIZE, SIZE } from "@/lib/rules";

export interface MoveResult {
  game?: Game;
  error?: string;
  words?: string[];
}

const isId = (id: unknown): id is string => typeof id === "string" && id.length > 0 && id.length <= 64;

const isPlacement = (p: unknown): p is Placement => {
  if (typeof p !== "object" || p === null) return false;
  const { row, col, letter } = p as Record<string, unknown>;
  return Number.isInteger(row) && Number.isInteger(col) && (row as number) >= 0 && (row as number) < SIZE &&
    (col as number) >= 0 && (col as number) < SIZE && typeof letter === "string" && /^[A-Z]$/.test(letter);
};

async function move(call: () => Promise<Game>): Promise<MoveResult> {
  try {
    return { game: await call() };
  } catch (e) {
    if (e instanceof ApiError) {
      if (e.status === 401) redirect("/auth/expired");
      return { error: e.code || "internal", words: e.words };
    }
    return { error: "internal" };
  }
}

export async function playAction(id: unknown, tiles: unknown): Promise<MoveResult> {
  if (!isId(id) || !Array.isArray(tiles) || tiles.length === 0 || tiles.length > RACK_SIZE || !tiles.every(isPlacement)) {
    return { error: "invalid_input" };
  }
  const clean = tiles.map(({ row, col, letter }) => ({ row, col, letter }));
  return move(() => api.play(id, clean));
}

export async function swapAction(id: unknown, letters: unknown): Promise<MoveResult> {
  if (!isId(id) || typeof letters !== "string" || !/^[A-Z]{1,7}$/.test(letters)) return { error: "invalid_input" };
  return move(() => api.swap(id, letters));
}

export async function passAction(id: unknown): Promise<MoveResult> {
  if (!isId(id)) return { error: "invalid_input" };
  return move(() => api.pass(id));
}

export async function endAction(id: unknown): Promise<MoveResult> {
  if (!isId(id)) return { error: "invalid_input" };
  return move(() => api.endGame(id));
}

export async function refreshAction(id: unknown): Promise<MoveResult> {
  if (!isId(id)) return { error: "invalid_input" };
  return move(() => api.game(id));
}

async function open(start: () => Promise<Game>): Promise<void> {
  let id: string;
  try {
    id = (await start()).id;
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) redirect("/auth/expired");
    throw e;
  }
  redirect(`/dashboard/play/${id}`);
}

export async function startGameAction(): Promise<void> {
  return open(api.startGame);
}

export async function startVersusAction(): Promise<void> {
  return open(api.startVersus);
}
