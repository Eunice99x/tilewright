"use server";

import { redirect } from "next/navigation";
import { ApiError, api, type Game } from "@/lib/api";
import { clearSession, getSessionToken, setSession, visitorIp } from "@/lib/session";

export interface PlayState {
  error?: string;
  name?: string;
  needName?: boolean;
}

export async function playNowAction(_prev: PlayState, fd: FormData): Promise<PlayState> {
  const intent = String(fd.get("intent") ?? "versus");
  const code = String(fd.get("code") ?? "");
  const name = String(fd.get("name") ?? "").trim();

  if (!(await getSessionToken())) {
    if (!name) return { error: "Type a name. It's what the other player sees.", name, needName: true };
    try {
      const { token } = await api.guest(name, await visitorIp());
      await setSession(token);
    } catch (e) {
      if (e instanceof ApiError) {
        if (e.status === 400) return { error: "Use 1 to 24 characters, without an @.", name, needName: true };
        if (e.status === 429) return { error: "Too many new players from here. Wait a while and try again.", name, needName: true };
        if (e.status === 503) return { error: "The game server isn't answering. Try again in a minute.", name, needName: true };
      }
      return { error: "Something went wrong. Try again.", name, needName: true };
    }
  }

  let game: Game;
  try {
    game = intent === "join" ? await api.joinGame(code) : intent === "solo" ? await api.startGame() : await api.startVersus();
  } catch (e) {
    if (e instanceof ApiError) {
      if (e.status === 401) {
        await clearSession();
        return { error: "Your last visit ended. Type your name to play.", name, needName: true };
      }
      if (e.status === 404) return { error: "This invite link doesn't lead to a game. Ask your friend to send it again." };
      if (e.code === "game_full") return { error: "Someone else already took the seat in this game." };
      if (e.code === "game_over") return { error: "This game is already over." };
    }
    return { error: "Couldn't start the game. Try again." };
  }
  redirect(`/dashboard/play/${game.id}`);
}
