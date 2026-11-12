export const n = (v: number) => v.toLocaleString("en-US");
export const fullDay = (iso: string) => new Date(iso).toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric", timeZone: "UTC" });

const METHODS: Record<string, string> = { password: "Password", google: "Google", github: "GitHub" };
export const signIn = (methods: string[]) => (methods.length > 0 ? methods.map((m) => METHODS[m] ?? m).join(", ") : "None");
