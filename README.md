# SaaS starter

Go API + Next.js 16 + Neon Postgres + Vercel Services + Paddle, with auth, billing, email, an admin area and a dashboard shell. Copy it to start a new app.

**Known limit: no customer has ever paid through this code.** The webhook is tested with signed fake events only. Do one real purchase and one real cancel on the first app before trusting it.

## New app, in order

1. **Copy.** `cp -R saas-starter myapp && cd myapp && rm -rf .git node_modules .next && git init`
2. **Submit the domain to Paddle on day one.** Paddle approves each domain separately and it takes time, so don't leave it for launch week. It looks at the site, so fill in the landing page, prices, terms, privacy and refunds early.
3. **Rename** (see below).
4. **Run it locally:**
   ```sh
   npm install
   cp .env.example .env.local && cp .env.example api/.env   # empty values are fine locally
   docker compose up -d --wait
   psql "postgres://postgres:postgres@localhost:5544/app?sslmode=disable" -f api/migrations/001_init.sql
   (cd api && go run ./cmd/api)   # :8180
   npm run dev                    # :3100
   ```
5. **Accounts.** Neon database, Resend domain, Google and GitHub OAuth clients, Paddle products and prices. Create the Paddle webhook destination at `https://<domain>/api/paddle/webhook`.
6. **Migrate Neon:** `psql "$DATABASE_URL_UNPOOLED" -f api/migrations/001_init.sql`
7. **Deploy** (see below), then sign up and make yourself admin:
   `UPDATE users SET is_admin = true WHERE email = 'you@yourdomain.com';`
8. **Test a real purchase and cancel** once Paddle approves the domain.

## Rename points

| What | Where |
|---|---|
| App name, Paddle product tag, support email, plans and prices | `api/internal/service/app.json` (Go and web both read it) |
| Colors, radius, fonts | `:root` in `src/app/globals.css` |
| Logo and favicon | `src/components/logo.tsx`, `src/app/icon.svg` |
| Landing page | `src/app/page.tsx` |
| Terms, privacy, refunds | `src/content/legal.ts` (`grep -rn TODO src` finds every placeholder) |
| Package names (optional) | `package.json`, and the module path in `api/go.mod` |

Each app needs its own `product` tag. Several products can share a Paddle account and all their events reach every webhook, so this tag is how an app knows which are its own. An event with another tag gets a 200 and is ignored.

Plan ids: `free` is required and `admin` is reserved. Every other plan needs a `PADDLE_PRICE_<ID>` variable. If you rename a plan, rename that variable and update `.env.example`. Delete the placeholder plans you don't sell: checkout stays closed until every plan listed has a price id.

Each paid plan has `"billing": "subscription"` or `"one_time"`. A subscription plan lasts while the subscription is live. A one-time plan gives lifetime access when `transaction.completed` arrives, and hides the portal link. In Paddle, create a one-time price for it, and subscribe the webhook to `transaction.completed` as well as `subscription.*`. If a buyer has both, a live subscription wins, and they fall back to the lifetime plan when it ends.

**Refunds of one-time purchases are not handled: access stays until you remove it.** Paddle's refund events (`adjustment.*`) don't carry our `custom_data`, a partial refund shouldn't revoke, and we have no real payload to test against. After a refund or chargeback, by hand:
`UPDATE users SET lifetime_plan = '', lifetime_transaction_id = '' WHERE email = '...';`
The admin user page shows the transaction id. Subscription refunds follow whatever status Paddle sends on `subscription.*`.

## Env vars

`.env.example` lists every variable and what happens when it's empty. Everything is optional locally. In production, **`SITE_URL` and `INTERNAL_KEY` are required**: without `INTERNAL_KEY`, every visitor counts as one IP and five signups block all signups for a day.

Set env vars on Vercel before the first deploy. `SITE_URL` is also read at build time (sitemap, robots).

## Migrations

Applied by hand, locally and on Neon. `001_init.sql` is the whole schema and is safe to run twice. Later changes go in `002_*.sql`. Go must use `DATABASE_URL_UNPOOLED` on Neon, because lib/pq breaks behind the pooler.

## Deploy

Vercel project with Services (`vercel.json`): the web app at the root and the Go container in `api/`. Only `/api/paddle/*` reaches the API from the internet.

```sh
vercel link
vercel --prod --yes
```

## Local testing without Resend

With no `RESEND_API_KEY`, email is skipped and logged. Tokens are stored only as hashes, so you can't read a link from the database. Set a known one instead:

```sql
UPDATE email_tokens SET token_hash = encode(sha256('dev-token'::bytea), 'hex') WHERE kind = 'verify';
```

Then open `/verify?token=dev-token` while logged in. For a password reset, use `kind = 'reset'` and `/reset?token=dev-token`.

## Tests

```sh
cd api && go vet ./... && go test ./...
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5544/app?sslmode=disable" go test ./internal/store
npm test && npm run lint && npm run build
```

## Not tested

Tested locally with fake config only. These have never run from this repo:

- A real Paddle checkout (subscription or one-time), webhook delivery and customer portal. Paddle.js never ran against this code here, and the one-time `transaction.completed` payload is from Paddle's documented shape, not a captured event.
- Real Google and GitHub sign-in. The service logic is tested against fake provider responses; the state-mismatch and not-configured paths were run locally.
- Real email delivery through Resend.
- A Vercel deploy of this repo.

Rate limits are in memory, so per instance. On serverless, the real limit is higher than the number in the code.

## If an app needs API keys

They were left out. MailVet has a working version, in `~/Dev/mailvet-live/api`:
`internal/service/keys.go`, `internal/store/keys.go`, per-key quotas in `internal/service/quota.go`, and the `api_keys` table in `migrations/001_init.sql`.

## Security fixes are manual

Nothing is shared between apps. **A security fix in auth or webhooks must be copied by hand to every app made from this starter: MailVet, Mentioned and any later ones.** Keep a list of those apps here:

- MailVet (auth and billing came from it, not from this starter)
- Mentioned (MailVet's auth and billing were first copied from it)
