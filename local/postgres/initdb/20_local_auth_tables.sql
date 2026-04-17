CREATE TABLE IF NOT EXISTS passwordless_tokens (
    token_id UUID PRIMARY KEY,
    user_email VARCHAR(255) NOT NULL,
    token_hash VARCHAR(255) NOT NULL,
    token_type VARCHAR(20) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    attempts INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_passwordless_tokens_email
    ON passwordless_tokens(user_email);

CREATE INDEX IF NOT EXISTS idx_passwordless_tokens_expires
    ON passwordless_tokens(expires_at);