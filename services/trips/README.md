# Trips Lifecycle Service

Go Lambda service that enforces trip lifecycle state transitions and operation permissions **driven entirely from the database** — no hardcoded state logic.

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/trips/{id}/operations?scope={scope}` | Return allowed operations for the trip's current status and caller scope |
| `POST` | `/trips/{id}/transition` | Validate and apply a status transition; appends to audit log |
| `GET` | `/trips/ui-hints/{status}` | Return UI directive row for a given trip status |

## Architecture

```
handler/main.go          — Lambda entry point, request routing
lifecycle/lifecycle.go   — DB-driven FSM helpers (CheckOperation, IsTransitionAllowed, GetUIHint, AppendStatusLog)
db/db.go                 — Connection pool helper (env-var driven)
```

## Database tables consumed

| Table | Purpose |
|-------|---------|
| `trip_status_transitions` | Defines legal from→to transitions and required actor scopes |
| `trip_operation_rules` | Per-status operation permission matrix |
| `trip_ui_hints` | Per-status UI directives loaded by the front-end |
| `trip_status_change_log` | Immutable append-only audit log |
| `trip_plans` | Status field read/updated on transition |

All rules are read at runtime — updating DB rows changes enforcement without a code deployment.

## Environment variables

| Variable | Description |
|----------|-------------|
| `DB_HOST` | PostgreSQL host |
| `DB_NAME` | Database name |
| `DB_USER` | Database user |
| `DB_PASSWORD` | Database password |

## Trip lifecycle (Option A — Strict Single-Use)

```
draft → active → shared → finalized → consumed
```

- `finalized` and `consumed` are permanently immutable (no edits, no reopen).
- Owner may abandon any non-consumed trip to `consumed` with a required reason.
- System can force `finalized → consumed` (post-booking webhook).

## Build

```bash
make build    # produces dist/bootstrap (linux/arm64)
make zip      # produces dist/trips.zip
make deploy   # updates Lambda function code (set FUNCTION_NAME env var)
```
