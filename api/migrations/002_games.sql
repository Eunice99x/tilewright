ALTER TABLE users ADD COLUMN IF NOT EXISTS display_name text NOT NULL DEFAULT '';
CREATE TABLE IF NOT EXISTS games (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mode        text NOT NULL DEFAULT 'solo' CHECK (mode IN ('solo', 'versus')),
    state       jsonb NOT NULL,
    score       integer NOT NULL DEFAULT 0,
    over        boolean NOT NULL DEFAULT false,
    invite_code text UNIQUE,
    version     integer NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS games_solo_best ON games (user_id, score DESC) WHERE over AND mode = 'solo';
CREATE UNIQUE INDEX IF NOT EXISTS games_one_solo ON games (user_id) WHERE NOT over AND mode = 'solo';

CREATE TABLE IF NOT EXISTS game_players (
    game_id uuid NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    seat    integer NOT NULL,
    PRIMARY KEY (game_id, seat),
    UNIQUE (game_id, user_id)
);
CREATE INDEX IF NOT EXISTS game_players_user ON game_players (user_id);
