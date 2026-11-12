import rules from "../../api/internal/game/rules.json";

export const RACK_SIZE = rules.rack_size;
export const MAX_TURNS = rules.max_turns;
export const ALL_TILES_BONUS = rules.all_tiles_bonus;
export const LAYOUT: string[] = rules.layout;
export const SIZE = LAYOUT.length;
export const CENTRE = Math.floor(SIZE / 2);

const TILES: Record<string, { value: number; count: number }> = rules.tiles;
export const TILE_COUNT = Object.values(TILES).reduce((n, t) => n + t.count, 0);

export const letterValue = (letter: string): number => TILES[letter]?.value ?? 0;

export type Bonus = "d" | "t" | "D" | "T" | "*" | ".";

export const BONUS: Record<Exclude<Bonus, ".">, { short: string; name: string }> = {
  d: { short: "2L", name: "double letter" },
  t: { short: "3L", name: "triple letter" },
  D: { short: "2W", name: "double word" },
  T: { short: "3W", name: "triple word" },
  "*": { short: "", name: "start square" },
};

export const bonusAt = (row: number, col: number): Bonus => (LAYOUT[row]?.[col] ?? ".") as Bonus;
