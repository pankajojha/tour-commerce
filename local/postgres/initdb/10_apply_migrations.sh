#!/usr/bin/env bash
set -euo pipefail

shopt -s nullglob

for migration in /docker-entrypoint-initdb.d/migrations/V*.sql; do
    echo "Applying migration ${migration}"
    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -f "$migration"
done