FROM postgres:15-bookworm

RUN apt-get update \
    && apt-get install -y --no-install-recommends postgresql-15-cron \
    && rm -rf /var/lib/apt/lists/*

COPY infrastructure/database/migrations /docker-entrypoint-initdb.d/migrations
COPY local/postgres/initdb/10_apply_migrations.sh /docker-entrypoint-initdb.d/10_apply_migrations.sh
COPY local/postgres/initdb/20_local_auth_tables.sql /docker-entrypoint-initdb.d/20_local_auth_tables.sql

RUN chmod +x /docker-entrypoint-initdb.d/10_apply_migrations.sh

EXPOSE 5432

CMD ["postgres", "-c", "shared_preload_libraries=pg_cron", "-c", "cron.database_name=postgres"]