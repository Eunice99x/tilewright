import { BONUS, bonusAt } from "@/lib/rules";
import { Tile } from "./tile";

export function BoardView({ board, label }: { board: string[]; label: string }) {
  return (
    <div className="board" role="img" aria-label={label}>
      {board.map((line, r) =>
        [...line].map((ch, c) => {
          const b = bonusAt(r, c);
          return (
            <span key={`${r},${c}`} className="sq" data-bonus={b}>
              {ch !== "." ? <Tile letter={ch} /> : b !== "." && b !== "*" ? <span className="bonus">{BONUS[b].short}</span> : null}
            </span>
          );
        }),
      )}
    </div>
  );
}
