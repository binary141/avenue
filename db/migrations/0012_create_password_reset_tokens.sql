CREATE TABLE IF NOT EXISTS password_reset_tokens (
    token      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    expires_at TIMESTAMP NOT NULL DEFAULT (now() + INTERVAL '1 hour')
);
