CREATE TABLE IF NOT EXISTS quiz_drafts (
    user_id   UUID        PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    data      JSONB       NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
