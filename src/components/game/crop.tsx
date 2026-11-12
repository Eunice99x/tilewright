import { BONUS, bonusAt } from "@/lib/rules";
import { Tile } from "./tile";

export function Crop({ board, r0, c0, rows, cols, fresh = [] }: { board: string[]; r0: number; c0: number; rows: number; cols: number; fresh?: string[] }) {
  return (
    <div className="crop" aria-hidden="true">
      {Array.from({ length: rows }, (_, i) => (
        <div className="crop-row" key={i}>
          {Array.from({ length: cols }, (_, j) => {
            const r = r0 + i, c = c0 + j;
            const ch = board[r]?.[c] ?? ".";
            const b = bonusAt(r, c);
            return (
              <span key={j} className="sq" data-bonus={b}>
                {ch !== "." ? <Tile letter={ch} fresh={fresh.includes(`${r},${c}`)} /> : b !== "." && b !== "*" ? <span className="bonus">{BONUS[b].short}</span> : null}
              </span>
            );
          })}
        </div>
      ))}
    </div>
  );
}

export function RackView({ letters, className = "" }: { letters: string; className?: string }) {
  return (
    <div className={`rack ${className}`} aria-hidden="true">
      {[...letters].map((l, i) => (
        <span key={i} className="rack-tile static"><Tile letter={l} /></span>
      ))}
    </div>
  );
}
