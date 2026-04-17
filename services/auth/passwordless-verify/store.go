package main

import (
	"context"
	"database/sql"
	"time"
)

func getTokenRecord(ctx context.Context, tokenID string) (string, string, time.Time, int, error) {
	query := `
		SELECT user_email, token_hash, expires_at, attempts
		FROM passwordless_tokens
		WHERE token_id = $1
	`

	var userEmail, tokenHash string
	var expiresAt time.Time
	var attempts int
	err := db.QueryRowContext(ctx, query, tokenID).Scan(&userEmail, &tokenHash, &expiresAt, &attempts)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", time.Time{}, 0, sql.ErrNoRows
		}
		return "", "", time.Time{}, 0, err
	}

	return userEmail, tokenHash, expiresAt, attempts, nil
}

func incrementAttempts(ctx context.Context, tokenID string) {
	updateQuery := `UPDATE passwordless_tokens SET attempts = attempts + 1 WHERE token_id = $1`
	_, _ = db.ExecContext(ctx, updateQuery, tokenID)
}

func deleteToken(ctx context.Context, tokenID string) error {
	deleteQuery := `DELETE FROM passwordless_tokens WHERE token_id = $1`
	_, err := db.ExecContext(ctx, deleteQuery, tokenID)
	return err
}
