-- What visitors tell us from the feedback box. The account is attached when they're signed in, and the row stays if the
-- account goes. Apply after 002. Safe to run twice.
CREATE TABLE IF NOT EXISTS feedback (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NULL REFERENCES users(id) ON DELETE SET NULL,
    email      text NOT NULL DEFAULT '',
    message    text NOT NULL,
    page       text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS feedback_created ON feedback (created_at);
CREATE INDEX IF NOT EXISTS feedback_user ON feedback (user_id);
