package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

// Open returns a connected *sql.DB using env vars DB_HOST / DB_NAME / DB_USER / DB_PASSWORD.
func Open() (*sql.DB, error) {
	host := os.Getenv("DB_HOST")
	name := os.Getenv("DB_NAME")
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASSWORD")

	if host == "" || name == "" || user == "" || pass == "" {
		return nil, fmt.Errorf("DB_HOST, DB_NAME, DB_USER and DB_PASSWORD must all be set")
	}

	dsn := fmt.Sprintf("host=%s port=5432 user=%s password=%s dbname=%s sslmode=require",
		host, user, pass, name)

	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	if err = conn.Ping(); err != nil {
		return nil, fmt.Errorf("db ping: %w", err)
	}
	return conn, nil
}
