import { letterValue } from "@/lib/rules";

export function Tile({ letter, fresh }: { letter: string; fresh?: boolean }) {
  return (
    <span className={fresh ? "tile fresh" : "tile"}>
      <span className="tile-l">{letter}</span>
      <span className="tile-v">{letterValue(letter)}</span>
    </span>
  );
}
