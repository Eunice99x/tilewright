export function safeNext(next: unknown): string | undefined {
  if (typeof next !== "string") return undefined;
  if (next.startsWith("/verify?token=")) return next;
  if (/^\/join\/[a-z0-9]{1,32}$/i.test(next)) return next;
  return undefined;
}

export const isInvite = (next?: string) => !!next?.startsWith("/join/");
