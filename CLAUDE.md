# Tilewright

A word-tile game (11x11 board, 80 tiles): versus a friend by invite link, taking turns; or solo practice in 15 turns. Built on the SaaS starter: Go API + Next.js 16 + Neon Postgres + Vercel Services + Paddle.

## Game
- The rules live only in `api/internal/game` (no HTTP, no SQL). The server owns every game: the bag order never leaves it, and every move is checked and scored there.
- Board layout, tile values and counts, rack size, turns and the all-tiles bonus live only in `api/internal/game/rules.json`. Go and the web (`src/lib/rules.ts`) both read it.
- `words.txt` is the ENABLE list (public domain), cut to 2-11 letters.
- A game is one row in `games` (`state` jsonb) plus its seats in `game_players` (seat 0 started it, seat 1 joined with `/join/<invite_code>`). `version` stops two requests from saving the same turn (409 `game_changed`). One solo game in progress per player; versus games, any number.
- A player only ever gets `View(seat)`: their own rack, never the bag or the other rack. Other players show by the part of their email before the @.
- The game page polls for the other player's move every 4 s while it's their turn. Login and signup may only send you back to `/verify?token=` or `/join/<code>` (`src/lib/next.ts`).

## Rules
- Go stays simple: chi, sqlx, handler → service → store. All SQL lives in `store`.
- Only the web server calls account routes. The one public API route is `/api/paddle/webhook` (`vercel.json`).
- App name, Paddle product tag and plans live only in `api/internal/service/app.json`. Colors live only in `:root` in `src/app/globals.css`.
- Use `src/components/ui/` for buttons, inputs, cards, badges and tables.
- Design: dark, built to the tasteskill rules (`~/.agents/skills/design-taste-frontend`). What it dealt (palette plum-black-pink, Epilogue, nav-mega, the photo hero with the board as key art, counter-rolls) is listed at the top of `src/app/globals.css` and `src/app/page.tsx`. Colours only from the `--p-*` tokens; one filled pink button per view, the main action "Start a game".
- The landing page's pictures in `public/art/` are captures of the real game (board, rack, invite box), made from the game's own components. Recapture them when the board's look changes; `invite.webp` still shows a localhost link.
- `public/assets/moments/` is the skill's kit, copied as is: never edit it.
- Check the landing page with the skill: `node ~/.agents/skills/design-taste-frontend/scripts/consistency.mjs http://localhost:3100/ --page gaming --category "word-tile game,tiles,board" --primary "Start a game"` (needs `CHROME_PATH`).
- Secrets live only in gitignored `api/.env` and `.env.local`.
- Migrations in `api/migrations` are applied by hand (local and Neon). Go must use `DATABASE_URL_UNPOOLED` on Neon.
- Paid plans are `"billing": "subscription"` or `"one_time"` in `app.json`. One-time access lives in `users.lifetime_plan`, apart from the subscription columns. Refunds of it are manual.
- `is_admin` is set by hand in the database, never from a request.
- A security fix in auth or webhooks must be copied by hand to every app made from this starter. See the README.

## Commands
- Database: `docker compose up -d --wait`, then apply `api/migrations/001_init.sql` and `002_games.sql` with `psql "postgres://postgres:postgres@localhost:5544/app?sslmode=disable" -f <file>`
- API (:8180): `cd api && go run ./cmd/api`
- Web (:3100): `npm run dev` (`npm start` after a build needs `API_URL=http://127.0.0.1:8180`)
- Go: `cd api && go vet ./... && go test ./...`. Add `TEST_DATABASE_URL=...` to run the store tests against Postgres.
- Web: `npm test && npm run lint && npm run build`
- Deploy: `vercel --prod --yes` from the repo root.
