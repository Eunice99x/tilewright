CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS users (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email               citext NOT NULL UNIQUE,
    password_hash       text NOT NULL,
    plan                text NOT NULL DEFAULT 'free',
    paddle_customer_id  text NOT NULL DEFAULT '',
    subscription_id     text NOT NULL DEFAULT '',
    subscription_status text NOT NULL DEFAULT '',
    paddle_updated_at   timestamptz,
    lifetime_plan           text NOT NULL DEFAULT '',
    lifetime_transaction_id text NOT NULL DEFAULT '',
    created_at          timestamptz NOT NULL DEFAULT now(),
    email_verified_at   timestamptz,
    is_admin            boolean NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS sessions (
    token_hash text PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS sessions_user_idx ON sessions(user_id);

CREATE TABLE IF NOT EXISTS oauth_identities (
    provider   text NOT NULL,
    subject    text NOT NULL,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (provider, subject)
);
CREATE INDEX IF NOT EXISTS oauth_identities_user_id ON oauth_identities (user_id);

CREATE TABLE IF NOT EXISTS email_tokens (
    token_hash text PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind       text NOT NULL CHECK (kind IN ('verify', 'reset')),
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS email_tokens_user_kind ON email_tokens (user_id, kind);

CREATE TABLE IF NOT EXISTS paddle_events (
    event_id    text PRIMARY KEY,
    event_type  text NOT NULL,
    received_at timestamptz NOT NULL DEFAULT now()
);
