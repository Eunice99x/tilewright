import { getSessionToken } from "@/lib/session";

function apiUrl(): string {
  const url = process.env.API_URL;
  if (!url && process.env.NODE_ENV === "production") {
    const message = "API_URL is not set: the web server can't reach the Go API.";
    console.error(message);
    throw new Error(message);
  }
  return (url ?? "http://127.0.0.1:8180").replace(/\/+$/, "");
}

export interface User {
  id: string;
  email: string;
  plan: string;
  created_at: string;
  email_verified: boolean;
  is_admin: boolean;
  name: string;
  guest: boolean;
}

export interface Me {
  user: User;
}

export interface PortalLinks {
  overview: string;
  cancel: string;
  update_payment: string;
}

export interface AdminUser {
  id: string;
  email: string;
  plan: string;
  status: string;
  confirmed: boolean;
  sign_in: string[];
  created_at: string;
  /** When the newest live session began. null once every session has ended or been signed out. */
  last_seen: string | null;
}

export interface AdminUsersPage {
  users: AdminUser[];
  total: number;
  offset: number;
  page_size: number;
}

export interface AdminUserDetail extends AdminUser {
  paddle_customer_id: string;
  subscription_id: string;
  lifetime_plan: string;
  lifetime_transaction_id: string;
}

/** One message from the feedback box. */
export interface Feedback {
  id: string;
  /** null for a visitor without an account, or one whose account is gone. */
  user_id: string | null;
  /** "" when they left none. */
  email: string;
  message: string;
  page: string;
  created_at: string;
}

export interface AdminFeedbackPage {
  feedback: Feedback[];
  total: number;
  offset: number;
  page_size: number;
}

export interface Placement {
  row: number;
  col: number;
  letter: string;
}

export interface GameMove {
  seat: number;
  kind: "play" | "swap" | "pass" | "end" | "resign" | "left";
  words?: string[];
  tiles?: number;
  points: number;
}

export interface Game {
  id: string;
  mode: "solo" | "versus";
  seat: number;
  opponent?: string;
  waiting: boolean;
  invite_code?: string;
  board: string[];
  rack: string;
  bag_left: number;
  scores: number[];
  to_move: number;
  turn: number;
  turns_left: number;
  over: boolean;
  winner: number;
  resigned: number;
  moves: GameMove[];
  last?: GameMove;
}

export interface GameListItem {
  id: string;
  mode: "solo" | "versus";
  seat: number;
  opponent?: string;
  waiting: boolean;
  invite_code?: string;
  scores: number[];
  to_move: number;
  turn: number;
  over: boolean;
  winner: number;
  updated_at: string;
}

export interface GameSummary {
  id: string;
  score: number;
  turns: number;
  finished_at: string;
}

export class ApiError extends Error {
  constructor(public status: number, public code: string, public words?: string[]) {
    super(code || `API error ${status}`);
  }
}

interface Opts {
  method?: string;
  json?: unknown;
  /** "optional" forwards the session when there is one and goes without it otherwise. */
  auth?: boolean | "optional";
  clientIp?: string;
}

async function call(path: string, { method = "GET", json, auth = true, clientIp }: Opts = {}): Promise<Response> {
  const token = auth ? await getSessionToken() : undefined;
  if (auth === true && !token) throw new ApiError(401, "no_session");
  const base = apiUrl();
  const internalKey = process.env.INTERNAL_KEY;
  let res: Response;
  try {
    res = await fetch(`${base}${path}`, {
      method,
      cache: "no-store",
      headers: {
        ...(json !== undefined ? { "content-type": "application/json" } : {}),
        ...(token ? { authorization: `Session ${token}` } : {}),
        ...(internalKey && clientIp ? { "x-internal-key": internalKey, "x-client-ip": clientIp } : {}),
      },
      body: json !== undefined ? JSON.stringify(json) : undefined,
    });
  } catch {
    throw new ApiError(503, "api_unreachable");
  }
  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as { error?: string; words?: string[] } | null;
    throw new ApiError(res.status, body?.error ?? "", body?.words);
  }
  return res;
}

async function request<T>(path: string, opts?: Opts): Promise<T> {
  const res = await call(path, opts);
  return (res.status === 204 ? undefined : await res.json()) as T;
}

async function authCall(path: string, json: unknown, clientIp?: string): Promise<{ token: string; user: User }> {
  const res = await call(path, { method: "POST", json, auth: false, clientIp });
  const body = (await res.json().catch(() => ({}))) as { token?: string; user: User };
  if (!body.token) throw new ApiError(502, "no_token");
  return { token: body.token, user: body.user };
}

export const api = {
  signup: (email: string, password: string, clientIp?: string) => authCall("/api/auth/signup", { email, password }, clientIp),
  login: (email: string, password: string, clientIp?: string) => authCall("/api/auth/login", { email, password }, clientIp),
  guest: (name: string, clientIp?: string) => authCall("/api/auth/guest", { name }, clientIp),
  oauth: (provider: "google" | "github", code: string, redirectUri: string, clientIp?: string) =>
    authCall("/api/auth/oauth", { provider, code, redirect_uri: redirectUri }, clientIp),
  logout: () => request<void>("/api/auth/logout", { method: "POST" }),
  me: () => request<Me>("/api/me"),
  forgot: (email: string, clientIp?: string) => request<void>("/api/auth/forgot", { method: "POST", json: { email }, auth: false, clientIp }),
  resetPassword: (token: string, password: string, clientIp?: string) => request<void>("/api/auth/reset", { method: "POST", json: { token, password }, auth: false, clientIp }),
  verifyEmail: (token: string) => request<void>("/api/auth/verify", { method: "POST", json: { token } }),
  resendVerification: () => request<void>("/api/auth/resend-verification", { method: "POST" }),
  adminUsers: ({ q, plan, confirmed, offset }: { q: string; plan: string; confirmed: string; offset: number }) =>
    request<AdminUsersPage>(`/api/admin/users?${new URLSearchParams({ q, plan, confirmed, offset: String(offset) })}`),
  adminUser: (id: string) => request<AdminUserDetail>(`/api/admin/users/${encodeURIComponent(id)}`),
  adminFeedback: (offset: number) => request<AdminFeedbackPage>(`/api/admin/feedback?offset=${offset}`),
  sendFeedback: (message: string, email: string, page: string, clientIp?: string) =>
    request<void>("/api/feedback", { method: "POST", json: { message, email, page }, auth: "optional", clientIp }),
  billingPortal: () => request<PortalLinks>("/api/billing/portal", { method: "POST" }),

  games: () => request<{ games: GameListItem[] }>("/api/games").then((r) => r.games),
  startGame: () => request<Game>("/api/games", { method: "POST" }),
  startVersus: () => request<Game>("/api/games/versus", { method: "POST" }),
  joinGame: (code: string) => request<Game>("/api/games/join", { method: "POST", json: { code } }),
  game: (id: string) => request<Game>(`/api/games/${encodeURIComponent(id)}`),
  bestGames: () => request<{ games: GameSummary[] }>("/api/games/best").then((r) => r.games),
  play: (id: string, tiles: Placement[]) => request<Game>(`/api/games/${encodeURIComponent(id)}/play`, { method: "POST", json: { tiles } }),
  swap: (id: string, letters: string) => request<Game>(`/api/games/${encodeURIComponent(id)}/swap`, { method: "POST", json: { letters } }),
  pass: (id: string) => request<Game>(`/api/games/${encodeURIComponent(id)}/pass`, { method: "POST" }),
  endGame: (id: string) => request<Game>(`/api/games/${encodeURIComponent(id)}/end`, { method: "POST" }),
};
