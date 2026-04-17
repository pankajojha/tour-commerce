# Local PostgreSQL Development

This folder provides a Linux-friendly local PostgreSQL container for developing the `tour-agent` API services against a local database.

## What It Does

- Builds a PostgreSQL 15 image with `pg_cron` enabled so the existing schema can initialize successfully.
- Applies all SQL migrations from `infrastructure/database/migrations/` on first container startup.
- Adds the `passwordless_tokens` table required by the auth services for local development.
- Exposes PostgreSQL on `127.0.0.1:5432`.
- Persists database files on the host at `local/postgres/data/` (outside the container filesystem).

## Why The Local Database Uses `postgres`

The current `V001__initial_schema.sql` migration contains database-level statements that target the `postgres` database explicitly:

- `GRANT CONNECT ON DATABASE postgres ...`
- `ALTER DATABASE postgres ...`
- `COMMENT ON DATABASE postgres ...`

Using `POSTGRES_DB=postgres` keeps the local container aligned with the current migration set without editing production SQL.

## Start The Database

From the `tour-agent/` root:

```bash
docker compose -f local/postgres/docker-compose.yml up --build -d
```

## Stop The Database

```bash
docker compose -f local/postgres/docker-compose.yml down
```

To wipe local data and rebuild from scratch:

```bash
docker compose -f local/postgres/docker-compose.yml down
rm -rf local/postgres/data/*
```

## Follow Logs

```bash
docker compose -f local/postgres/docker-compose.yml logs -f postgres
```

## Connect With `psql`

```bash
docker compose -f local/postgres/docker-compose.yml exec postgres psql -U postgres -d postgres
```

## Local Service Environment

Use these values when running the backend services locally:

```bash
DB_HOST=127.0.0.1
DB_PORT=5432
DB_NAME=postgres
DB_USER=postgres
DB_PASSWORD=postgres
```

## Notes

- Migrations only run on the first initialization of a fresh data directory.
- If you add a new SQL migration and want Docker init to replay from scratch, stop the container and clear `local/postgres/data/*`.
- The image also creates the `passwordless_tokens` table locally because it is required by the auth services but is not yet present in the shared base migration.