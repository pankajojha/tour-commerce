package main

import (
	"context"
	"fmt"
	"time"
)

func checkRateLimit(ctx context.Context, email string) error {
	query := `
		SELECT COUNT(*) FROM passwordless_tokens
		WHERE user_email = $1 AND created_at > NOW() - INTERVAL '15 minutes'
	`

	var count int
	err := db.QueryRowContext(ctx, query, email).Scan(&count)
	if err != nil {
		return err
	}

	if count >= 3 {
		return fmt.Errorf("rate limit exceeded")
	}

	return nil
}

func saveToken(ctx context.Context, tokenID, email, tokenHash, tokenType string, expiresAt time.Time) error {
	query := `
		INSERT INTO passwordless_tokens (token_id, user_email, token_hash, token_type, expires_at, attempts, created_at)
		VALUES ($1, $2, $3, $4, $5, 0, NOW())
	`
	_, err := db.ExecContext(ctx, query, tokenID, email, tokenHash, tokenType, expiresAt)
	return err
}
